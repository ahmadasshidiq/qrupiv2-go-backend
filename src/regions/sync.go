package regions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"clasenna-go-backend/libs/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultBaseURL = "https://wilayah.id/api"
	syncSource     = "wilayah.id"
	regionLockKey  = "qrupi-region-sync-wilayah-id"
)

var errSyncAlreadyRunning = errors.New("region sync is already running")

type APIArea struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type APIMeta struct {
	AdministrativeAreaLevel int    `json:"administrative_area_level"`
	UpdatedAt               string `json:"updated_at"`
}

type APIResponse struct {
	Data []APIArea `json:"data"`
	Meta APIMeta   `json:"meta"`
}

type childArea struct {
	ParentCode string
	Code       string
	Name       string
}

type Synchronizer struct {
	DB          *gorm.DB
	Client      *http.Client
	BaseURL     string
	Concurrency int
	Logger      *slog.Logger
}

func NewSynchronizer(db *gorm.DB, logger *slog.Logger, baseURL string, concurrency int) *Synchronizer {
	baseURL, validURL := normalizeBaseURL(baseURL)
	if !validURL {
		logger.Warn("invalid REGION_API_BASE_URL; using default", "base_url", baseURL, "default", defaultBaseURL)
		baseURL = defaultBaseURL
	}
	if concurrency < 1 {
		concurrency = 8
	}
	return &Synchronizer{
		DB: db, BaseURL: strings.TrimRight(baseURL, "/"), Concurrency: concurrency, Logger: logger,
		Client: &http.Client{Timeout: 20 * time.Second},
	}
}

func normalizeBaseURL(value string) (string, bool) {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if open := strings.Index(value, "]("); strings.HasPrefix(value, "[") && open > 1 && strings.HasSuffix(value, ")") {
		value = value[open+2 : len(value)-1]
	}
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return value, false
	}
	return value, true
}

func (s *Synchronizer) Sync(ctx context.Context) error {
	return s.DB.WithContext(ctx).Connection(func(connection *gorm.DB) error {
		cleanConnection := func() *gorm.DB {
			return connection.Session(&gorm.Session{NewDB: true}).WithContext(ctx)
		}
		var acquired bool
		if err := cleanConnection().Raw("SELECT pg_try_advisory_lock(hashtext(?))", regionLockKey).Scan(&acquired).Error; err != nil {
			return err
		}
		if !acquired {
			s.Logger.InfoContext(ctx, "region sync skipped because another instance is running")
			return errSyncAlreadyRunning
		}
		defer func() {
			if err := cleanConnection().Exec("SELECT pg_advisory_unlock(hashtext(?))", regionLockKey).Error; err != nil {
				s.Logger.ErrorContext(ctx, "failed to release region sync lock", "error", err)
			}
		}()
		return s.syncLocked(ctx, cleanConnection())
	})
}

