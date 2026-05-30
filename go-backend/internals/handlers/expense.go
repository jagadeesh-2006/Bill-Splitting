package handlers

import (
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/models"
)

// AddExpense handles POST /api/expenses
func AddExpense(c *gin.Context) {
	var input struct {
		// name type and json tag is used for binding and validation
		GroupID      int     `json:"groupId"     binding:"required"`
		Description  string  `json:"description" binding:"required"`
		Amount       float64 `json:"amount"      binding:"required,gt=0"`
		PaidByID     int     `json:"paidById"    binding:"required"`
		SplitBetween []struct {
			MemberID   int     `json:"memberId"   binding:"required"`
			AmountOwed float64 `json:"amountOwed" binding:"required,gt=0"`
		} `json:"splitBetween" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "All fields including splitBetween are required"})
		return
	}

	ctx := c.Request.Context()

	// Verify group exists
	var groupExists bool
	if err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM groups WHERE id=$1)", input.GroupID,
	).Scan(&groupExists); err != nil || !groupExists {
		c.JSON(http.StatusNotFound, gin.H{"message": "Group not found"})
		return
	}
	if _, ok := userCanAccessGroup(c, input.GroupID); !ok {
		return
	}

	// isMember checks if a member belongs to this group
	isMember := func(memberID int) (bool, error) {
		var exists bool
		err := db.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM members WHERE id=$1 AND group_id=$2)",
			memberID, input.GroupID,
		).Scan(&exists)
		return exists, err
	}

	// Payer must be in the group
	if ok, err := isMember(input.PaidByID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	} else if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Payer is not a member of this group"})
		return
	}

	// All split members must be in the group
	for _, s := range input.SplitBetween {
		if ok, err := isMember(s.MemberID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
			return
		} else if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"message": "A member in splitBetween does not belong to this group"})
			return
		}
	}

	// Split amounts must sum to total (±0.01 tolerance)
	var total float64
	for _, s := range input.SplitBetween {
		total += s.AmountOwed
	}
	if math.Abs(total-input.Amount) > 0.01 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Sum of split amounts must equal total expense amount"})
		return
	}

	// Insert expense
	var expense models.Expense
	err := db.QueryRow(ctx, `
		INSERT INTO expenses(group_id, description, amount, paid_by_id)
		VALUES($1, $2, $3, $4)
		RETURNING id, group_id, description, amount, paid_by_id, created_at
	`, input.GroupID, input.Description, input.Amount, input.PaidByID,
	).Scan(&expense.ID, &expense.GroupID, &expense.Description, &expense.Amount, &expense.PaidByID, &expense.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving expense"})
		return
	}

	// Insert splits
	var splits []models.ExpenseSplit
	for _, s := range input.SplitBetween {
		var split models.ExpenseSplit
		err = db.QueryRow(ctx, `
			INSERT INTO expense_splits(expense_id, member_id, amount_owed)
			VALUES($1, $2, $3)
			RETURNING id, expense_id, member_id, amount_owed
		`, expense.ID, s.MemberID, s.AmountOwed,
		).Scan(&split.ID, &split.ExpenseID, &split.MemberID, &split.AmountOwed)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving expense split"})
			return
		}
		splits = append(splits, split)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Expense added",
		"expense": expense,
		"splits":  splits,
	})
}

// GetExpensesByGroup handles GET /api/expenses/:groupId
func GetExpensesByGroup(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	expenseRows, err := db.Query(ctx, `
		SELECT id, group_id, description, amount, paid_by_id, created_at
		FROM expenses WHERE group_id = $1 ORDER BY created_at DESC
	`, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer expenseRows.Close()

	type ExpenseWithSplits struct {
		models.Expense
		Splits []models.ExpenseSplit `json:"splits"`
	}

	var result []ExpenseWithSplits
	for expenseRows.Next() {
		var e models.Expense
		if err := expenseRows.Scan(&e.ID, &e.GroupID, &e.Description, &e.Amount, &e.PaidByID, &e.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading expenses"})
			return
		}

		splitRows, err := db.Query(ctx,
			"SELECT id, expense_id, member_id, amount_owed FROM expense_splits WHERE expense_id=$1", e.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching splits"})
			return
		}

		var splits []models.ExpenseSplit
		for splitRows.Next() {
			var s models.ExpenseSplit
			if err := splitRows.Scan(&s.ID, &s.ExpenseID, &s.MemberID, &s.AmountOwed); err != nil {
				splitRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading splits"})
				return
			}
			splits = append(splits, s)
		}
		splitRows.Close()

		result = append(result, ExpenseWithSplits{Expense: e, Splits: splits})
	}

	c.JSON(http.StatusOK, result)
}

// GetBalances handles GET /api/groups/:groupId/balances
// Net balance per member: positive = owed to them, negative = they owe
func GetBalances(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	memberRows, err := db.Query(ctx,
		"SELECT id, name FROM members WHERE group_id=$1", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer memberRows.Close()

	type memberBalance struct {
		Name   string
		Amount float64
	}
	balanceMap := map[int]*memberBalance{} // memberID -> balance info
	for memberRows.Next() {
		var id int
		var name string
		if err := memberRows.Scan(&id, &name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading members"})
			return
		}
		balanceMap[id] = &memberBalance{Name: name}
	}

	// What each member paid (they are owed this)
	paidRows, err := db.Query(ctx,
		"SELECT paid_by_id, SUM(amount) FROM expenses WHERE group_id=$1 GROUP BY paid_by_id", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer paidRows.Close()
	for paidRows.Next() {
		var id int
		var total float64
		if err := paidRows.Scan(&id, &total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading paid totals"})
			return
		}
		if b, ok := balanceMap[id]; ok {
			b.Amount += total
		}
	}

	// What each member owes (their split share)
	owedRows, err := db.Query(ctx, `
		SELECT es.member_id, SUM(es.amount_owed)
		FROM expense_splits es
		JOIN expenses e ON e.id = es.expense_id
		WHERE e.group_id = $1 GROUP BY es.member_id
	`, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer owedRows.Close()
	for owedRows.Next() {
		var id int
		var total float64
		if err := owedRows.Scan(&id, &total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading owed totals"})
			return
		}
		if b, ok := balanceMap[id]; ok {
			b.Amount -= total
		}
	}

	// Factor in settlements
	settlementRows, err := db.Query(ctx,
		"SELECT from_member, to_member, amount FROM settlements WHERE group_id=$1", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer settlementRows.Close()
	for settlementRows.Next() {
		var from, to int
		var amount float64
		if err := settlementRows.Scan(&from, &to, &amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading settlements"})
			return
		}
		if b, ok := balanceMap[from]; ok {
			b.Amount += amount
		}
		if b, ok := balanceMap[to]; ok {
			b.Amount -= amount
		}
	}

	var balances []models.Balance
	for memberID, b := range balanceMap {
		balances = append(balances, models.Balance{
			MemberID:   memberID,
			MemberName: b.Name,
			Amount:     math.Round(b.Amount*100) / 100,
		})
	}

	c.JSON(http.StatusOK, balances)
}

// SettleUp handles POST /api/groups/:groupId/settle
func SettleUp(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}

	var input struct {
		FromMember int     `json:"fromMember" binding:"required"`
		ToMember   int     `json:"toMember"   binding:"required"`
		Amount     float64 `json:"amount"     binding:"required,gt=0"`
		Note       string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "fromMember, toMember and a positive amount are required"})
		return
	}
	if input.FromMember == input.ToMember {
		c.JSON(http.StatusBadRequest, gin.H{"message": "fromMember and toMember cannot be the same person"})
		return
	}

	ctx := c.Request.Context()

	// Both members must belong to this group
	for _, memberID := range []int{input.FromMember, input.ToMember} {
		var exists bool
		if err := db.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM members WHERE id=$1 AND group_id=$2)",
			memberID, groupID,
		).Scan(&exists); err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"message": "One or both members do not belong to this group"})
			return
		}
	}

	var settlement models.Settlement
	err := db.QueryRow(ctx, `
		INSERT INTO settlements(group_id, from_member, to_member, amount, note)
		VALUES($1, $2, $3, $4, $5)
		RETURNING id, group_id, from_member, to_member, amount, note, paid_at
	`, groupID, input.FromMember, input.ToMember, input.Amount, input.Note,
	).Scan(
		&settlement.ID, &settlement.GroupID,
		&settlement.FromMember, &settlement.ToMember,
		&settlement.Amount, &settlement.Note, &settlement.PaidAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error recording settlement"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Settlement recorded",
		"settlement": settlement,
	})
}

// GetPaymentHistory handles GET /api/groups/:groupId/settlements
func GetPaymentHistory(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	rows, err := db.Query(ctx, `
		SELECT s.id, s.group_id,
		       s.from_member, fm.name AS from_name,
		       s.to_member,   tm.name AS to_name,
		       s.amount, s.note, s.paid_at
		FROM settlements s
		JOIN members fm ON fm.id = s.from_member
		JOIN members tm ON tm.id = s.to_member
		WHERE s.group_id = $1
		ORDER BY s.paid_at DESC
	`, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer rows.Close()

	type SettlementWithNames struct {
		models.Settlement
		FromName string `json:"fromName"`
		ToName   string `json:"toName"`
	}

	var history []SettlementWithNames
	for rows.Next() {
		var s SettlementWithNames
		if err := rows.Scan(
			&s.ID, &s.GroupID,
			&s.FromMember, &s.FromName,
			&s.ToMember, &s.ToName,
			&s.Amount, &s.Note, &s.PaidAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading payment history"})
			return
		}
		history = append(history, s)
	}

	c.JSON(http.StatusOK, history)
}

// UpdateExpense handles PUT /api/expenses/:expenseId
func UpdateExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	ctx := c.Request.Context()

	var input struct {
		Description  string  `json:"description"`
		Amount       float64 `json:"amount"`
		PaidByID     int     `json:"paidById"`
		SplitBetween []struct {
			MemberID   int     `json:"memberId"   binding:"required"`
			AmountOwed float64 `json:"amountOwed" binding:"required,gt=0"`
		} `json:"splitBetween"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	// Get existing expense to validate group
	var groupID int
	err := db.QueryRow(ctx,
		"SELECT group_id FROM expenses WHERE id=$1", expenseID,
	).Scan(&groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Expense not found"})
		return
	}
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}

	// Update expense fields if provided
	if input.Description != "" || input.Amount > 0 || input.PaidByID > 0 {
		query := "UPDATE expenses SET "
		args := []interface{}{}
		argCount := 1

		if input.Description != "" {
			query += fmt.Sprintf("description=$%d, ", argCount)
			args = append(args, input.Description)
			argCount++
		}
		if input.Amount > 0 {
			query += fmt.Sprintf("amount=$%d, ", argCount)
			args = append(args, input.Amount)
			argCount++
		}
		if input.PaidByID > 0 {
			// Verify payer is in group
			var exists bool
			db.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM members WHERE id=$1 AND group_id=$2)",
				input.PaidByID, groupID,
			).Scan(&exists)
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Payer is not a member of this group"})
				return
			}
			query += fmt.Sprintf("paid_by_id=$%d, ", argCount)
			args = append(args, input.PaidByID)
			argCount++
		}

		// Remove trailing comma and add WHERE clause
		if len(query) > 22 { // More than just "UPDATE expenses SET "
			query = query[:len(query)-2] + fmt.Sprintf(" WHERE id=$%d", argCount)
			args = append(args, expenseID)

			_, err := db.Exec(ctx, query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating expense"})
				return
			}
		}
	}

	// Update splits if provided
	if len(input.SplitBetween) > 0 {
		// Verify amounts sum to total
		var total float64
		err := db.QueryRow(ctx, "SELECT amount FROM expenses WHERE id=$1", expenseID).Scan(&total)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching expense"})
			return
		}

		var splitTotal float64
		for _, s := range input.SplitBetween {
			splitTotal += s.AmountOwed
		}
		if math.Abs(splitTotal-total) > 0.01 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Sum of split amounts must equal total expense amount"})
			return
		}

		// Delete old splits
		_, err = db.Exec(ctx, "DELETE FROM expense_splits WHERE expense_id=$1", expenseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating splits"})
			return
		}

		// Insert new splits
		for _, s := range input.SplitBetween {
			// Verify member is in group
			var exists bool
			db.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM members WHERE id=$1 AND group_id=$2)",
				s.MemberID, groupID,
			).Scan(&exists)
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"message": "A member in splitBetween does not belong to this group"})
				return
			}

			_, err = db.Exec(ctx, `
				INSERT INTO expense_splits(expense_id, member_id, amount_owed)
				VALUES($1, $2, $3)
			`, expenseID, s.MemberID, s.AmountOwed)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving expense split"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense updated successfully"})
}

