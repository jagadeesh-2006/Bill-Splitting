package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/models"
)

// CreateGroup handles POST /api/groups
// Creator ID comes from JWT context set by middlewares.AuthMiddleware().
func CreateGroup(c *gin.Context) {
	// Read userID injected by AuthMiddleware — never trust the request body for this
	creatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	creatorIDInt, ok := creatorID.(int)
	if !ok || creatorIDInt == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var input struct {
		Name    string `json:"name" binding:"required"`
		Members []struct {
			Name  string `json:"name"  binding:"required"`
			Phone string `json:"phone" binding:"required"`
		} `json:"members" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Group name and at least one member are required"})
		return
	}

	ctx := c.Request.Context()

	// Verify the creator exists in users table
	var creatorExists bool
	if err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", creatorIDInt,
	).Scan(&creatorExists); err != nil || !creatorExists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Creator user not found"})
		return
	}

	// Insert the group
	var group models.Group
	err := db.QueryRow(ctx, `
		INSERT INTO groups(name, created_by)
		VALUES($1, $2)
		RETURNING id, name, created_by, created_at, updated_at
	`, input.Name, creatorIDInt,
	).Scan(&group.ID, &group.Name, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating group"})
		return
	}

	// Insert each member (name + phone, no user account needed)
	var members []models.Member
	//insert user by default when group is created
	// we need to get username and phn using creatorId frim usertable and insert that as member in the group
	var member models.Member
	err = db.QueryRow(ctx, `
		INSERT INTO members(group_id, name, phone)
		VALUES($1, (SELECT username FROM users WHERE id=$2), (SELECT phone FROM users WHERE id=$2))
		RETURNING id, group_id, name, phone
	`, group.ID, creatorIDInt,
	).Scan(&member.ID, &member.GroupID, &member.Name, &member.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error adding creator as member"})
		return
	}
	members = append(members, member)
	

	for _, m := range input.Members {
		var member models.Member
		err = db.QueryRow(ctx, `
			INSERT INTO members(group_id, name, phone)
			VALUES($1, $2, $3)
			RETURNING id, group_id, name, phone
		`, group.ID, m.Name, m.Phone,
		).Scan(&member.ID, &member.GroupID, &member.Name, &member.Phone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Error adding member: %s", m.Name)})
			return
		}
		members = append(members, member)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Group created",
		"group":   group,
		"members": members,
	})
}

// GetUserGroups handles GET /api/groups/creator/:userId
func GetUserGroups(c *gin.Context) {
	userID := c.Param("userId")
	ctx := c.Request.Context()

	groupRows, err := db.Query(ctx, `
		SELECT id, name, created_by, created_at, updated_at
		FROM groups WHERE created_by = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer groupRows.Close()

	type GroupWithMembers struct {
		models.Group
		Members []models.Member `json:"members"`
	}

	var groups []GroupWithMembers
	for groupRows.Next() {
		var g models.Group
		if err := groupRows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading groups"})
			return
		}

		memberRows, err := db.Query(ctx,
			"SELECT id, group_id, name, phone FROM members WHERE group_id=$1 ORDER BY name", g.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching members"})
			return
		}

		var members []models.Member
		for memberRows.Next() {
			var m models.Member
			if err := memberRows.Scan(&m.ID, &m.GroupID, &m.Name, &m.Phone); err != nil {
				memberRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading members"})
				return
			}
			members = append(members, m)
		}
		memberRows.Close()

		groups = append(groups, GroupWithMembers{Group: g, Members: members})
	}

	c.JSON(http.StatusOK, groups)
}

// GetGroupMembers handles GET /api/groups/:groupId/members
func GetGroupMembers(c *gin.Context) {
	groupID := c.Param("groupId")
	ctx := c.Request.Context()

	rows, err := db.Query(ctx,
		"SELECT id, group_id, name, phone FROM members WHERE group_id=$1 ORDER BY name", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.ID, &m.GroupID, &m.Name, &m.Phone); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading members"})
			return
		}
		members = append(members, m)
	}

	c.JSON(http.StatusOK, members)
}
