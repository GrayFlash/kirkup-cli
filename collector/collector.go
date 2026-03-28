package collector

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"


	"github.com/fsnotify/fsnotify"

	"github.com/GrayFlash/kirkup-cli/agent"
	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
)

// Collector watches agent log files and writes new prompt events to the store.
type Collector struct {
	agents       *agent.Registry
	store        store.Store
	cfg          *config.Config
	log          *slog.Logger
	seen         map[string]struct{}
	seenProjects map[string]struct{}
	mu           sync.Mutex
	cancel       context.CancelFunc
	done         chan struct{}
}

// New creates a Collector. Call Start to begin watching.
func New(agents *agent.Registry, s store.Store, cfg *config.Config, log *slog.Logger) *Collector {
	if log == nil {
		log = slog.Default()
	}
	return &Collector{
		agents:       agents,
		store:        s,
		cfg:          cfg,
		log:          log,
		seen:         make(map[string]struct{}),
		seenProjects: make(map[string]struct{}),
		done:         make(chan struct{}),
	}
}

// Start performs an initial scan of all agent files, then watches for changes.
// It blocks until ctx is cancelled or Stop is called.
func (c *Collector) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	// Collect glob patterns from all registered adapters and watch their
	// parent directories.
	globs := c.collectGlobs()
	watchDirs := uniqueDirs(globs)
	for _, dir := range watchDirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		if err := watcher.Add(dir); err != nil {
			c.log.Warn("cannot watch dir", "dir", dir, "err", err)
		}
	}

	c.syncConfigProjects(ctx)

	// Initial scan.
	c.scanAll(ctx, globs)

	poll := time.Duration(c.cfg.Daemon.PollIntervalSeconds) * time.Second
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	defer close(c.done)

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				c.processMatchingFile(ctx, event.Name, globs)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			c.log.Warn("watcher error", "err", err)

		case <-ticker.C:
			// Periodic poll as fallback for missed fsnotify events.
			c.scanAll(ctx, globs)
		}
	}
}

// Stop signals the collector to shut down and waits for it to finish.
func (c *Collector) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	<-c.done
}

// scanAll expands all globs and processes each matching file.
func (c *Collector) scanAll(ctx context.Context, globs []globEntry) {
	for _, g := range globs {
		matches, err := filepath.Glob(g.pattern)
		if err != nil || len(matches) == 0 {
			continue
		}
		for _, path := range matches {
			c.processFile(ctx, g.adapter, path)
		}
	}
}

// processMatchingFile checks whether path matches any registered glob and, if
// so, processes it with the corresponding adapter.
func (c *Collector) processMatchingFile(ctx context.Context, path string, globs []globEntry) {
	// Normalise to the OS separator so filepath.Match works correctly on
	// Windows where fsnotify may return paths with either separator.
	path = filepath.FromSlash(path)
	for _, g := range globs {
		pattern := filepath.FromSlash(g.pattern)
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			c.processFile(ctx, g.adapter, path)
			return
		}
	}
}

// processFile reads all events from path via the adapter, enriches them, and
// stores any that have not been seen before.
func (c *Collector) processFile(ctx context.Context, a agent.Adapter, path string) {
	events, err := a.Events(ctx, path)
	if err != nil {
		c.log.Debug("parse error", "agent", a.Name(), "path", path, "err", err)
		return
	}

	for i := range events {
		e := &events[i]

		// Deterministic ID for deduplication.
		e.ID = eventID(e)

		c.mu.Lock()
		_, already := c.seen[e.ID]
		if !already {
			c.seen[e.ID] = struct{}{}
		}
		c.mu.Unlock()

		if already {
			continue
		}

		// Enrich with git context if we have a working directory.
		if e.WorkingDir != "" && (e.GitRemote == "" || e.GitBranch == "") {
			gi := GitContext(e.WorkingDir)
			if e.GitRemote == "" {
				e.GitRemote = gi.Remote
			}
			if e.GitBranch == "" {
				e.GitBranch = gi.Branch
			}
		}

		// Resolve project name.
		if e.Project == "" {
			e.Project = ResolveProject(c.cfg.Projects, e.GitRemote, e.WorkingDir)
		}

		if err := c.store.InsertPromptEvent(ctx, e); err != nil {
			c.log.Error("store insert", "err", err)
		} else {
			c.log.Debug("stored event",
				"agent", e.Agent,
				"project", e.Project,
				"prompt_prefix", truncate(e.Prompt, 60),
			)
			if e.Project != "" {
				c.ensureProject(ctx, e)
			}
		}
	}
}

