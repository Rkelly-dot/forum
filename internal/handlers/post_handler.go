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
	db        *sql.DB
	templates *template.Template
}

func NewPostHandler(db *sql.DB, tmpl *template.Template) *PostHandler {
	return &PostHandler{db: db, templates: tmpl}
}

// GET /posts
func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := getAllPosts(h.db)
	if err != nil {
		http.Error(w, "could not fetch posts", http.StatusInternalServerError)
		return
	}

	user, _ := auth.GetSessionUser(r, h.db)

	data := map[string]interface{}{
		"Posts": posts,
		"User":  user,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// GET /posts/new
func (h *PostHandler) NewPostGET(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetSessionUser(r, h.db)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"User": user,
	}

	if err := h.templates.ExecuteTemplate(w, "new_post.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// POST /posts/new
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
	body := strings.TrimSpace(r.FormValue("body"))
	categories := r.Form["categories"]

	if title == "" || body == "" {
		http.Error(w, "title and body are required", http.StatusBadRequest)
		return
	}

	p := &models.Post{
		UserID:     user.ID,
		Title:      title,
		Body:       body,
		Categories: categories,
	}

	id, err := insertPost(h.db, p)
	if err != nil {
		http.Error(w, "could not create post", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

// GET /posts/{id}
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

	comments, err := getCommentsByPostID(h.db, postID)
	if err != nil {
		http.Error(w, "could not fetch comments", http.StatusInternalServerError)
		return
	}

	user, _ := auth.GetSessionUser(r, h.db)

	data := map[string]interface{}{
		"Post":     post,
		"Comments": comments,
		"User":     user,
	}

	if err := h.templates.ExecuteTemplate(w, "post.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}