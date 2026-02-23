package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONB is a generic type for PostgreSQL JSONB columns.
type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB("{}")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = JSONB(v)
	case string:
		*j = JSONB(v)
	default:
		return fmt.Errorf("unsupported type for JSONB: %T", value)
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("{}"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = JSONB(data)
	return nil
}

// --- Models ---

type User struct {
	ID               string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ClerkID          *string `gorm:"type:text;uniqueIndex" json:"clerk_id,omitempty"`
	AccountStatus    string  `gorm:"type:text;not null;default:'active'" json:"account_status"`
	PlanID           string  `gorm:"type:text;not null;default:'free'" json:"plan_id"`
	DisplayName      *string `gorm:"type:text" json:"display_name,omitempty"`
	AvatarURL        *string `gorm:"type:text" json:"avatar_url,omitempty"`
	Email            *string `gorm:"type:text" json:"email,omitempty"`
	Role             string  `gorm:"type:text;not null;default:'user'" json:"role"`
	StripeCustomerID *string `gorm:"type:text;uniqueIndex" json:"stripe_customer_id,omitempty"`
	Settings         JSONB   `gorm:"type:jsonb;default:'{}'" json:"settings"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (User) TableName() string { return "mcpist.users" }

type Plan struct {
	ID            string `gorm:"primaryKey;type:text" json:"id"`
	Name          string `gorm:"type:text;not null" json:"name"`
	DailyLimit    int    `gorm:"not null" json:"daily_limit"`
	PriceMonthly  int    `gorm:"default:0" json:"price_monthly"`
	StripePriceID *string `gorm:"type:text" json:"stripe_price_id,omitempty"`
	Features      JSONB  `gorm:"type:jsonb;default:'{}'" json:"features"`
}

func (Plan) TableName() string { return "mcpist.plans" }

type Module struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:text;not null;uniqueIndex" json:"name"`
	Status    string    `gorm:"type:text;not null;default:'active'" json:"status"`
	Tools     JSONB     `gorm:"type:jsonb;default:'[]'" json:"tools"`
	CreatedAt time.Time `json:"created_at"`
}

func (Module) TableName() string { return "mcpist.modules" }

type ModuleSetting struct {
	UserID      string    `gorm:"primaryKey;type:uuid" json:"user_id"`
	ModuleID    string    `gorm:"primaryKey;type:uuid" json:"module_id"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	Description string    `gorm:"type:text;not null;default:''" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ModuleSetting) TableName() string { return "mcpist.module_settings" }

type ToolSetting struct {
	UserID    string    `gorm:"primaryKey;type:uuid" json:"user_id"`
	ModuleID  string    `gorm:"primaryKey;type:uuid" json:"module_id"`
	ToolID    string    `gorm:"primaryKey;type:text" json:"tool_id"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

func (ToolSetting) TableName() string { return "mcpist.tool_settings" }

type Prompt struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID      string    `gorm:"type:uuid;not null" json:"user_id"`
	ModuleID    *string   `gorm:"type:uuid" json:"module_id,omitempty"`
	Name        string    `gorm:"type:text;not null" json:"name"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Prompt) TableName() string { return "mcpist.prompts" }

type APIKey struct {
	ID         string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID     string     `gorm:"type:uuid;not null" json:"user_id"`
	JwtKID     string     `gorm:"column:jwt_kid;type:text" json:"jwt_kid"`
	KeyPrefix  string     `gorm:"type:text;not null" json:"key_prefix"`
	Name       string     `gorm:"type:text;not null" json:"name"`
	ExpiresAt  *time.Time `gorm:"type:timestamptz" json:"expires_at,omitempty"`
	LastUsedAt *time.Time `gorm:"type:timestamptz" json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (APIKey) TableName() string { return "mcpist.api_keys" }

type UserCredential struct {
	ID                   string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID               string    `gorm:"type:uuid;not null" json:"user_id"`
	Module               string    `gorm:"type:text;not null" json:"module"`
	Credentials          string    `gorm:"-" json:"-"`
	EncryptedCredentials *string   `gorm:"type:text" json:"encrypted_credentials,omitempty"`
	KeyVersion           int       `gorm:"not null;default:1" json:"key_version"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (UserCredential) TableName() string { return "mcpist.user_credentials" }

type OAuthApp struct {
	ID                    string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Provider              string    `gorm:"type:text;not null;uniqueIndex" json:"provider"`
	ClientID              string    `gorm:"type:text;not null" json:"client_id"`
	ClientSecret          string    `gorm:"-" json:"-"`
	EncryptedClientSecret *string   `gorm:"type:text" json:"encrypted_client_secret,omitempty"`
	RedirectURI           string    `gorm:"type:text;not null" json:"redirect_uri"`
	Enabled               bool      `gorm:"default:true" json:"enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (OAuthApp) TableName() string { return "mcpist.oauth_apps" }

type UsageLog struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"type:uuid;not null" json:"user_id"`
	MetaTool  string    `gorm:"type:text;not null" json:"meta_tool"`
	RequestID *string   `gorm:"type:text" json:"request_id,omitempty"`
	Details   JSONB     `gorm:"type:jsonb;not null" json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

func (UsageLog) TableName() string { return "mcpist.usage_log" }

type ProcessedWebhookEvent struct {
	EventID     string    `gorm:"primaryKey;type:text" json:"event_id"`
	UserID      string    `gorm:"type:uuid;not null" json:"user_id"`
	ProcessedAt time.Time `gorm:"not null;default:now()" json:"processed_at"`
}

func (ProcessedWebhookEvent) TableName() string { return "mcpist.processed_webhook_events" }
