package ogenserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mcpist/server/internal/db"

	"gorm.io/gorm"
)

const stripeTimestampTolerance = 300 // 5 minutes

// NewStripeWebhookHandler returns an http.HandlerFunc for POST /v1/stripe/webhook.
// This is kept outside ogen scope because Stripe webhooks require raw body
// reading and custom signature verification.
func NewStripeWebhookHandler(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if secret == "" {
			writeErrorJSON(w, http.StatusInternalServerError, "webhook secret not configured")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "failed to read body")
			return
		}
		defer r.Body.Close()

		sigHeader := r.Header.Get("Stripe-Signature")
		if err := verifyStripeSignature(body, sigHeader, secret); err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, err.Error())
			return
		}

		var event struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				Object json.RawMessage `json:"object"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid event JSON")
			return
		}

		log.Printf("[stripe] Received event: %s (%s)", event.Type, event.ID)

		switch event.Type {
		case "invoice.paid":
			handleInvoicePaid(database, event.ID, event.Data.Object)
		case "customer.subscription.deleted":
			handleSubscriptionDeleted(database, event.ID, event.Data.Object)
		default:
			log.Printf("[stripe] Ignoring event type: %s", event.Type)
		}

		writeSuccessJSON(w, http.StatusOK)
	}
}

func handleInvoicePaid(database *gorm.DB, eventID string, data json.RawMessage) {
	var invoice struct {
		Customer string `json:"customer"`
		Metadata struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &invoice); err != nil {
		log.Printf("[stripe] Failed to parse invoice: %v", err)
		return
	}

	userID := invoice.Metadata.UserID
	if userID == "" && invoice.Customer != "" {
		user, err := db.GetUserByStripeCustomer(database, invoice.Customer)
		if err != nil {
			log.Printf("[stripe] Failed to find user for customer %s: %v", invoice.Customer, err)
			return
		}
		userID = user.ID
	}

	if userID == "" {
		log.Printf("[stripe] No user_id found for invoice event %s", eventID)
		return
	}

	if err := db.ActivateSubscription(database, userID, "plus", eventID); err != nil {
		log.Printf("[stripe] Failed to activate subscription for %s: %v", userID, err)
		return
	}
	log.Printf("[stripe] Subscription activated for user %s", userID)
}

func handleSubscriptionDeleted(database *gorm.DB, eventID string, data json.RawMessage) {
	var subscription struct {
		Metadata struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &subscription); err != nil {
		log.Printf("[stripe] Failed to parse subscription: %v", err)
		return
	}

	userID := subscription.Metadata.UserID
	if userID == "" {
		log.Printf("[stripe] No user_id in subscription.deleted metadata for event %s", eventID)
		return
	}

	if err := db.ActivateSubscription(database, userID, "free", eventID); err != nil {
		log.Printf("[stripe] Failed to downgrade user %s: %v", userID, err)
		return
	}
	log.Printf("[stripe] Subscription downgraded to free for user %s", userID)
}

func verifyStripeSignature(payload []byte, header, secret string) error {
	if header == "" {
		return fmt.Errorf("missing signature")
	}

	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return fmt.Errorf("invalid signature format")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	if math.Abs(float64(time.Now().Unix()-ts)) > stripeTimestampTolerance {
		return fmt.Errorf("timestamp outside tolerance")
	}

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}

	return fmt.Errorf("signature mismatch")
}

// writeErrorJSON writes a JSON error response (used outside ogen).
func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeSuccessJSON writes {"received": true} response.
func writeSuccessJSON(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]bool{"received": true})
}
