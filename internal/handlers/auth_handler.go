package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"forum/internal/auth"
)

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

func (h *AuthHandler) RegisterGET(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/register.html",
	)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{})
}

func (h *AuthHandler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// Validate required fields before attempting to register
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/register.html")
		w.WriteHeader(http.StatusBadRequest) // 400 — malformed/incomplete request
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
			"Error": "All fields are required.",
		})
		return
	}

	err := auth.RegisterUser(h.DB, username, email, password)
	if err != nil {
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/register.html")
		w.WriteHeader(http.StatusBadRequest) // 400 — e.g. email already taken, invalid input
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
			"Error": err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/login.html",
	)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{})
}

func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Empty fields = malformed request → 400
	if email == "" || password == "" {
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/login.html")
		w.WriteHeader(http.StatusBadRequest)
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
			"Error": "Email and password are required.",
		})
		return
	}

	user, err := auth.LoginUser(h.DB, email, password)
	if err != nil {
		// Fields were present but credentials didn't match → 401
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/login.html")
		w.WriteHeader(http.StatusUnauthorized)
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
			"Error": "Invalid email or password.",
		})
		return
	}

	session, err := auth.CreateSession(h.DB, user.ID)
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Path:     "/",
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		auth.DeleteSession(h.DB, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}