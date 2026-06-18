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
	// FIXED: parse layout first, then page — execute "layout.html"
	// so {{block "content"}} picks up the page's {{define "content"}}
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
	err := auth.RegisterUser(h.DB, r.FormValue("username"), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/register.html")
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{"Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	// FIXED: same pattern — layout first, execute "layout.html"
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
	user, err := auth.LoginUser(h.DB, r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/login.html")
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{"Error": "Invalid email or password."})
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
