package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/magnusohle/openanki-backend/internal/media"
)

func main() {
	// 1. Load Enviroment
	// Try loading from .env.apple or .env in current CWD
	if err := godotenv.Load(".env.apple"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
            // Fallback to relative path if running from inside cmd/verify_r2 (via vs code debug etc)
            if err := godotenv.Load("../../.env.apple"); err != nil {
                 godotenv.Load("../../.env")
            }
		} else {
            log.Println("Loaded .env")
        }
	} else {
        log.Println("Loaded .env.apple")
    }

	// 2. Initialize Service
	log.Println("Initializing S3 Service...")
	s3Service, err := media.InitS3()
	if err != nil {
		log.Fatalf("❌ Failed to init S3 Service: %v", err)
	}
	if !s3Service.IsConfigured {
		log.Fatal("❌ S3 Service not configured (Environment variables missing?)")
	}
	log.Printf("✅ S3 Service Valid (Bucket: %s, Endpoint: https://%s.r2.cloudflarestorage.com)\n", 
        s3Service.Bucket, os.Getenv("R2_ACCOUNT_ID"))

	// 3. Test Presigned PUT
	testKey := fmt.Sprintf("test_verification_%d.txt", time.Now().Unix())
	log.Printf("Testing Upload for key: %s...", testKey)

	putURL, err := s3Service.GetPresignedPutURL(testKey, "text/plain")
	if err != nil {
		log.Fatalf("❌ Failed to generate PUT URL: %v", err)
	}
	log.Println("✅ Generated Presigned PUT URL")

	// perform upload
	content := []byte("Hello R2 from OpenAnki verification script!")
	req, _ := http.NewRequest("PUT", putURL, bytes.NewReader(content))
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("❌ Upload request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("❌ Upload returned status %d: %s", resp.StatusCode, string(body))
	}
	log.Println("✅ Upload Successful (200 OK)")

	// 4. Test Presigned GET
	log.Println("Testing Download...")
	getURL, err := s3Service.GetPresignedGetURL(testKey)
	if err != nil {
		log.Fatalf("❌ Failed to generate GET URL: %v", err)
	}

	getResp, err := http.Get(getURL)
	if err != nil {
		log.Fatalf("❌ Download request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != 200 {
		log.Fatalf("❌ Download returned status %d", getResp.StatusCode)
	}

	downloadedBody, _ := io.ReadAll(getResp.Body)
	if !strings.EqualFold(string(downloadedBody), string(content)) {
		log.Printf("⚠️ Content mismatch! Expected '%s', got '%s'", content, downloadedBody)
	} else {
		log.Println("✅ Content Verified")
	}

	log.Println("🎉 R2 INTEGRATION VERIFIED & WORKING!")
}
