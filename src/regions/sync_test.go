package regions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchUsesOneRequestOnSuccess(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"code":"36","name":"Banten"}],"meta":{"administrative_area_level":1,"updated_at":"2025-07-04"}}`))
	}))
	defer server.Close()

	synchronizer := &Synchronizer{Client: server.Client(), BaseURL: server.URL}
	payload, err := synchronizer.fetch(context.Background(), "provinces.json")
	if err != nil {
		t.Fatalf("fetch returned an error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one request, got %d", calls.Load())
	}
	if len(payload.Data) != 1 || payload.Data[0].Code != "36" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestFetchRetriesTemporaryFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(writer, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte(`{"data":[],"meta":{"administrative_area_level":1,"updated_at":"2025-07-04"}}`))
	}))
	defer server.Close()

	synchronizer := &Synchronizer{Client: server.Client(), BaseURL: server.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := synchronizer.fetch(ctx, "provinces.json"); err != nil {
		t.Fatalf("fetch did not recover after retry: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected two requests, got %d", calls.Load())
	}
}

func TestNormalizeBaseURLAcceptsMarkdownLink(t *testing.T) {
	value, valid := normalizeBaseURL("[https://wilayah.id/api](https://wilayah.id/api)")
	if !valid {
		t.Fatal("expected Markdown URL to be valid after normalization")
	}
	if value != "https://wilayah.id/api" {
		t.Fatalf("unexpected normalized URL: %s", value)
	}
}

func TestNormalizeBaseURLRejectsInvalidScheme(t *testing.T) {
	if _, valid := normalizeBaseURL("file:///tmp/regions"); valid {
		t.Fatal("expected unsupported URL scheme to be rejected")
	}
}

func TestFetchLiveWilayahID(t *testing.T) {
	if os.Getenv("REGION_LIVE_TEST") != "1" {
		t.Skip("set REGION_LIVE_TEST=1 to call wilayah.id")
	}
	synchronizer := &Synchronizer{
		Client:  &http.Client{Timeout: 30 * time.Second},
		BaseURL: defaultBaseURL,
	}
	payload, err := synchronizer.fetch(context.Background(), "provinces.json")
	if err != nil {
		t.Fatalf("live fetch failed: %v", err)
	}
	if len(payload.Data) == 0 {
		t.Fatal("live fetch returned no provinces")
	}
}
