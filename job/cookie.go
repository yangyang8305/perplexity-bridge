package job

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"pplx2api/config"
	"pplx2api/core"
)

const ConfigFileName = "sessions.json"

var (
	sessionUpdaterInstance *SessionUpdater
	sessionUpdaterOnce     sync.Once
)

type SessionConfig struct {
	Sessions []config.SessionInfo `json:"sessions"`
}

type SessionUpdater struct {
	interval    time.Duration
	stopChan    chan struct{}
	isRunning   bool
	runningLock sync.Mutex
	configPath  string
}

func GetSessionUpdater(interval time.Duration) *SessionUpdater {
	sessionUpdaterOnce.Do(func() {
		sessionUpdaterInstance = &SessionUpdater{
			interval:   interval,
			stopChan:   make(chan struct{}),
			isRunning:  false,
			configPath: ConfigFileName,
		}
		sessionUpdaterInstance.loadSessionsFromFile()
	})
	return sessionUpdaterInstance
}

func (su *SessionUpdater) loadSessionsFromFile() {
	if _, err := os.Stat(su.configPath); os.IsNotExist(err) {
		log.Println("No sessions config file found, will create on first update")
		return
	}
	// A8 fix: os.ReadFile instead of deprecated ioutil.ReadFile
	data, err := os.ReadFile(su.configPath)
	if err != nil {
		log.Printf("Failed to read sessions config file: %v", err)
		return
	}
	var sessionConfig SessionConfig
	if err := json.Unmarshal(data, &sessionConfig); err != nil {
		log.Printf("Failed to parse sessions config file: %v", err)
		return
	}
	config.ConfigInstance.RwMutex.Lock()
	config.ConfigInstance.Sessions = sessionConfig.Sessions
	config.ConfigInstance.RwMutex.Unlock()
	log.Printf("Loaded %d sessions from config file", len(sessionConfig.Sessions))
}

func (su *SessionUpdater) saveSessionsToFile(sessions []config.SessionInfo) error {
	data, err := json.MarshalIndent(SessionConfig{Sessions: sessions}, "", "  ")
	if err != nil {
		return err
	}
	// A8 fix: os.WriteFile instead of deprecated ioutil.WriteFile
	if err := os.WriteFile(su.configPath, data, 0644); err != nil {
		return err
	}
	log.Printf("Saved %d sessions to %s", len(sessions), su.configPath)
	return nil
}

func (su *SessionUpdater) Start() {
	su.runningLock.Lock()
	defer su.runningLock.Unlock()
	if su.isRunning {
		log.Println("Session updater already running")
		return
	}
	su.isRunning = true
	su.stopChan = make(chan struct{})
	go su.runUpdateLoop()
	log.Println("Session updater started, interval:", su.interval)
}

func (su *SessionUpdater) Stop() {
	su.runningLock.Lock()
	defer su.runningLock.Unlock()
	if !su.isRunning {
		return
	}
	close(su.stopChan)
	su.isRunning = false
	log.Println("Session updater stopped")
}

func (su *SessionUpdater) runUpdateLoop() {
	ticker := time.NewTicker(su.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			su.updateAllSessions()
		case <-su.stopChan:
			log.Println("Update loop terminated")
			return
		}
	}
}

// firstValidModel returns the first key in ModelMap, used as a neutral model
// for cookie refresh (A2 fix: was hardcoded to a stale model name).
func firstValidModel() string {
	for k := range config.ModelMap {
		return k
	}
	return "sonar" // absolute fallback
}

func (su *SessionUpdater) updateAllSessions() {
	log.Println("Starting session refresh...")
	config.ConfigInstance.RwMutex.RLock()
	sessionsCopy := make([]config.SessionInfo, len(config.ConfigInstance.Sessions))
	copy(sessionsCopy, config.ConfigInstance.Sessions)
	proxy := config.ConfigInstance.Proxy
	config.ConfigInstance.RwMutex.RUnlock()

	if len(sessionsCopy) == 0 {
		log.Println("No sessions to refresh")
		return
	}

	model := firstValidModel()

	// A3 fix: collect only successfully refreshed sessions; drop failed ones.
	type result struct {
		index   int
		session config.SessionInfo
		ok      bool
	}
	results := make([]result, len(sessionsCopy))
	var wg sync.WaitGroup
	for i, session := range sessionsCopy {
		wg.Add(1)
		go func(idx int, orig config.SessionInfo) {
			defer wg.Done()
			client := core.NewClient(orig.SessionKey, proxy, model, false)
			newCookie, err := client.GetNewCookie()
			if err != nil {
				log.Printf("Session %d refresh failed: %v — dropping", idx, err)
				results[idx] = result{index: idx, ok: false}
				return
			}
			results[idx] = result{index: idx, session: config.SessionInfo{SessionKey: newCookie}, ok: true}
		}(i, session)
	}
	wg.Wait()

	// Build updated list with only live sessions
	updatedSessions := make([]config.SessionInfo, 0, len(sessionsCopy))
	for _, r := range results {
		if r.ok {
			updatedSessions = append(updatedSessions, r.session)
		}
	}
	log.Printf("Session refresh done: %d/%d alive", len(updatedSessions), len(sessionsCopy))

	config.ConfigInstance.RwMutex.Lock()
	config.ConfigInstance.Sessions = updatedSessions
	config.ConfigInstance.RetryCount = len(updatedSessions)
	// A9 fix: reset round-robin index after pool replacement to avoid out-of-range
	config.Sr.ResetIndex()
	config.ConfigInstance.RwMutex.Unlock()

	if err := su.saveSessionsToFile(updatedSessions); err != nil {
		log.Printf("Failed to save sessions: %v", err)
	}
}
