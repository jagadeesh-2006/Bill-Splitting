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

	// ── CORS ──────────────────────────────────────────────────────────────────
	// Read allowed origins from env, e.g. "http://localhost:3000,https://myapp.com"
	// Falls back to localhost:3000 for local dev
	originsEnv := os.Getenv("CORS_ORIGINS")
	var allowedOrigins []string
	if originsEnv == "" {
		allowedOrigins = []string{"http://localhost:3000"}
	} else {
		for _, o := range strings.Split(originsEnv, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(o))
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		// OPTIONS preflight is handled automatically by gin-contrib/cors
	}))

	// ── ROUTES ────────────────────────────────────────────────────────────────
	// CORS must be applied BEFORE routes are registered
	SetupRoutes(r)

	return r
}
