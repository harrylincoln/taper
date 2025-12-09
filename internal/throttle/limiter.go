package throttle

import (
	"io"
	"sync"
	"time"
)

type Profile struct {
	Name                string
	Level               int
	LatencyMs           int
	DownloadBytesPerSec int64
	UploadBytesPerSec   int64
}

type Manager struct {
	mu       sync.RWMutex
	profiles map[int]Profile
	current  int
}

func NewManager(profiles []Profile, initialLevel int) *Manager {
	m := &Manager{
		profiles: make(map[int]Profile),
		current:  initialLevel,
	}
	for _, p := range profiles {
		m.profiles[p.Level] = p
	}
	return m
}

func (m *Manager) GetProfile() Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profiles[m.current]
}

func (m *Manager) SetLevel(level int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[level]; ok {
		m.current = level
	}
}

func (m *Manager) CurrentLevel() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// ThrottledCopy copies data with simple bandwidth limiting.
// direction: "down" or "up"
func ThrottledCopy(dst io.Writer, src io.Reader, bytesPerSec int64) (int64, error) {
	if bytesPerSec <= 0 {
		// unlimited
		return io.Copy(dst, src)
	}

	const chunkSize = 32 * 1024
	buf := make([]byte, chunkSize)
	var total int64

	interval := time.Second
	quota := bytesPerSec

	for {
		start := time.Now()
		remaining := quota

		for remaining > 0 {
			toRead := int64(chunkSize)
			if toRead > remaining {
				toRead = remaining
			}

			n, readErr := src.Read(buf[:toRead])
			if n > 0 {
				wn, writeErr := dst.Write(buf[:n])
				total += int64(wn)
				remaining -= int64(wn)
				if writeErr != nil {
					return total, writeErr
				}
			}

			if readErr != nil {
				if readErr == io.EOF {
					return total, nil
				}
				return total, readErr
			}
		}

		elapsed := time.Since(start)
		if elapsed < interval {
			time.Sleep(interval - elapsed)
		}
		quota = bytesPerSec
	}
}
