package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"forum/internal/auth"
)

import (
	"encoding/json"
)

func acceptsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept == "application/json" || accept == "application/json, */*" || accept == "*/*" || accept != "" && (accept == "application/json")
}

func writeJSONCounts(w http.ResponseWriter, likes, dislikes int) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"likes": likes, "dislikes": dislikes})
}

type LikeHandler struct {
	db *sql.DB
}

func NewLikeHandler(db *sql.DB) *LikeHandler {
	return &LikeHandler{db: db}
}

func (h *LikeHandler) Like(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetSessionUser(r, h.db)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	value, err := strconv.Atoi(r.FormValue("value"))
	if err != nil || (value != 1 && value != -1) {
		http.Error(w, "value must be 1 or -1", http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("post_id")
	commentIDStr := r.FormValue("comment_id")

	switch {
	case postIDStr != "" && commentIDStr != "":
		http.Error(w, "specify only one of post_id or comment_id", http.StatusBadRequest)
		return

	case postIDStr != "":
		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid post id", http.StatusBadRequest)
			return
		}
		if _, err := getPostByID(h.db, postID); err == sql.ErrNoRows {
			http.Error(w, "post not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "could not verify post", http.StatusInternalServerError)
			return
		}
		if err := upsertPostLike(h.db, postID, user.ID, value); err != nil {
			http.Error(w, "could not save vote", http.StatusInternalServerError)
			return
		}
		// If the client expects JSON, return updated counts for AJAX UI
		if acceptsJSON(r) {
			likes, dislikes, _ := countPostLikes(h.db, postID)
			writeJSONCounts(w, likes, dislikes)
			return
		}
		http.Redirect(w, r, "/posts/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)

	case commentIDStr != "":
		commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid comment id", http.StatusBadRequest)
			return
		}
		if err := upsertCommentLike(h.db, commentID, user.ID, value); err != nil {
			http.Error(w, "could not save vote", http.StatusInternalServerError)
			return
		}
		if acceptsJSON(r) {
			likes, dislikes, _ := countCommentLikes(h.db, commentID)
			writeJSONCounts(w, likes, dislikes)
			return
		}
		referer := r.Header.Get("Referer")
		if referer == "" {
			referer = "/posts"
		}
		http.Redirect(w, r, referer, http.StatusSeeOther)

	default:
		http.Error(w, "post_id or comment_id is required", http.StatusBadRequest)
	}
}
