package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"mcpist/server/internal/observability"
)

// Recovery is HTTP middleware that recovers from panics.
// It logs the stack trace and returns a 500 Internal Server Error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				log.Printf("PANIC recovered: %v\n%s", err, stack)

				// Log to Loki for alerting
				requestID := GetRequestID(r.Context())
				authCtx := GetAuthContext(r.Context())
				userID := ""
				if authCtx != nil {
					userID = authCtx.UserID
				}
				observability.LogSecurityEvent(requestID, userID, "panic_recovered", map[string]any{
					"error": fmt.Sprintf("%v", err),
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"internal_server_error","message":"An unexpected error occurred"}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
