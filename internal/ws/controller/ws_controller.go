package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	
	authService "github.com/hafiztri123/document-api/internal/auth/service"
	wsService "github.com/hafiztri123/document-api/internal/ws/service"
)

type Controller interface {
	HandleWebSocket(c *gin.Context)
}

type wsController struct {
	wsService   wsService.Service
	authService authService.Service
	logger      *zap.Logger
	upgrader    websocket.Upgrader
}

func NewWSController(wsService wsService.Service, authService authService.Service, logger *zap.Logger) Controller {
	return &wsController{
		wsService:   wsService,
		authService: authService,
		logger:      logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// In production, this should be restricted
				return true
			},
		},
	}
}

func (ctrl *wsController) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "Missing token",
		}})
		return
	}
	
	claims, err := ctrl.authService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "Invalid or expired token",
		}})
		return
	}
	
	conn, err := ctrl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		ctrl.logger.Error("Failed to upgrade connection to WebSocket", zap.Error(err))
		return
	}
	
	ctrl.wsService.HandleConnection(conn, claims.UserID, claims.Email)
}