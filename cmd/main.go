package main

import (
	"database/sql"
	"fmt"
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
	postHandler := handlers.NewPostHandler(db)
	commentHandler := handlers.NewCommentHandler(db)
	filterHandler := handlers.NewFilterHandler(db)
	likeHandler := handlers.NewLikeHandler(db)

	// Home — handles ?filter= and ?category= from sidebar too
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("filter") != "" || r.URL.Query().Get("category") != "" {
			filterHandler.FilteredPosts(w, r)
			return
		}
		postHandler.ListPosts(w, r)
	})

	// Categories
	mux.HandleFunc("/categories", filterHandler.FilteredPosts)

	// New post
	mux.HandleFunc("/posts/new", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			postHandler.NewPostGET(w, r)
		case http.MethodPost:
			postHandler.NewPostPOST(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Single post, comments, likes
	mux.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/posts/" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		switch {
		case isLikePath(r.URL.Path) && r.Method == http.MethodPost:
			likeHandler.Like(w, r)
		case isCommentPath(r.URL.Path) && r.Method == http.MethodPost:
			commentHandler.CreateComment(w, r)
		case r.Method == http.MethodGet:
			postHandler.ViewPost(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
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

func isLikePath(path string) bool {
	return len(path) > 5 && path[len(path)-5:] == "/like"
}

func isCommentPath(path string) bool {
	return len(path) > 9 && path[len(path)-9:] == "/comments"
}
