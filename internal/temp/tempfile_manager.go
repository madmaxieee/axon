package temp

import (
	"fmt"
	"os"
	"sync"
)

// Manager manages temporary files mapped by string handles.
type Manager struct {
	dir     string
	handles map[string]*Handle
	mu      sync.Mutex
}

// NewManager creates a new tempfile Manager.
// If dir is empty, it uses the default OS temp directory.
func NewManager(dir string) *Manager {
	if dir == "" {
		dir = os.TempDir()
	}
	return &Manager{
		dir:     dir,
		handles: make(map[string]*Handle),
	}
}

// Handle represents a temporary file.
type Handle struct {
	path string
}

// Path returns the underlying file path.
func (h *Handle) Path() string {
	return h.path
}

// WriteText overwrites the file with the given text.
func (h *Handle) WriteText(text string) error {
	return os.WriteFile(h.path, []byte(text), 0644)
}

// AppendText appends the given text to the file.
func (h *Handle) AppendText(text string) error {
	f, err := os.OpenFile(h.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	return err
}

// GetTempFile returns a Handle for the given string ID.
// It creates a new temp file if one does not exist for the ID.
func (m *Manager) GetTempFile(id string) (*Handle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if handle, exists := m.handles[id]; exists {
		return handle, nil
	}

	file, err := os.CreateTemp(m.dir, fmt.Sprintf("%s-*", id))
	if err != nil {
		return nil, err
	}
	path := file.Name()
	file.Close() // Close it since we only needed to reserve the name and create the file

	handle := &Handle{path: path}
	m.handles[id] = handle
	return handle, nil
}

// Cleanup removes all tracked temporary files from disk.
func (m *Manager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for id, handle := range m.handles {
		if err := os.Remove(handle.path); err != nil && !os.IsNotExist(err) {
			if firstErr == nil {
				firstErr = err
			}
		}
		delete(m.handles, id)
	}
	return firstErr
}
