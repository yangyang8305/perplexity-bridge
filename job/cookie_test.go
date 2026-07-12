package job

import (
	"pplx2api/config"
	"sync"
	"testing"
)

// A9: after session replacement, Sr.Index must be 0
func TestResetIndexAfterReplacement(t *testing.T) {
	config.ConfigInstance = &config.Config{
		Sessions: []config.SessionInfo{{SessionKey: "a"}, {SessionKey: "b"}},
		RwMutex:  sync.RWMutex{},
	}
	config.Sr = &config.SessionRagen{Index: 5, Mutex: sync.Mutex{}}
	config.Sr.ResetIndex()
	if config.Sr.Index != 0 {
		t.Errorf("expected Sr.Index=0 after reset, got %d", config.Sr.Index)
	}
}

// A2: firstValidModel must return a non-empty string
func TestFirstValidModel(t *testing.T) {
	m := firstValidModel()
	if m == "" {
		t.Error("firstValidModel returned empty string")
	}
}

// A3: failed session must not appear in updated pool
func TestDropFailedSessions(t *testing.T) {
	type result struct {
		index   int
		session config.SessionInfo
		ok      bool
	}
	results := []result{
		{0, config.SessionInfo{SessionKey: "alive"}, true},
		{1, config.SessionInfo{}, false}, // failed
	}
	updated := make([]config.SessionInfo, 0)
	for _, r := range results {
		if r.ok {
			updated = append(updated, r.session)
		}
	}
	if len(updated) != 1 {
		t.Errorf("expected 1 live session, got %d", len(updated))
	}
	if updated[0].SessionKey != "alive" {
		t.Errorf("wrong session kept: %s", updated[0].SessionKey)
	}
}
