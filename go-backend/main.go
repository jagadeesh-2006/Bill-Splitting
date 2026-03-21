import (
	"os"
	"config"
)

func main(){
	config.Init()

	r := setupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" 
	}
	r.Run(":" + port) 
	fmt.Printf("server is running on port %d", port)

}