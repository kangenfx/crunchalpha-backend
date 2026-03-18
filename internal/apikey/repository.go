package apikey

import (
	"database/sql"
	
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateAPIKey stores new API key with bound accounts
func (r *Repository) CreateAPIKey(userID, keyHash, keyPrefix, name string, accountIDs []string, allowedIPs []string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	
	// Insert API key
	var keyID string
	query := `
		INSERT INTO api_keys (user_id, key_hash, key_prefix, name, allowed_ips, active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id
	`
	
	var ipsArray interface{}
	if len(allowedIPs) > 0 {
		ipsArray = pq.Array(allowedIPs)
	} else {
		ipsArray = nil
	}
	
	err = tx.QueryRow(query, userID, keyHash, keyPrefix, name, ipsArray).Scan(&keyID)
	if err != nil {
		return "", err
	}
	
	// Link to accounts
	for _, accountID := range accountIDs {
		_, err = tx.Exec(`
			INSERT INTO api_key_accounts (api_key_id, account_id)
			VALUES ($1, $2)
		`, keyID, accountID)
		if err != nil {
			return "", err
		}
	}
	
	if err = tx.Commit(); err != nil {
		return "", err
	}
	
	return keyID, nil
}

// GetAPIKeyByHash retrieves API key by hash
func (r *Repository) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	query := `
		SELECT 
			k.id, k.user_id, k.key_hash, k.key_prefix, k.name,
			k.allowed_ips, k.active, k.created_at, k.last_used_at, k.revoked_at
		FROM api_keys k
		WHERE k.key_hash = $1 AND k.active = true AND k.revoked_at IS NULL
	`
	
	var key APIKey
	var allowedIPs sql.NullString
	var lastUsed, revoked sql.NullTime
	
	err := r.db.QueryRow(query, keyHash).Scan(
		&key.ID, &key.UserID, &key.KeyHash, &key.KeyPrefix, &key.Name,
		&allowedIPs, &key.Active, &key.CreatedAt, &lastUsed, &revoked,
	)
	
	if err != nil {
		return nil, err
	}
	
	if lastUsed.Valid {
		key.LastUsedAt = &lastUsed.Time
	}
	if revoked.Valid {
		key.RevokedAt = &revoked.Time
	}
	
	// Get bound accounts
	rows, err := r.db.Query(`
		SELECT account_id FROM api_key_accounts WHERE api_key_id = $1
	`, key.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var accountID string
		if err := rows.Scan(&accountID); err != nil {
			return nil, err
		}
		key.AccountIDs = append(key.AccountIDs, accountID)
	}
	
	// Parse allowed IPs
	if allowedIPs.Valid && allowedIPs.String != "" {
		if err := pq.Array(&key.AllowedIPs).Scan([]byte(allowedIPs.String)); err == nil {
			// Successfully parsed
		}
	}
	
	return &key, nil
}

// UpdateLastUsed updates last_used_at timestamp
func (r *Repository) UpdateLastUsed(keyID string) error {
	_, err := r.db.Exec(`
		UPDATE api_keys SET last_used_at = NOW() WHERE id = $1
	`, keyID)
	return err
}

// ListAPIKeys returns all API keys for user
func (r *Repository) ListAPIKeys(userID string) ([]*APIKey, error) {
	query := `
		SELECT 
			k.id, k.key_prefix, k.name, k.allowed_ips,
			k.active, k.created_at, k.last_used_at, k.revoked_at
		FROM api_keys k
		WHERE k.user_id = $1
		ORDER BY k.created_at DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var allowedIPs sql.NullString
		var lastUsed, revoked sql.NullTime
		
		err := rows.Scan(
			&key.ID, &key.KeyPrefix, &key.Name, &allowedIPs,
			&key.Active, &key.CreatedAt, &lastUsed, &revoked,
		)
		if err != nil {
			return nil, err
		}
		
		if lastUsed.Valid {
			key.LastUsedAt = &lastUsed.Time
		}
		if revoked.Valid {
			key.RevokedAt = &revoked.Time
		}
		
		// Get bound accounts
		accountRows, err := r.db.Query(`
			SELECT account_id FROM api_key_accounts WHERE api_key_id = $1
		`, key.ID)
		if err == nil {
			for accountRows.Next() {
				var accountID string
				if err := accountRows.Scan(&accountID); err == nil {
					key.AccountIDs = append(key.AccountIDs, accountID)
				}
			}
			accountRows.Close()
		}
		
		keys = append(keys, &key)
	}
	
	return keys, nil
}

// RevokeAPIKey marks key as revoked
func (r *Repository) RevokeAPIKey(keyID, userID string) error {
	_, err := r.db.Exec(`
		UPDATE api_keys 
		SET active = false, revoked_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, keyID, userID)
	return err
}

// LogAPIKeyUsage records API key usage
func (r *Repository) LogAPIKeyUsage(keyID, ip, endpoint, method string, statusCode int) error {
	_, err := r.db.Exec(`
		INSERT INTO api_key_logs (api_key_id, ip_address, endpoint, method, status_code)
		VALUES ($1, $2, $3, $4, $5)
	`, keyID, ip, endpoint, method, statusCode)
	return err
}
