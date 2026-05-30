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
	creatorIDInt, ok := currentUserID(c)
	if !ok {
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

// GetUserGroups handles GET /api/groups/mine.
func GetUserGroups(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()

	groupRows, err := db.Query(ctx, `
		SELECT DISTINCT g.id, g.name, g.created_by, g.created_at, g.updated_at
		FROM groups g
		JOIN users u ON u.id = $1
		LEFT JOIN members m ON m.group_id = g.id AND m.phone = u.phone
		WHERE g.created_by = u.id OR m.id IS NOT NULL
		ORDER BY g.created_at DESC
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
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
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

// GetGroupByID handles GET /api/groups/:groupId
func GetGroupByID(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	var group models.Group
	err := db.QueryRow(ctx,
		"SELECT id, name, created_by, created_at, updated_at FROM groups WHERE id=$1", groupID,
	).Scan(&group.ID, &group.Name, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Group not found"})
		return
	}

	// Fetch members
	memberRows, err := db.Query(ctx,
		"SELECT id, group_id, name, phone FROM members WHERE group_id=$1 ORDER BY name", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching members"})
		return
	}
	defer memberRows.Close()

	var members []models.Member
	for memberRows.Next() {
		var m models.Member
		if err := memberRows.Scan(&m.ID, &m.GroupID, &m.Name, &m.Phone); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading members"})
			return
		}
		members = append(members, m)
	}

	type GroupWithMembers struct {
		models.Group
		Members []models.Member `json:"members"`
	}

	c.JSON(http.StatusOK, GroupWithMembers{Group: group, Members: members})
}

// AddMember handles POST /api/groups/:groupId/members
func AddMember(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	var input struct {
		Name  string `json:"name" binding:"required"`
		Phone string `json:"phone" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Name and phone are required"})
		return
	}

	// Check if group exists
	var groupExists bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM groups WHERE id=$1)", groupID,
	).Scan(&groupExists)
	if err != nil || !groupExists {
		c.JSON(http.StatusNotFound, gin.H{"message": "Group not found"})
		return
	}

	// Check if member already exists
	var memberExists bool
	err = db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM members WHERE group_id=$1 AND phone=$2)", groupID, input.Phone,
	).Scan(&memberExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	if memberExists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Member already exists in this group"})
		return
	}

	// Add member
	var member models.Member
	err = db.QueryRow(ctx, `
		INSERT INTO members(group_id, name, phone)
		VALUES($1, $2, $3)
		RETURNING id, group_id, name, phone
	`, groupID, input.Name, input.Phone,
	).Scan(&member.ID, &member.GroupID, &member.Name, &member.Phone)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error adding member"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Member added successfully",
		"member":  member,
	})
}

// UpdateMember handles PUT /api/groups/:groupId/members/:memberId
func UpdateMember(c *gin.Context) {
	groupID := c.Param("groupId")
	memberID := c.Param("memberId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	var input struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	if input.Name == "" && input.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "At least one field must be provided"})
		return
	}

	// Check if member exists
	var memberExists bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM members WHERE id=$1 AND group_id=$2)", memberID, groupID,
	).Scan(&memberExists)
	if err != nil || !memberExists {
		c.JSON(http.StatusNotFound, gin.H{"message": "Member not found"})
		return
	}

	// Update member
	var member models.Member
	query := "UPDATE members SET "
	args := []interface{}{}
	argCount := 1

	if input.Name != "" {
		query += fmt.Sprintf("name=$%d, ", argCount)
		args = append(args, input.Name)
		argCount++
	}
	if input.Phone != "" {
		query += fmt.Sprintf("phone=$%d, ", argCount)
		args = append(args, input.Phone)
		argCount++
	}

	query = query[:len(query)-2] // Remove trailing comma and space
	query += fmt.Sprintf(" WHERE id=$%d AND group_id=$%d RETURNING id, group_id, name, phone", argCount, argCount+1)
	args = append(args, memberID, groupID)

	err = db.QueryRow(ctx, query, args...).Scan(&member.ID, &member.GroupID, &member.Name, &member.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member updated successfully",
		"member":  member,
	})
}

// DeleteMember handles DELETE /api/groups/:groupId/members/:memberId
func DeleteMember(c *gin.Context) {
	groupID := c.Param("groupId")
	memberID := c.Param("memberId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	// Check if member exists
	var memberExists bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM members WHERE id=$1 AND group_id=$2)", memberID, groupID,
	).Scan(&memberExists)
	if err != nil || !memberExists {
		c.JSON(http.StatusNotFound, gin.H{"message": "Member not found"})
		return
	}

	// Delete member
	_, err = db.Exec(ctx, "DELETE FROM members WHERE id=$1", memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member deleted successfully"})
}

// UpdateGroup handles PUT /api/groups/:groupId
func UpdateGroup(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	var input struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Group name is required"})
		return
	}

	var group models.Group
	err := db.QueryRow(ctx, `
		UPDATE groups SET name=$1, updated_at=NOW()
		WHERE id=$2
		RETURNING id, name, created_by, created_at, updated_at
	`, input.Name, groupID,
	).Scan(&group.ID, &group.Name, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Group updated successfully",
		"group":   group,
	})
}

// DeleteGroup handles DELETE /api/groups/:groupId
func DeleteGroup(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	// Check if group exists
	var groupExists bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM groups WHERE id=$1)", groupID,
	).Scan(&groupExists)
	if err != nil || !groupExists {
		c.JSON(http.StatusNotFound, gin.H{"message": "Group not found"})
		return
	}

	// Delete all members first
	_, err = db.Exec(ctx, "DELETE FROM members WHERE group_id=$1", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting members"})
		return
	}

	// Delete the group
	_, err = db.Exec(ctx, "DELETE FROM groups WHERE id=$1", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}
