package helpers

import "testing"

func TestFormatResponseTreatsAcceptedAsSuccess(t *testing.T) {
	response := FormatResponse("POST", "activities", 202, map[string]string{"status": "pending"}, nil, nil)
	if response["status"] != "success" {
		t.Fatalf("expected successful 202 response, got %#v", response)
	}
}
