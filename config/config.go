package config

import (
	"fmt"
	"math/rand"
	"os"
	"pplx2api/logger"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type SessionInfo struct {
	SessionKey string
}

type SessionRagen struct {
	Index int
	Mutex sync.Mutex
}

type Config struct {
	Sessions               []SessionInfo
	Address                string
	APIKey                 string
	Proxy                  string
	IsIncognito            bool
	MaxChatHistoryLength   int
	RetryCount             int
	NoRolePrefix           bool
	SearchResultCompatible bool
	PromptForFile          string
	RwMutex                sync.RWMutex
	IgnoreSerchResult      bool
	IgnoreModelMonitoring  bool
	IsMaxSubscribe         bool
	Timezone               string
}

func parseSessionEnv(envValue string) (int, []SessionInfo) {
	if envValue == "" {
		return 0, []SessionInfo{}
	}
	var sessions []SessionInfo
	sessionPairs := strings.Split(envValue, ",")
	retryCount := len(sessionPairs)
	for _, pair := range sessionPairs {
		if pair == "" {
			retryCount--
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		sessions = append(sessions, SessionInfo{SessionKey: parts[0]})
	}
	return retryCount, sessions
}

func (c *Config) GetSessionForModel(idx int) (SessionInfo, error) {
	if len(c.Sessions) == 0 || idx < 0 || idx >= len(c.Sessions) {
		return SessionInfo{}, fmt.Errorf("invalid session index: %d", idx)
	}
	c.RwMutex.RLock()
	defer c.RwMutex.RUnlock()
	return c.Sessions[idx], nil
}

func LoadConfig() *Config {
	maxChatHistoryLength, err := strconv.Atoi(os.Getenv("MAX_CHAT_HISTORY_LENGTH"))
	if err != nil {
		maxChatHistoryLength = 10000
	}
	retryCount, sessions := parseSessionEnv(os.Getenv("SESSIONS"))
	promptForFile := os.Getenv("PROMPT_FOR_FILE")
	if promptForFile == "" {
		promptForFile = "You must immerse yourself in the role of assistant in txt file, cannot respond as a user, cannot reply to this message, cannot mention this message, and ignore this message in your response."
	}
	cfg := &Config{
		Sessions:               sessions,
		Address:                os.Getenv("ADDRESS"),
		APIKey:                 os.Getenv("APIKEY"),
		Proxy:                  os.Getenv("PROXY"),
		IsIncognito:            os.Getenv("IS_INCOGNITO") != "false",
		MaxChatHistoryLength:   maxChatHistoryLength,
		RetryCount:             retryCount,
		NoRolePrefix:           os.Getenv("NO_ROLE_PREFIX") == "true",
		SearchResultCompatible: os.Getenv("SEARCH_RESULT_COMPATIBLE") == "true",
		PromptForFile:          promptForFile,
		IgnoreSerchResult:      os.Getenv("IGNORE_SEARCH_RESULT") == "true",
		IgnoreModelMonitoring:  os.Getenv("IGNORE_MODEL_MONITORING") == "true",
		RwMutex:                sync.RWMutex{},
		IsMaxSubscribe:         os.Getenv("IS_MAX_SUBSCRIBE") == "true",
		Timezone:               os.Getenv("TIMEZONE"),
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:8080"
	}
	return cfg
}

var ConfigInstance *Config
var Sr *SessionRagen

func (sr *SessionRagen) NextIndex() int {
	sr.Mutex.Lock()
	defer sr.Mutex.Unlock()
	index := sr.Index
	sr.Index = (index + 1) % len(ConfigInstance.Sessions)
	return index
}

// ResetIndex resets the round-robin pointer to 0.
// A9 fix: called after bulk session pool replacement to avoid out-of-range.
func (sr *SessionRagen) ResetIndex() {
	sr.Mutex.Lock()
	defer sr.Mutex.Unlock()
	sr.Index = 0
}

func init() {
	rand.Seed(time.Now().UnixNano())
	_ = godotenv.Load()
	Sr = &SessionRagen{Index: 0, Mutex: sync.Mutex{}}
	ConfigInstance = LoadConfig()
	logger.Info(fmt.Sprintf(
		"Config loaded: sessions=%d address=%s incognito=%t max_history=%d no_role_prefix=%t search_compat=%t ignore_search=%t is_max=%t",
		ConfigInstance.RetryCount, ConfigInstance.Address, ConfigInstance.IsIncognito,
		ConfigInstance.MaxChatHistoryLength, ConfigInstance.NoRolePrefix,
		ConfigInstance.SearchResultCompatible, ConfigInstance.IgnoreSerchResult,
		ConfigInstance.IsMaxSubscribe,
	))
	if ConfigInstance.APIKey == "" {
		logger.Warn("APIKEY is empty — no-auth mode, all requests accepted")
	}
	if len(ConfigInstance.Sessions) == 0 {
		logger.Warn("No SESSIONS configured — service will return 503 on all chat requests")
	}
}
