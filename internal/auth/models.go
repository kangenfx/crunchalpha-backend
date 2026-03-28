package auth

import "time"

// User represents platform user
type User struct {
	ID          string    `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	PasswordHash string   `json:"-" db:"password_hash"`
	PrimaryRole string    `json:"primary_role" db:"primary_role"`
	Name        *string   `json:"name,omitempty" db:"name"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
}

// LoginRequest for login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterRequest for registration
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	PrimaryRole string `json:"primary_role"`
	RefCode     string `json:"ref_code"`
}

// LoginResponse after successful login
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	User        *User  `json:"user"`
}

// ForgotPasswordRequest for requesting password reset
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest for resetting password with token
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ValidateTokenResponse for token validation
type ValidateTokenResponse struct {
	Valid bool `json:"valid"`
}

// PasswordResetToken represents password reset token in database
type PasswordResetToken struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
	Used      bool      `db:"used"`
	CreatedAt time.Time `db:"created_at"`
}

// RefreshTokenRequest for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse after successful refresh
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// RefreshToken represents refresh token in database
type RefreshToken struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
}

// VerifyEmailRequest for email verification
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// ResendVerificationRequest for resending verification email
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}
