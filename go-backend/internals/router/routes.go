package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/handlers"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/middlewares"
)

func SetupRoutes(r *gin.Engine) {

	//  PUBLIC ROUTES 
	r.POST("/api/register", handlers.RegisterUser)
	r.POST("/api/login", handlers.LoginUser)

	// Read-only — members have no accounts so these are public
	r.GET("/api/groups/:groupId/members", handlers.GetGroupMembers)
	r.GET("/api/expenses/:groupId", handlers.GetExpensesByGroup)
	r.GET("/api/groups/:groupId/balances", handlers.GetBalances)
	r.GET("/api/groups/:groupId/settlements", handlers.GetPaymentHistory)

	// PROTECTED ROUTES 
	// middlewares.AuthMiddleware() validates JWT and injects userID into context
	auth := r.Group("/api")
	auth.Use(middlewares.AuthMiddleware())
	{
		// Groups
		auth.POST("/groups", handlers.CreateGroup)
		auth.GET("/groups/creator/:userId", handlers.GetUserGroups)

		// Expenses
		auth.POST("/expenses", handlers.AddExpense)

		// Settle up
		auth.POST("/groups/:groupId/settle", handlers.SettleUp)

		// Users
		auth.GET("/users", handlers.GetAllUsers)
	}
}
