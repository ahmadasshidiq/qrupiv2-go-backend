package kafka

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	Version       int             `json:"version"`
	Timestamp     time.Time       `json:"timestamp"`
	Source        string          `json:"source"`
	InstitutionID string          `json:"institution_id"`
	CorrelationID string          `json:"correlation_id"`
	Data          json.RawMessage `json:"data"`
}

func NewEnvelope(eventType, source, institutionID, correlationID string, data any) (Envelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, err
	}
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	e := Envelope{
		EventID: uuid.NewString(), EventType: eventType, Version: 1,
		Timestamp: time.Now().UTC(), Source: source, InstitutionID: institutionID,
		CorrelationID: correlationID, Data: payload,
	}
	return e, e.Validate()
}

func (e Envelope) Validate() error {
	if e.EventID == "" || e.EventType == "" || e.Version < 1 || e.Timestamp.IsZero() ||
		e.Source == "" || e.InstitutionID == "" || e.CorrelationID == "" || len(e.Data) == 0 {
		return errors.New("invalid event envelope: all required fields must be set")
	}
	return nil
}
