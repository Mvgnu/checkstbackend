package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/magnusohle/openanki-backend/internal/mailer"
)

func main() {
	// Try loading .env file (ignore error if not present, env vars might be set globally)
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/test_email/main.go <recipient_email>")
	}

	recipient := os.Args[1]
	
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	
	log.Printf("📧 Configured SMTP Settings:")
	log.Printf("   Host: %s", host)
	log.Printf("   Port: %s", port)
	log.Printf("   User: %s", user)
	log.Printf("\n🚀 Sending test reset email to: %s...", recipient)

	err := mailer.SendResetEmail(recipient, "TEST-123456")
	if err != nil {
		log.Fatalf("\n❌ FAILED to send email: %v", err)
	}

	log.Println("\n✅ Email sent successfully! Check your inbox (and spam).")
}