func (s *Synchronizer) syncLocked(ctx context.Context, db *gorm.DB) error {
	checkedAt := time.Now().UTC()
	provincesResponse, err := s.fetch(ctx, "provinces.json")
	if err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	sourceUpdatedAt, err := time.Parse("2006-01-02", provincesResponse.Meta.UpdatedAt)
	if err != nil {
		err = fmt.Errorf("parse wilayah.id updated_at: %w", err)
		s.markFailed(checkedAt, err)
		return err
	}

	var state models.RegionSyncState
	if err := db.Where("source = ?", syncSource).Limit(1).Find(&state).Error; err != nil {
		return err
	}
	var provinceCount int64
	if err := db.Model(&models.Province{}).Where("is_active = true").Count(&provinceCount).Error; err != nil {
		return err
	}
	if state.SourceUpdatedAt != nil && state.SourceUpdatedAt.Equal(sourceUpdatedAt) && provinceCount > 0 {
		return s.saveState(db, models.RegionSyncState{
			Source: syncSource, SourceUpdatedAt: &sourceUpdatedAt, LastCheckedAt: &checkedAt,
			LastSyncedAt: state.LastSyncedAt, Status: "current",
		})
	}
	if err := s.saveState(db, models.RegionSyncState{
		Source: syncSource, SourceUpdatedAt: state.SourceUpdatedAt, LastCheckedAt: &checkedAt,
		LastSyncedAt: state.LastSyncedAt, Status: "running",
	}); err != nil {
		return err
	}

	provinces := make([]models.Province, len(provincesResponse.Data))
	for index, area := range provincesResponse.Data {
		provinces[index] = models.Province{Code: area.Code, Name: strings.TrimSpace(area.Name), IsActive: true, SourceUpdatedAt: sourceUpdatedAt}
	}
	if err := upsertRegions(db, provinces, []string{"name", "is_active", "source_updated_at", "updated_at"}); err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	s.Logger.InfoContext(ctx, "region sync level completed", "level", "provinces", "count", len(provinces))

	provinceCodes := areaCodes(provincesResponse.Data)
	regencyAreas, err := s.fetchChildren(ctx, provinceCodes, "regencies")
	if err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	regencies := make([]models.Regency, len(regencyAreas))
	for index, area := range regencyAreas {
		regencies[index] = models.Regency{Code: area.Code, ProvinceCode: area.ParentCode, Name: area.Name, IsActive: true, SourceUpdatedAt: sourceUpdatedAt}
	}
	if err := upsertRegions(db, regencies, []string{"province_code", "name", "is_active", "source_updated_at", "updated_at"}); err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	s.Logger.InfoContext(ctx, "region sync level completed", "level", "regencies", "count", len(regencies))

	districtAreas, err := s.fetchChildren(ctx, childCodes(regencyAreas), "districts")
	if err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	districts := make([]models.District, len(districtAreas))
	for index, area := range districtAreas {
		districts[index] = models.District{Code: area.Code, RegencyCode: area.ParentCode, Name: area.Name, IsActive: true, SourceUpdatedAt: sourceUpdatedAt}
	}
	if err := upsertRegions(db, districts, []string{"regency_code", "name", "is_active", "source_updated_at", "updated_at"}); err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	s.Logger.InfoContext(ctx, "region sync level completed", "level", "districts", "count", len(districts))

	villageCount, err := s.fetchAndStoreVillages(ctx, db, childCodes(districtAreas), sourceUpdatedAt)
	if err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	s.Logger.InfoContext(ctx, "region sync level completed", "level", "villages", "count", villageCount)

	syncedAt := time.Now().UTC()
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, model := range []interface{}{&models.Province{}, &models.Regency{}, &models.District{}, &models.Village{}} {
			if err := tx.Model(model).
				Where("is_active = true AND source_updated_at <> ?", sourceUpdatedAt).
				Update("is_active", false).Error; err != nil {
				return err
			}
		}
		return s.saveState(tx, models.RegionSyncState{
			Source: syncSource, SourceUpdatedAt: &sourceUpdatedAt, LastCheckedAt: &checkedAt,
			LastSyncedAt: &syncedAt, Status: "completed",
		})
	})
	if err != nil {
		s.markFailed(checkedAt, err)
		return err
	}
	s.Logger.InfoContext(ctx, "region sync completed",
		"source_updated_at", provincesResponse.Meta.UpdatedAt, "provinces", len(provinces),
		"regencies", len(regencies), "districts", len(districts), "villages", villageCount)
	return nil
}

func (s *Synchronizer) fetchAndStoreVillages(ctx context.Context, db *gorm.DB, districtCodes []string, sourceUpdatedAt time.Time) (int, error) {
	stored := 0
	batch := make([]models.Village, 0, 500)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := upsertRegions(db, batch, []string{"district_code", "name", "is_active", "source_updated_at", "updated_at"}); err != nil {
			return err
		}
		stored += len(batch)
		batch = batch[:0]
		s.Logger.InfoContext(ctx, "region sync batch stored", "level", "villages", "count", stored)
		return nil
	}

	_, err := s.walkChildren(ctx, districtCodes, "villages", func(areas []childArea) error {
		for _, area := range areas {
			batch = append(batch, models.Village{
				Code: area.Code, DistrictCode: area.ParentCode, Name: area.Name,
				IsActive: true, SourceUpdatedAt: sourceUpdatedAt,
			})
		}
		if len(batch) >= 500 {
			return flush()
		}
		return nil
	})
	if err != nil {
		return stored, err
	}
	if err := flush(); err != nil {
		return stored, err
	}
	return stored, nil
}

func (s *Synchronizer) fetchChildren(ctx context.Context, parentCodes []string, segment string) ([]childArea, error) {
	var all []childArea
	_, err := s.walkChildren(ctx, parentCodes, segment, func(areas []childArea) error {
		all = append(all, areas...)
		return nil
	})
	return all, err
}

