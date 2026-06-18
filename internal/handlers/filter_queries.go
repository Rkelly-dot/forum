package handlers

import (
	"database/sql"

	"forum/internal/models"
)

// ADDED: was missing — needed by NewPostGET and FilteredPosts
func getAllCategories(db *sql.DB) ([]models.Category, error) {
	rows, err := db.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func getPostsByCategory(db *sql.DB, category string) ([]models.Post, error) {
	rows, err := db.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at
		 FROM posts p
		 JOIN users u ON u.id = p.user_id
		 JOIN post_categories pc ON pc.post_id = p.id
		 JOIN categories c ON c.id = pc.category_id
		 WHERE c.name = ?
		 ORDER BY p.created_at DESC`,
		category,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostsWithCategories(db, rows)
}

func getPostsByUser(db *sql.DB, userID int64) ([]models.Post, error) {
	rows, err := db.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at
		 FROM posts p
		 JOIN users u ON u.id = p.user_id
		 WHERE p.user_id = ?
		 ORDER BY p.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostsWithCategories(db, rows)
}

func getPostsLikedByUser(db *sql.DB, userID int64) ([]models.Post, error) {
	rows, err := db.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.created_at
		 FROM posts p
		 JOIN users u ON u.id = p.user_id
		 JOIN likes l ON l.post_id = p.id
		 WHERE l.user_id = ? AND l.value = 1
		 ORDER BY p.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostsWithCategories(db, rows)
}

func scanPostsWithCategories(db *sql.DB, rows *sql.Rows) ([]models.Post, error) {
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
