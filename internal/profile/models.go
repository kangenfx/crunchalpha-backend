package profile

import "time"

// UserProfile represents user profile information
type UserProfile struct {
	ID           string     `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	Name         *string    `json:"name,omitempty" db:"name"`
	PrimaryRole  string     `json:"primary_role" db:"primary_role"`
	PhoneNumber  *string    `json:"phone_number,omitempty" db:"phone_number"`
	Country      *string    `json:"country,omitempty" db:"country"`
	Bio          *string    `json:"bio,omitempty" db:"bio"`
	AvatarURL    *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// UpdateProfileRequest for updating profile
type UpdateProfileRequest struct {
	Name        *string `json:"name,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	Country     *string `json:"country,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}
