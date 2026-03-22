package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/classifier"
	"github.com/GrayFlash/kirkup-cli/store"
)

var (
	classifyReclassify bool
)

var classifyCmd = &cobra.Command{
	Use:   "classify",
	Short: "Run the rule classifier on unclassified prompt events",
	RunE:  runClassify,
}

func init() {
	classifyCmd.Flags().BoolVar(&classifyReclassify, "reclassify", false, "Re-classify all events, not just unclassified ones")
	rootCmd.AddCommand(classifyCmd)
}

func runClassify(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	ctx := context.Background()

	rc := classifier.NewRuleClassifier()

	if classifyReclassify {
		all, err := s.QueryPromptEvents(ctx, store.EventFilter{})
		if err != nil {
			return fmt.Errorf("query events: %w", err)
		}
		classifications, err := rc.Classify(ctx, all)
		if err != nil {
			return err
		}
		inserted := 0
		for i := range classifications {
			if err := s.InsertClassification(ctx, &classifications[i]); err == nil {
				inserted++
			}
		}
		fmt.Printf("reclassified %d / %d events\n", inserted, len(all))
		return nil
	}

	unclassified, err := s.GetUnclassified(ctx, 0)
	if err != nil {
		return fmt.Errorf("query unclassified events: %w", err)
	}

	if len(unclassified) == 0 {
		fmt.Println("nothing to classify")
		return nil
	}

	classifications, err := rc.Classify(ctx, unclassified)
	if err != nil {
		return err
	}

	inserted := 0
	for i := range classifications {
		if err := s.InsertClassification(ctx, &classifications[i]); err == nil {
			inserted++
		}
	}

	skipped := len(unclassified) - len(classifications)
	fmt.Printf("classified %d events", inserted)
	if skipped > 0 {
		fmt.Printf(" (%d did not match any rule)", skipped)
	}
	fmt.Println()
	return nil
}
