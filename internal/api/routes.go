package api

import (
	"github.com/gin-gonic/gin"
	analyticsRepo "github.com/hafiztri123/document-api/internal/analytics/repository"
	analyticsService "github.com/hafiztri123/document-api/internal/analytics/service"
	authController "github.com/hafiztri123/document-api/internal/auth/controller"
	authRepository "github.com/hafiztri123/document-api/internal/auth/repository"
	authService "github.com/hafiztri123/document-api/internal/auth/service"
	docController "github.com/hafiztri123/document-api/internal/document/controller"
	docRepository "github.com/hafiztri123/document-api/internal/document/repository"
	docService "github.com/hafiztri123/document-api/internal/document/service"
	wsController "github.com/hafiztri123/document-api/internal/ws/controller"
	wsRepository "github.com/hafiztri123/document-api/internal/ws/repository"
	wsService "github.com/hafiztri123/document-api/internal/ws/service"
	"github.com/hafiztri123/document-api/internal/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

)


func SetupRoutes(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, logger *zap.Logger) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// API routes
	api := router.Group("/api/v1")

	// Repositories
	authRepo := authRepository.NewAuthRepository(db)
	docRepo := docRepository.NewDocumentRepository(db, logger)
	analyticsRepo := analyticsRepo.NewAnalyticsRepository(db, logger)
	wsRepo := wsRepository.NewWSRepository(logger)

	// Services
	authSvc := authService.NewAuthService(authRepo, redisClient, logger)
	analyticsService := analyticsService.NewAnalyticsService(analyticsRepo, logger)
	docSvc := docService.NewDocumentService(docRepo, authRepo, analyticsRepo, logger)
	wsSvc := wsService.NewWSService(wsRepo, docRepo, logger)

	// Controllers
	authCtrl := authController.NewAuthController(authSvc, logger)
	docCtrl := docController.NewDocumentController(docSvc, logger)
	wsCtrl := wsController.NewWSController(wsSvc, authSvc, logger)

	// Auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/login", authCtrl.Login)
		auth.POST("/refresh", authCtrl.RefreshToken)
		auth.POST("/logout", authCtrl.Logout)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(authSvc))
	{
		// Document routes
		docs := protected.Group("/documents")
		{
			docs.POST("", docCtrl.CreateDocument)
			docs.GET("", docCtrl.GetDocuments)
			docs.GET("/:id", docCtrl.GetDocumentByID)
			docs.PUT("/:id", docCtrl.UpdateDocument)
			docs.DELETE("/:id", docCtrl.DeleteDocument)

			// Document history
			docs.GET("/:id/history", docCtrl.GetDocumentHistory)
			docs.POST("/:id/history/:version", docCtrl.RestoreDocumentVersion)

			// Collaboration
			docs.POST("/:id/share", docCtrl.ShareDocument)
			docs.PUT("/:id/share/:user_id", docCtrl.UpdateCollaboratorPermission)
			docs.DELETE("/:id/share/:user_id", docCtrl.RemoveCollaborator)

			// Analytics
			docs.GET("/:id/analytics", docCtrl.GetDocumentAnalytics)
		}

		// User analytics
		protected.GET("/users/me/analytics", docCtrl.GetUserAnalytics)
	}

	// WebSocket endpoint
	router.GET("/ws/documents/:id", wsCtrl.HandleWebSocket)
}