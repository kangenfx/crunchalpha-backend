package ratelimit

import (
	"database/sql"
	"fmt"
	"time"
)

type Config struct {
	MaxAttempts int
	Window      time.Duration
}

type Limiter struct {
	repo *Repository
}

func NewLimiter(db *sql.DB) *Limiter {
	return &Limiter{
		repo: NewRepository(db),
	}
}

func (l *Limiter) Check(key string, config Config) (bool, error) {
	attempts, windowStart, err := l.repo.GetAttempts(key)
	if err == sql.ErrNoRows {
		err = l.repo.CreateLimit(key, time.Now())
		return true, err
	}
	
	if err != nil {
		return false, err
	}

	if time.Since(windowStart) > config.Window {
		err = l.repo.ResetLimit(key, time.Now())
		return true, err
	}

	if attempts >= config.MaxAttempts {
		return false, fmt.Errorf("rate limit exceeded")
	}

	err = l.repo.IncrementAttempts(key)
	return true, err
}
