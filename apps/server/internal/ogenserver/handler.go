package ogenserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	db          *gorm.DB
	adminEmails map[string]bool
}

var _ gen.Handler = (*handler)(nil)

// NewHandler creates the ogen Handler implementation.
func NewHandler(database *gorm.DB, adminEmailsCSV string) gen.Handler {
	emails := map[string]bool{}
	for _, e := range strings.Split(adminEmailsCSV, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			emails[strings.ToLower(e)] = true
		}
	}
	return &handler{db: database, adminEmails: emails}
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
			Tools:  jx.Raw(m.Tools),
		}
		if !m.CreatedAt.IsZero() {
			out[i].CreatedAt = gen.NewOptDateTime(m.CreatedAt)
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
	user, err := db.FindByID(h.db, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	admin := user.Email != nil && h.adminEmails[strings.ToLower(*user.Email)]
	profile, err := db.GetMyProfile(h.db, userID, admin)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	out := &gen.UserProfile{
		ID:             profile.ID,
		AccountStatus:  profile.AccountStatus,
		PlanID:         profile.PlanID,
		Role:           profile.Role,
		Settings:       jx.Raw(profile.Settings),
		ConnectedCount: profile.ConnectedCount,
	}
	if profile.DisplayName != nil {
		out.DisplayName = gen.NewOptNilString(*profile.DisplayName)
	}
	if profile.AvatarURL != nil {
		out.AvatarURL = gen.NewOptNilString(*profile.AvatarURL)
	}
	if profile.Email != nil {
		out.Email = gen.NewOptNilString(*profile.Email)
	}
	return out, nil
}

func (h *handler) UpdateSettings(ctx context.Context, req *gen.UpdateSettingsBody) (*gen.SuccessResult, error) {
	if err := db.UpdateSettings(h.db, getUserID(ctx), json.RawMessage(req.Settings)); err != nil {
		return nil, fmt.Errorf("failed to update settings")
	}
	return &gen.SuccessResult{Success: true}, nil
}

func (h *handler) CompleteUserOnboarding(ctx context.Context, req *gen.CompleteOnboardingBody) (*gen.SuccessResult, error) {
	if req.EventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	if err := db.CompleteOnboarding(h.db, getUserID(ctx), req.EventID); err != nil {
		return nil, fmt.Errorf("failed to complete onboarding")
	}
	return &gen.SuccessResult{Success: true}, nil
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
		out[i] = gen.Credential{
			Module:    c.Module,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}
	return out, nil
}

func (h *handler) UpsertCredential(ctx context.Context, req *gen.UpsertCredentialBody, params gen.UpsertCredentialParams) (*gen.SuccessResult, error) {
	if err := db.UpsertCredential(h.db, getUserID(ctx), params.Module, string(req.Credentials)); err != nil {
		return nil, fmt.Errorf("failed to upsert credential")
	}
	return &gen.SuccessResult{Success: true}, nil
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
			ID:        k.ID,
			KeyPrefix: k.KeyPrefix,
			Name:      k.Name,
			CreatedAt: gen.NewOptDateTime(k.CreatedAt),
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
	}
	return out, nil
}

