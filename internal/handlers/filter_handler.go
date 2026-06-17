package handlers

import (
	"html/template"
	"net/http"

	"database/sql"

	"forum/internal/auth"
	"forum/internal/models"
)

type FilterHandler struct {
	db        *sql.DB
	templates *template.Template
}

func NewFilterHandler(db *sql.DB, tmpl *template.Template) *FilterHandler {
	return &FilterHandler{db: db, templates: tmpl}
}

func (h *FilterHandler) FilteredPosts(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	filter := r.URL.Query().Get("filter")

	user, _ := auth.GetSessionUser(r, h.db)

	var (
		posts []models.Post
		err   error
	)

	switch {
	case category != "":
		posts, err = getPostsByCategory(h.db, category)

	case filter == "mine":
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		posts, err = getPostsByUser(h.db, user.ID)

	case filter == "liked":
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		posts, err = getPostsLikedByUser(h.db, user.ID)

	default:
		posts, err = getAllPosts(h.db)
	}

	if err != nil {
		http.Error(w, "could not fetch posts", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Posts":    posts,
		"User":     user,
		"Category": category,
		"Filter":   filter,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}