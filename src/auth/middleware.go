package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates middleware for API token authentication
func (s *Service) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from header
		tokenString, err := ExtractTokenFromHeader(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "unauthorized: " + err.Error(),
			})
			c.Abort()
			return
		}

		// Validate token
		err = s.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "unauthorized: invalid token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
