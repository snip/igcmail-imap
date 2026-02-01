package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/emersion/go-imap"
)

// State holds the last seen UID for incremental IMAP fetch.
type State struct {
	LastUID uint32 `json:"last_uid"`
}

// Load reads state from the JSON file. If the file does not exist, returns LastUID 0.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{LastUID: 0}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes state to the JSON file. Creates the parent directory if needed (e.g. on macOS).
func Save(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// UpdateLastUID updates state with the highest UID from the given set and saves.
func UpdateLastUID(path string, s *State, uids []uint32) error {
	var max uint32
	for _, u := range uids {
		if u > max {
			max = u
		}
	}
	if max > 0 {
		s.LastUID = max
		return Save(path, s)
	}
	return nil
}

// UIDSet returns an imap.SeqSet for "UID lastUID+1:*" (messages after last seen).
// In go-imap SeqSet, 0 for Stop means "*".
func (s *State) UIDSet() *imap.SeqSet {
	set := new(imap.SeqSet)
	if s.LastUID > 0 {
		set.AddRange(s.LastUID+1, 0) // 0 = *
	} else {
		set.AddRange(1, 0) // 1:*
	}
	return set
}
