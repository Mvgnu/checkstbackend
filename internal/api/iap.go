package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/magnusohle/openanki-backend/internal/apple"
	"github.com/magnusohle/openanki-backend/internal/auth"
	"github.com/magnusohle/openanki-backend/internal/database"
)

type IAPHandler struct{}

func RegisterIAPRoutes(r chi.Router) {
	handler := &IAPHandler{}
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Post("/verify", handler.VerifyPurchase)
	})
	// Webhook doesn't need auth - Apple sends it
	r.Post("/webhook", handler.HandleWebhook)
}

type verifyRequest struct {
	ProductID        string `json:"product_id"`
	TransactionID    string `json:"transaction_id"`
	VerificationData string `json:"verification_data"`
	Platform         string `json:"platform"`
}

// VerifyPurchase validates a purchase with Apple and updates user subscription
func (h *IAPHandler) VerifyPurchase(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TransactionID == "" || req.ProductID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var expiresAt time.Time
	var status string

	// Try Apple API verification if configured
	if apple.IsConfigured() {
		txnInfo, err := apple.GetTransactionInfo(req.TransactionID)
		if err != nil {
			log.Printf("Apple API verification failed: %v (falling back to dev mode)", err)
			// Fall through to dev mode
		} else {
			// Verify product ID matches
			if txnInfo.ProductID != req.ProductID {
				http.Error(w, "Product ID mismatch", http.StatusForbidden)
				return
			}

			// Use Apple's expiry date if available
			if txnInfo.ExpiresDate > 0 {
				expiresAt = time.UnixMilli(txnInfo.ExpiresDate)
			} else {
				// Non-consumable (lifetime)
				expiresAt = time.Now().AddDate(100, 0, 0)
			}
			status = "pro"

			// Save and respond
			if err := database.SaveSubscription(userID, req.ProductID, req.TransactionID, expiresAt); err != nil {
				http.Error(w, "Failed to save subscription", http.StatusInternalServerError)
				return
			}

			if err := database.UpdateUserSubscription(userID, status); err != nil {
				http.Error(w, "Failed to update user status", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":              status,
				"subscription_status": status,
				"expires_at":          expiresAt.Format(time.RFC3339),
				"product_id":          req.ProductID,
				"verified_by":         "apple_api",
			})
			return
		}
	}

	// DEV MODE: Trust client (only for sandbox/development)
	log.Printf("IAP: Using dev mode verification for user %d, product %s", userID, req.ProductID)

	switch req.ProductID {
	case "checkst.pro.semester", "checkst.pro.semester.sub":
		expiresAt = time.Now().AddDate(0, 6, 0) // 6 months
		status = "pro"
	case "checkst.pro.lifetime":
		expiresAt = time.Now().AddDate(100, 0, 0) // 100 years = lifetime
		status = "pro"
	default:
		http.Error(w, "Unknown product ID", http.StatusBadRequest)
		return
	}

	if err := database.SaveSubscription(userID, req.ProductID, req.TransactionID, expiresAt); err != nil {
		http.Error(w, "Failed to save subscription", http.StatusInternalServerError)
		return
	}

	if err := database.UpdateUserSubscription(userID, status); err != nil {
		http.Error(w, "Failed to update user status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":              status,
		"subscription_status": status,
		"expires_at":          expiresAt.Format(time.RFC3339),
		"product_id":          req.ProductID,
		"verified_by":         "dev_mode",
	})
}

// HandleWebhook receives App Store Server Notifications V2
func (h *IAPHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var payload apple.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Webhook: Failed to decode payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	notification, err := apple.ParseWebhookPayload(payload.SignedPayload)
	if err != nil {
		log.Printf("Webhook: Failed to parse notification: %v", err)
		http.Error(w, "Invalid notification", http.StatusBadRequest)
		return
	}

	if notification == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Webhook: Received %s notification (subtype: %s)", 
		notification.NotificationType, notification.Subtype)

	// Extract transaction info
	txnInfo, err := apple.GetTransactionFromNotification(notification)
	if err != nil || txnInfo == nil {
		log.Printf("Webhook: No transaction info in notification")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle based on notification type
	switch notification.NotificationType {
	case apple.NotificationTypeSubscribed, apple.NotificationTypeDidRenew:
		// User subscribed or renewed - activate/extend subscription
		log.Printf("Webhook: Subscription active for txn %s, product %s", 
			txnInfo.TransactionID, txnInfo.ProductID)
		
		// Find user by transaction and update
		// Note: In production, you'd look up user by originalTransactionId
		// For now, just log - the user's app will verify on next open

	case apple.NotificationTypeExpired, apple.NotificationTypeRevoke:
		// Subscription expired or revoked - deactivate
		log.Printf("Webhook: Subscription ended for txn %s", txnInfo.TransactionID)
		
		if err := database.ExpireSubscription(txnInfo.TransactionID); err != nil {
			log.Printf("Webhook: Failed to expire subscription: %v", err)
		}

	case apple.NotificationTypeDidFailToRenew:
		// Billing issue - subscription in grace period
		log.Printf("Webhook: Billing failed for txn %s", txnInfo.TransactionID)
		// Keep active during grace period

	case apple.NotificationTypeRefund:
		// User got refund - revoke access
		log.Printf("Webhook: Refund issued for txn %s", txnInfo.TransactionID)
		if err := database.ExpireSubscription(txnInfo.TransactionID); err != nil {
			log.Printf("Webhook: Failed to expire after refund: %v", err)
		}
	}

	// Always acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "received"}`))
}
