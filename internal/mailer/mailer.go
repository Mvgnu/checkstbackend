package mailer

import (
	"fmt"
	"net/smtp"
	"os"
)

// SendResetEmail sends a password reset code to the specified email address
func SendResetEmail(toEmail, code string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP configuration missing")
	}

	if smtpFrom == "" {
		smtpFrom = "noreply@checkst.app"
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	
	subject := "Reset your Checkst Password"
	body := fmt.Sprintf(`Hello,

You requested a password reset for your Checkst account.
Your reset code is:

%s

This code will expire in 10 minutes.
If you did not request this reset, please ignore this email.
Also, check your spam folder if you don't use this often!

Best regards,
Checkst Team`, code)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", toEmail, smtpFrom, subject, body))
	
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	
	err := smtp.SendMail(addr, auth, smtpFrom, []string{toEmail}, msg)
	if err != nil {
		return err
	}
	
	return nil
}
