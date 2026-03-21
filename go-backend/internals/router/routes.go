import(
	"github.com/gin-gonic/gin"
	"github.com/jagadeesh-2006/Bill-Splitting/internals/handlers"
)

func setuproutes(r *gin.Engine) {
	//public
	r.POST("/register", handlers.RegisterHandler) 
	r.POST("/login", handlers.LoginHandler)
	

	//protected routes
	auth := r.Group("/")
	auth.Use(handlers.AuthMiddleware()) 

	//group routes

	auth.POST("/groups/create", handlers.CreateGroup) 
	auth.GET("/groups/user/user_id", handlers.Getusergroup) 

	//expense routes

	auth.POST("/expenses/create", handlers.AddExpense) 
	auth.GET("/expenses/group/group_id", handlers.GetGroupExpenses) 
	//user routes

	auth.GET("/users",handlers.getAllusers)


}