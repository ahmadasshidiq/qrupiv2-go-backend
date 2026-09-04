package activities

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"clasenna-go-backend/libs/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BulkValidationError struct {
	Index   int    `json:"index"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *BulkValidationError) Error() string {
	if e.Index < 0 {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("activities[%d].%s: %s", e.Index, e.Field, e.Message)
}

type parsedBulkEntry struct {
	itemID      uuid.UUID
	userID      uuid.UUID
	description string
	pointValue  *int
	occurredAt  time.Time
}

type activityLimitBucket struct {
	key     string
	userID  uuid.UUID
	itemID  uuid.UUID
	start   *time.Time
	end     *time.Time
	limit   int
	field   string
	message string
}

type activityLimitCount struct {
	Key   string `gorm:"column:bucket_key"`
	Count int64  `gorm:"column:activity_count"`
}

func (s *ActivityService) createBulk(dto BulkCreateDTO) (*BulkCreateResult, error) {
	if len(dto.Activities) == 0 || len(dto.Activities) > MaxBulkActivities {
		return nil, &BulkValidationError{Index: -1, Field: "activities", Message: fmt.Sprintf("must contain between 1 and %d records", MaxBulkActivities)}
	}

	recordedUserID, err := uuid.Parse(dto.RecordedUserID)
	if err != nil {
		return nil, &BulkValidationError{Index: -1, Field: "recorded_user_id", Message: "invalid UUID format"}
	}
	learningGroupID, err := parseOptionalUUID(dto.LearningGroupID)
	if err != nil {
		return nil, &BulkValidationError{Index: -1, Field: "learning_group_id", Message: "invalid UUID format"}
	}

	entries := make([]parsedBulkEntry, len(dto.Activities))
	itemIDs := make(map[uuid.UUID]struct{})
	userIDs := map[uuid.UUID]struct{}{recordedUserID: {}}
	for index, entry := range dto.Activities {
		itemID, err := uuid.Parse(entry.ActivityItemID)
		if err != nil {
			return nil, bulkEntryError(index, "activity_item_id", "invalid UUID format")
		}
		userID, err := uuid.Parse(entry.UserID)
		if err != nil {
			return nil, bulkEntryError(index, "user_id", "invalid UUID format")
		}
		entries[index] = parsedBulkEntry{
			itemID: itemID, userID: userID, description: entry.Description,
			pointValue: entry.PointValue, occurredAt: entry.OccurredAt,
		}
		itemIDs[itemID] = struct{}{}
		userIDs[userID] = struct{}{}
	}

	var created []models.Activity
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := validateBulkReferences(tx, entries, recordedUserID, learningGroupID, userIDs); err != nil {
			return err
		}

		items, err := lockBulkItems(tx, itemIDs)
		if err != nil {
			return err
		}
		for index, entry := range entries {
			if _, found := items[entry.itemID]; !found {
				return bulkEntryError(index, "activity_item_id", "activity item not found")
			}
		}
		bucketsByEntry, buckets, err := buildLimitBuckets(entries, items)
		if err != nil {
			return err
		}
		counts, err := loadLimitCounts(tx, buckets)
		if err != nil {
			return err
		}

		created = make([]models.Activity, 0, len(entries))
		for index, entry := range entries {
			if exceeded := consumeLimitBuckets(counts, bucketsByEntry[index], buckets); exceeded != nil {
				return bulkEntryError(index, exceeded.field, exceeded.message)
			}

			pointValue := items[entry.itemID].PointValue
			if entry.pointValue != nil {
				pointValue = *entry.pointValue
			}
			created = append(created, models.Activity{
				ActivityItemID:  entry.itemID,
				UserID:          entry.userID,
				LearningGroupID: learningGroupID,
				RecordedUserID:  recordedUserID,
				Description:     entry.description,
				PointValue:      pointValue,
				Platform:        dto.Platform,
				OccurredAt:      entry.occurredAt,
			})
		}
		return tx.CreateInBatches(&created, 500).Error
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(created))
	for index := range created {
		ids[index] = created[index].ID.String()
	}
	return &BulkCreateResult{Count: len(created), IDs: ids}, nil
}

func consumeLimitBuckets(counts map[string]int64, keys []string, buckets map[string]activityLimitBucket) *activityLimitBucket {
	for _, key := range keys {
		bucket := buckets[key]
		if counts[key] >= int64(bucket.limit) {
			return &bucket
		}
	}
	for _, key := range keys {
		counts[key]++
	}
	return nil
}

func validateBulkReferences(tx *gorm.DB, entries []parsedBulkEntry, recordedUserID uuid.UUID, learningGroupID *uuid.UUID, userIDs map[uuid.UUID]struct{}) error {
	if learningGroupID != nil {
		var count int64
		if err := tx.Model(&models.LearningGroup{}).Where("id = ?", *learningGroupID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return &BulkValidationError{Index: -1, Field: "learning_group_id", Message: "learning group not found"}
		}
	}

	userIDList := mapUUIDKeys(userIDs)
	var foundUserIDs []uuid.UUID
	if err := tx.Model(&models.User{}).Where("id IN ?", userIDList).Pluck("id", &foundUserIDs).Error; err != nil {
		return err
	}
	foundUsers := uuidSet(foundUserIDs)
	if _, found := foundUsers[recordedUserID]; !found {
		return &BulkValidationError{Index: -1, Field: "recorded_user_id", Message: "user not found"}
	}
	for index, entry := range entries {
		if _, found := foundUsers[entry.userID]; !found {
			return bulkEntryError(index, "user_id", "user not found")
		}
	}

	return nil
}

func lockBulkItems(tx *gorm.DB, ids map[uuid.UUID]struct{}) (map[uuid.UUID]models.ActivityItem, error) {
	idList := mapUUIDKeys(ids)
	sort.Slice(idList, func(i, j int) bool { return idList[i].String() < idList[j].String() })
	var items []models.ActivityItem
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", idList).Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	itemMap := make(map[uuid.UUID]models.ActivityItem, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}
	return itemMap, nil
}

func buildLimitBuckets(entries []parsedBulkEntry, items map[uuid.UUID]models.ActivityItem) ([][]string, map[string]activityLimitBucket, error) {
	bucketsByEntry := make([][]string, len(entries))
	buckets := make(map[string]activityLimitBucket)
	for index, entry := range entries {
		item := items[entry.itemID]
		if item.DailyLimit > 0 {
			start := startOfDay(entry.occurredAt)
			end := start.AddDate(0, 0, 1)
			bucket := newLimitBucket(entry.userID, entry.itemID, &start, &end, item.DailyLimit, "daily_limit", fmt.Sprintf("maximum %d activities per day has been reached", item.DailyLimit))
			buckets[bucket.key] = bucket
			bucketsByEntry[index] = append(bucketsByEntry[index], bucket.key)
		}
		if item.PeriodLimit > 0 && item.PeriodType != "" && item.PeriodType != models.ActivityLimitNone {
			start, end, hasPeriod, err := activityLimitPeriod(item.PeriodType, entry.occurredAt)
			if err != nil {
				return nil, nil, bulkEntryError(index, "period_type", err.Error())
			}
			var startPointer, endPointer *time.Time
			if hasPeriod {
				startPointer, endPointer = &start, &end
			}
			bucket := newLimitBucket(entry.userID, entry.itemID, startPointer, endPointer, item.PeriodLimit, "period_limit", fmt.Sprintf("maximum %d activities per %s has been reached", item.PeriodLimit, item.PeriodType))
			buckets[bucket.key] = bucket
			bucketsByEntry[index] = append(bucketsByEntry[index], bucket.key)
		}
	}
	return bucketsByEntry, buckets, nil
}

func newLimitBucket(userID, itemID uuid.UUID, start, end *time.Time, limit int, field, message string) activityLimitBucket {
	startKey, endKey := "lifetime", "lifetime"
	if start != nil {
		startKey = start.Format(time.RFC3339Nano)
		endKey = end.Format(time.RFC3339Nano)
	}
	key := strings.Join([]string{userID.String(), itemID.String(), startKey, endKey}, "|")
	return activityLimitBucket{key: key, userID: userID, itemID: itemID, start: start, end: end, limit: limit, field: field, message: message}
}

func loadLimitCounts(tx *gorm.DB, buckets map[string]activityLimitBucket) (map[string]int64, error) {
	counts := make(map[string]int64, len(buckets))
	if len(buckets) == 0 {
		return counts, nil
	}
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for start := 0; start < len(keys); start += 1000 {
		end := start + 1000
		if end > len(keys) {
			end = len(keys)
		}
		if err := loadLimitCountChunk(tx, keys[start:end], buckets, counts); err != nil {
			return nil, err
		}
	}
	return counts, nil
}

func loadLimitCountChunk(tx *gorm.DB, keys []string, buckets map[string]activityLimitBucket, counts map[string]int64) error {
	values := make([]string, 0, len(keys))
	arguments := make([]interface{}, 0, len(keys)*5)
	for _, key := range keys {
		bucket := buckets[key]
		values = append(values, "(?, ?::uuid, ?::uuid, ?::timestamptz, ?::timestamptz)")
		arguments = append(arguments, bucket.key, bucket.userID, bucket.itemID, bucket.start, bucket.end)
		counts[key] = 0
	}
	query := `WITH requested(bucket_key, user_id, item_id, start_at, end_at) AS (VALUES ` + strings.Join(values, ",") + `)
		SELECT requested.bucket_key, COUNT(activities.id) AS activity_count
		FROM requested
		LEFT JOIN activities ON activities.user_id = requested.user_id
			AND activities.item_id = requested.item_id
			AND activities.deleted_at IS NULL
			AND (requested.start_at IS NULL OR activities.occurred_at >= requested.start_at)
			AND (requested.end_at IS NULL OR activities.occurred_at < requested.end_at)
		GROUP BY requested.bucket_key`
	var rows []activityLimitCount
	if err := tx.Raw(query, arguments...).Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		counts[row.Key] = row.Count
	}
	return nil
}

func parseOptionalUUID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func mapUUIDKeys(values map[uuid.UUID]struct{}) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}

func uuidSet(values []uuid.UUID) map[uuid.UUID]struct{} {
	result := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func bulkEntryError(index int, field, message string) error {
	return &BulkValidationError{Index: index, Field: field, Message: message}
}

var _ error = (*BulkValidationError)(nil)
