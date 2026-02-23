package ogenserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-faster/jx"

	"mcpist/server/internal/auth"
	"mcpist/server/internal/db"
	gen "mcpist/server/internal/ogenserver/gen"

	"gorm.io/gorm"
)

// handler implements gen.Handler.
type handler struct {
	gen.UnimplementedHandler
	db *gorm.DB
}

var _ gen.Handler = (*handler)(nil)

// NewHandler creates the ogen Handler implementation.
func NewHandler(database *gorm.DB) gen.Handler {
	return &handler{db: database}
}

// ── Modules ──────────────────────────────────────────────────

func (h *handler) ListModules(ctx context.Context) ([]gen.ModuleWithTools, error) {
	modules, err := db.ListModules(h.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list modules")
	}
	out := make([]gen.ModuleWithTools, len(modules))
	for i, m := range modules {
		out[i] = gen.ModuleWithTools{
			ID:     m.ID,
			Name:   m.Name,
			Status: m.Status,
		}
		// Tools is []jx.Raw; unmarshal via json.RawMessage (jx.Raw lacks json.Unmarshaler)
		var raw []json.RawMessage
		if err := json.Unmarshal(m.Tools, &raw); err == nil && len(raw) > 0 {
			items := make([]jx.Raw, len(raw))
			for j, r := range raw {
				items[j] = jx.Raw(r)
			}
			out[i].Tools = items
		} else {
			out[i].Tools = []jx.Raw{}
		}
	}
	return out, nil
}

// ── Plans ────────────────────────────────────────────────────

func (h *handler) ListPlans(ctx context.Context) ([]gen.PlanInfo, error) {
	var plans []db.Plan
	if err := h.db.Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("failed to list plans")
	}
	out := make([]gen.PlanInfo, len(plans))
	for i, p := range plans {
		out[i] = gen.PlanInfo{
			ID:           p.ID,
			Name:         p.Name,
			DailyLimit:   p.DailyLimit,
			PriceMonthly: p.PriceMonthly,
		}
		if p.StripePriceID != nil {
			out[i].StripePriceID = gen.NewOptNilString(*p.StripePriceID)
		}
		if len(p.Features) > 0 {
			out[i].Features.SetTo(gen.PlanInfoFeatures{})
		}
	}
	return out, nil
}

// ── Registration ─────────────────────────────────────────────

func (h *handler) RegisterUser(ctx context.Context) (gen.RegisterUserRes, error) {
	// SecurityHandler stored clerk_id as userID and email in context
	clerkID := getUserID(ctx)
	email := getEmail(ctx)
	if clerkID == "" {
		return &gen.ErrorResponse{Error: "missing clerk_id in gateway token"}, nil
	}
	if email == "" {
		return &gen.ErrorResponse{Error: "missing email in gateway token"}, nil
	}

	userID, err := db.FindOrCreateByClerkID(h.db, clerkID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to register user")
	}
	return &gen.RegisterResult{ID: userID}, nil
}

// ── Profile ──────────────────────────────────────────────────

