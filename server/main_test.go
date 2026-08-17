package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	recorder := httptest.NewRecorder()

	healthHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %q", got)
	}

	want := `{"status":"ok","service":"otter-link"}`
	if got := recorder.Body.String(); got != want+"\n" {
		t.Fatalf("expected %q, got %q", want+"\n", got)
	}
}

func TestGetenv(t *testing.T) {
	const key = "OTTERLINK_TEST_VALUE"
	t.Setenv(key, "configured")

	if got := getenv(key, "fallback"); got != "configured" {
		t.Fatalf("expected configured value, got %q", got)
	}

	if got := getenv("OTTERLINK_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback value, got %q", got)
	}
}
