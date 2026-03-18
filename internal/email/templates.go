package email

import (
	"strings"
)

// GetPasswordResetHTML generates password reset email HTML
func GetPasswordResetHTML(data PasswordResetData) string {
	template := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password - CrunchAlpha</title>
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
    <table width="100%" cellpadding="0" cellspacing="0" style="background-color: #f4f4f4; padding: 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" style="background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
                    <tr>
                        <td style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 40px 20px; text-align: center;">
                            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: bold;">CrunchAlpha</h1>
                            <p style="color: #ffffff; margin: 10px 0 0 0; font-size: 14px;">Professional Trading Analytics Platform</p>
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 40px 30px;">
                            <h2 style="color: #333333; margin: 0 0 20px 0; font-size: 24px;">Reset Your Password</h2>
                            <p style="color: #666666; line-height: 1.6; margin: 0 0 20px 0;">
                                You requested to reset your password for: <strong>{{EMAIL}}</strong>
                            </p>
                            <p style="color: #666666; line-height: 1.6; margin: 0 0 30px 0;">
                                Click the button below to reset your password. This link will expire in <strong>{{EXPIRES}}</strong>.
                            </p>
                            <table width="100%" cellpadding="0" cellspacing="0">
                                <tr>
                                    <td align="center" style="padding: 0 0 30px 0;">
                                        <a href="{{RESET_LINK}}" style="display: inline-block; padding: 16px 40px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: bold; font-size: 16px;">Reset Password</a>
                                    </td>
                                </tr>
                            </table>
                            <p style="color: #999999; font-size: 14px; line-height: 1.6; margin: 0 0 10px 0;">
                                Or copy this link: <span style="color: #667eea; word-break: break-all;">{{RESET_LINK}}</span>
                            </p>
                        </td>
                    </tr>
                    <tr>
                        <td style="background-color: #f8f9fa; padding: 20px 30px; text-align: center;">
                            <p style="color: #999999; font-size: 12px; margin: 0;">© 2026 CrunchAlpha. All rights reserved.</p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`
	template = strings.ReplaceAll(template, "{{EMAIL}}", data.UserEmail)
	template = strings.ReplaceAll(template, "{{RESET_LINK}}", data.ResetLink)
	template = strings.ReplaceAll(template, "{{EXPIRES}}", data.ExpiresIn)
	return template
}

// GetWelcomeHTML generates welcome email HTML
func GetWelcomeHTML(data WelcomeData) string {
	userName := data.UserName
	if userName == "" {
		userName = strings.Split(data.UserEmail, "@")[0]
	}
	
	template := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to CrunchAlpha</title>
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
    <table width="100%" cellpadding="0" cellspacing="0" style="background-color: #f4f4f4; padding: 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" style="background-color: #ffffff; border-radius: 8px; overflow: hidden;">
                    <tr>
                        <td style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 40px 20px; text-align: center;">
                            <h1 style="color: #ffffff; margin: 0; font-size: 32px;">Welcome to CrunchAlpha! 🎉</h1>
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 40px 30px;">
                            <h2 style="color: #333333; margin: 0 0 20px 0;">Hi {{NAME}}!</h2>
                            <p style="color: #666666; line-height: 1.8; margin: 0 0 20px 0; font-size: 16px;">
                                Thank you for joining <strong>CrunchAlpha</strong> - Indonesia's premier trading analytics platform!
                            </p>
                            <p style="color: #666666; line-height: 1.8; margin: 0 0 30px 0;">
                                Your account is ready. Start tracking your performance with AlphaRank™!
                            </p>
                            <table width="100%" cellpadding="0" cellspacing="0">
                                <tr>
                                    <td align="center">
                                        <a href="https://crunchalpha.com/dashboard" style="display: inline-block; padding: 16px 40px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: bold;">Go to Dashboard</a>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    <tr>
                        <td style="background-color: #f8f9fa; padding: 20px 30px; text-align: center;">
                            <p style="color: #999999; font-size: 12px; margin: 0;">© 2026 CrunchAlpha</p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`
	template = strings.ReplaceAll(template, "{{NAME}}", userName)
	return template
}
