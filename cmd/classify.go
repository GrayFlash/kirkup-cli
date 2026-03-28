package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/classifier"
	"github.com/GrayFlash/kirkup-cli/store"
)

var (
	classifyReclassify  bool
	classifyMode        string
	classifyReconfigure bool
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
		return openInEditor(cfgPath)
	}

	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	mode := cfg.Classifier.Mode
	if classifyMode != "" {
		mode = classifyMode
	}

	var cl classifier.Classifier
	switch mode {
	case "llm":
		cl = classifier.NewLLMClassifier(cfg.Classifier.LLM)
	case "rules", "":
		rc := classifier.NewRuleClassifier()
		for _, r := range cfg.Classifier.CustomRules {
			rc.AddRule(r.Category, r.Keywords, r.Patterns, r.Priority)
		}
		cl = rc
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}

	ctx := context.Background()

	if classifyReclassify {
		all, err := s.QueryPromptEvents(ctx, store.EventFilter{})
		if err != nil {
			return fmt.Errorf("query events: %w", err)
		}
		fmt.Printf("classifying %d events using %s...\n", len(all), cl.Name())
		classifications, err := cl.Classify(ctx, all)
		if err != nil {
			return err
		}
		inserted := 0
		errs := 0
		for i := range classifications {
			if err := s.InsertClassification(ctx, &classifications[i]); err == nil {
				inserted++
			} else {
				errs++
			}
		}
		fmt.Printf("reclassified %d / %d events\n", inserted, len(all))
		if errs > 0 {
			fmt.Printf("warning: failed to insert %d classifications\n", errs)
		}
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

	fmt.Printf("classifying %d events using %s...\n", len(unclassified), cl.Name())
	classifications, err := cl.Classify(ctx, unclassified)
	if err != nil {
		return err
	}

	inserted := 0
	errs := 0
	for i := range classifications {
		if err := s.InsertClassification(ctx, &classifications[i]); err == nil {
			inserted++
		} else {
			errs++
		}
	}

	skipped := len(unclassified) - len(classifications)
	fmt.Printf("classified %d events", inserted)
	if skipped > 0 {
		fmt.Printf(" (%d did not match any rule)", skipped)
	}
	fmt.Println()
	if errs > 0 {
		fmt.Printf("warning: failed to insert %d classifications\n", errs)
	}
	return nil
}
