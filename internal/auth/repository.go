package auth

import (
	"database/sql"
	"errors"
	"time"
	
	"golang.org/x/crypto/bcrypt"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user already exists")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(email, password, role string) (*User, error) {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if role == "" {
		role = "trader"
	}

	var user User
	err = r.db.QueryRow(`
		INSERT INTO users (email, password_hash, primary_role)
		VALUES ($1, $2, $3)
		RETURNING id, email, primary_role, created_at
	`, email, string(hash), role).Scan(&user.ID, &user.Email, &user.PrimaryRole, &user.CreatedAt)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return nil, ErrUserExists
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByEmail(email string) (*User, error) {
	var user User
	err := r.db.QueryRow(`
		SELECT id, email, password_hash, primary_role, created_at, COALESCE(email_verified, false)
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.PrimaryRole, &user.CreatedAt, &user.EmailVerified)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// CreatePasswordResetToken creates a new password reset token
func (r *Repository) CreatePasswordResetToken(userID, token string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO password_reset_tokens (user_id, token, expires_at, used)
		VALUES ($1, $2, $3, false)
	`, userID, token, expiresAt)
	return err
}

// GetPasswordResetToken retrieves a password reset token
func (r *Repository) GetPasswordResetToken(token string) (*PasswordResetToken, error) {
	var prt PasswordResetToken
	err := r.db.QueryRow(`
		SELECT id, user_id, token, expires_at, used, created_at
		FROM password_reset_tokens
		WHERE token = $1
	`, token).Scan(&prt.ID, &prt.UserID, &prt.Token, &prt.ExpiresAt, &prt.Used, &prt.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("token not found")
	}
	return &prt, err
}

// MarkTokenAsUsed marks a password reset token as used
func (r *Repository) MarkTokenAsUsed(token string) error {
	_, err := r.db.Exec(`
		UPDATE password_reset_tokens
		SET used = true
		WHERE token = $1
	`, token)
	return err
}

// UpdatePassword updates user password
func (r *Repository) UpdatePassword(userID, newPasswordHash string) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET password_hash = $1
		WHERE id = $2
	`, newPasswordHash, userID)
	return err
}

// InvalidateAllRefreshTokens deletes all refresh tokens for a user
func (r *Repository) InvalidateAllRefreshTokens(userID string) error {
	_, err := r.db.Exec(`
		DELETE FROM refresh_tokens
		WHERE user_id = $1
	`, userID)
	return err
}

// GetUserByID retrieves user by ID
func (r *Repository) GetUserByID(userID string) (*User, error) {
	var user User
	err := r.db.QueryRow(`
		SELECT id, email, password_hash, primary_role, created_at, COALESCE(email_verified, false)
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.PrimaryRole, &user.CreatedAt, &user.EmailVerified)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	return &user, err
}

// StoreRefreshToken saves refresh token
func (r *Repository) StoreRefreshToken(userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, revoked)
		VALUES ($1, $2, $3, false)
	`, userID, tokenHash, expiresAt)
	return err
}

// GetRefreshToken retrieves refresh token by hash
func (r *Repository) GetRefreshToken(tokenHash string) (*RefreshToken, error) {
	var rt RefreshToken
	err := r.db.QueryRow(`
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens WHERE token_hash = $1
	`, tokenHash).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("token not found")
	}
	return &rt, err
}

// RevokeRefreshToken marks token as revoked
func (r *Repository) RevokeRefreshToken(tokenHash string) error {
	_, err := r.db.Exec(`
		UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1
	`, tokenHash)
	return err
}

// RevokeAllUserRefreshTokens revokes all tokens for a user
func (r *Repository) RevokeAllUserRefreshTokens(userID string) error {
	_, err := r.db.Exec(`
		UPDATE refresh_tokens SET revoked = true
		WHERE user_id = $1 AND revoked = false
	`, userID)
	return err
}

// CreateEmailVerificationToken creates email verification token
func (r *Repository) CreateEmailVerificationToken(userID, token string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO email_verification_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, userID, token, expiresAt)
	return err
}

// GetEmailVerificationToken retrieves verification token
func (r *Repository) GetEmailVerificationToken(token string) (string, time.Time, error) {
	var userID string
	var expiresAt time.Time
	
	err := r.db.QueryRow(`
		SELECT user_id, expires_at
		FROM email_verification_tokens
		WHERE token = $1
	`, token).Scan(&userID, &expiresAt)
	
	return userID, expiresAt, err
}

// MarkEmailAsVerified marks user email as verified
func (r *Repository) MarkEmailAsVerified(userID string) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET email_verified = true,
		    email_verified_at = NOW()
		WHERE id = $1
	`, userID)
	return err
}

// DeleteEmailVerificationToken deletes verification token
func (r *Repository) DeleteEmailVerificationToken(token string) error {
	_, err := r.db.Exec(`
		DELETE FROM email_verification_tokens
		WHERE token = $1
	`, token)
	return err
}

// IsEmailVerified checks if email is verified
func (r *Repository) IsEmailVerified(userID string) (bool, error) {
	var verified bool
	err := r.db.QueryRow(`
		SELECT email_verified FROM users WHERE id = $1
	`, userID).Scan(&verified)
	return verified, err
}
