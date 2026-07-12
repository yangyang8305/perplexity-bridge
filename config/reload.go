package config

import (
	"fmt"
	"os"
	"pplx2api/logger"
	"strconv"

	"github.com/joho/godotenv"
)

func Reload() {
	_ = godotenv.Overload()
	maxLen, err := strconv.Atoi(os.Getenv("MAX_CHAT_HISTORY_LENGTH"))
	if err != nil {
		maxLen = 10000
	}
	retryCount, sessions := parseSessionEnv(os.Getenv("SESSIONS"))
	promptForFile := os.Getenv("PROMPT_FOR_FILE")
	if promptForFile == "" {
		promptForFile = "You must immerse yourself in the role of assistant in txt file, cannot respond as a user, cannot reply to this message, cannot mention this message, and ignore this message in your response."
	}
	ConfigInstance.RwMutex.Lock()
	ConfigInstance.Sessions = sessions
	ConfigInstance.RetryCount = retryCount
	ConfigInstance.APIKey = os.Getenv("APIKEY")
	ConfigInstance.Proxy = os.Getenv("PROXY")
	ConfigInstance.IsIncognito = os.Getenv("IS_INCOGNITO") != "false"
	ConfigInstance.MaxChatHistoryLength = maxLen
	ConfigInstance.NoRolePrefix = os.Getenv("NO_ROLE_PREFIX") == "true"
	ConfigInstance.SearchResultCompatible = os.Getenv("SEARCH_RESULT_COMPATIBLE") == "true"
	ConfigInstance.PromptForFile = promptForFile
	ConfigInstance.IgnoreSerchResult = os.Getenv("IGNORE_SEARCH_RESULT") == "true"
	ConfigInstance.IgnoreModelMonitoring = os.Getenv("IGNORE_MODEL_MONITORING") == "true"
	ConfigInstance.IsMaxSubscribe = os.Getenv("IS_MAX_SUBSCRIBE") == "true"
	ConfigInstance.Timezone = os.Getenv("TIMEZONE") // Fix #7
	ConfigInstance.RwMutex.Unlock()
	// #4 fix: reset round-robin index after session pool replacement
	Sr.ResetIndex()
	// #4 fix: rebuild ResponseModels so /v1/models reflects updated IsMaxSubscribe
	buildResponseModels()
	logger.Info(fmt.Sprintf("Config reloaded: sessions=%d", retryCount))
}
