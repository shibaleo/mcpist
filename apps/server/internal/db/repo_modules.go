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

// ModuleConfig is the response for GET /v1/me/modules/config.
type ModuleConfig struct {
	ModuleName  string   `json:"module_name"`
	Enabled     bool     `json:"enabled"`
	Description string   `json:"description"`
	Tools       []string `json:"enabled_tools"`
}

// GetModuleConfig returns all module configs for a user.
func GetModuleConfig(db *gorm.DB, userID string) ([]ModuleConfig, error) {
	var modules []Module
	if err := db.Where("status IN ('active', 'beta')").Find(&modules).Error; err != nil {
		return nil, err
	}

	var msRows []ModuleSetting
	db.Where("user_id = ?", userID).Find(&msRows)
	msMap := map[string]ModuleSetting{}
	for _, ms := range msRows {
		msMap[ms.ModuleID] = ms
	}

	var tsRows []ToolSetting
	db.Where("user_id = ? AND enabled = true", userID).Find(&tsRows)
	tsMap := map[string][]string{}
	for _, ts := range tsRows {
		tsMap[ts.ModuleID] = append(tsMap[ts.ModuleID], ts.ToolID)
	}

	configs := make([]ModuleConfig, 0, len(modules))
	for _, m := range modules {
		ms, hasSetting := msMap[m.ID]
		enabled := !hasSetting || ms.Enabled
		desc := ""
		if hasSetting {
			desc = ms.Description
		}

		configs = append(configs, ModuleConfig{
			ModuleName:  m.Name,
			Enabled:     enabled,
			Description: desc,
			Tools:       tsMap[m.ID],
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

	return db.Transaction(func(tx *gorm.DB) error {
		for _, toolID := range enabled {
			ts := ToolSetting{UserID: userID, ModuleID: mod.ID, ToolID: toolID, Enabled: true}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "module_id"}, {Name: "tool_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"enabled"}),
			}).Create(&ts).Error; err != nil {
				return err
			}
		}
		for _, toolID := range disabled {
			ts := ToolSetting{UserID: userID, ModuleID: mod.ID, ToolID: toolID, Enabled: false}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "module_id"}, {Name: "tool_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"enabled"}),
			}).Create(&ts).Error; err != nil {
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
