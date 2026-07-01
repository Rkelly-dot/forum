# ---- Build stage ----
FROM golang:1.24-alpine AS builder

# go-sqlite3 uses cgo, so we need gcc/musl headers to compile it
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -o forum ./cmd/main.go

# ---- Runtime stage ----
FROM alpine:3.20

# sqlite3 lib needed at runtime since we built dynamically against musl
RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /app

# Binary
COPY --from=builder /app/forum .

# Static assets + templates (read via relative path at runtime)
COPY --from=builder /app/web ./web

# Pre-create forum.db as an empty file. This ensures that when Docker
# mounts a named volume at this exact path, it mounts a FILE (not a
# directory) -- Docker infers the mount type from what currently exists
# at the target path inside the image.
RUN touch forum.db

EXPOSE 8080

CMD ["./forum"]