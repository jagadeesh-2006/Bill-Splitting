package router

import(
	"os"
	
	"github.com/gin-gonic/gin" 
	"github.com/gin-gonic/gin/cors" 
	"github.com/jagadeesh-2006/Bill-Splitting/internals/router" 

func setupRouter() *gin.Engine {
	r := gin.Default() //create a new instance of the Gin router with default middleware (such as logging and recovery).
	origins := os.Getenv("CORS_ORIGINS") 

	config:= cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}
	setuproutes(r)
	r.Use(cors.New(config))

	return r
}