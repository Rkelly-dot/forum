package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"forum/internal/database"
	"forum/internal/handlers"
)

func main() {
	db, err := database.Connect("forum.db")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, db)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	addr := ":8080"
	fmt.Printf("Forum running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func registerRoutes(mux *http.ServeMux, db *sql.DB) {
	authHandler := handlers.NewAuthHandler(db)

	// Home / feed
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles(
			"./web/templates/layout.html",
			"./web/templates/index.html",
		))
		tmpl.ExecuteTemplate(w, "layout.html", nil)
	})

	// Categories — data will come from DB later
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles(
			"./web/templates/layout.html",
			"./web/templates/categories.html",
		))
		tmpl.ExecuteTemplate(w, "layout.html", nil)
	})

	// New post
	mux.HandleFunc("/posts/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		tmpl := template.Must(template.ParseFiles(
			"./web/templates/layout.html",
			"./web/templates/new_post.html",
		))
		tmpl.ExecuteTemplate(w, "layout.html", nil)
	})

	// Auth
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