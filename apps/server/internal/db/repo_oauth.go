package db

import (
	"fmt"

	"gorm.io/gorm"
)

// GetOAuthAppCredentials returns the OAuth app config for a provider.
// Used by TokenBroker for token refresh.
func GetOAuthAppCredentials(db *gorm.DB, provider string) (*OAuthApp, error) {
	var app OAuthApp
	if err := db.Where("provider = ? AND enabled = true", provider).First(&app).Error; err != nil {
		return nil, fmt.Errorf("oauth app not found for provider %s: %w", provider, err)
	}
	return &app, nil
}

// ListOAuthApps returns all OAuth apps.
func ListOAuthApps(db *gorm.DB) ([]OAuthApp, error) {
	var apps []OAuthApp
	if err := db.Order("provider").Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// UpsertOAuthApp creates or updates an OAuth app.
func UpsertOAuthApp(db *gorm.DB, app *OAuthApp) error {
	return db.Save(app).Error
}

// DeleteOAuthApp deletes an OAuth app by provider.
func DeleteOAuthApp(db *gorm.DB, provider string) error {
	result := db.Where("provider = ?", provider).Delete(&OAuthApp{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("oauth app not found: %s", provider)
	}
	return result.Error
}

// OAuthConsent represents a user's OAuth consent (credential connection).
type OAuthConsent struct {
	ID        string `json:"id"`
	Module    string `json:"module"`
	CreatedAt string `json:"created_at"`
}

// ListOAuthConsents returns a user's connected OAuth services.
func ListOAuthConsents(db *gorm.DB, userID string) ([]OAuthConsent, error) {
	var creds []UserCredential
	if err := db.Select("id", "module", "created_at").
		Where("user_id = ?", userID).
		Order("module").
		Find(&creds).Error; err != nil {
		return nil, err
	}

	result := make([]OAuthConsent, len(creds))
	for i, c := range creds {
		result[i] = OAuthConsent{
			ID:        c.ID,
			Module:    c.Module,
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return result, nil
}

// ListAllOAuthConsents returns all users' OAuth consents (admin).
func ListAllOAuthConsents(db *gorm.DB) ([]UserCredential, error) {
	var creds []UserCredential
	if err := db.Select("id", "user_id", "module", "created_at").
		Order("created_at DESC").
		Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

// RevokeOAuthConsent deletes a user's OAuth credential by ID.
func RevokeOAuthConsent(db *gorm.DB, userID, consentID string) error {
	result := db.Where("id = ? AND user_id = ?", consentID, userID).Delete(&UserCredential{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("consent not found")
	}
	return result.Error
}
