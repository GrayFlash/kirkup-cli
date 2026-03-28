package retro

import (
	"context"
	"sort"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
)

// Summary holds all aggregated data for a retrospective period.
type Summary struct {
	From          time.Time
	To            time.Time
	Projects      []ProjectStat
	Categories    []CategoryStat
	Agents        []AgentStat
	Daily         []DayStat
	TotalPrompts  int
	TotalSessions int
	TotalEstTime  time.Duration
}

type ProjectStat struct {
	Name     string
	Sessions int
	Prompts  int
	EstTime  time.Duration
	Branches []BranchStat
}

type BranchStat struct {
	Name    string
	Prompts int
}

type CategoryStat struct {
	Category string
	Count    int
	Percent  float64
}

type AgentStat struct {
	Agent   string
	Count   int
	Percent float64
}

type DayStat struct {
	Date            time.Time
	Prompts         int
	ContextSwitches int
}

// Aggregate queries the store for the given time range and builds a Summary.
// project filters to a single project when non-empty.
func Aggregate(ctx context.Context, s store.Store, from, to time.Time, project string, gapMinutes int) (*Summary, error) {
	f := store.EventFilter{Since: &from, Until: &to, Project: project}
	events, err := s.QueryPromptEvents(ctx, f)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return &Summary{From: from, To: to}, nil
	}

	// Collect event IDs for classification lookup.
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	classifications, err := s.QueryClassifications(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Map eventID -> categories.
	catByEvent := make(map[string][]string, len(classifications))
	for _, c := range classifications {
		catByEvent[c.PromptEventID] = append(catByEvent[c.PromptEventID], c.Category)
	}

	// Infer sessions from events (query time, not ingestion time).
	sessions := inferSessions(events, gapMinutes)

	sum := &Summary{
		From:          from,
		To:            to,
		TotalPrompts:  len(events),
		TotalSessions: len(sessions),
	}

	// -- Projects --
	projectPrompts := make(map[string]int)
	projectSessions := make(map[string]int)
	projectTime := make(map[string]time.Duration)
	projectBranches := make(map[string]map[string]int)
	for _, e := range events {
		projectPrompts[e.Project]++
		if e.GitBranch != "" {
			if projectBranches[e.Project] == nil {
				projectBranches[e.Project] = make(map[string]int)
			}
			projectBranches[e.Project][e.GitBranch]++
		}
	}
	for _, sess := range sessions {
		projectSessions[sess.Project]++
		dur := sess.EndedAt.Sub(sess.StartedAt)
		if dur < 5*time.Minute {
			dur = 5 * time.Minute // minimum session estimate
		}
		projectTime[sess.Project] += dur
		sum.TotalEstTime += dur
	}
	for name, count := range projectPrompts {
		var branches []BranchStat
		for b, n := range projectBranches[name] {
			branches = append(branches, BranchStat{Name: b, Prompts: n})
		}
		sort.Slice(branches, func(i, j int) bool {
			return branches[i].Prompts > branches[j].Prompts
		})
		sum.Projects = append(sum.Projects, ProjectStat{
			Name:     name,
			Prompts:  count,
			Sessions: projectSessions[name],
			EstTime:  projectTime[name],
			Branches: branches,
		})
	}
	sort.Slice(sum.Projects, func(i, j int) bool {
		return sum.Projects[i].Prompts > sum.Projects[j].Prompts
	})

	// -- Categories --
	catCounts := make(map[string]int)
	for _, cats := range catByEvent {
		for _, cat := range cats {
			catCounts[cat]++
		}
	}
	totalCat := 0
	for _, n := range catCounts {
		totalCat += n
	}
	for cat, n := range catCounts {
		pct := 0.0
		if totalCat > 0 {
			pct = float64(n) / float64(totalCat) * 100
		}
		sum.Categories = append(sum.Categories, CategoryStat{Category: cat, Count: n, Percent: pct})
	}
	sort.Slice(sum.Categories, func(i, j int) bool {
		return sum.Categories[i].Count > sum.Categories[j].Count
	})

	// -- Agents --
	agentCounts := make(map[string]int)
	for _, e := range events {
		agentCounts[e.Agent]++
	}
	for ag, n := range agentCounts {
		pct := float64(n) / float64(len(events)) * 100
		sum.Agents = append(sum.Agents, AgentStat{Agent: ag, Count: n, Percent: pct})
	}
	sort.Slice(sum.Agents, func(i, j int) bool {
		return sum.Agents[i].Count > sum.Agents[j].Count
	})

	// -- Daily activity + context switches --
	sum.Daily = dailyStats(events, from, to)

	return sum, nil
}

