package models

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // never serialize password in JSON responses
	Phone    string `json:"phone"`
}