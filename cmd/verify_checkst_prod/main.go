package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const BaseURL = "https://checkst.app/api/v1"

func main() {
	log.Println("🚀 Starting PROD Verification for checkst.app")

	// 1. Register User (Fresh every time to avoid conflicts)
	// Use timestamp to guarantee unique email
	uniqueID := time.Now().Unix()
	email := fmt.Sprintf("verify_%d@checkst.app", uniqueID)
	password := "TestVerify123!"
	username := fmt.Sprintf("verify_%d", uniqueID)

	log.Printf("1️⃣ Registering User: %s", email)
	token, userID := registerUser(email, password, username)
	if token == "" {
		log.Fatal("❌ Registration failed")
	}
	log.Printf("✅ Registered (User ID: %d)", userID)

	// 2. Test Media Sync (R2)
	log.Println("2️⃣ Testing Media Upload (R2 Presigned URL)")
	
	// A. Get Presigned URL
	initUploadReq, _ := json.Marshal(map[string]interface{}{
		"filename": "test_image.jpg",
		"hash":     fmt.Sprintf("hash_%d", uniqueID),
		"size":     1024,
	})
	
	resp, err := authenticatedRequest("POST", BaseURL+"/sync/media/upload", token, initUploadReq)
	if err != nil {
		log.Fatalf("❌ Failed to get upload URL: %v", err)
	}
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        log.Fatalf("❌ Upload init failed (%d): %s", resp.StatusCode, string(body))
    }
    
    var uploadResp map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&uploadResp)
    
	uploadURL, ok := uploadResp["upload_url"].(string)
    if !ok || uploadURL == "" {
        // If it's 200 but no URL, maybe it says "exists"?
        log.Fatalf("❌ No upload_url in response: %v", uploadResp)
    }
    log.Println("   Got Presigned URL! 🔗")

    // B. Upload Dummy Data to R2
    log.Println("   Uploading 1KB dummy data to R2...")
    dummyData := bytes.Repeat([]byte("A"), 1024)
    uploadReq, _ := http.NewRequest("PUT", uploadURL, bytes.NewReader(dummyData))
    uploadReq.Header.Set("Content-Type", "application/octet-stream")
    
    client := &http.Client{}
    r2Resp, err := client.Do(uploadReq)
    if err != nil {
         log.Fatalf("❌ Failed R2 upload: %v", err)
    }
    defer r2Resp.Body.Close()
    
    if r2Resp.StatusCode != 200 {
        log.Fatalf("❌ R2 Upload rejected (%d)", r2Resp.StatusCode)
    }
    log.Println("✅ R2 Upload Successful!")

    // 3. Create Group (Deep Link / Invite Code Check)
    log.Println("3️⃣ Testing Group Creation & Invite Code")
    groupReq, _ := json.Marshal(map[string]interface{}{
        "name": "Verification Group",
        "description": "Testing Invite Codes",
    })
    
    gResp, err := authenticatedRequest("POST", BaseURL+"/groups", token, groupReq)
    if err != nil {
         log.Fatalf("❌ Failed to create group: %v", err)
    }
    defer gResp.Body.Close()
    
    if gResp.StatusCode != 201 && gResp.StatusCode != 200 {
         body, _ := io.ReadAll(gResp.Body)
         log.Fatalf("❌ Group creation failed (%d): %s", gResp.StatusCode, string(body))
    }
    
    var groupResp map[string]interface{}
    json.NewDecoder(gResp.Body).Decode(&groupResp)
    
    inviteCode, _ := groupResp["invite_code"].(string)
    if inviteCode == "" {
        log.Fatalf("❌ No invite_code returned! Schema migration might have failed. Resp: %v", groupResp)
    }
    log.Printf("✅ Group Created with Code: %s", inviteCode)

    log.Println("🎉 ALL SYSTEMS GO! Production is Healthy.")
}

func registerUser(email, password, username string) (string, int) {
	reqBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
		"username": username,
	})

	resp, err := http.Post(BaseURL+"/auth/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("Req failed: %v", err)
		return "", 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Register status %d: %s", resp.StatusCode, string(body))
		return "", 0
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	token := res["token"].(string)
	user := res["user"].(map[string]interface{})
	id := int(user["id"].(float64))
	return token, id
}

func authenticatedRequest(method, url, token string, body []byte) (*http.Response, error) {
    req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    return client.Do(req)
}
