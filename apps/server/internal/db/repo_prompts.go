package db

import (
	"fmt"

	"gorm.io/gorm"
)

// ListPrompts returns all prompts for a user, optionally filtered by module.
func ListPrompts(db *gorm.DB, userID string, moduleName *string) ([]Prompt, error) {
	q := db.Where("user_id = ?", userID)
	if moduleName != nil {
		q = q.Joins("JOIN mcpist.modules m ON m.id = prompts.module_id").
			Where("m.name = ?", *moduleName)
	}
	var prompts []Prompt
	if err := q.Order("created_at DESC").Find(&prompts).Error; err != nil {
		return nil, err
	}
	return prompts, nil
}

// GetPrompt returns a single prompt by ID, scoped to user.
func GetPrompt(db *gorm.DB, userID, promptID string) (*Prompt, error) {
	var prompt Prompt
	if err := db.Where("id = ? AND user_id = ?", promptID, userID).First(&prompt).Error; err != nil {
		return nil, fmt.Errorf("prompt not found: %w", err)
	}
	return &prompt, nil
}

// CreatePrompt creates a new prompt.
func CreatePrompt(db *gorm.DB, p *Prompt) error {
	return db.Create(p).Error
}

// UpdatePrompt updates an existing prompt.
func UpdatePrompt(db *gorm.DB, userID, promptID string, updates map[string]interface{}) error {
	result := db.Model(&Prompt{}).Where("id = ? AND user_id = ?", promptID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return fmt.Errorf("prompt not found")
	}
	return result.Error
}

// DeletePrompt deletes a prompt.
func DeletePrompt(db *gorm.DB, userID, promptID string) error {
	result := db.Where("id = ? AND user_id = ?", promptID, userID).Delete(&Prompt{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("prompt not found")
	}
	return result.Error
}

// GetEnabledPrompts returns all enabled prompts for a user (used by MCP handler).
func GetEnabledPrompts(db *gorm.DB, userID string) ([]Prompt, error) {
	var prompts []Prompt
	if err := db.Where("user_id = ? AND enabled = true", userID).
		Order("name").
		Find(&prompts).Error; err != nil {
		return nil, err
	}
	return prompts, nil
}

// GetEnabledPromptByName returns a single enabled prompt by name.
func GetEnabledPromptByName(db *gorm.DB, userID, name string) (*Prompt, error) {
	var prompt Prompt
	if err := db.Where("user_id = ? AND name = ? AND enabled = true", userID, name).
		First(&prompt).Error; err != nil {
		return nil, nil // not found is not an error for MCP
	}
	return &prompt, nil
}
