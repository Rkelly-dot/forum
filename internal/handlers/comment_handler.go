package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/auth"
	"forum/internal/models"
)

type CommentHandler struct {
	db *sql.DB
}

// FIXED: no longer takes *template.Template
func NewCommentHandler(db *sql.DB) *CommentHandler {
	return &CommentHandler{db: db}
}

func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetSessionUser(r, h.db)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/posts/")
	path = strings.TrimSuffix(path, "/comments")
	postID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

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

	// FIXED: was r.FormValue("body") but the textarea is name="content"
	body := strings.TrimSpace(r.FormValue("content"))
	if body == "" {
		http.Error(w, "comment cannot be empty", http.StatusBadRequest)
		return
	}

	if _, err := insertComment(h.db, &models.Comment{
		PostID: postID,
		UserID: user.ID,
		Body:   body,
	}); err != nil {
		http.Error(w, "could not save comment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}
