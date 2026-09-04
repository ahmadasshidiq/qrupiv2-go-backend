package activities

import (
	"testing"
	"time"

	"clasenna-go-backend/libs/models"
	"github.com/google/uuid"
)

func TestBulkLimitRejectsSecondActivityOnSameDay(t *testing.T) {
	itemID, userID := uuid.New(), uuid.New()
	occurredAt := time.Date(2026, time.August, 10, 8, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	entries := []parsedBulkEntry{
		{itemID: itemID, userID: userID, occurredAt: occurredAt},
		{itemID: itemID, userID: userID, occurredAt: occurredAt.Add(2 * time.Hour)},
	}
	items := map[uuid.UUID]models.ActivityItem{
		itemID: {ID: itemID, DailyLimit: 1, PeriodLimit: 5, PeriodType: models.ActivityLimitMonthly},
	}

	entryBuckets, buckets, err := buildLimitBuckets(entries, items)
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int64)
	if exceeded := consumeLimitBuckets(counts, entryBuckets[0], buckets); exceeded != nil {
		t.Fatalf("first activity unexpectedly exceeded %s", exceeded.field)
	}
	if exceeded := consumeLimitBuckets(counts, entryBuckets[1], buckets); exceeded == nil || exceeded.field != "daily_limit" {
		t.Fatalf("expected daily_limit error, got %#v", exceeded)
	}
}

func TestBulkLimitRejectsActivityAfterMonthlyLimit(t *testing.T) {
	itemID, userID := uuid.New(), uuid.New()
	start := time.Date(2026, time.August, 1, 8, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	entries := make([]parsedBulkEntry, 6)
	for index := range entries {
		entries[index] = parsedBulkEntry{itemID: itemID, userID: userID, occurredAt: start.AddDate(0, 0, index)}
	}
	items := map[uuid.UUID]models.ActivityItem{
		itemID: {ID: itemID, DailyLimit: 1, PeriodLimit: 5, PeriodType: models.ActivityLimitMonthly},
	}

	entryBuckets, buckets, err := buildLimitBuckets(entries, items)
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int64)
	for index := 0; index < 5; index++ {
		if exceeded := consumeLimitBuckets(counts, entryBuckets[index], buckets); exceeded != nil {
			t.Fatalf("activity %d unexpectedly exceeded %s", index, exceeded.field)
		}
	}
	if exceeded := consumeLimitBuckets(counts, entryBuckets[5], buckets); exceeded == nil || exceeded.field != "period_limit" {
		t.Fatalf("expected period_limit error, got %#v", exceeded)
	}
}
