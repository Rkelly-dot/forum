package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"forum/internal/models"
)

type contextKey string

const UserIDKey contextKey = "userID"

// RequireAuth blocks requests without a valid, unexpired session.
// Looks up the session token against the sessions table.
func RequireAuth(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var session models.Session
		err = db.QueryRow(
			"SELECT user_id, expires_at FROM sessions WHERE token = ?", cookie.Value,
		).Scan(&session.UserID, &session.ExpiresAt)
		if err == sql.ErrNoRows {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// session expired — clean it up and redirect
		if time.Now().After(session.ExpiresAt) {
			db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID is a helper other handlers call to read the userID from context
func GetUserID(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserIDKey).(int)
	return userID, ok
}
