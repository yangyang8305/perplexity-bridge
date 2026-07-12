package router

import (
	"pplx2api/middleware"
	"pplx2api/service"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.Use(middleware.CORSMiddleware())

	// UI and public health -- no auth required
	r.GET("/", service.DashboardHandler)
	r.GET("/health", service.HealthCheckHandler)
	r.GET("/status", service.StatusHandler)

	// API routes -- require Bearer token when APIKEY is set
	api := r.Group("/")
	api.Use(middleware.AuthMiddleware())
	{
		api.POST("/v1/chat/completions", service.ChatCompletionsHandler)
		api.GET("/v1/models", service.ModelsHandler)
		api.POST("/admin/refresh", service.AdminRefreshHandler)
		api.POST("/admin/reload", service.AdminReloadHandler)
		hf := api.Group("/hf/v1")
		{
			hf.POST("/chat/completions", service.ChatCompletionsHandler)
			hf.GET("/models", service.ModelsHandler)
		}
	}
}
