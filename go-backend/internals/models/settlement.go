package models
import "time"

type Settlement struct {
	ID         int       `json:"id"`
	GroupID    int       `json:"groupId"`
	FromMember int       `json:"fromMember"` // who paid
	ToMember   int       `json:"toMember"`   // who received
	Amount     float64   `json:"amount"`
	Note       string    `json:"note"`
	PaidAt     time.Time `json:"paidAt"`
}