package rest

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/bentalebwael/faceit-users-service/internal/api/rest/middleware"
	"github.com/bentalebwael/faceit-users-service/internal/platform/ratelimiter"
)

// setupRouter configures all the routes and middleware for the API
func setupRouter(handler *Handler, limiter *ratelimiter.RateLimiter, logger *slog.Logger) *gin.Engine {
	router := gin.New()

	router.Use(
		gin.Recovery(),
		middleware.Logger(logger),
		middleware.RateLimit(limiter),
	)

	// Health check
	router.GET("/healthz", handler.Health)

	// API routes
	v1 := router.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("", handler.AddUser)
			users.GET("", handler.ListUsers)
			users.GET("/:id", handler.GetUser)
			users.PUT("/:id", handler.UpdateUser)
			users.DELETE("/:id", handler.DeleteUser)
		}
	}

	// Serve Swagger Documentation
	router.StaticFile("/swagger.yaml", "./doc/swagger.yaml")
	url := ginSwagger.URL("/swagger.yaml")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	return router
}
