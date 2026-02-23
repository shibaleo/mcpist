package db

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListModules returns all active/beta modules with their tools.
func ListModules(db *gorm.DB) ([]Module, error) {
	var modules []Module
	if err := db.Where("status IN ('active', 'beta')").
		Order("name").
		Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}

// SyncModuleEntry is the input for SyncModules.
type SyncModuleEntry struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Tools  interface{} `json:"tools"`
}

// SyncModules upserts module+tool data at server startup.
func SyncModules(db *gorm.DB, entries []SyncModuleEntry) (int, error) {
	upserted := 0
	for _, e := range entries {
		toolsJSON, err := json.Marshal(e.Tools)
		if err != nil {
			return upserted, fmt.Errorf("failed to marshal tools for %s: %w", e.Name, err)
		}

		mod := Module{
			Name:   e.Name,
			Status: e.Status,
			Tools:  JSONB(toolsJSON),
		}

		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "tools"}),
		}).Create(&mod)

		if result.Error != nil {
			return upserted, fmt.Errorf("failed to sync module %s: %w", e.Name, result.Error)
		}
		upserted++
	}
	return upserted, nil
}

// ModuleConfig is a flat per-tool row for GET /v1/me/modules/config.
type ModuleConfig struct {
	ModuleName  string `json:"module_name"`
	Description *string `json:"description"`
	ToolID      string `json:"tool_id"`
	Enabled     bool   `json:"enabled"`
}

// GetModuleConfig returns per-tool settings for a user (one row per tool_setting).
func GetModuleConfig(db *gorm.DB, userID string) ([]ModuleConfig, error) {
	// Build module ID → name map and ID → description map
	var modules []Module
	if err := db.Where("status IN ('active', 'beta')").Find(&modules).Error; err != nil {
		return nil, err
	}
	moduleNames := map[string]string{}
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}

	// Module descriptions from module_settings
	var msRows []ModuleSetting
	db.Where("user_id = ?", userID).Find(&msRows)
	msDescMap := map[string]string{}
	for _, ms := range msRows {
		msDescMap[ms.ModuleID] = ms.Description
	}

	// All tool_settings for this user (both enabled and disabled)
	var tsRows []ToolSetting
	db.Where("user_id = ?", userID).Find(&tsRows)

	configs := make([]ModuleConfig, 0, len(tsRows))
	for _, ts := range tsRows {
		name, ok := moduleNames[ts.ModuleID]
		if !ok {
			continue // skip orphaned settings for inactive modules
		}
		var desc *string
		if d, has := msDescMap[ts.ModuleID]; has {
			desc = &d
		}
		configs = append(configs, ModuleConfig{
			ModuleName:  name,
			Description: desc,
			ToolID:      ts.ToolID,
			Enabled:     ts.Enabled,
		})
	}
	return configs, nil
}

// UpsertToolSettings updates tool enable/disable settings for a module.
func UpsertToolSettings(db *gorm.DB, userID, moduleName string, enabled, disabled []string) error {
	var mod Module
	if err := db.Where("name = ?", moduleName).First(&mod).Error; err != nil {
		return fmt.Errorf("module not found: %s", moduleName)
	}

	// Use raw SQL to avoid GORM's zero-value problem:
	// GORM treats bool false as "unset" and omits it from INSERT,
	// causing DB DEFAULT (true) to always be applied.
	const upsertSQL = `INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, module_id, tool_id)
		DO UPDATE SET enabled = EXCLUDED.enabled`

	return db.Transaction(func(tx *gorm.DB) error {
		for _, toolID := range enabled {
			if err := tx.Exec(upsertSQL, userID, mod.ID, toolID, true).Error; err != nil {
				return err
			}
		}
		for _, toolID := range disabled {
			if err := tx.Exec(upsertSQL, userID, mod.ID, toolID, false).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpsertModuleDescription sets the user's custom description for a module.
func UpsertModuleDescription(db *gorm.DB, userID, moduleName, description string) error {
	var mod Module
	if err := db.Where("name = ?", moduleName).First(&mod).Error; err != nil {
		return fmt.Errorf("module not found: %s", moduleName)
	}

	ms := ModuleSetting{UserID: userID, ModuleID: mod.ID, Description: description}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "module_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"description"}),
	}).Create(&ms).Error
}