// DeleteExpense handles DELETE /api/expenses/:expenseId
func DeleteExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	ctx := c.Request.Context()

	// Check if expense exists
	var groupID int
	err := db.QueryRow(ctx,
		"SELECT group_id FROM expenses WHERE id=$1", expenseID,
	).Scan(&groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Expense not found"})
		return
	}
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}

	// Delete splits first
	_, err = db.Exec(ctx, "DELETE FROM expense_splits WHERE expense_id=$1", expenseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting expense splits"})
		return
	}

	// Delete expense
	_, err = db.Exec(ctx, "DELETE FROM expenses WHERE id=$1", expenseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting expense"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted successfully"})
}

// DeleteSettlement handles DELETE /api/groups/:groupId/settlements/:settlementId
func DeleteSettlement(c *gin.Context) {
	groupID := c.Param("groupId")
	settlementID := c.Param("settlementId")
	if _, ok := userCanAccessGroup(c, groupID); !ok {
		return
	}
	ctx := c.Request.Context()

	// Check if settlement exists and belongs to this group
	var settlementExists bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM settlements WHERE id=$1 AND group_id=$2)", settlementID, groupID,
	).Scan(&settlementExists)
	if err != nil || !settlementExists {
		c.JSON(http.StatusNotFound, gin.H{"message": "Settlement not found"})
		return
	}

	// Delete settlement
	_, err = db.Exec(ctx, "DELETE FROM settlements WHERE id=$1", settlementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting settlement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settlement deleted successfully"})
}
