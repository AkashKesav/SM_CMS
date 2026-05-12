package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// EmailService handles sending emails via HTTP APIs (not SMTP, which is blocked on DigitalOcean)
type EmailService struct {
	client *http.Client
}

func NewEmailService() *EmailService {
	return &EmailService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendPasswordResetEmail sends a password reset email using the first available provider
// Returns the reset link and any error encountered
func (s *EmailService) SendPasswordResetEmail(targetEmail, resetLink string) (string, error) {
	// Try SendGrid first (uses HTTPS, not blocked by DigitalOcean)
	if sendGridKey := os.Getenv("SENDGRID_API_KEY"); sendGridKey != "" {
		if err := s.sendViaSendGrid(sendGridKey, targetEmail, resetLink); err == nil {
			return resetLink, nil
		}
	}

	// Try Mailgun next
	if mailgunKey := os.Getenv("MAILGUN_API_KEY"); mailgunKey != "" {
		domain := os.Getenv("MAILGUN_DOMAIN")
		if domain != "" {
			if err := s.sendViaMailgun(mailgunKey, domain, targetEmail, resetLink); err == nil {
				return resetLink, nil
			}
		}
	}

	// Try Brevo (formerly Sendinblue)
	if brevoKey := os.Getenv("BREVO_API_KEY"); brevoKey != "" {
		if err := s.sendViaBrevo(brevoKey, targetEmail, resetLink); err == nil {
			return resetLink, nil
		}
	}

	// If no email provider worked, return the link for manual delivery
	// The caller can decide what to do with it
	return resetLink, fmt.Errorf("no email provider configured or all failed")
}

// sendViaSendGrid sends email using SendGrid HTTP API
func (s *EmailService) sendViaSendGrid(apiKey, toEmail, resetLink string) error {
	type Personalization struct {
		To []struct {
			Email string `json:"email"`
		} `json:"to"`
		Subject string `json:"subject"`
	}

	type Content struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}

	payload := map[string]interface{}{
		"personalizations": []Personalization{{
			To: []struct {
				Email string `json:"email"`
			}{{Email: toEmail}},
			Subject: "Password Reset Request",
		}},
		"from": struct {
			Email string `json:"email"`
			Name  string `json:"name,omitempty"`
		}{
			Email: os.Getenv("SENDGRID_FROM_EMAIL"),
			Name:  os.Getenv("SENDGRID_FROM_NAME"),
		},
		"content": []Content{{
			Type: "text/html",
			Value: fmt.Sprintf(`
				<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e1e1e1; border-radius: 10px;">
					<h2 style="color: #333;">Password Reset</h2>
					<p>Hello,</p>
					<p>We received a request to reset your password. Click the button below to set a new password. This link is valid for 15 minutes.</p>
					<div style="margin: 30px 0;">
						<a href="%s" style="background-color: #6366f1; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">Reset Password</a>
					</div>
					<p style="color: #666; font-size: 14px;">If the button above doesn't work, copy and paste this link into your browser:</p>
					<p style="color: #666; font-size: 14px; word-break: break-all;">%s</p>
					<hr style="margin: 30px 0; border: 0; border-top: 1px solid #eee;" />
					<p style="color: #999; font-size: 12px;">If you didn't request this, you can safely ignore this email.</p>
				</div>
			`, resetLink, resetLink),
		}},
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SendGrid error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendViaMailgun sends email using Mailgun HTTP API
func (s *EmailService) sendViaMailgun(apiKey, domain, toEmail, resetLink string) error {
	formData := fmt.Sprintf(
		"from=%s&to=%s&subject=Password Reset Request&html=%s",
		os.Getenv("MAILGUN_FROM_EMAIL"),
		toEmail,
		fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e1e1e1; border-radius: 10px;">
				<h2 style="color: #333;">Password Reset</h2>
				<p>Hello,</p>
				<p>We received a request to reset your password. Click the button below to set a new password. This link is valid for 15 minutes.</p>
				<div style="margin: 30px 0;">
					<a href="%s" style="background-color: #6366f1; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">Reset Password</a>
				</div>
				<p style="color: #666; font-size: 14px;">If the button above doesn't work, copy and paste this link into your browser:</p>
				<p style="color: #666; font-size: 14px; word-break: break-all;">%s</p>
				<hr style="margin: 30px 0; border: 0; border-top: 1px solid #eee;" />
				<p style="color: #999; font-size: 12px;">If you didn't request this, you can safely ignore this email.</p>
			</div>
		`, resetLink, resetLink),
	)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", domain), bytes.NewBufferString(formData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Mailgun error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendViaBrevo sends email using Brevo (Sendinblue) HTTP API
func (s *EmailService) sendViaBrevo(apiKey, toEmail, resetLink string) error {
	payload := map[string]interface{}{
		"sender": map[string]string{
			"email": os.Getenv("BREVO_FROM_EMAIL"),
			"name":  os.Getenv("BREVO_FROM_NAME"),
		},
		"to": []map[string]string{{
			"email": toEmail,
		}},
		"subject": "Password Reset Request",
		"htmlContent": fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e1e1e1; border-radius: 10px;">
				<h2 style="color: #333;">Password Reset</h2>
				<p>Hello,</p>
				<p>We received a request to reset your password. Click the button below to set a new password. This link is valid for 15 minutes.</p>
				<div style="margin: 30px 0;">
					<a href="%s" style="background-color: #6366f1; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">Reset Password</a>
				</div>
				<p style="color: #666; font-size: 14px;">If the button above doesn't work, copy and paste this link into your browser:</p>
				<p style="color: #666; font-size: 14px; word-break: break-all;">%s</p>
				<hr style="margin: 30px 0; border: 0; border-top: 1px solid #eee;" />
				<p style="color: #999; font-size: 12px;">If you didn't request this, you can safely ignore this email.</p>
			</div>
		`, resetLink, resetLink),
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Brevo error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
