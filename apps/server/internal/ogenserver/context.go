package ogenserver

import "context"

type contextKey string

const (
	userIDKey contextKey = "userID"
	emailKey  contextKey = "email"
)

// withUserID stores the resolved internal user ID in the context.
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// getUserID extracts the internal user ID from the context.
func getUserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// withEmail stores the user email in the context.
func withEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey, email)
}

// getEmail extracts the user email from the context.
func getEmail(ctx context.Context) string {
	e, _ := ctx.Value(emailKey).(string)
	return e
}
