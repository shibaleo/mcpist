package db

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CredentialMeta is the metadata-only view returned for listing.
type CredentialMeta struct {
	Module    string `json:"module"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListCredentials returns credential metadata (no secrets) for a user.
func ListCredentials(db *gorm.DB, userID string) ([]CredentialMeta, error) {
	var creds []UserCredential
	if err := db.Select("module", "created_at", "updated_at").
		Where("user_id = ?", userID).
		Order("module").
		Find(&creds).Error; err != nil {
		return nil, err
	}

	result := make([]CredentialMeta, len(creds))
	for i, c := range creds {
		result[i] = CredentialMeta{
			Module:    c.Module,
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return result, nil
}

// GetCredential returns the full credential for a user/module.
func GetCredential(db *gorm.DB, userID, module string) (*UserCredential, error) {
	var cred UserCredential
	if err := db.Where("user_id = ? AND module = ?", userID, module).First(&cred).Error; err != nil {
		return nil, fmt.Errorf("credential not found for module %s: %w", module, err)
	}
	return &cred, nil
}

// UpsertCredential creates or updates a credential.
// Also auto-enables all tools for the module if no tool_settings exist yet.
func UpsertCredential(db *gorm.DB, userID, module, credentials string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		cred := UserCredential{
			UserID:      userID,
			Module:      module,
			Credentials: credentials,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "module"}},
			DoUpdates: clause.AssignmentColumns([]string{"credentials", "updated_at"}),
		}).Create(&cred).Error; err != nil {
			return err
		}

		// Auto-enable all tools for this module if user has no tool_settings yet
		var mod Module
		if err := tx.Where("name = ?", module).First(&mod).Error; err != nil {
			return nil // Module not in DB — skip
		}

		var existingCount int64
		tx.Model(&ToolSetting{}).Where("user_id = ? AND module_id = ?", userID, mod.ID).Count(&existingCount)
		if existingCount > 0 {
			return nil // Already has settings
		}

		type toolAnnotations struct {
			DestructiveHint *bool `json:"destructiveHint,omitempty"`
		}
		type toolDef struct {
			ID          string          `json:"id"`
			Annotations toolAnnotations `json:"annotations"`
		}
		var tools []toolDef
		if err := json.Unmarshal(mod.Tools, &tools); err != nil {
			return nil // Can't parse — skip
		}

		// Use raw SQL to avoid GORM's zero-value problem with bool fields.
		const upsertSQL = `INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (user_id, module_id, tool_id)
			DO UPDATE SET enabled = EXCLUDED.enabled`
		for _, t := range tools {
			// Skip destructive tools — user must explicitly enable them
			if t.Annotations.DestructiveHint != nil && *t.Annotations.DestructiveHint {
				continue
			}
			if err := tx.Exec(upsertSQL, userID, mod.ID, t.ID, true).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// DeleteCredential removes a credential.
func DeleteCredential(db *gorm.DB, userID, module string) error {
	result := db.Where("user_id = ? AND module = ?", userID, module).Delete(&UserCredential{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential not found for module %s", module)
	}
	return result.Error
}
