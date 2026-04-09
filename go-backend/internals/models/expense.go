package models

import "time"

type Expense struct {
	ID          int       `json:"id"`
	GroupID     int       `json:"groupId"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	PaidByID    int       `json:"paidById"`   // FK → members.id (not users.id)
	CreatedAt   time.Time `json:"createdAt"`
}