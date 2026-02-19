package db

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// RecordUsage inserts a usage log entry. Fire-and-forget style.
func RecordUsage(db *gorm.DB, userID, metaTool, requestID string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	entry := UsageLog{
		UserID:   userID,
		MetaTool: metaTool,
		Details:  JSONB(detailsJSON),
	}
	if requestID != "" {
		entry.RequestID = &requestID
	}

	return db.Create(&entry).Error
}

// UsageSummary is the response for GET /v1/me/usage.
type UsageSummary struct {
	DailyUsed  int `json:"daily_used"`
	DailyLimit int `json:"daily_limit"`
}

// GetUsage returns the user's current daily usage and limit.
func GetUsage(db *gorm.DB, userID string) (*UsageSummary, error) {
	var plan Plan
	if err := db.Table("mcpist.plans AS p").
		Select("p.daily_limit").
		Joins("JOIN mcpist.users u ON u.plan_id = p.id").
		Where("u.id = ?", userID).
		First(&plan).Error; err != nil {
		return nil, err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	var dailyUsed int64
	db.Model(&UsageLog{}).Where("user_id = ? AND created_at >= ?", userID, today).Count(&dailyUsed)

	return &UsageSummary{
		DailyUsed:  int(dailyUsed),
		DailyLimit: plan.DailyLimit,
	}, nil
}
