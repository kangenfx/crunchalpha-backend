package auth

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}

	if req.PrimaryRole == "" {
		req.PrimaryRole = "trader"
	}

	user, token, err := h.service.Register(req.Email, req.Password, req.PrimaryRole)
	if err != nil {
		if err == ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed", "message": err.Error()})
		return
	}

	// Handle referral code
	if req.RefCode != "" {
		go func() {
			var affID string
			_ = h.service.repo.db.QueryRow(
				`SELECT id::text FROM affiliates WHERE code=$1`, req.RefCode,
			).Scan(&affID)
			if affID != "" {
				h.service.repo.db.Exec(`
					INSERT INTO affiliate_referrals (affiliate_id, referred_user_id, referred_email, status)
					VALUES ($1::uuid, $2::uuid, $3, 'ACTIVE')
					ON CONFLICT DO NOTHING`,
					affID, user.ID, user.Email)
				h.service.repo.db.Exec(`
					UPDATE affiliates SET total_referrals = total_referrals + 1,
					active_referrals = active_referrals + 1
					WHERE id=$1::uuid`, affID)
			}
		}()
	}
	c.JSON(http.StatusCreated, LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        user,
	})
}


// POST /api/auth/impersonate — exchange impersonate token for JWT
func (h *Handler) Impersonate(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user_id from impersonate_tokens
	var userID string
	err := h.service.repo.db.QueryRow(`
		SELECT user_id::text FROM impersonate_tokens
		WHERE token = $1 AND expires_at > NOW()
	`, req.Token).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Delete token after use
	h.service.repo.db.Exec(`DELETE FROM impersonate_tokens WHERE token = $1`, req.Token)

	// Get user info
	var email, role string
	h.service.repo.db.QueryRow(`SELECT email, primary_role FROM users WHERE id=$1::uuid`, userID).Scan(&email, &role)

	// Generate JWT
	accessToken, err := GenerateToken(userID, email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"email":        email,
	})
}
// Login now handles refresh token
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}

	user, accessToken, refreshToken, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed", "message": err.Error()})
		return
	}

	// Set refresh token in httpOnly cookie
	c.SetCookie("refresh_token", refreshToken, 7*24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		User:        user,
	})
}

// Refresh handles token refresh
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
			return
		}
		refreshToken = req.RefreshToken
	}

	accessToken, _, err := h.service.RefreshAccessToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token refresh failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
	})
}

// Logout handles logout
func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
			return
		}
		refreshToken = req.RefreshToken
	}

	err = h.service.Logout(refreshToken)
	if err != nil {
		log.Printf("Logout error: %v", err)
	}

	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutAll handles logout from all devices
func (h *Handler) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := h.service.LogoutAll(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices"})
}

// Password reset handlers
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}

	token, err := h.service.ForgotPassword(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	response := gin.H{"message": "If your email is registered, you will receive a password reset link."}
	if token != "" {
		response["token"] = token
		response["test_note"] = "Token shown for testing only"
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) ValidateResetToken(c *gin.Context) {
	token := c.Param("token")
	err := h.service.ValidateResetToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Token is valid"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}

	err := h.service.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password reset failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password reset successfully"})
}

// VerifyEmail handles email verification
func (h *Handler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		var req VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request",
				"message": "Token is required",
			})
			return
		}
		token = req.Token
	}

	err := h.service.VerifyEmail(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email verification failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Email verified successfully! You can now access all features.",
	})
}

// ResendVerification resends verification email
func (h *Handler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"message": err.Error(),
		})
		return
	}

	err := h.service.ResendVerificationEmail(req.Email)
	if err != nil {
		if err.Error() == "email already verified" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email already verified",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send verification email",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If your email is registered and not verified, you will receive a verification link.",
	})
}
