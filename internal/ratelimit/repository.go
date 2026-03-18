package ratelimit

import (
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAttempts(key string) (int, time.Time, error) {
	var attempts int
	var windowStart time.Time
	
	err := r.db.QueryRow(`
		SELECT attempts, window_start
		FROM rate_limits
		WHERE key = $1
	`, key).Scan(&attempts, &windowStart)
	
	return attempts, windowStart, err
}

func (r *Repository) CreateLimit(key string, windowStart time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO rate_limits (key, attempts, window_start)
		VALUES ($1, 1, $2)
		ON CONFLICT (key) DO UPDATE
		SET attempts = 1, window_start = $2
	`, key, windowStart)
	return err
}

func (r *Repository) IncrementAttempts(key string) error {
	_, err := r.db.Exec(`
		UPDATE rate_limits
		SET attempts = attempts + 1
		WHERE key = $1
	`, key)
	return err
}

func (r *Repository) ResetLimit(key string, windowStart time.Time) error {
	_, err := r.db.Exec(`
		UPDATE rate_limits
		SET attempts = 1, window_start = $2
		WHERE key = $1
	`, key, windowStart)
	return err
}
