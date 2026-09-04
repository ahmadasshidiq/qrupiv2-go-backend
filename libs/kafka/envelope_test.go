package kafka

import "testing"

func TestNewEnvelope(t *testing.T) {
	event, err := NewEnvelope("student.created", "core-service", "tenant-1", "correlation-1", map[string]string{"id": "student-1"})
	if err != nil {
		t.Fatal(err)
	}
	if event.EventID == "" || event.Version != 1 || event.EventType != "student.created" {
		t.Fatalf("unexpected envelope: %+v", event)
	}
}

func TestEnvelopeRequiresInstitution(t *testing.T) {
	_, err := NewEnvelope("student.created", "core-service", "", "correlation-1", map[string]string{"id": "student-1"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
