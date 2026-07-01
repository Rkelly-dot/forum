# Forum

A discussion board built with Go and SQLite. Users register, create posts, tag them with categories, leave comments, and vote on both posts and comments. The application compiles to a single binary and stores everything in a SQLite file on disk.

---

## Features

- Registration and login with bcrypt-hashed passwords and cookie-based sessions (24-hour expiry, one active session per account)
- Posts with a title, body, and one or more categories
- Threaded comments on any post
- Up/downvotes on posts and comments; one vote per user per target, toggle to change your vote
- Feed filters: all posts, your own posts, posts you have liked, or posts in a given category
- Seven default categories seeded on first run: General, Technology, Gaming, Science, Sports, Music, Other

---

## Dependencies

Three external packages, all in `go.mod`:

- `github.com/mattn/go-sqlite3` for the SQLite driver (uses cgo)
- `golang.org/x/crypto` for bcrypt
- `github.com/google/uuid` for session tokens

Everything else is the Go standard library. No ORM, no HTTP framework, no templating engine beyond `html/template`.

---

## Prerequisites

To run locally you need Go 1.24 or later and a C compiler on your PATH because `go-sqlite3` requires cgo. On Debian/Ubuntu that means `gcc`; on macOS it comes with Xcode Command Line Tools; on Windows you can use TDM-GCC.

If you would rather skip the Go setup, Docker covers everything.

---

## Running locally

```bash
git clone https://github.com/yourname/forum.git
cd forum
go run ./cmd/main.go
```

Open http://localhost:8080 in a browser. On first startup the application creates `forum.db` in the working directory and runs migrations. Migrations use `CREATE TABLE IF NOT EXISTS` throughout, so restarting the server is always safe.

---

## Running with Docker

Two Compose files are included. They behave identically at runtime and differ only in how the database file is persisted.

**Bind mount** (maps `./forum.db` on the host into the container, good for development):

```bash
docker compose up --build
```

**Named volume** (Docker manages the storage, cleaner for a remote server):

```bash
docker compose -f docker-compose.prod.yml up --build
```

Both serve the application on port 8080.

The Dockerfile is a two-stage build. The first stage uses `golang:1.24-alpine` and installs `gcc` and `musl-dev` so that cgo can link against SQLite. The second stage copies the compiled binary into a minimal `alpine:3.20` image alongside the `web/` directory. The image pre-creates an empty `forum.db` file before the volume is mounted; this is intentional because Docker infers whether to mount a file or a directory from what already exists at the target path inside the image.

---

## Project structure

```
forum/
  cmd/
    main.go               # entry point, route registration
  internal/
    auth/
      auth.go             # RegisterUser, LoginUser, bcrypt comparison
      session.go          # CreateSession, ValidateSession, DeleteSession
      auth_test.go
    database/
      db.go               # SQLite connection helper
      migration.go        # schema creation and category seeding
    handlers/
      auth_handler.go     # /register, /login, /logout
      post_handler.go     # post listing, creation, single-post view
      post_queries.go     # SQL helpers: insert, fetch by ID, fetch all
      comment_handler.go  # comment creation
      like_handler.go     # upvote/downvote, JSON response for AJAX callers
      like_queries.go     # upsert and count queries for likes
      like_test.go
      filter_handler.go   # category browsing, my-posts, liked-posts
      filter_queries.go   # queries for each filter mode
      post_test.go
    models/
      user.go
      post.go
      comment.go
      like.go
      session.go
  middleware/
    auth_middleware.go    # RequireAuth, GetUserID from context
  web/
    static/
      style.css
      main.js             # AJAX voting and filter button state
    templates/
      layout.html
      index.html          # post feed
      post.html           # single post with comments
      new_post.html
      categories.html
      login.html
      register.html
      error.html
```

---

## Routes

| Method | Path | Auth required | Description |
|--------|------|:---:|-------------|
| GET | `/` | no | Post feed; accepts `?filter=mine`, `?filter=liked`, `?category=<id>` |
| GET | `/categories` | no | Browse all categories |
| GET | `/posts/new` | yes | New post form |
| POST | `/posts/new` | yes | Create a post |
| GET | `/posts/:id` | no | View a post and its comments |
| POST | `/posts/:id/comments` | yes | Submit a comment |
| POST | `/posts/:id/like` | yes | Cast or update a vote |
| GET | `/register` | no | Registration form |
| POST | `/register` | no | Create an account |
| GET | `/login` | no | Login form |
| POST | `/login` | no | Authenticate and set session cookie |
| POST | `/logout` | no | Clear session and redirect to `/login` |

The like endpoint accepts `post_id` or `comment_id` (not both) and `value` (`1` or `-1`) in the POST body. When the request carries an `Accept: application/json` header, it returns `{"likes": N, "dislikes": N}` instead of redirecting. `main.js` uses this to update vote counts in the DOM without a full page reload.

Unauthenticated requests to protected routes redirect to `/login`.

---

## Database

Seven tables in one SQLite file.

`users` holds credentials. Passwords are bcrypt hashes. Both `email` and `username` have UNIQUE constraints.

`sessions` stores one token per user at a time. Signing in deletes any previous session before creating a new one, so you cannot have two active sessions for the same account.

`posts` and `comments` reference `users` with a cascading delete, so removing a user cleans up their content automatically.

`categories` is a lookup table seeded with seven defaults. `post_categories` is the join table, so one post can belong to multiple categories.

`likes` is the most constrained table. A row must target either a post or a comment, never both and never neither, enforced by a CHECK constraint. Two separate UNIQUE constraints (`user_id, post_id`) and (`user_id, comment_id`) prevent double votes at the database level regardless of what the application layer does.

Vote values are stored as `1` or `-1` and a CHECK constraint rejects anything else.

Timestamps throughout the application are formatted in East Africa Time (UTC+3) via a fixed timezone in the `FormattedDate` method on `Post` and `Comment`.

---

## Tests

```bash
go test ./...
```

Test files sit alongside the packages they cover. They open an in-memory SQLite database and create the necessary schema themselves, so running them does not touch `forum.db` and no environment setup is required.

---

## Contributors

- Ronnie Jeff
- Walter Onyango
- Alvin James
- Ryan Kelly
- Juma Tony