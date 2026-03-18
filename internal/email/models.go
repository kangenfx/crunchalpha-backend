package email

// EmailRequest represents an email to be sent
type EmailRequest struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// EmailTemplate types
const (
	TemplatePasswordReset     = "password_reset"
	TemplateWelcome          = "welcome"
	TemplateEmailVerification = "email_verification"
)

// PasswordResetData for password reset email
type PasswordResetData struct {
	UserEmail string
	ResetLink string
	ExpiresIn string
}

// WelcomeData for welcome email
type WelcomeData struct {
	UserEmail string
	UserName  string
}

// EmailVerificationData for email verification
type EmailVerificationData struct {
	UserEmail        string
	VerificationLink string
}
