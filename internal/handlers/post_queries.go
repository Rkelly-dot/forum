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