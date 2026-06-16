package auth

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"forum/internal/models"
)

func RegisterUser(db *sql.DB, username, email, password string) error {
	if username == "" || email == "" || password == "" {
		return fmt.Errorf("all fields are required")
	}

	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("email already registered")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("username already taken")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("could not hash password: %w", err)
	}

	_, err = db.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, string(hashed),
	)
	if err != nil {
		return fmt.Errorf("could not create user: %w", err)
	}

	return nil
}

func LoginUser(db *sql.DB, email, password string) (*models.User, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	var user models.User
	err := db.QueryRow(
		"SELECT id, username, email, password FROM users WHERE email = ?", email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	return &user, nil
}
