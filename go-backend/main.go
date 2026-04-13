package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/config"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/handlers"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/router"
)

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// config.ConnectDB()
	handlers.InitDB(config.DB)

	r := router.SetupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		fmt.Printf("failed to run server: %v\n", err)
		return
	}
	fmt.Printf("server is running on port %s\n", port)
}
