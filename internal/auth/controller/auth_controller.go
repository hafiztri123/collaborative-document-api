package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hafiztri123/document-api/internal/auth/service"
	"github.com/hafiztri123/document-api/internal/user/model"
	"go.uber.org/zap"
)

type Controller interface {
	Register(ctx *gin.Context)
	Login(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	Logout(ctx *gin.Context)
	GetProfile(ctx *gin.Context)
}

type authController struct {
	service service.Service
	logger  *zap.Logger
}

func NewAuthController(service service.Service, logger *zap.Logger) Controller {
	return &authController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *authController) Register(ctx *gin.Context) {
	var req model.UserRegistration

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}

	user, err := ctrl.service.Register(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			ctx.JSON(http.StatusConflict, gin.H{"error": gin.H{
				"code":    "conflict",
				"message": "User already exists with this email",
			}})
			return
		}

		ctrl.logger.Error("Error registering user", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to register user",
		}})
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

func (ctrl *authController) Login(ctx *gin.Context) {
	var req model.UserLogin

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}

	tokens, err := ctrl.service.Login(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "unauthorized",
				"message": "Invalid email or password",
			}})
			return
		}

		ctrl.logger.Error("Error logging in user", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to login",
		}})
		return
	}

	ctx.JSON(http.StatusOK, tokens)
}

func (ctrl *authController) RefreshToken(ctx *gin.Context) {
	var req model.RefreshTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}

	tokens, err := ctrl.service.RefreshToken(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "unauthorized",
				"message": "Invalid or expired refresh token",
			}})
			return
		}

		ctrl.logger.Error("Error refreshing token", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to refresh token",
		}})
		return
	}

	ctx.JSON(http.StatusOK, tokens)
}

func (ctrl *authController) Logout(ctx *gin.Context) {
	var req model.RefreshTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}

	if err := ctrl.service.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		ctrl.logger.Error("Error logging out user", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to logout",
		}})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (ctrl *authController) GetProfile(ctx *gin.Context) {
	userID, ok  := ctx.Get("userID")
	if !ok {
		ctrl.logger.Error("Error getting userID")
		ctx.JSON(http.StatusNotFound, gin.H{
			"code": "not_found",
			"message": "Failed to get user ID",
		})
		return		
	}


	user, err := ctrl.service.GetProfile(context.Background(), userID.(uuid.UUID))
	if err != nil {
		ctrl.logger.Error("Error getting profile")
		ctx.JSON(http.StatusNotFound, gin.H{
			"code": "not_found",
			"message": "Failed to get profile",
		})
		return
	}

	ctx.JSON(http.StatusOK, user)
}