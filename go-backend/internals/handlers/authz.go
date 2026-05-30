package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func currentUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return 0, false
	}

	id, ok := userID.(int)
	if !ok || id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return 0, false
	}

	return id, true
}

func userCanAccessGroup(c *gin.Context, groupID interface{}) (bool, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return false, false
	}

	var canAccess bool
	err := db.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1
			FROM groups g
			JOIN users u ON u.id = $2
			LEFT JOIN members m ON m.group_id = g.id AND m.phone = u.phone
			WHERE g.id = $1
			  AND (g.created_by = u.id OR m.id IS NOT NULL)
		)
	`, groupID, userID).Scan(&canAccess)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return false, false
	}
	if !canAccess {
		c.JSON(http.StatusForbidden, gin.H{"message": "You do not have access to this group"})
		return false, false
	}

	return true, true
}
