package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type EventType string

type EventScope string

const (
	EventScopeTenant EventScope = "tenant"
	EventScopeUser   EventScope = "user"
	EventScopeSystem EventScope = "system"
)

const (
	EventTypeAttendanceCreated EventType = "attendance.created"
	EventTypeQuizCreated       EventType = "quiz.created"
	EventTypeMaterialCreated   EventType = "material.created"
	EventTypeLoginDetected     EventType = "auth.login_detected"
	EventTypeUserCreated       EventType = "user.created"
	EventTypePasswordReset     EventType = "user.password_reset"
)

type Event struct {
	Type          EventType              `json:"type"`
	Scope         EventScope             `json:"scope,omitempty"`
	Title         string                 `json:"title"`
	Message       string                 `json:"message"`
	Category      string                 `json:"category"`
	CategoryKey   string                 `json:"category_key"`
	InstitutionID string                 `json:"institution_id,omitempty"`
	GroupName     string                 `json:"group_name,omitempty"`
	GroupCode     string                 `json:"group_code,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	ActorID       string                 `json:"actor_id,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, Event) error {
	return nil
}

type HTTPPublisher struct {
	BaseURL string
	Client  *http.Client
}

func NewPublisherFromEnv() Publisher {
	baseURL := os.Getenv("NOTIFICATION_SERVICE_URL")
	if baseURL == "" {
		return NoopPublisher{}
	}

	return HTTPPublisher{BaseURL: baseURL}
}

func (p HTTPPublisher) Publish(ctx context.Context, event Event) error {
	if p.BaseURL == "" {
		return nil
	}

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/internal/events", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build notification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send notification event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}
