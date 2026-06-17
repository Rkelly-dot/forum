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

type CommentHandler struct {
	db        *sql.DB
	templates *template.Template
}

func NewCommentHandler(db *sql.DB, tmpl *template.Template) *CommentHandler {
	return &CommentHandler{db: db, templates: tmpl}
}

// POST /posts/{id}/comments
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetSessionUser(r, h.db)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// extract post id from /posts/{id}/comments
	path := strings.TrimPrefix(r.URL.Path, "/posts/")
	path = strings.TrimSuffix(path, "/comments")
	postID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	// confirm the post exists
	_, err = getPostByID(h.db, postID)
	if err == sql.ErrNoRows {
		http.Error(w, "post not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "could not verify post", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "comment body is required", http.StatusBadRequest)
		return
	}

	c := &models.Comment{
		PostID: postID,
		UserID: user.ID,
		Body:   body,
	}

	if _, err := insertComment(h.db, c); err != nil {
		http.Error(w, "could not save comment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}