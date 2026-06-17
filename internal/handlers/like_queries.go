package handlers

import (
	"database/sql"
)

func upsertPostLike(db *sql.DB, postID, userID int64, value int) error {
	var existing int
	err := db.QueryRow(
		`SELECT value FROM likes WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
		postID, userID,
	).Scan(&existing)

	switch {
	case err == sql.ErrNoRows:
		_, err = db.Exec(
			`INSERT INTO likes (post_id, comment_id, user_id, value, created_at)
			 VALUES (?, NULL, ?, ?, datetime('now'))`,
			postID, userID, value,
		)
		return err

	case err != nil:
		return err

	case existing == value:
		_, err = db.Exec(
			`DELETE FROM likes WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
			postID, userID,
		)
		return err

	default:
		_, err = db.Exec(
			`UPDATE likes SET value = ?, created_at = datetime('now')
			 WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
			value, postID, userID,
		)
		return err
	}
}
