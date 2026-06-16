package handlers

import (
	"database/sql"
	"fmt"

	"forum/internal/models"
)

// ── Posts ──────────────────────────────────────────────────────────────────

func insertPost(db *sql.DB, p *models.Post) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO posts (user_id, title, content, created_at) VALUES (?, ?, ?, datetime('now'))`,
		p.UserID, p.Title, p.Body,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// categories is now a normalized lookup table (id, name); post_categories
	// stores category_id, not the raw name, so resolve each name first.
	for _, cat := range p.Categories {
		var categoryID int64
		err = db.QueryRow(`SELECT id FROM categories WHERE name = ?`, cat).Scan(&categoryID)
		if err != nil {
			return 0, fmt.Errorf("unknown category %q: %w", cat, err)
		}

		_, err = db.Exec(
			`INSERT OR IGNORE INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			id, categoryID,
		)
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

func getPostByID(db *sql.DB, postID int64) (*models.Post, error) {
	row := db.QueryRow(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at
		 FROM posts p
		 JOIN users u ON u.id = p.user_id
		 WHERE p.id = ?`,
		postID,
	)

	p := &models.Post{}
	err := row.Scan(&p.ID, &p.UserID, &p.Username, &p.Title, &p.Body, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	cats, err := getCategoriesForPost(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Categories = cats
	return p, nil
}

func getAllPosts(db *sql.DB) ([]models.Post, error) {
	rows, err := db.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at
		 FROM posts p
		 JOIN users u ON u.id = p.user_id
		 ORDER BY p.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Title, &p.Body, &p.CreatedAt); err != nil {
			return nil, err
		}
		cats, err := getCategoriesForPost(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Categories = cats
		posts = append(posts, p)
	}
	return posts, rows.Err()
}