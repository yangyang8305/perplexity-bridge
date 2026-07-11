package service

import (
	"net/http"
	"pplx2api/config"
	"pplx2api/job"
	"time"

	"github.com/gin-gonic/gin"
)

func DashboardHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", dashboardHTML)
}

func StatusHandler(c *gin.Context) {
	config.ConfigInstance.RwMutex.RLock()
	sessionCount := len(config.ConfigInstance.Sessions)
	config.ConfigInstance.RwMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"sessions":     sessionCount,
		"address":      config.ConfigInstance.Address,
		"is_incognito": config.ConfigInstance.IsIncognito,
		"is_max":       config.ConfigInstance.IsMaxSubscribe,
		"model_count":  len(config.ModelMap),
		"time":         time.Now().UTC().Format(time.RFC3339),
	})
}

func AdminRefreshHandler(c *gin.Context) {
	su := job.GetSessionUpdater(24 * time.Hour)
	go su.TriggerNow()
	c.JSON(http.StatusOK, gin.H{"message": "Cookie refresh triggered"})
}

func AdminReloadHandler(c *gin.Context) {
	config.Reload()
	c.JSON(http.StatusOK, gin.H{"message": "Config reloaded"})
}
