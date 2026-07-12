package service

import (
	"net/http"
	"pplx2api/config"
	"pplx2api/job"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// refreshRunning is 1 when a refresh goroutine is already in progress.
// #5 fix: prevents multiple concurrent updateAllSessions() when AdminRefreshHandler
// is called repeatedly in quick succession.
var refreshRunning int32

func DashboardHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", dashboardHTML)
}

func StatusHandler(c *gin.Context) {
	config.ConfigInstance.RwMutex.RLock()
	sessionCount := len(config.ConfigInstance.Sessions)
	address := config.ConfigInstance.Address
	isIncognito := config.ConfigInstance.IsIncognito
	isMax := config.ConfigInstance.IsMaxSubscribe
	config.ConfigInstance.RwMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"sessions":     sessionCount,
		"address":      address,
		"is_incognito": isIncognito,
		"is_max":       isMax,
		"model_count":  len(config.ModelMap),
		"time":         time.Now().UTC().Format(time.RFC3339),
	})
}

func AdminRefreshHandler(c *gin.Context) {
	if !atomic.CompareAndSwapInt32(&refreshRunning, 0, 1) {
		c.JSON(http.StatusTooManyRequests, gin.H{"message": "Refresh already in progress"})
		return
	}
	su := job.GetSessionUpdater(24 * time.Hour)
	go func() {
		defer atomic.StoreInt32(&refreshRunning, 0)
		su.TriggerNow()
	}()
	c.JSON(http.StatusOK, gin.H{"message": "Cookie refresh triggered"})
}

func AdminReloadHandler(c *gin.Context) {
	config.Reload()
	c.JSON(http.StatusOK, gin.H{"message": "Config reloaded"})
}

// ensure sync is imported (used transitively; explicit blank import avoids lint warning)
var _ sync.Mutex
