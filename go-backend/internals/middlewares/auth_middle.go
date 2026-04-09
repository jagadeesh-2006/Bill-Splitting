package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates the JWT token from the Authorization header
// and injects the authenticated userID into the Gin context.
//
// Usage in routes.go:
//
//	import "github.com/jagadeesh-2006/Bill-Splitting/internals/middlewares"
//
//	auth := r.Group("/api")
//	auth.Use(middlewares.AuthMiddleware())
//
// Handlers read the userID via:
//
//	userID, _ := c.Get("userID")
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Authorization header missing",
			})
			return
		}

		// Must be "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Authorization header format must be: Bearer <token>",
			})
			return
		}

		tokenStr := parts[1]
		secretKey := []byte(os.Getenv("JWT_SECRET"))

		// Parse and validate — enforce HMAC to prevent alg:none attack
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid or expired token",
			})
			return
		}

		// Extract user_id claim — JWT stores numbers as float64
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid token claims",
			})
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid token: missing user_id",
			})
			return
		}

		// Inject into Gin context — handlers read with c.GetInt("userID")
		c.Set("userID", int(userIDFloat))
		c.Next()
	}
}