package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/models"
	"golang.org/x/crypto/bcrypt"
)

var db *pgxpool.Pool

// InitDB injects the database pool — call this once from main.go
func InitDB(pool *pgxpool.Pool) {
	db = pool
}

// RegisterUser handles POST /api/register
func RegisterUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request payload"})
		return
	}

	ctx := c.Request.Context()

	// Check duplicate email
	var exists bool
	if err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", user.Email,
	).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email already registered"})
		return
	}

	// Check duplicate phone
	if err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE phone=$1)", user.Phone,
	).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Phone number already registered"})
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error hashing password"})
		return
	}
	user.Password = string(hashed)

	// Insert user
	_, err = db.Exec(ctx,
		"INSERT INTO users(username, email, password, phone) VALUES($1, $2, $3, $4)",
		user.Username, user.Email, user.Password, user.Phone,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// LoginUser handles POST /api/login
func LoginUser(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request payload"})
		return
	}

	ctx := c.Request.Context()

	// Fetch user by email
	var user models.User
	err := db.QueryRow(ctx,
		"SELECT id, username, email, password, phone FROM users WHERE email=$1",
		input.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid credentials"})
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid credentials"})
		return
	}

	// Generate JWT
	token, err := generateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error generating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"phone":    user.Phone,
		},
	})
}

// GetAllUsers handles GET /api/users
func GetAllUsers(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.Query(ctx, "SELECT id, username, email, phone FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Phone); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error reading users"})
			return
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, users)
}

// generateJWT creates a signed JWT for the given userID
func generateJWT(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 12).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
