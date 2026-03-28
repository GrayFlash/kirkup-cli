package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/GrayFlash/kirkup-cli/models"
)

func TestInsertClassification_FKViolation(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	
	// Classification for non-existent event ID
	c := &models.Classification{
		PromptEventID: "non-existent",
		Category:      "coding",
		Confidence:    1.0,
		Classifier:    "test",
	}

	err := s.InsertClassification(ctx, c)
	if err == nil {
		t.Fatal("expected error for foreign key violation, got nil")
	}
	if !strings.Contains(err.Error(), "FOREIGN KEY") {
		t.Errorf("expected foreign key error, got %v", err)
	}
}

func setupTestDB(t *testing.T) (*Store, func()) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s, func() { _ = s.Close() }
}
