package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/rifat977/standup/internal/config"
)

const maxEntries = 30

// Entry is one saved standup.
type Entry struct {
	Date    time.Time `json:"date"`
	Summary string    `json:"summary"`
	Today   string    `json:"today,omitempty"`
	Blocker string    `json:"blocker,omitempty"`
}

func file() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// Load returns the saved entries (newest first). Missing file → empty slice.
func Load() ([]Entry, error) {
	p, err := file()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Save prepends the entry, caps at maxEntries, and writes atomically.
func Save(e Entry) error {
	entries, err := Load()
	if err != nil {
		return err
	}
	entries = append([]Entry{e}, entries...)
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	p, err := file()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
