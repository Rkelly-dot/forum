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

package handlers

import (
	"database/sql"
)

// ── Likes ──────────────────────────────────────────────────────────────────

// upsertPostLike inserts a like/dislike for a post, or updates it if the
// user already voted, or removes it if the same value is sent again (toggle off).
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

// upsertCommentLike does the same as upsertPostLike but scoped to a comment.
func upsertCommentLike(db *sql.DB, commentID, userID int64, value int) error {
	var existing int
	err := db.QueryRow(
		`SELECT value FROM likes WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
		commentID, userID,
	).Scan(&existing)

	switch {
	case err == sql.ErrNoRows:
		_, err = db.Exec(
			`INSERT INTO likes (post_id, comment_id, user_id, value, created_at)
			 VALUES (NULL, ?, ?, ?, datetime('now'))`,
			commentID, userID, value,
		)
		return err

	case err != nil:
		return err

	case existing == value:
		_, err = db.Exec(
			`DELETE FROM likes WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
			commentID, userID,
		)
		return err

	default:
		_, err = db.Exec(
			`UPDATE likes SET value = ?, created_at = datetime('now')
			 WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
			value, commentID, userID,
		)
		return err
	}
}

func countPostLikes(db *sql.DB, postID int64) (likes, dislikes int, err error) {
	row := db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM likes WHERE post_id = ?`,
		postID,
	)
	err = row.Scan(&likes, &dislikes)
	return likes, dislikes, err
}

func countCommentLikes(db *sql.DB, commentID int64) (likes, dislikes int, err error) {
	row := db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM likes WHERE comment_id = ?`,
		commentID,
	)
	err = row.Scan(&likes, &dislikes)
	return likes, dislikes, err
}

func getUserPostVote(db *sql.DB, postID, userID int64) (int, error) {
	var value int
	err := db.QueryRow(
		`SELECT value FROM likes WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
		postID, userID,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}
 
func getUserCommentVote(db *sql.DB, commentID, userID int64) (int, error) {
	var value int
	err := db.QueryRow(
		`SELECT value FROM likes WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
		commentID, userID,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}