func (h *handler) GenerateApiKey(ctx context.Context, req *gen.GenerateApiKeyBody) (*gen.GenerateApiKeyResult, error) {
	// Accept both "name" and "display_name"
	name := req.Name.Or("")
	if name == "" {
		name = req.DisplayName
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	userID := getUserID(ctx)

	// Compute expiration
	var expiresAt *time.Time
	if ea, ok := req.ExpiresAt.Get(); ok {
		expiresAt = &ea
	} else if ei, ok := req.ExpiresIn.Get(); ok {
		t := time.Now().Add(time.Duration(ei) * time.Second)
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

	out := &gen.GenerateApiKeyResult{
		ID:          key.ID,
		APIKey:      token,
		Key:         gen.NewOptString(token),
		KeyPrefix:   prefix,
		Name:        name,
		DisplayName: gen.NewOptString(name),
		CreatedAt:   gen.NewOptDateTime(key.CreatedAt),
	}
	if expiresAt != nil {
		out.ExpiresAt = gen.NewOptNilDateTime(*expiresAt)
	}
	return out, nil
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
	out := make([]gen.Prompt, len(prompts))
	for i, p := range prompts {
		out[i] = dbPromptToGen(p)
	}
	return out, nil
}

func (h *handler) GetPrompt(ctx context.Context, params gen.GetPromptParams) (*gen.Prompt, error) {
	p, err := db.GetPrompt(h.db, getUserID(ctx), params.ID)
	if err != nil {
		return nil, fmt.Errorf("prompt not found")
	}
	out := dbPromptToGen(*p)
	return &out, nil
}

func (h *handler) CreatePrompt(ctx context.Context, req *gen.CreatePromptBody) (*gen.Prompt, error) {
	p := db.Prompt{
		UserID:  getUserID(ctx),
		Name:    req.Name,
		Content: req.Content,
		Enabled: req.Enabled,
	}
	if desc, ok := req.Description.Get(); ok {
		p.Description = &desc
	}
	// module_name is handled via the Prompt struct's json binding
	if err := db.CreatePrompt(h.db, &p); err != nil {
		return nil, fmt.Errorf("failed to create prompt")
	}
	out := dbPromptToGen(p)
	return &out, nil
}

func (h *handler) UpdatePrompt(ctx context.Context, req *gen.UpdatePromptBody, params gen.UpdatePromptParams) (*gen.SuccessResult, error) {
	updates := map[string]interface{}{}
	if v, ok := req.Name.Get(); ok {
		updates["name"] = v
	}
	if v, ok := req.Content.Get(); ok {
		updates["content"] = v
	}
	if v, ok := req.Enabled.Get(); ok {
		updates["enabled"] = v
	}
	if v, ok := req.Description.Get(); ok {
		updates["description"] = v
	}
	if v, ok := req.ModuleName.Get(); ok {
		updates["module_name"] = v
	}

	if err := db.UpdatePrompt(h.db, getUserID(ctx), params.ID, updates); err != nil {
		return nil, fmt.Errorf("prompt not found")
	}
	return &gen.SuccessResult{Success: true}, nil
}

func (h *handler) DeletePrompt(ctx context.Context, params gen.DeletePromptParams) (*gen.SuccessResult, error) {
	if err := db.DeletePrompt(h.db, getUserID(ctx), params.ID); err != nil {
		return nil, fmt.Errorf("prompt not found")
	}
	return &gen.SuccessResult{Success: true}, nil
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

func (h *handler) UpsertToolSettings(ctx context.Context, req *gen.UpsertToolSettingsBody, params gen.UpsertToolSettingsParams) (*gen.SuccessResult, error) {
	if err := db.UpsertToolSettings(h.db, getUserID(ctx), params.Name, req.Enabled, req.Disabled); err != nil {
		return nil, fmt.Errorf("module not found")
	}
	return &gen.SuccessResult{Success: true}, nil
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
		out[i] = gen.OAuthConsent{
			ID:        c.ID,
			Module:    c.Module,
			CreatedAt: c.CreatedAt,
		}
	}
	return out, nil
}

func (h *handler) RevokeOAuthConsent(ctx context.Context, params gen.RevokeOAuthConsentParams) (*gen.SuccessResult, error) {
	if err := db.RevokeOAuthConsent(h.db, getUserID(ctx), params.ID); err != nil {
		return nil, fmt.Errorf("consent not found")
	}
	return &gen.SuccessResult{Success: true}, nil
}

// ── OAuth App Credentials ────────────────────────────────────

func (h *handler) GetOAuthAppCredentials(ctx context.Context, params gen.GetOAuthAppCredentialsParams) (*gen.OAuthAppCredentials, error) {
	app, err := db.GetOAuthAppCredentials(h.db, params.Provider)
	if err != nil {
		return nil, fmt.Errorf("OAuth app not configured for provider: %s", params.Provider)
	}
	return &gen.OAuthAppCredentials{
		Provider:     app.Provider,
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
			ID:           gen.NewOptString(a.ID),
			Provider:     gen.NewOptString(a.Provider),
			ClientID:     gen.NewOptString(a.ClientID),
			ClientSecret: gen.NewOptString(a.ClientSecret),
			RedirectURI:  gen.NewOptString(a.RedirectURI),
			Enabled:      gen.NewOptBool(a.Enabled),
			CreatedAt:    gen.NewOptDateTime(a.CreatedAt),
			UpdatedAt:    gen.NewOptDateTime(a.UpdatedAt),
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
		Enabled:      req.Enabled.Or(true),
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
			Module:    c.Module,
			CreatedAt: c.CreatedAt,
		}
	}
	return out, nil
}

// ── Helpers ──────────────────────────────────────────────────

func dbPromptToGen(p db.Prompt) gen.Prompt {
	out := gen.Prompt{
		ID:        p.ID,
		UserID:    gen.NewOptString(p.UserID),
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
		out.ModuleID = gen.NewOptNilString(*p.ModuleID)
	}
	return out
}
