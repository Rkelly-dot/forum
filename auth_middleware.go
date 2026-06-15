package middleware

import (
	"context"
	"net/http"

	"forum/handlers"
)

// contextKey is unexported to avoid collisions with other packages
type contextKey string

const UserIDKey contextKey = "userID"

// RequireAuth blocks unauthenticated requests and redirects to /login
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, ok := handlers.Sessions[cookie.Value]
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// attach userID to request context so handlers can read it
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID is a helper other handlers call to read the userID from context
func GetUserID(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserIDKey).(int)
	return userID, ok
}
