package router

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors" // correct import — NOT gin/cors
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		panic(err)
	}

	// CORS 
	// Read allowed origins from env, e.g. "http://localhost:3000,https://myapp.com".
	// Falls back to common local static-server origins for this plain HTML frontend.
	originsEnv := os.Getenv("CORS_ORIGINS")
	var allowedOrigins []string
	allowFileOrigin := false
	if originsEnv == "" {
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:5500",
			"http://127.0.0.1:5500",
		}
		allowFileOrigin = true
	} else {
		for _, o := range strings.Split(originsEnv, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(o))
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowOriginFunc: func(origin string) bool {
			if allowFileOrigin && origin == "null" {
				return true
			}
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		// OPTIONS preflight is handled automatically by gin-contrib/cors
	}))

	//  ROUTES 
	// CORS must be applied BEFORE routes are registered
	SetupRoutes(r)

	return r
}
