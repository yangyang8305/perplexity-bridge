package main

import (
	"net/http"
	"pplx2api/config"
	"pplx2api/job"
	"pplx2api/logger"
	"pplx2api/router"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// A5 fix: gin.New() instead of gin.Default() to avoid stack-trace leaks via
	// the built-in recovery middleware. We install our own recovery that logs
	// without dumping full goroutine stacks to stdout.
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))
	r.Use(func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered: %v\n%s", err, debug.Stack())
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	})

	router.SetupRoutes(r)

	sessionUpdater := job.GetSessionUpdater(24 * time.Hour)
	sessionUpdater.Start()
	defer sessionUpdater.Stop()

	r.Run(config.ConfigInstance.Address)
}
