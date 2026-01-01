package apple

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AppStoreConfig holds credentials for App Store Server API
type AppStoreConfig struct {
	KeyID      string // From App Store Connect
	IssuerID   string // From App Store Connect
	BundleID   string // Your app's bundle ID
	PrivateKey *ecdsa.PrivateKey
	IsSandbox  bool
}

// TransactionInfo represents decoded transaction data from Apple
type TransactionInfo struct {
	TransactionID       string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	ProductID           string `json:"productId"`
	PurchaseDate        int64  `json:"purchaseDate"`
	ExpiresDate         int64  `json:"expiresDate,omitempty"`
	Type                string `json:"type"` // Auto-Renewable Subscription, Non-Consumable, etc.
	InAppOwnershipType  string `json:"inAppOwnershipType"`
	Environment         string `json:"environment"` // Sandbox or Production
}

// SubscriptionStatus from Get Subscription Status API
type SubscriptionStatus struct {
	BundleID           string `json:"bundleId"`
	Environment        string `json:"environment"`
	SubscriptionGroupID string `json:"subscriptionGroupIdentifier"`
	Items              []struct {
		LastTransactions []struct {
			Status              int    `json:"status"` // 1=active, 2=expired, 3=billing retry, 4=grace, 5=revoked
			SignedTransactionInfo string `json:"signedTransactionInfo"`
		} `json:"lastTransactions"`
	} `json:"data"`
}

var config *AppStoreConfig

// Initialize loads the Apple credentials
func Initialize(keyPath, keyID, issuerID, bundleID string, sandbox bool) error {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return errors.New("failed to parse PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return errors.New("key is not ECDSA")
	}

	config = &AppStoreConfig{
		KeyID:      keyID,
		IssuerID:   issuerID,
		BundleID:   bundleID,
		PrivateKey: ecdsaKey,
		IsSandbox:  sandbox,
	}

	return nil
}

// IsConfigured returns true if Apple API is configured
func IsConfigured() bool {
	return config != nil
}

// GenerateJWT creates a signed JWT for Apple API authentication
func GenerateJWT() (string, error) {
	if config == nil {
		return "", errors.New("apple API not configured")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": config.IssuerID,
		"iat": now.Unix(),
		"exp": now.Add(20 * time.Minute).Unix(), // Apple allows up to 20 min
		"aud": "appstoreconnect-v1",
		"bid": config.BundleID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = config.KeyID

	return token.SignedString(config.PrivateKey)
}

// GetTransactionInfo verifies and retrieves transaction info from Apple
func GetTransactionInfo(transactionID string) (*TransactionInfo, error) {
	if config == nil {
		return nil, errors.New("apple API not configured - using dev mode")
	}

	jwt, err := GenerateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	baseURL := "https://api.storekit.itunes.apple.com"
	if config.IsSandbox {
		baseURL = "https://api.storekit-sandbox.itunes.apple.com"
	}

	url := fmt.Sprintf("%s/inApps/v1/transactions/%s", baseURL, transactionID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Apple API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse the JWS (we trust Apple's signature for now - production should verify)
	return ParseSignedTransaction(result.SignedTransactionInfo)
}

// ParseSignedTransaction extracts transaction info from JWS
func ParseSignedTransaction(signedInfo string) (*TransactionInfo, error) {
	// JWS format: header.payload.signature
	parts := strings.Split(signedInfo, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWS format")
	}

	// Decode payload (base64url)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var info TransactionInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	return &info, nil
}

// GetSubscriptionStatus checks current subscription status
func GetSubscriptionStatus(originalTransactionID string) (*SubscriptionStatus, error) {
	if config == nil {
		return nil, errors.New("apple API not configured")
	}

	jwt, err := GenerateJWT()
	if err != nil {
		return nil, err
	}

	baseURL := "https://api.storekit.itunes.apple.com"
	if config.IsSandbox {
		baseURL = "https://api.storekit-sandbox.itunes.apple.com"
	}

	url := fmt.Sprintf("%s/inApps/v1/subscriptions/%s", baseURL, originalTransactionID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple API error %d", resp.StatusCode)
	}

	var status SubscriptionStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}
