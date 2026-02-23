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

// UsageData is the response for GET /v1/me/usage (matches OpenAPI spec).
type UsageData struct {
	TotalUsed int            `json:"total_used"`
	ByModule  map[string]int `json:"by_module"`
	Period    UsagePeriod    `json:"period"`
}

// UsagePeriod represents the date range for usage data.
type UsagePeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// moduleCount is a helper struct for the GROUP BY query.
type moduleCount struct {
	Module string
	Count  int
}

// GetUsageByDateRange returns usage counts grouped by module for a date range.
func GetUsageByDateRange(database *gorm.DB, userID string, start, end time.Time) (*UsageData, error) {
	// Total count in date range
	var totalUsed int64
	if err := database.Model(&UsageLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end).
		Count(&totalUsed).Error; err != nil {
		return nil, err
	}

	// Count by module: extract module from details JSONB array
	// details is an array like [{"module":"notion","tool":"search_pages"}]
	var counts []moduleCount
	database.Raw(`
		SELECT elem->>'module' AS module, COUNT(*) AS count
		FROM mcpist.usage_log,
		     jsonb_array_elements(details) AS elem
		WHERE user_id = ? AND created_at >= ? AND created_at < ?
		  AND elem->>'module' IS NOT NULL
		GROUP BY elem->>'module'
		ORDER BY count DESC
	`, userID, start, end).Scan(&counts)

	byModule := make(map[string]int, len(counts))
	for _, c := range counts {
		byModule[c.Module] = c.Count
	}

	return &UsageData{
		TotalUsed: int(totalUsed),
		ByModule:  byModule,
		Period: UsagePeriod{
			Start: start.Format("2006-01-02"),
			End:   end.Format("2006-01-02"),
		},
	}, nil
}
