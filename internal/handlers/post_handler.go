package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/auth"
	"forum/internal/models"
)

type PostHandler struct {
	db *sql.DB
}

// FIXED: no longer takes *template.Template — each method parses its own
func NewPostHandler(db *sql.DB) *PostHandler {
	return &PostHandler{db: db}
}

func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := getAllPosts(h.db)
	if err != nil {
		http.Error(w, "could not fetch posts", http.StatusInternalServerError)
		return
	}

	// FIXED: populate like counts so votes show up
	for i := range posts {
		posts[i].Likes, posts[i].Dislikes, _ = countPostLikes(h.db, posts[i].ID)
	}

	user, _ := auth.GetSessionUser(r, h.db)

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/index.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"Posts": posts,
		"User":  user,
	})
}

func (h *PostHandler) NewPostGET(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetSessionUser(r, h.db)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// FIXED: fetch categories from DB so checkboxes are populated
	cats, err := getAllCategories(h.db)
	if err != nil {
		http.Error(w, "could not fetch categories", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/new_post.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"User":       user,
		"Categories": cats,
	})
}

func (h *PostHandler) NewPostPOST(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetSessionUser(r, h.db)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("content")) // FIXED: was "body", form sends "content"
	categoryIDs := r.Form["categories"]

	if title == "" || body == "" {
		cats, _ := getAllCategories(h.db)
		tmpl, _ := template.ParseFiles("web/templates/layout.html", "web/templates/new_post.html")
		tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
			"User":       user,
			"Categories": cats,
			"Error":      "Title and content are required.",
		})
		return
	}

	// FIXED: form sends category IDs but insertPost looks up by name
	// so resolve each ID to its name first
	var categoryNames []string
	for _, idStr := range categoryIDs {
		var name string
		if err := h.db.QueryRow(`SELECT name FROM categories WHERE id = ?`, idStr).Scan(&name); err == nil {
			categoryNames = append(categoryNames, name)
		}
	}

	id, err := insertPost(h.db, &models.Post{
		UserID:     user.ID,
		Title:      title,
		Body:       body,
		Categories: categoryNames,
	})
	if err != nil {
		http.Error(w, "could not create post", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (h *PostHandler) ViewPost(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/posts/")
	postID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	post, err := getPostByID(h.db, postID)
	if err == sql.ErrNoRows {
		http.Error(w, "post not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "could not fetch post", http.StatusInternalServerError)
		return
	}

	// FIXED: populate like counts
	post.Likes, post.Dislikes, _ = countPostLikes(h.db, post.ID)

	comments, err := getCommentsByPostID(h.db, postID)
	if err != nil {
		http.Error(w, "could not fetch comments", http.StatusInternalServerError)
		return
	}

	// FIXED: populate like counts on comments too
	for i := range comments {
		comments[i].Likes, comments[i].Dislikes, _ = countCommentLikes(h.db, comments[i].ID)
	}

	user, _ := auth.GetSessionUser(r, h.db)

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/post.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"Post":     post,
		"Comments": comments,
		"User":     user,
	})
}
