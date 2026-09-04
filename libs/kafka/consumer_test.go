package kafka

import (
	"errors"
	"testing"
)

func TestInvalidEnvelopeCanBeWrappedForDeadLetter(t *testing.T) {
	invalid := Envelope{EventID: "legacy-event"}
	if invalid.Validate() == nil {
		t.Fatal("expected incomplete legacy envelope to be invalid")
	}

	wrapped, err := buildDeadLetterEnvelope(invalid, "activity-service", []byte(`{"event_id":"legacy-event"}`), errors.New("invalid envelope"))
	if err != nil {
		t.Fatal(err)
	}
	if err := wrapped.Validate(); err != nil {
		t.Fatalf("wrapped DLQ envelope must be valid: %v", err)
	}
}
