package kafka

import (
	"context"
	"testing"
)

func TestEnsureTopicsRequiresBroker(t *testing.T) {
	if err := EnsureTopics(context.Background(), nil, []string{"events"}, 1, 1); err == nil {
		t.Fatal("expected missing broker error")
	}
}
