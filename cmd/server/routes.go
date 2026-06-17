package server

import (
    "database/sql"
    "net/http"

    "forum/internal/handlers"
)

func registerRoutes(mux *http.ServeMux, db *sql.DB) {
	authHandler := handlers.NewAuthHandler(db)

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authHandler.RegisterGET(w, r)
		case http.MethodPost:
			authHandler.RegisterPOST(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authHandler.LoginGET(w, r)
		case http.MethodPost:
			authHandler.LoginPOST(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/logout", authHandler.Logout)
}
