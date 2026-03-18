package apikey

import (
	"time"
)

type APIKey struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	KeyHash     string    `json:"-"` // Never expose hash
	KeyPrefix   string    `json:"key_prefix"`
	Name        string    `json:"name"`
	AllowedIPs  []string  `json:"allowed_ips,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	AccountIDs  []string  `json:"account_ids,omitempty"` // Bound accounts
}

type CreateAPIKeyRequest struct {
	Name       string   `json:"name" binding:"required,min=3,max=100"`
	AccountIDs []string `json:"account_ids" binding:"required,min=1"`
	AllowedIPs []string `json:"allowed_ips,omitempty"`
}

type CreateAPIKeyResponse struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"` // Plain key, shown ONCE only
	KeyPrefix string    `json:"key_prefix"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
