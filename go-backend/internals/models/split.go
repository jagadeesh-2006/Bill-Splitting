package models

type ExpenseSplit struct {
	ID         int     `json:"id"`
	ExpenseID  int     `json:"expenseId"`
	MemberID   int     `json:"memberId"`   // FK → members.id
	AmountOwed float64 `json:"amountOwed"`
}