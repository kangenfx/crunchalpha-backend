package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"crunchalpha-v3/internal/email"
)

type Service struct {
	repo        *Repository
	emailSender *email.Sender
}

func NewService(db *sql.DB) *Service {
	return &Service{
		repo:        NewRepository(db),
		emailSender: email.NewSender(),
	}
}

func (s *Service) Register(email, password, role string) (*User, string, error) {
	user, err := s.repo.CreateUser(email, password, role)
	if err != nil {
		return nil, "", err
	}

	token, err := GenerateToken(user.ID, user.Email, user.PrimaryRole)
	if err != nil {
		return nil, "", err
	}

	// Send welcome email (async - don't block registration)
	go func() {
		if err := s.emailSender.SendWelcome(user.Email, ""); err != nil {
			log.Printf("Failed to send welcome email to %s: %v", user.Email, err)
		}
	}()

	// Send verification email (async)
	go func() {
		if err := s.SendVerificationEmail(user.ID, user.Email); err != nil {
			log.Printf("Failed to send verification email to %s: %v", user.Email, err)
		}
	}()

	return user, token, nil
}

func (s *Service) Login(email, password string) (*User, string, string, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, "", "", errors.New("invalid email or password")
		}
		return nil, "", "", err
	}

	if err := s.repo.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, "", "", errors.New("invalid email or password")
	}

	// Block login if email not verified
	if !user.EmailVerified {
		return nil, "", "", errors.New("email not verified. Please check your inbox and verify your email first")
	}

	accessToken, err := GenerateToken(user.ID, user.Email, user.PrimaryRole)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", "", err
	}

	tokenHash := HashToken(refreshToken)
	expiresAt := time.Now().Add(RefreshTokenDuration)
	err = s.repo.StoreRefreshToken(user.ID, tokenHash, expiresAt)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *Service) RefreshAccessToken(refreshToken string) (string, string, error) {
	tokenHash := HashToken(refreshToken)
	rt, err := s.repo.GetRefreshToken(tokenHash)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}
	if rt.Revoked {
		return "", "", errors.New("refresh token has been revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return "", "", errors.New("refresh token has expired")
	}
	user, err := s.repo.GetUserByID(rt.UserID)
	if err != nil {
		return "", "", err
	}
	accessToken, err := GenerateToken(user.ID, user.Email, user.PrimaryRole)
	if err != nil {
		return "", "", err
	}
	return accessToken, user.ID, nil
}

func (s *Service) Logout(refreshToken string) error {
	tokenHash := HashToken(refreshToken)
	return s.repo.RevokeRefreshToken(tokenHash)
}

func (s *Service) LogoutAll(userID string) error {
	return s.repo.RevokeAllUserRefreshTokens(userID)
}

func (s *Service) ForgotPassword(userEmail string) (string, error) {
	user, err := s.repo.GetUserByEmail(userEmail)
	if err != nil {
		// Don't reveal if email exists - but still return nil
		return "", nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(1 * time.Hour)
	err = s.repo.CreatePasswordResetToken(user.ID, token, expiresAt)
	if err != nil {
		return "", err
	}

	// Send password reset email (async)
	go func() {
		if err := s.emailSender.SendPasswordReset(user.Email, token); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", user.Email, err)
		}
	}()

	// For mock mode, return token for testing
	// In production with real SMTP, return empty string
	return token, nil
}

func (s *Service) ValidateResetToken(token string) error {
	prt, err := s.repo.GetPasswordResetToken(token)
	if err != nil {
		return errors.New("invalid token")
	}
	if prt.Used {
		return errors.New("token already used")
	}
	if time.Now().After(prt.ExpiresAt) {
		return errors.New("token expired")
	}
	return nil
}

func (s *Service) ResetPassword(token, newPassword string) error {
	prt, err := s.repo.GetPasswordResetToken(token)
	if err != nil {
		return errors.New("invalid token")
	}
	if prt.Used {
		return errors.New("token already used")
	}
	if time.Now().After(prt.ExpiresAt) {
		return errors.New("token expired")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	err = s.repo.UpdatePassword(prt.UserID, string(hash))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	err = s.repo.MarkTokenAsUsed(token)
	if err != nil {
		return fmt.Errorf("failed to invalidate token: %w", err)
	}
	err = s.repo.InvalidateAllRefreshTokens(prt.UserID)
	if err != nil {
		log.Printf("Warning: failed to invalidate refresh tokens: %v", err)
	}
	return nil
}

// SendVerificationEmail sends email verification link
func (s *Service) SendVerificationEmail(userID, userEmail string) error {
	// Generate verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	// Store token (valid for 24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)
	err := s.repo.CreateEmailVerificationToken(userID, token, expiresAt)
	if err != nil {
		return err
	}

	// Send verification email (async)
	go func() {
		verificationLink := fmt.Sprintf("http://45.32.118.117:5176/verify-email?token=%s", token)
		
		// Create email HTML
		htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 8px; padding: 40px;">
        <h1 style="color: #667eea; margin: 0 0 20px 0;">Verify Your Email</h1>
        <p style="color: #666666; line-height: 1.6; margin: 0 0 20px 0;">
            Thank you for registering with CrunchAlpha! Please verify your email address to activate your account.
        </p>
        <p style="margin: 0 0 30px 0;">
            <a href="%s" style="display: inline-block; padding: 16px 40px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: bold;">Verify Email</a>
        </p>
        <p style="color: #999999; font-size: 13px; margin: 0;">
            Or copy this link: <br>%s
        </p>
        <p style="color: #999999; font-size: 13px; margin: 20px 0 0 0;">
            This link will expire in 24 hours.
        </p>
    </div>
</body>
</html>
`, verificationLink, verificationLink)

		req := email.EmailRequest{
			To:      userEmail,
			Subject: "Verify Your CrunchAlpha Email",
			Body:    htmlBody,
			IsHTML:  true,
		}
		
		if err := s.emailSender.Send(req); err != nil {
			log.Printf("Failed to send verification email to %s: %v", userEmail, err)
		}
	}()

	return nil
}

// VerifyEmail verifies user email with token
func (s *Service) VerifyEmail(token string) error {
	// Get token info
	userID, expiresAt, err := s.repo.GetEmailVerificationToken(token)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("invalid verification token")
		}
		return err
	}

	// Check expiry
	if time.Now().After(expiresAt) {
		return errors.New("verification token has expired")
	}

	// Mark email as verified
	err = s.repo.MarkEmailAsVerified(userID)
	if err != nil {
		return err
	}

	// Delete token
	s.repo.DeleteEmailVerificationToken(token)

	return nil
}

// ResendVerificationEmail resends verification email
func (s *Service) ResendVerificationEmail(userEmail string) error {
	user, err := s.repo.GetUserByEmail(userEmail)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Check if already verified
	verified, err := s.repo.IsEmailVerified(user.ID)
	if err != nil {
		return err
	}
	if verified {
		return errors.New("email already verified")
	}

	// Send new verification email
	return s.SendVerificationEmail(user.ID, user.Email)
}
