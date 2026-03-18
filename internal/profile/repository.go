package profile

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

// GetProfile retrieves user profile by user ID
func (r *Repository) GetProfile(userID string) (*UserProfile, error) {
	var profile UserProfile
	
	err := r.db.QueryRow(`
		SELECT id, email, name, primary_role, phone_number, 
		       country, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&profile.ID, &profile.Email, &profile.Name, &profile.PrimaryRole,
		&profile.PhoneNumber, &profile.Country, &profile.Bio,
		&profile.AvatarURL, &profile.CreatedAt, &profile.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	
	return &profile, err
}

// UpdateProfile updates user profile
func (r *Repository) UpdateProfile(userID string, req UpdateProfileRequest) (*UserProfile, error) {
	now := time.Now()
	
	_, err := r.db.Exec(`
		UPDATE users
		SET name = COALESCE($1, name),
		    phone_number = COALESCE($2, phone_number),
		    country = COALESCE($3, country),
		    bio = COALESCE($4, bio),
		    avatar_url = COALESCE($5, avatar_url),
		    updated_at = $6
		WHERE id = $7
	`, req.Name, req.PhoneNumber, req.Country, req.Bio, req.AvatarURL, now, userID)
	
	if err != nil {
		return nil, err
	}
	
	// Get updated profile
	return r.GetProfile(userID)
}
