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

func getCategoriesForPost(db *sql.DB, postID int64) ([]string, error) {
	rows, err := db.Query(
		`SELECT c.name
		 FROM post_categories pc
		 JOIN categories c ON c.id = pc.category_id
		 WHERE pc.post_id = ?`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// ── Comments ───────────────────────────────────────────────────────────────

func insertComment(db *sql.DB, c *models.Comment) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO comments (post_id, user_id, content, created_at) VALUES (?, ?, ?, datetime('now'))`,
		c.PostID, c.UserID, c.Body,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getCommentsByPostID(db *sql.DB, postID int64) ([]models.Comment, error) {
	rows, err := db.Query(
		`SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.created_at
		 FROM comments c
		 JOIN users u ON u.id = c.user_id
		 WHERE c.post_id = ?
		 ORDER BY c.created_at ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var cm models.Comment
		if err := rows.Scan(&cm.ID, &cm.PostID, &cm.UserID, &cm.Username, &cm.Body, &cm.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, cm)
	}
	return comments, rows.Err()
}
