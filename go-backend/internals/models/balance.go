package models

type Balance struct {
	MemberID   int     `json:"memberId"`
	MemberName string  `json:"memberName"`
	Amount     float64 `json:"amount"` // positive = owed to them, negative = they owe
}
 