func (s *Synchronizer) walkChildren(ctx context.Context, parentCodes []string, segment string, consume func([]childArea) error) (int, error) {
	workerCount := s.Concurrency
	if workerCount > len(parentCodes) {
		workerCount = len(parentCodes)
	}
	if workerCount == 0 {
		return 0, nil
	}
	type result struct {
		areas []childArea
		err   error
	}
	jobs := make(chan string)
	results := make(chan result, workerCount)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for parentCode := range jobs {
				response, err := s.fetch(ctx, fmt.Sprintf("%s/%s.json", segment, parentCode))
				areas := make([]childArea, len(response.Data))
				for index, area := range response.Data {
					areas[index] = childArea{ParentCode: parentCode, Code: area.Code, Name: strings.TrimSpace(area.Name)}
				}
				results <- result{areas: areas, err: err}
			}
		}()
	}
	go func() {
		for _, code := range parentCodes {
			jobs <- code
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	processed := 0
	var firstError error
	for result := range results {
		if result.err != nil && firstError == nil {
			firstError = result.err
		}
		if result.err == nil && firstError == nil {
			if err := consume(result.areas); err != nil {
				firstError = err
			}
		}
		processed++
	}
	return processed, firstError
}

func (s *Synchronizer) fetch(ctx context.Context, path string) (APIResponse, error) {
	var lastError error
	for attempt := 1; attempt <= 3; attempt++ {
		payload, err := s.fetchOnce(ctx, path)
		if err == nil {
			if payload.Meta.UpdatedAt != "" {
				return payload, nil
			}
			err = fmt.Errorf("wilayah.id %s response has no updated_at", path)
		}
		lastError = err
		if attempt < 3 {
			select {
			case <-ctx.Done():
				return APIResponse{}, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
	}
	return APIResponse{}, lastError
}

func (s *Synchronizer) fetchOnce(ctx context.Context, path string) (APIResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+"/"+path, nil)
	if err != nil {
		return APIResponse{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Qrupi-Region-Sync/1.0")
	response, err := s.Client.Do(request)
	if err != nil {
		return APIResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return APIResponse{}, fmt.Errorf("wilayah.id %s returned status %d", path, response.StatusCode)
	}
	var payload APIResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 10<<20)).Decode(&payload); err != nil {
		return APIResponse{}, fmt.Errorf("decode wilayah.id %s: %w", path, err)
	}
	return payload, nil
}

func (s *Synchronizer) saveState(db *gorm.DB, state models.RegionSyncState) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source"}}, DoUpdates: clause.AssignmentColumns([]string{
			"source_updated_at", "last_checked_at", "last_synced_at", "status", "error_message", "updated_at",
		}),
	}).Create(&state).Error
}

func (s *Synchronizer) markFailed(checkedAt time.Time, cause error) {
	state := models.RegionSyncState{
		Source: syncSource, LastCheckedAt: &checkedAt, Status: "failed", ErrorMessage: cause.Error(),
	}
	if err := s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"last_checked_at": checkedAt,
			"status":          "failed",
			"error_message":   cause.Error(),
			"updated_at":      time.Now().UTC(),
		}),
	}).Create(&state).Error; err != nil {
		s.Logger.Error("failed to store region sync error", "error", err)
	}
}

func upsertRegions[T any](db *gorm.DB, values []T, updateColumns []string) error {
	if len(values) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns(updateColumns),
	}).CreateInBatches(values, 500).Error
}

func areaCodes(areas []APIArea) []string {
	codes := make([]string, len(areas))
	for index, area := range areas {
		codes[index] = area.Code
	}
	return codes
}

func childCodes(areas []childArea) []string {
	codes := make([]string, len(areas))
	for index, area := range areas {
		codes[index] = area.Code
	}
	return codes
}

func RunScheduler(ctx context.Context, synchronizer *Synchronizer, intervalMonths int, retryDelay time.Duration) {
	if intervalMonths < 1 {
		intervalMonths = 3
	}
	if retryDelay <= 0 {
		retryDelay = 15 * time.Minute
	}
	for {
		nextRun := time.Now().AddDate(0, intervalMonths, 0)
		if err := synchronizer.Sync(ctx); err != nil && ctx.Err() == nil {
			if errors.Is(err, errSyncAlreadyRunning) {
				nextRun = time.Now().Add(time.Minute)
			} else {
				synchronizer.Logger.ErrorContext(ctx, "region sync failed", "error", err)
				nextRun = time.Now().Add(retryDelay)
			}
		}
		timer := time.NewTimer(time.Until(nextRun))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
