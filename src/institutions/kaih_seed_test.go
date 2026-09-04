package institutions

import (
	"testing"

	"clasenna-go-backend/libs/models"
	"github.com/google/uuid"
)

func TestDefaultKAIHActivities(t *testing.T) {
	institutionID := uuid.New()
	category, items := defaultKAIHActivities(institutionID)

	if category.Name != "7 KAIH" || category.InstitutionID == nil || *category.InstitutionID != institutionID {
		t.Fatalf("unexpected category: %#v", category)
	}
	expectedNames := []string{
		"Bangun Pagi", "Beribadah", "Bermasyarakat", "Berolahraga",
		"Gemar Belajar", "Makan Sehat dan Bergizi", "Tidur Cepat",
	}
	if len(items) != len(expectedNames) {
		t.Fatalf("expected %d items, got %d", len(expectedNames), len(items))
	}
	for index, item := range items {
		if item.Name != expectedNames[index] {
			t.Errorf("item %d name = %q, want %q", index, item.Name, expectedNames[index])
		}
		if item.InstitutionID == nil || *item.InstitutionID != institutionID {
			t.Errorf("item %q has an invalid institution", item.Name)
		}
		if item.Type != models.ActivityItemTypePositive || item.PointValue != 1 {
			t.Errorf("item %q has invalid point configuration", item.Name)
		}
		if item.DailyLimit != 1 || item.PeriodLimit != 0 || item.PeriodType != models.ActivityLimitNone {
			t.Errorf("item %q has invalid limit configuration", item.Name)
		}
	}
}
