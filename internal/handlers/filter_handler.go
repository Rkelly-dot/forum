package handlers

import (
	"database/sql"
	"html/template"
	"net/http"

	"forum/internal/auth"
	"forum/internal/models"
)

type FilterHandler struct {
	db *sql.DB
}

// FIXED: no longer takes *template.Template
func NewFilterHandler(db *sql.DB) *FilterHandler {
	return &FilterHandler{db: db}
}

func (h *FilterHandler) FilteredPosts(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	filter   := r.URL.Query().Get("filter")
	user, _  := auth.GetSessionUser(r, h.db)

	// FIXED: no params = show categories browse page with real DB data
	if category == "" && filter == "" {
		cats, err := getAllCategories(h.db)
		if err != nil {
			http.Error(w, "could not fetch categories", http.StatusInternalServerError)
			return
		}
		tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/categories.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
			"Categories": cats,
			"User":       user,
		})
		return
	}

	var (
		posts []models.Post
		err   error
	)

	switch {
	case category != "":
		// FIXED: categories page links ?category={{.ID}} but query filters by name
		// so resolve ID → name first
		var catName string
		if lookupErr := h.db.QueryRow(`SELECT name FROM categories WHERE id = ?`, category).Scan(&catName); lookupErr != nil {
			catName = category // fallback: treat value as name directly
		}
		posts, err = getPostsByCategory(h.db, catName)

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

	// FIXED: populate like counts
	for i := range posts {
		posts[i].Likes, posts[i].Dislikes, _ = countPostLikes(h.db, posts[i].ID)
	}

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/index.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"Posts":    posts,
		"User":     user,
		"Category": category,
		"Filter":   filter,
	})
}