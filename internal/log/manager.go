package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

func NewManager(outputPath string) (*Manager, error) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return &Manager{path: outputPath, f: f}, nil
}

func (m *Manager) Write(r CommitRecord) error {
	data, err := marshal(r)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := m.f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write record: %w", err)
	}
	return m.f.Sync()
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.f.Close()
}
