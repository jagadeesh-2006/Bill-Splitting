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

	// PROTECTED ROUTES 
	auth := r.Group("/api")
	auth.Use(middlewares.AuthMiddleware())
	{
		// Groups 
		auth.GET("/groups/mine", handlers.GetUserGroups)
		auth.GET("/groups/:groupId", handlers.GetGroupByID)
		auth.POST("/groups", handlers.CreateGroup)
		auth.PUT("/groups/:groupId", handlers.UpdateGroup)
		auth.DELETE("/groups/:groupId", handlers.DeleteGroup)

		// Members
		auth.GET("/groups/:groupId/members", handlers.GetGroupMembers)
		auth.POST("/groups/:groupId/members", handlers.AddMember)
		auth.PUT("/groups/:groupId/members/:memberId", handlers.UpdateMember)
		auth.DELETE("/groups/:groupId/members/:memberId", handlers.DeleteMember)

		// Expenses
		auth.GET("/expenses/:groupId", handlers.GetExpensesByGroup)
		auth.POST("/expenses", handlers.AddExpense)
		auth.PUT("/expenses/:expenseId", handlers.UpdateExpense)
		auth.DELETE("/expenses/:expenseId", handlers.DeleteExpense)

		// Settle up
		auth.GET("/groups/:groupId/balances", handlers.GetBalances)
		auth.GET("/groups/:groupId/settlements", handlers.GetPaymentHistory)
		auth.POST("/groups/:groupId/settle", handlers.SettleUp)
		auth.DELETE("/groups/:groupId/settlements/:settlementId", handlers.DeleteSettlement)

		// Users
		auth.GET("/users", handlers.GetAllUsers)
	}
}
