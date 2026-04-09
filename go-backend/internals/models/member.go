package models

type Member struct {
	ID      int    `json:"id"`
	GroupID int    `json:"groupId"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
}