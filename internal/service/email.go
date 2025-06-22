package service

import (
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	// LINT-FIX-2025: Import golang.org/x/text/cases for proper title casing
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/pageza/alchemorsel-v2/backend/internal/models"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	fromName     string
	adminEmail   string
}

// readSecret reads a Docker secret from the secrets directory
func readSecret(name string) string {
	secretsDir := os.Getenv("SECRETS_DIR")
	if secretsDir == "" {
		secretsDir = "/run/secrets"
	}
	secretPath := filepath.Join(secretsDir, name)
	if data, err := os.ReadFile(secretPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

func NewEmailService() IEmailService {
	service := &EmailService{
		smtpHost:     readSecret("smtp_host"),
		smtpPort:     readSecret("smtp_port"),
		smtpUsername: readSecret("smtp_username"),
		smtpPassword: readSecret("smtp_password"),
		fromEmail:    readSecret("email_from"),
		fromName:     readSecret("email_from_name"),
		adminEmail:   readSecret("admin_email"),
	}

	// Debug logging
	fmt.Printf("Email service initialized with ADMIN_EMAIL: %s\n", service.adminEmail)
	fmt.Printf("Email service initialized with SMTP_HOST: %s\n", service.smtpHost)
	fmt.Printf("Email service initialized with SMTP_USERNAME: %s\n", service.smtpUsername)

	return service
}

func (s *EmailService) SendFeedbackNotification(feedback *models.Feedback, user *models.User) error {
	// Use admin email or fallback to fromEmail
	toEmail := s.adminEmail
	if toEmail == "" {
		toEmail = s.fromEmail
	}

	// LINT-FIX-2025: Replace deprecated strings.Title with golang.org/x/text/cases
	// strings.Title is deprecated since Go 1.18 due to Unicode handling issues
	caser := cases.Title(language.English)
	// Create email subject
	subject := fmt.Sprintf("[Alchemorsel] New %s: %s", caser.String(feedback.Type), feedback.Title)

	// Create detailed email body
	body := s.buildFeedbackEmailBody(feedback, user)

	return s.SendEmail(toEmail, subject, body)
}

func (s *EmailService) SendEmail(to, subject, body string) error {
	// If SMTP is not configured, log the email instead
	if s.smtpHost == "" || s.smtpPort == "" {
		fmt.Printf("SMTP not configured, logging email:\n")
		fmt.Printf("To: %s\n", to)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Body:\n%s\n", body)
		fmt.Printf("--- End Email ---\n")
		return nil
	}

	// Set up authentication
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	// Compose message
	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", to, from, subject, body))

	// Send email
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	err := smtp.SendMail(addr, auth, s.fromEmail, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *EmailService) SendVerificationEmail(user *models.User, token string) error {
	subject := "Verify Your Email - Alchemorsel"
	body := s.buildVerificationEmailBody(user, token)
	return s.SendEmail(user.Email, subject, body)
}

func (s *EmailService) SendWelcomeEmail(user *models.User) error {
	subject := "Welcome to Alchemorsel!"
	body := s.buildWelcomeEmailBody(user)
	return s.SendEmail(user.Email, subject, body)
}

func (s *EmailService) buildVerificationEmailBody(user *models.User, token string) string {
	// Construct verification URL
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173" // Development fallback
	}
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Verify Your Email - Alchemorsel</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
	<div style="background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0;">
		<h1 style="margin: 0; font-size: 28px;">🍳 Alchemorsel</h1>
		<p style="margin: 10px 0 0 0; font-size: 16px;">Your AI-Powered Recipe Companion</p>
	</div>
	
	<div style="background-color: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
		<h2 style="color: #4CAF50; margin-top: 0;">Welcome, %s!</h2>
		<p>Thank you for signing up for Alchemorsel. To start creating amazing recipes with AI, please verify your email address.</p>
		
		<div style="text-align: center; margin: 30px 0;">
			<a href="%s" style="background-color: #4CAF50; color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; font-size: 16px; display: inline-block;">
				Verify Email Address
			</a>
		</div>
		
		<p style="color: #666; font-size: 14px;">If the button above doesn't work, copy and paste this link into your browser:</p>
		<p style="background-color: #eee; padding: 10px; border-radius: 5px; word-break: break-all; font-size: 12px;">%s</p>
		
		<div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd;">
			<p style="color: #666; font-size: 12px; margin: 0;">
				This verification link will expire in 24 hours. If you didn't sign up for Alchemorsel, you can safely ignore this email.
			</p>
		</div>
	</div>
</body>
</html>
	`, user.Name, verificationURL, verificationURL)
}

func (s *EmailService) buildWelcomeEmailBody(user *models.User) string {
	// Get frontend URL with fallback
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Development fallback
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Welcome to Alchemorsel!</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
	<div style="background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0;">
		<h1 style="margin: 0; font-size: 28px;">🎉 Welcome to Alchemorsel!</h1>
		<p style="margin: 10px 0 0 0; font-size: 16px;">Your AI-Powered Recipe Journey Begins</p>
	</div>
	
	<div style="background-color: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
		<h2 style="color: #4CAF50; margin-top: 0;">Hello %s!</h2>
		<p>Your email has been verified successfully! Welcome to the Alchemorsel community.</p>
		
		<h3 style="color: #4CAF50;">What can you do now?</h3>
		<ul style="padding-left: 20px;">
			<li style="margin-bottom: 10px;">🤖 <strong>Generate AI Recipes:</strong> Describe what you want to cook and let our AI create personalized recipes</li>
			<li style="margin-bottom: 10px;">🍴 <strong>Fork Recipes:</strong> Take any recipe and modify it to your taste</li>
			<li style="margin-bottom: 10px;">⭐ <strong>Save Favorites:</strong> Build your personal recipe collection</li>
			<li style="margin-bottom: 10px;">🔍 <strong>Smart Search:</strong> Find recipes using AI-powered semantic search</li>
		</ul>
		
		<div style="text-align: center; margin: 30px 0;">
			<a href="%s" style="background-color: #4CAF50; color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; font-size: 16px; display: inline-block;">
				Start Cooking with AI
			</a>
		</div>
		
		<div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd;">
			<p style="color: #666; font-size: 12px; margin: 0;">
				Happy cooking! 🍳<br>
				The Alchemorsel Team
			</p>
		</div>
	</div>
</body>
</html>
	`, user.Name, frontendURL)
}

func (s *EmailService) buildFeedbackEmailBody(feedback *models.Feedback, user *models.User) string {
	// LINT-FIX-2025: Create title caser for proper case handling
	caser := cases.Title(language.English)
	var userInfo string
	if user != nil {
		userInfo = fmt.Sprintf(`
			<p><strong>User Information:</strong></p>
			<ul>
				<li>Email: %s</li>
				<li>User ID: %s</li>
				<li>Created: %s</li>
			</ul>
		`, user.Email, user.ID, user.CreatedAt.Format("2006-01-02 15:04:05"))
	} else {
		userInfo = "<p><strong>User:</strong> Anonymous</p>"
	}

	var technicalInfo string
	if feedback.UserAgent != "" || feedback.URL != "" {
		technicalInfo = fmt.Sprintf(`
			<p><strong>Technical Information:</strong></p>
			<ul>
				%s
				%s
			</ul>
		`,
			func() string {
				if feedback.URL != "" {
					return fmt.Sprintf("<li>Page URL: %s</li>", feedback.URL)
				}
				return ""
			}(),
			func() string {
				if feedback.UserAgent != "" {
					return fmt.Sprintf("<li>User Agent: %s</li>", feedback.UserAgent)
				}
				return ""
			}(),
		)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>New Feedback - Alchemorsel</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<h2>New %s Report</h2>
	
	<div style="background-color: #f9f9f9; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0;">
		<h3>%s</h3>
		<p><strong>Type:</strong> %s</p>
		<p><strong>Priority:</strong> %s</p>
		<p><strong>Status:</strong> %s</p>
		<p><strong>Submitted:</strong> %s</p>
	</div>

	<div style="margin: 20px 0;">
		<h4>Description:</h4>
		<div style="background-color: #f5f5f5; padding: 15px; border-radius: 5px;">
			%s
		</div>
	</div>

	%s

	%s

	<div style="margin-top: 30px; padding: 15px; background-color: #e9ecef; border-radius: 5px;">
		<p><strong>Feedback ID:</strong> %s</p>
		<p style="font-size: 12px; color: #666;">
			This is an automated notification from the Alchemorsel feedback system.
		</p>
	</div>
</body>
</html>
	`,
		// LINT-FIX-2025: Use proper title casing with golang.org/x/text/cases
		caser.String(feedback.Type),
		feedback.Title,
		caser.String(feedback.Type),
		caser.String(feedback.Priority),
		caser.String(feedback.Status),
		feedback.CreatedAt.Format("2006-01-02 15:04:05 MST"),
		strings.ReplaceAll(feedback.Description, "\n", "<br>"),
		userInfo,
		technicalInfo,
		feedback.ID,
	)
}
