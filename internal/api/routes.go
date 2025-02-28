package api

import (
	"github.com/gin-gonic/gin"
	authController "github.com/hafiztri123/document-api/internal/auth/controller"
	authRepo "github.com/hafiztri123/document-api/internal/auth/repository"
	authService "github.com/hafiztri123/document-api/internal/auth/service"
	"github.com/hafiztri123/document-api/internal/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)


func SetupRoutes(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, logger *zap.Logger) {

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	api := router.Group("/api/v1")

	authRepo := authRepo.NewAuthRepository(db)
	authService := authService.NewAuthService(authRepo, redisClient, logger)
	authCtrl := authController.NewAuthController(authService, logger)

	auth := api.Group("/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/login", authCtrl.Login)
		auth.POST("/refresh", authCtrl.RefreshToken)
		auth.POST("/logout", authCtrl.Logout)
	}

	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		
	}
}