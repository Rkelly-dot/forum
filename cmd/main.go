package main

import (
	"fmt"
	"log"
	"net/http"

	"forum/internal/database"
)

func main() {
	// Initialise SQLite — creates forum.db if it doesn't exist
	db, err := database.Connect("forum.db")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run all CREATE TABLE IF NOT EXISTS statements
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Register routes
	mux := http.NewServeMux()
	registerRoutes(mux, db)

	// Serve static files (CSS, JS)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	addr := ":8080"
	fmt.Printf("Forum running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}