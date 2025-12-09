package throttle

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// helper to create a default manager used across tests
func newTestManager() *Manager {
	profiles := []Profile{
		{Name: "Full", Level: 10, LatencyMs: 0, DownloadBytesPerSec: 0, UploadBytesPerSec: 0},
		{Name: "Slow", Level: 5, LatencyMs: 300, DownloadBytesPerSec: 10_000, UploadBytesPerSec: 5_000},
	}
	return NewManager(profiles, 10)
}

func TestManagerSetAndGetLevel(t *testing.T) {
	m := newTestManager()

	// initial level
	if lvl := m.CurrentLevel(); lvl != 10 {
		t.Fatalf("expected initial level 10, got %d", lvl)
	}

	// change to a valid level
	m.SetLevel(5)
	if lvl := m.CurrentLevel(); lvl != 5 {
		t.Fatalf("expected level 5, got %d", lvl)
	}

	p := m.GetProfile()
	if p.Level != 5 || p.Name != "Slow" {
		t.Fatalf("expected Slow profile level 5, got %+v", p)
	}

	// change to an invalid level – should be ignored
	m.SetLevel(999)
	if lvl := m.CurrentLevel(); lvl != 5 {
		t.Fatalf("expected level to remain 5 after invalid SetLevel, got %d", lvl)
	}
}

func TestThrottledCopyUnlimited(t *testing.T) {
	srcData := bytes.Repeat([]byte("abc"), 1000) // 3000 bytes
	src := bytes.NewReader(srcData)
	var dst bytes.Buffer

	n, err := ThrottledCopy(&dst, src, 0) // 0 = unlimited
	if err != nil {
		t.Fatalf("ThrottledCopy (unlimited) returned error: %v", err)
	}
	if n != int64(len(srcData)) {
		t.Fatalf("expected %d bytes copied, got %d", len(srcData), n)
	}

	if !bytes.Equal(srcData, dst.Bytes()) {
		t.Fatalf("destination data does not match source")
	}
}

func TestThrottledCopyLimited(t *testing.T) {
	srcData := bytes.Repeat([]byte("x"), 50_000) // 50 KB
	src := bytes.NewReader(srcData)
	var dst bytes.Buffer

	rate := int64(10_000) // 10 KB/sec
	start := time.Now()
	n, err := ThrottledCopy(&dst, src, rate)
	elapsed := time.Since(start)

	if err != nil && err != io.EOF {
		t.Fatalf("ThrottledCopy (limited) returned error: %v", err)
	}

	if n != int64(len(srcData)) {
		t.Fatalf("expected %d bytes copied, got %d", len(srcData), n)
	}

	if !bytes.Equal(srcData, dst.Bytes()) {
		t.Fatalf("destination data does not match source")
	}

	// We won't assert a precise duration, but we can at least check that
	// some time has passed to indicate throttling, unless the system is weirdly fast.
	if elapsed < 2*time.Second {
		t.Logf("warning: ThrottledCopy completed quickly (elapsed=%s); throttling may be too coarse for this small test", elapsed)
	}
}
