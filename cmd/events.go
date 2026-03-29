package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/store"
)

var (
	eventsToday bool
	eventsTail  bool
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show raw prompt events (JSON or tail mode)",
	RunE:  runEvents,
}

func init() {
	eventsCmd.Flags().BoolVar(&eventsToday, "today", false, "Show events from today")
	eventsCmd.Flags().BoolVar(&eventsTail, "tail", false, "Tail new events as they are recorded")
	rootCmd.AddCommand(eventsCmd)
}

func runEvents(_ *cobra.Command, _ []string) error {
	_, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	if eventsTail {
		return tailEvents(ctx, s)
	}

	filter := store.EventFilter{}
	if eventsToday {
		midnight := today()
		filter.Since = &midnight
	}

	events, err := s.QueryPromptEvents(ctx, filter)
	if err != nil {
		return fmt.Errorf("query events: %w", err)
	}

	for _, e := range events {
		printEvent(e.Timestamp, e.Agent, e.Project, e.Prompt)
	}

	return nil
}

func tailEvents(ctx context.Context, s store.Store) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	fmt.Println("tailing events (ctrl+c to stop)...")

	last := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			events, err := s.QueryPromptEvents(ctx, store.EventFilter{Since: &last})
			if err != nil {
				return err
			}
			if len(events) > 0 {
				for i := len(events) - 1; i >= 0; i-- {
					e := events[i]
					printEvent(e.Timestamp, e.Agent, e.Project, e.Prompt)
				}
				last = events[0].Timestamp.Add(time.Nanosecond)
			}
		}
	}
}

func printEvent(ts time.Time, agent, project, prompt string) {
	proj := project
	if proj == "" {
		proj = "-"
	}
	truncated := truncateStr(prompt, 80)
	fmt.Printf("%s  %-12s  %-16s  %s\n",
		ts.Local().Format("2006-01-02 15:04:05"),
		agent, proj, truncated,
	)
}