func (h *handler) GetMyProfile(ctx context.Context) (*gen.UserProfile, error) {
	userID := getUserID(ctx)
	profile, err := db.GetMyProfile(h.db, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	email := ""
	if profile.Email != nil {
		email = *profile.Email
	}

	// Compute daily usage
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	usage, _ := db.GetUsageByDateRange(h.db, userID, startOfDay, endOfDay)
	dailyUsed := 0
	if usage != nil {
		dailyUsed = usage.TotalUsed
	}
	// Get daily limit from plan
	dailyLimit := 0
	var plan db.Plan
	if h.db.Select("daily_limit").Where("id = ?", profile.PlanID).First(&plan).Error == nil {
		dailyLimit = plan.DailyLimit
	}

	out := &gen.UserProfile{
		UserID:         profile.ID,
		Email:          email,
		AccountStatus:  profile.AccountStatus,
		PlanID:         profile.PlanID,
		DailyUsed:      dailyUsed,
		DailyLimit:     dailyLimit,
		Role:           profile.Role,
		Settings:       jx.Raw(profile.Settings),
		ConnectedCount: profile.ConnectedCount,
	}
	if profile.DisplayName != nil {
		out.DisplayName = gen.NewOptNilString(*profile.DisplayName)
	}
	return out, nil
}

func (h *handler) UpdateSettings(ctx context.Context, req *gen.UpdateSettingsBody) (*gen.SuccessResult, error) {
	if err := db.UpdateSettings(h.db, getUserID(ctx), json.RawMessage(req.Settings)); err != nil {
		return nil, fmt.Errorf("failed to update settings")
	}
	return &gen.SuccessResult{Success: true}, nil
}

func (h *handler) CompleteUserOnboarding(ctx context.Context, req *gen.CompleteOnboardingBody) (*gen.OnboardingResult, error) {
	if req.EventID == "" {
		return &gen.OnboardingResult{Success: false, Error: gen.NewOptString("event_id is required")}, nil
	}
	if err := db.CompleteOnboarding(h.db, getUserID(ctx), req.EventID); err != nil {
		return &gen.OnboardingResult{Success: false, Error: gen.NewOptString("failed to complete onboarding")}, nil
	}
	return &gen.OnboardingResult{Success: true}, nil
}

// ── Usage ────────────────────────────────────────────────────

func (h *handler) GetUsage(ctx context.Context, params gen.GetUsageParams) (*gen.UsageData, error) {
	start := params.Start
	end := params.End.AddDate(0, 0, 1) // end date is inclusive
	usage, err := db.GetUsageByDateRange(h.db, getUserID(ctx), start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage")
	}
	return &gen.UsageData{
		TotalUsed: usage.TotalUsed,
		ByModule:  gen.UsageDataByModule(usage.ByModule),
		Period: gen.UsagePeriod{
			Start: usage.Period.Start,
			End:   usage.Period.End,
		},
	}, nil
}

// ── Stripe ───────────────────────────────────────────────────

func (h *handler) GetStripeCustomerId(ctx context.Context) (*gen.StripeCustomer, error) {
	customerID, err := db.GetStripeCustomerID(h.db, getUserID(ctx))
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	out := &gen.StripeCustomer{}
	if customerID != nil {
		out.StripeCustomerID = gen.NewOptNilString(*customerID)
	} else {
		out.StripeCustomerID.SetToNull()
	}
	return out, nil
}

func (h *handler) LinkStripeCustomer(ctx context.Context, req *gen.LinkStripeCustomerBody) (*gen.SuccessResult, error) {
	if req.StripeCustomerID == "" {
		return nil, fmt.Errorf("stripe_customer_id is required")
	}
	if err := db.LinkStripeCustomer(h.db, getUserID(ctx), req.StripeCustomerID); err != nil {
		return nil, fmt.Errorf("failed to link stripe customer")
	}
	return &gen.SuccessResult{Success: true}, nil
}

// ── Credentials ──────────────────────────────────────────────

func (h *handler) ListCredentials(ctx context.Context) ([]gen.Credential, error) {
	creds, err := db.ListCredentials(h.db, getUserID(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials")
	}
	out := make([]gen.Credential, len(creds))
	for i, c := range creds {
		createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, c.UpdatedAt)
		out[i] = gen.Credential{
			Module:    c.Module,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	}
	return out, nil
}

func (h *handler) UpsertCredential(ctx context.Context, req *gen.UpsertCredentialBody, params gen.UpsertCredentialParams) (*gen.UpsertCredentialResult, error) {
	if err := db.UpsertCredential(h.db, getUserID(ctx), params.Module, string(req.Credentials)); err != nil {
		return nil, fmt.Errorf("failed to upsert credential")
	}
	return &gen.UpsertCredentialResult{Success: true, Module: params.Module}, nil
}

func (h *handler) DeleteCredential(ctx context.Context, params gen.DeleteCredentialParams) (*gen.SuccessResult, error) {
	if err := db.DeleteCredential(h.db, getUserID(ctx), params.Module); err != nil {
		return nil, fmt.Errorf("credential not found")
	}
	return &gen.SuccessResult{Success: true}, nil
}

// ── API Keys ─────────────────────────────────────────────────

func (h *handler) ListApiKeys(ctx context.Context) ([]gen.ApiKey, error) {
	keys, err := db.ListAPIKeys(h.db, getUserID(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys")
	}
	out := make([]gen.ApiKey, len(keys))
	for i, k := range keys {
		out[i] = gen.ApiKey{
			ID:          k.ID,
			KeyPrefix:   k.KeyPrefix,
			DisplayName: k.Name,
		}
		if k.ExpiresAt != nil {
			out[i].ExpiresAt = gen.NewOptNilDateTime(*k.ExpiresAt)
		} else {
			out[i].ExpiresAt.SetToNull()
		}
		if k.LastUsedAt != nil {
			out[i].LastUsedAt = gen.NewOptNilDateTime(*k.LastUsedAt)
		} else {
			out[i].LastUsedAt.SetToNull()
		}
		// DB does not track revoked_at (keys are deleted on revoke)
		out[i].RevokedAt.SetToNull()
	}
	return out, nil
}

func (h *handler) GenerateApiKey(ctx context.Context, req *gen.GenerateApiKeyBody) (*gen.GenerateApiKeyResult, error) {
	name := req.DisplayName
	if name == "" {
		return nil, fmt.Errorf("display_name is required")
	}

	userID := getUserID(ctx)

	// Compute expiration:
	//   no_expiry=true  → nil (no expiration)
	//   expires_at set  → use that value
	//   neither         → default 90 days
	const defaultExpiryDays = 90
	var expiresAt *time.Time
	if noExp, ok := req.NoExpiry.Get(); ok && noExp {
		// explicit no expiry
	} else if ea, ok := req.ExpiresAt.Get(); ok {
		expiresAt = &ea
	} else {
		t := time.Now().AddDate(0, 0, defaultExpiryDays)
		expiresAt = &t
	}

	key, err := db.CreateAPIKey(h.db, userID, "", "mpt_", name, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key record")
	}

	token, err := auth.GenerateAPIKeyJWT(userID, key.ID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key")
	}

	prefix := token[:12] + "..."
	h.db.Model(&db.APIKey{}).Where("id = ?", key.ID).Updates(map[string]interface{}{
		"jwt_kid":    auth.GetKeyPair().KID,
		"key_prefix": prefix,
	})

	return &gen.GenerateApiKeyResult{
		APIKey:    token,
		KeyPrefix: prefix,
	}, nil
}

func (h *handler) GetApiKeyStatus(ctx context.Context, params gen.GetApiKeyStatusParams) (gen.GetApiKeyStatusRes, error) {
	key, err := db.GetAPIKeyByID(h.db, params.ID)
	if err != nil {
		return &gen.ErrorResponse{Error: "api key not found"}, nil
	}

	// Check expiration
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return &gen.ErrorResponse{Error: "api key expired"}, nil
	}

	result := &gen.ApiKeyStatus{
		Active: true,
		KeyID:  key.ID,
		UserID: key.UserID,
	}
	if key.ExpiresAt != nil {
		result.ExpiresAt = gen.NewOptNilDateTime(*key.ExpiresAt)
	} else {
		result.ExpiresAt.SetToNull()
	}

	// Update last_used_at
	go db.UpdateAPIKeyLastUsed(h.db, key.ID)

	return result, nil
}

func (h *handler) RevokeApiKey(ctx context.Context, params gen.RevokeApiKeyParams) (*gen.SuccessResult, error) {
	if err := db.RevokeAPIKey(h.db, getUserID(ctx), params.ID); err != nil {
		return nil, fmt.Errorf("API key not found")
	}
	return &gen.SuccessResult{Success: true}, nil
}

// ── Prompts ──────────────────────────────────────────────────

func (h *handler) ListPrompts(ctx context.Context, params gen.ListPromptsParams) ([]gen.Prompt, error) {
	var modulePtr *string
	if m, ok := params.Module.Get(); ok {
		modulePtr = &m
	}
	prompts, err := db.ListPrompts(h.db, getUserID(ctx), modulePtr)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts")
	}
	moduleNameMap := h.buildModuleNameMap()
	out := make([]gen.Prompt, len(prompts))
	for i, p := range prompts {
		out[i] = dbPromptToGen(p, moduleNameMap)
	}
	return out, nil
}

func (h *handler) GetPrompt(ctx context.Context, params gen.GetPromptParams) (*gen.GetPromptResult, error) {
	p, err := db.GetPrompt(h.db, getUserID(ctx), params.ID)
	if err != nil {
		return &gen.GetPromptResult{Found: false, Error: gen.NewOptString("prompt not found")}, nil
	}
	moduleName := h.resolveModuleName(p.ModuleID)
	return &gen.GetPromptResult{
		Found:       true,
		ID:          gen.NewOptString(p.ID),
		Name:        gen.NewOptString(p.Name),
		Content:     gen.NewOptString(p.Content),
		Enabled:     gen.NewOptBool(p.Enabled),
		CreatedAt:   gen.NewOptDateTime(p.CreatedAt),
		UpdatedAt:   gen.NewOptDateTime(p.UpdatedAt),
		ModuleName:  optNilStringFromPtr(moduleName),
		Description: optNilStringFromPtr(p.Description),
	}, nil
}

func (h *handler) CreatePrompt(ctx context.Context, req *gen.CreatePromptBody) (*gen.UpsertPromptResult, error) {
	p := db.Prompt{
		UserID:  getUserID(ctx),
		Name:    req.Name,
		Content: req.Content,
		Enabled: req.Enabled,
	}
	if desc, ok := req.Description.Get(); ok {
		p.Description = &desc
	}
	if mn, ok := req.ModuleName.Get(); ok {
		moduleID := h.resolveModuleID(mn)
		p.ModuleID = moduleID
	}
	if err := db.CreatePrompt(h.db, &p); err != nil {
		return &gen.UpsertPromptResult{Success: false, Error: gen.NewOptString("failed to create prompt")}, nil
	}
	return &gen.UpsertPromptResult{
		Success: true,
		ID:      gen.NewOptString(p.ID),
		Action:  gen.NewOptString("created"),
	}, nil
}

func (h *handler) UpdatePrompt(ctx context.Context, req *gen.UpdatePromptBody, params gen.UpdatePromptParams) (*gen.UpsertPromptResult, error) {
	updates := map[string]interface{}{
		"name":    req.Name,
		"content": req.Content,
		"enabled": req.Enabled,
	}
	if v, ok := req.Description.Get(); ok {
		updates["description"] = v
	}
	if v, ok := req.ModuleName.Get(); ok {
		moduleID := h.resolveModuleID(v)
		if moduleID != nil {
			updates["module_id"] = *moduleID
		}
	}

	if err := db.UpdatePrompt(h.db, getUserID(ctx), params.ID, updates); err != nil {
		return &gen.UpsertPromptResult{Success: false, Error: gen.NewOptString("prompt not found")}, nil
	}
	return &gen.UpsertPromptResult{
		Success: true,
		ID:      gen.NewOptString(params.ID),
		Action:  gen.NewOptString("updated"),
	}, nil
}

func (h *handler) DeletePrompt(ctx context.Context, params gen.DeletePromptParams) (*gen.DeletePromptResult, error) {
	if err := db.DeletePrompt(h.db, getUserID(ctx), params.ID); err != nil {
		return &gen.DeletePromptResult{Success: false, Error: gen.NewOptString("prompt not found")}, nil
	}
	return &gen.DeletePromptResult{Success: true}, nil
}

// ── Module Config ────────────────────────────────────────────

func (h *handler) GetModuleConfig(ctx context.Context) ([]gen.ModuleConfig, error) {
	configs, err := db.GetModuleConfig(h.db, getUserID(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get module config")
	}
	out := make([]gen.ModuleConfig, len(configs))
	for i, c := range configs {
		out[i] = gen.ModuleConfig{
			ModuleName: c.ModuleName,
			ToolID:     c.ToolID,
			Enabled:    c.Enabled,
		}
		if c.Description != nil {
			out[i].Description = gen.NewOptNilString(*c.Description)
		}
	}
	return out, nil
}

func (h *handler) UpsertToolSettings(ctx context.Context, req *gen.UpsertToolSettingsBody, params gen.UpsertToolSettingsParams) (*gen.UpsertToolSettingsResult, error) {
	if err := db.UpsertToolSettings(h.db, getUserID(ctx), params.Name, req.EnabledTools, req.DisabledTools); err != nil {
		return nil, fmt.Errorf("module not found")
	}
	return &gen.UpsertToolSettingsResult{
		Success:       true,
		EnabledCount:  gen.NewOptInt(len(req.EnabledTools)),
		DisabledCount: gen.NewOptInt(len(req.DisabledTools)),
	}, nil
}

func (h *handler) UpsertModuleDescription(ctx context.Context, req *gen.UpsertModuleDescriptionBody, params gen.UpsertModuleDescriptionParams) (*gen.SuccessResult, error) {
	if err := db.UpsertModuleDescription(h.db, getUserID(ctx), params.Name, req.Description); err != nil {
		return nil, fmt.Errorf("module not found")
	}
	return &gen.SuccessResult{Success: true}, nil
}

// ── OAuth Consents ───────────────────────────────────────────

func (h *handler) ListOAuthConsents(ctx context.Context) ([]gen.OAuthConsent, error) {
	consents, err := db.ListOAuthConsents(h.db, getUserID(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list consents")
	}
	out := make([]gen.OAuthConsent, len(consents))
	for i, c := range consents {
		grantedAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
		out[i] = gen.OAuthConsent{
			ID:        c.ID,
			ClientID:  c.Module,
			Scopes:    "",
			GrantedAt: grantedAt,
		}
	}
	return out, nil
}

func (h *handler) RevokeOAuthConsent(ctx context.Context, params gen.RevokeOAuthConsentParams) (*gen.RevokeConsentResult, error) {
	if err := db.RevokeOAuthConsent(h.db, getUserID(ctx), params.ID); err != nil {
		return nil, fmt.Errorf("consent not found")
	}
	return &gen.RevokeConsentResult{Revoked: gen.NewOptBool(true)}, nil
}

// ── OAuth App Credentials ────────────────────────────────────

func (h *handler) GetOAuthAppCredentials(ctx context.Context, params gen.GetOAuthAppCredentialsParams) (*gen.OAuthAppCredentials, error) {
	app, err := db.GetOAuthAppCredentials(h.db, params.Provider)
	if err != nil {
		return nil, fmt.Errorf("OAuth app not configured for provider: %s", params.Provider)
	}
	return &gen.OAuthAppCredentials{
		Provider:     gen.NewOptString(app.Provider),
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		RedirectURI:  app.RedirectURI,
	}, nil
}

// ── Admin ────────────────────────────────────────────────────

func (h *handler) ListOAuthApps(ctx context.Context) ([]gen.OAuthApp, error) {
	apps, err := db.ListOAuthApps(h.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list OAuth apps")
	}
	out := make([]gen.OAuthApp, len(apps))
	for i, a := range apps {
		out[i] = gen.OAuthApp{
			Provider:  gen.NewOptString(a.Provider),
			ClientID:  gen.NewOptString(a.ClientID),
			RedirectURI: gen.NewOptString(a.RedirectURI),
			Enabled:   gen.NewOptBool(a.Enabled),
			CreatedAt: gen.NewOptDateTime(a.CreatedAt),
		}
	}
	return out, nil
}

func (h *handler) UpsertOAuthApp(ctx context.Context, req *gen.UpsertOAuthAppBody, params gen.UpsertOAuthAppParams) (*gen.SuccessResult, error) {
	app := &db.OAuthApp{
		Provider:     params.Provider,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURI:  req.RedirectURI,
		Enabled:      req.Enabled,
	}
	if err := db.UpsertOAuthApp(h.db, app); err != nil {
		return nil, fmt.Errorf("failed to upsert OAuth app")
	}
	return &gen.SuccessResult{Success: true}, nil
}

func (h *handler) DeleteOAuthApp(ctx context.Context, params gen.DeleteOAuthAppParams) (*gen.SuccessResult, error) {
	if err := db.DeleteOAuthApp(h.db, params.Provider); err != nil {
		return nil, fmt.Errorf("OAuth app not found")
	}
	return &gen.SuccessResult{Success: true}, nil
}

func (h *handler) ListAllOAuthConsents(ctx context.Context) ([]gen.OAuthConsentAdmin, error) {
	creds, err := db.ListAllOAuthConsents(h.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list consents")
	}
	out := make([]gen.OAuthConsentAdmin, len(creds))
	for i, c := range creds {
		out[i] = gen.OAuthConsentAdmin{
			ID:        c.ID,
			UserID:    c.UserID,
			ClientID:  c.Module,
			Scopes:    "",
			GrantedAt: c.CreatedAt, // UserCredential.CreatedAt is time.Time
		}
	}
	return out, nil
}

// ── Helpers ──────────────────────────────────────────────────

func dbPromptToGen(p db.Prompt, moduleNameMap map[string]string) gen.Prompt {
	out := gen.Prompt{
		ID:        p.ID,
		Name:      p.Name,
		Content:   p.Content,
		Enabled:   p.Enabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.Description != nil {
		out.Description = gen.NewOptNilString(*p.Description)
	}
	if p.ModuleID != nil {
		if name, ok := moduleNameMap[*p.ModuleID]; ok {
			out.ModuleName = gen.NewOptNilString(name)
		}
	}
	return out
}

func optNilStringFromPtr(s *string) gen.OptNilString {
	if s != nil {
		return gen.NewOptNilString(*s)
	}
	return gen.OptNilString{}
}

// buildModuleNameMap returns a map of module ID → module name.
func (h *handler) buildModuleNameMap() map[string]string {
	var modules []db.Module
	h.db.Select("id", "name").Find(&modules)
	m := make(map[string]string, len(modules))
	for _, mod := range modules {
		m[mod.ID] = mod.Name
	}
	return m
}

// resolveModuleName returns a module name pointer from a module ID pointer.
func (h *handler) resolveModuleName(moduleID *string) *string {
	if moduleID == nil {
		return nil
	}
	var mod db.Module
	if h.db.Select("name").Where("id = ?", *moduleID).First(&mod).Error == nil {
		return &mod.Name
	}
	return nil
}

// resolveModuleID returns a module ID pointer from a module name.
func (h *handler) resolveModuleID(moduleName string) *string {
	var mod db.Module
	if h.db.Select("id").Where("name = ?", moduleName).First(&mod).Error == nil {
		return &mod.ID
	}
	return nil
}
