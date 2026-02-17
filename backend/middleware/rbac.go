// ABOUTME: Role-based access control middleware for API endpoints
// ABOUTME: Gates endpoints by required role derived from JWT scopes

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
)

// roleHierarchy defines the privilege level for each role.
// Higher value means more privilege. Unknown caller roles resolve to 0,
// which denies access to any protected endpoint (fail-closed).
var roleHierarchy = map[string]int{
	RoleViewer:   1,
	RoleOperator: 2,
}

// RequireRole returns middleware that enforces a minimum role.
// Panics if requiredRole is not in the role hierarchy (catches config errors at startup).
// Anonymous requests (no UserClaims in context) are treated as viewer.
// Returns 403 Forbidden if the caller's role is insufficient.
func RequireRole(requiredRole string) func(http.HandlerFunc) http.HandlerFunc {
	requiredLevel, ok := roleHierarchy[requiredRole]
	if !ok {
		panic(fmt.Sprintf("RequireRole: unknown role %q; valid roles: %v", requiredRole, []string{RoleViewer, RoleOperator}))
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Determine caller's role
			callerRole := RoleViewer // default for anonymous
			claims := GetUserClaims(r)
			if claims != nil && claims.Role != "" {
				callerRole = claims.Role
			}

			callerLevel := roleHierarchy[callerRole]
			if callerLevel < requiredLevel {
				username := ""
				if claims != nil {
					username = claims.Username
				}
				slog.Warn("RBAC authorization denied",
					"path", r.URL.Path,
					"method", r.Method,
					"required_role", requiredRole,
					"user_role", callerRole,
					"username", username,
				)
				writeJSONError(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}
