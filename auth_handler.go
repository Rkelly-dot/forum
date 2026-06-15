package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// DB is set once in main.go and shared across handlers
var DB *sql.DB

// Sessions stores sessionToken -> userID in memory
// In production this would be a DB table, but this is fine for the project
var Sessions = map[string]int{}

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

// RegisterGET renders the registration page
func (h *AuthHandler) RegisterGET(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// RegisterPOST handles form submission for new users
func (h *AuthHandler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// basic validation
	if username == "" || email == "" || password == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	// check if email already exists
	var exists int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		http.Error(w, "email already registered", http.StatusConflict)
		return
	}

	// check if username already exists
	err = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		http.Error(w, "username already taken", http.StatusConflict)
		return
	}

	// hash the password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "could not hash password", http.StatusInternalServerError)
		return
	}

	// insert user into DB
	_, err = h.DB.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, string(hashed),
	)
	if err != nil {
		http.Error(w, "could not create user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// LoginGET renders the login page
func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// LoginPOST validates credentials and creates a session
func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	// fetch user by email
	var userID int
	var storedHash string
	err := h.DB.QueryRow(
		"SELECT id, password FROM users WHERE email = ?", email,
	).Scan(&userID, &storedHash)
	if err == sql.ErrNoRows {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	// compare password
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// generate session token
	sessionToken := uuid.New().String()
	Sessions[sessionToken] = userID

	// set cookie — expires in 24 hours
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout clears the session and cookie
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(Sessions, cookie.Value)
	}

	// overwrite cookie with expired one to clear it
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
