package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/magnusohle/openanki-backend/internal/database"
)

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func main() {
	// 1. Load Enviroment
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: No .env file found, relying on system vars")
	}

	// 2. Init DB
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	email := "appreview@checkst.app"
	password := "AppleReview123!"
	username := "Reviewer"

	// 3. Check / Create User
	user, err := database.GetUserByEmail(email)
	if err != nil {
		log.Fatalf("DB Error checking user: %v", err)
	}

	if user == nil {
		log.Println("Creating Reviewer Account...")
		hashed := hashPassword(password)
		user, err = database.CreateUser(email, hashed, username)
		if err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		log.Printf("✅ User Created: ID %d\n", user.ID)
	} else {
		log.Printf("ℹ️ User already exists: ID %d\n", user.ID)
	}

	// 4. Set Pro Status
	_, err = database.DB.Exec(`UPDATE users SET subscription_status='pro', subscription_expiry='2030-01-01 00:00:00' WHERE id=?`, user.ID)
	if err != nil {
		log.Fatalf("Failed to set pro status: %v", err)
	}
	log.Println("✅ Set Subscription to PRO (Expires 2030)")

	// 5. Create / Check Group
	vargroupID := 0
	err = database.DB.QueryRow("SELECT id FROM groups WHERE creator_id = ? AND name = ?", user.ID, "Review Team").Scan(&vargroupID)
	if err != nil {
		// Not found, create
		g, err := database.CreateGroup("Review Team", "Official Reviewers", "Apple", "Review", user.ID)
		if err != nil {
			log.Fatalf("Failed to create group: %v", err)
		}
		log.Printf("✅ Created Group: %s (Code: %s)\n", g.Name, g.InviteCode)
		vargroupID = g.ID
	} else {
		log.Printf("ℹ️ Group 'Review Team' already exists: ID %d\n", vargroupID)
	}

	// 6. Create Deck
	// Check if exists
	var deckID int
	err = database.DB.QueryRow("SELECT id FROM group_decks WHERE uploader_id = ? AND name = ?", user.ID, "Capitals Demo").Scan(&deckID)
	if err != nil {
		// Create
		deckData := map[string]interface{}{
			"notes": []map[string]interface{}{
				{
					"fields": []string{"France", "Paris"},
					"tags":   []string{"geography"},
				},
				{
					"fields": []string{"Germany", "Berlin"},
					"tags":   []string{"geography"},
				},
				{
					"fields": []string{"Spain", "Madrid"},
					"tags":   []string{"geography"},
				},
			},
		}
		jsonBytes, _ := json.Marshal(deckData)

		_, err = database.DB.Exec(`INSERT INTO group_decks (group_id, uploader_id, name, card_count, deck_data) VALUES (?, ?, ?, ?, ?)`,
			vargroupID, user.ID, "Capitals Demo", 3, string(jsonBytes))
		
		if err != nil {
			log.Printf("❌ Failed to create deck: %v\n", err)
		} else {
			log.Println("✅ Created 'Capitals Demo' Deck")
		}
	} else {
		log.Println("ℹ️ Deck 'Capitals Demo' already exists")
	}

	log.Println("🎉 SEEDING COMPLETE")
	log.Printf("Credentials:\nEmail: %s\nPass: %s\n", email, password)
}
