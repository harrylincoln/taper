package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harrylincoln/taper/internal/throttle"
)

// helper to create a proxy server with unlimited throttling
func newTestProxyServer(t *testing.T) *Server {
	t.Helper()

	profiles := []throttle.Profile{
		{Name: "Full", Level: 10, LatencyMs: 0, DownloadBytesPerSec: 0, UploadBytesPerSec: 0},
	}
	mgr := throttle.NewManager(profiles, 10)

	// addr is irrelevant because we call handle directly
	return NewServer(":0", mgr)
}

func TestHandleHTTPProxiesRequest(t *testing.T) {
	// Upstream test server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tiny assertion to verify proxy forwarded correctly
		if r.URL.Path != "/test" {
			t.Errorf("expected path /test, got %s", r.URL.Path)
		}
		w.Header().Set("X-Upstream", "ok")
		_, _ = w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	// Proxy server
	ps := newTestProxyServer(t)

	// Build a request as if it came to the proxy: for HTTP proxies,
	// RequestURI is the full URL to the target.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RequestURI = upstream.URL + "/test"

	rr := httptest.NewRecorder()

	ps.handle(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 from proxy, got %d", res.StatusCode)
	}

	if got := res.Header.Get("X-Upstream"); got != "ok" {
		t.Fatalf("expected X-Upstream=ok header, got %q", got)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "hello from upstream") {
		t.Fatalf("unexpected body from proxy: %q", body)
	}
}

// This test just verifies we take the CONNECT path without panicking.
// It doesn't fully exercise the TCP tunnel.
func TestHandleHTTPSConnectDoesNotPanic(t *testing.T) {
	ps := newTestProxyServer(t)

	req := httptest.NewRequest(http.MethodConnect, "https://example.com:443", nil)
	req.Host = "example.com:443"

	rr := httptest.NewRecorder()

	// We can't fully simulate the hijacker in httptest, but we at least
	// ensure the handler doesn't panic when Hijacker is unavailable and
	// returns a 500 error as implemented.
	ps.handle(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 status when Hijacker not supported, got %d", res.StatusCode)
	}
}
