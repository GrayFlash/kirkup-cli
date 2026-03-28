package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/classifier"
	"github.com/GrayFlash/kirkup-cli/store"
)

var (
	classifyReclassify   bool
	classifyMode         string
	classifyReconfigure  bool
)

var classifyCmd = &cobra.Command{
	Use:   "classify",
	Short: "Run the rule classifier on unclassified prompt events",
	RunE:  runClassify,
}

func init() {
	classifyCmd.Flags().BoolVar(&classifyReclassify, "reclassify", false, "Re-classify all events, not just unclassified ones")
	classifyCmd.Flags().StringVar(&classifyMode, "mode", "rules", "Classifier to use: rules")
	classifyCmd.Flags().BoolVar(&classifyReconfigure, "reconfigure", false, "Open the classifier config in $EDITOR")
	rootCmd.AddCommand(classifyCmd)
}

func runClassify(_ *cobra.Command, _ []string) error {
	if classifyReconfigure {
		cfgPath, err := defaultConfigPath()
		if err != nil {
			return err
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			return fmt.Errorf("$EDITOR is not set; open %s manually", cfgPath)
		}
		cmd := exec.Command(editor, cfgPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if classifyMode != "rules" {
		return fmt.Errorf("unsupported mode %q: only \"rules\" is available in this version", classifyMode)
	}

	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	rc := classifier.NewRuleClassifier()
	for _, r := range cfg.Classifier.CustomRules {
		rc.AddRule(r.Category, r.Keywords, r.Patterns, r.Priority)
	}

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
