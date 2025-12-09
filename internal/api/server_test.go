package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harrylincoln/taper/internal/throttle"
)

func newTestAPI(t *testing.T) (*Server, *throttle.Manager, *httptest.Server) {
	t.Helper()

	profiles := []throttle.Profile{
		{Name: "Full", Level: 10, LatencyMs: 0, DownloadBytesPerSec: 0, UploadBytesPerSec: 0},
		{Name: "Bad", Level: 3, LatencyMs: 800, DownloadBytesPerSec: 64_000, UploadBytesPerSec: 32_000},
	}
	mgr := throttle.NewManager(profiles, 10)

	srv := NewServer(":0", mgr) // addr doesn't matter for httptest

	// Use the handler directly with httptest server
	ts := httptest.NewServer(srv.HttpHandler()) // we'll add HttpHandler() helper below
	return srv, mgr, ts
}

// To make testing easier, expose the http.Handler from Server
// (add this method in your actual api/server.go file):
//
// func (s *Server) HttpHandler() http.Handler {
//     return s.httpSrv.Handler
// }

func TestStatusEndpoint(t *testing.T) {
	_, mgr, ts := newTestAPI(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/status")
	if err != nil {
		t.Fatalf("GET /status error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var payload struct {
		Level   int              `json:"level"`
		Profile throttle.Profile `json:"profile"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}

	if payload.Level != mgr.CurrentLevel() {
		t.Fatalf("expected level %d, got %d", mgr.CurrentLevel(), payload.Level)
	}
	if payload.Profile.Level != payload.Level {
		t.Fatalf("profile level %d does not match reported level %d", payload.Profile.Level, payload.Level)
	}
}

func TestLevelEndpoint(t *testing.T) {
	_, mgr, ts := newTestAPI(t)
	defer ts.Close()

	body := []byte(`{"level":3}`)
	resp, err := http.Post(ts.URL+"/level", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /level error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	if lvl := mgr.CurrentLevel(); lvl != 3 {
		t.Fatalf("expected manager level to be 3 after POST, got %d", lvl)
	}
}

func TestLevelEndpointRejectsBadMethod(t *testing.T) {
	_, _, ts := newTestAPI(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/level", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /level error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /level, got %d", resp.StatusCode)
	}
}
