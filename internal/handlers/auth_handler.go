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
	tmpl, err := template.ParseFiles("web/templates/register.html", "web/templates/layout.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "register.html", nil); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	err := auth.RegisterUser(
		h.DB,
		r.FormValue("username"),
		r.FormValue("email"),
		r.FormValue("password"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/login.html", "web/templates/layout.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "login.html", nil); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	user, err := auth.LoginUser(h.DB, r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
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