// inferSessions groups events into sessions by (project, agent) with a gap
// inferSessions groups events into sessions separated by more than the gap
// threshold. Events are sorted by timestamp within each group.
func inferSessions(events []models.PromptEvent, gapMinutes int) []models.Session {
	type key struct{ project, agent, branch string }
	groups := make(map[key][]models.PromptEvent)
	for _, e := range events {
		k := key{e.Project, e.Agent, e.GitBranch}
		groups[k] = append(groups[k], e)
	}

	gap := time.Duration(gapMinutes) * time.Minute
	var sessions []models.Session

	for k, group := range groups {
		sort.Slice(group, func(i, j int) bool {
			return group[i].Timestamp.Before(group[j].Timestamp)
		})

		start := group[0].Timestamp
		prev := group[0].Timestamp
		count := 1

		for _, e := range group[1:] {
			if e.Timestamp.Sub(prev) > gap {
				// End current session
				ended := prev
				// If session is very short, assume at least 5 mins of work.
				if ended.Sub(start) < 5*time.Minute {
					ended = start.Add(5 * time.Minute)
				}

				sessions = append(sessions, models.Session{
					Project:             k.project,
					Agent:               k.agent,
					StartedAt:           start,
					EndedAt:             ended,
					PromptCount:         count,
					GapThresholdMinutes: gapMinutes,
				})
				start = e.Timestamp
				count = 0
			}
			prev = e.Timestamp
			count++
		}

		// Final session
		ended := prev
		if ended.Sub(start) < 5*time.Minute {
			ended = start.Add(5 * time.Minute)
		}
		sessions = append(sessions, models.Session{
			Project:             k.project,
			Agent:               k.agent,
			StartedAt:           start,
			EndedAt:             ended,
			PromptCount:         count,
			GapThresholdMinutes: gapMinutes,
		})
	}

	return sessions
}

// dailyStats builds per-day stats sorted chronologically over the range.
func dailyStats(events []models.PromptEvent, from, to time.Time) []DayStat {
	type dayKey string
	dayOf := func(t time.Time) dayKey {
		y, m, d := t.Local().Date()
		return dayKey(time.Date(y, m, d, 0, 0, 0, 0, time.Local).Format("2006-01-02"))
	}
	dateOf := func(t time.Time) time.Time {
		y, m, d := t.Local().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	}

	// Sort events by timestamp ascending.
	sorted := make([]models.PromptEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	type dayData struct {
		date   time.Time
		count  int
		events []models.PromptEvent
	}
	days := make(map[dayKey]*dayData)
	for _, e := range sorted {
		k := dayOf(e.Timestamp)
		if days[k] == nil {
			days[k] = &dayData{date: dateOf(e.Timestamp)}
		}
		days[k].count++
		days[k].events = append(days[k].events, e)
	}

	// Enumerate each day in the range.
	var stats []DayStat
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		k := dayKey(d.Local().Format("2006-01-02"))
		data := days[k]
		if data == nil {
			continue
		}
		switches := countContextSwitches(data.events)
		stats = append(stats, DayStat{
			Date:            data.date,
			Prompts:         data.count,
			ContextSwitches: switches,
		})
	}
	return stats
}

// countContextSwitches counts the number of times the project changes in a
// sorted-by-time slice of events.
func countContextSwitches(events []models.PromptEvent) int {
	if len(events) == 0 {
		return 0
	}
	switches := 0
	prev := events[0].Project
	for _, e := range events[1:] {
		if e.Project != prev {
			switches++
			prev = e.Project
		}
	}
	return switches
}