// collectGlobs returns all (adapter, glob pattern) pairs from registered agents.
// If the config specifies log_paths for an agent, those override the adapter defaults.
func (c *Collector) collectGlobs() []globEntry {
	var entries []globEntry
	for _, a := range c.agents.All() {
		globs := a.WatchGlobs()
		if cfg, ok := c.cfg.Agents[a.Name()]; ok && len(cfg.LogPaths) > 0 {
			globs = cfg.LogPaths
		}
		for _, g := range globs {
			entries = append(entries, globEntry{adapter: a, pattern: g})
		}
	}
	return entries
}

// syncConfigProjects upserts projects defined in the config file so that
// ListProjects returns them even before any events are collected.
func (c *Collector) syncConfigProjects(ctx context.Context) {
	for _, p := range c.cfg.Projects {
		proj := &models.Project{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Paths:       p.Match.Paths,
		}
		if p.Match.GitRemote != "" {
			proj.GitRemotes = []string{p.Match.GitRemote}
		}
		if err := c.store.UpsertProject(ctx, proj); err != nil {
			c.log.Warn("upsert config project", "name", p.Name, "err", err)
		}
		c.mu.Lock()
		c.seenProjects[p.Name] = struct{}{}
		c.mu.Unlock()
	}
}

// ensureProject persists a project record the first time the collector
// encounters a new project name from an event.
func (c *Collector) ensureProject(ctx context.Context, e *models.PromptEvent) {
	c.mu.Lock()
	_, known := c.seenProjects[e.Project]
	if !known {
		c.seenProjects[e.Project] = struct{}{}
	}
	c.mu.Unlock()

	if known {
		return
	}

	proj := &models.Project{Name: e.Project}
	if e.GitRemote != "" {
		proj.GitRemotes = []string{e.GitRemote}
	}
	if e.WorkingDir != "" {
		proj.Paths = []string{e.WorkingDir}
	}
	if err := c.store.UpsertProject(ctx, proj); err != nil {
		c.log.Warn("upsert discovered project", "name", e.Project, "err", err)
	}
}

type globEntry struct {
	adapter agent.Adapter
	pattern string
}

// uniqueDirs returns the unique watch directories derived from a set of glob
// patterns (the longest non-wildcard prefix of each pattern).
func uniqueDirs(globs []globEntry) []string {
	seen := make(map[string]struct{})
	var dirs []string
	for _, g := range globs {
		dir := globWatchDir(g.pattern)
		if _, ok := seen[dir]; !ok {
			seen[dir] = struct{}{}
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// globWatchDir returns the deepest non-wildcard directory component of a glob
// pattern, which is the directory we should hand to fsnotify.
func globWatchDir(pattern string) string {
	for i, ch := range pattern {
		if ch == '*' || ch == '?' || ch == '[' {
			return filepath.Dir(pattern[:i])
		}
	}
	return filepath.Dir(pattern)
}

// eventID returns a deterministic ID for an event based on its content so that
// re-reading the same file never produces duplicate rows.
func eventID(e *models.PromptEvent) string {
	h := sha256.Sum256([]byte(e.Agent + "|" + e.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + e.Prompt))
	return fmt.Sprintf("%x", h[:16])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

