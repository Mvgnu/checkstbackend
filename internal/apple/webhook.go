package apple

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// NotificationType represents App Store Server Notification types
type NotificationType string

const (
	NotificationTypeSubscribed        NotificationType = "SUBSCRIBED"
	NotificationTypeDidRenew          NotificationType = "DID_RENEW"
	NotificationTypeExpired           NotificationType = "EXPIRED"
	NotificationTypeDidFailToRenew    NotificationType = "DID_FAIL_TO_RENEW"
	NotificationTypeGracePeriodExpired NotificationType = "GRACE_PERIOD_EXPIRED"
	NotificationTypeRefund            NotificationType = "REFUND"
	NotificationTypeRevoke            NotificationType = "REVOKE"
)

// Subtype for notifications
type NotificationSubtype string

const (
	SubtypeInitialBuy       NotificationSubtype = "INITIAL_BUY"
	SubtypeResubscribe      NotificationSubtype = "RESUBSCRIBE"
	SubtypeAutoRenewEnabled NotificationSubtype = "AUTO_RENEW_ENABLED"
	SubtypeVoluntary        NotificationSubtype = "VOLUNTARY"
	SubtypeBillingRecovery  NotificationSubtype = "BILLING_RECOVERY"
)

// WebhookPayload is the signed notification from Apple
type WebhookPayload struct {
	SignedPayload string `json:"signedPayload"`
}

// DecodedNotification is the parsed notification
type DecodedNotification struct {
	NotificationType NotificationType    `json:"notificationType"`
	Subtype          NotificationSubtype `json:"subtype,omitempty"`
	Data             NotificationData    `json:"data"`
	NotificationUUID string              `json:"notificationUUID"`
	SignedDate       int64               `json:"signedDate"`
}

type NotificationData struct {
	AppAppleID            int64  `json:"appAppleId"`
	BundleID              string `json:"bundleId"`
	Environment           string `json:"environment"`
	SignedTransactionInfo string `json:"signedTransactionInfo"`
	SignedRenewalInfo     string `json:"signedRenewalInfo,omitempty"`
}

// ParseWebhookPayload decodes the signed notification from Apple
func ParseWebhookPayload(signedPayload string) (*DecodedNotification, error) {
	// JWS format: header.payload.signature
	parts := strings.Split(signedPayload, ".")
	if len(parts) != 3 {
		return nil, nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var notification DecodedNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return nil, err
	}

	return &notification, nil
}

// GetTransactionFromNotification extracts transaction info from notification
func GetTransactionFromNotification(n *DecodedNotification) (*TransactionInfo, error) {
	if n.Data.SignedTransactionInfo == "" {
		return nil, nil
	}
	return ParseSignedTransaction(n.Data.SignedTransactionInfo)
}
