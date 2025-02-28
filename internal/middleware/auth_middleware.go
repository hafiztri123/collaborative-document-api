package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hafiztri123/document-api/internal/auth/service"
)

func AuthMiddleware(authService service.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code": "unauthorized",
					"message": "Missing authorization header",
				},
			})
			ctx.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code": "unauthorized",
					"message": "Invalid authorization header format",
				},
			})
			ctx.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code": "unauthorized",
					"message": "Invalid or expired token",
				},
			})
			ctx.Abort()
			return
		}

		ctx.Set("userID", claims.UserID)
		ctx.Set("userEmail", claims.Email)
		ctx.Next()


	}
}