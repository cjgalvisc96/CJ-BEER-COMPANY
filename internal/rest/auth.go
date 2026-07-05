package rest

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/auth"
)

// The RBAC vocabulary, in the ubiquitous language: who may do what.
const (
	// RoleViewer may read every projection.
	RoleViewer = "viewer"
	// RoleSalesManager may place sales orders.
	RoleSalesManager = "sales-manager"
	// RoleWarehouseOperator may declare production orders.
	RoleWarehouseOperator = "warehouse-operator"
)

const principalKey = "auth.principal"

// TokenVerifier validates a bearer token. Nil disables authentication
// entirely (zero-dependency dev/test mode); the OIDC implementation lives
// in platform/auth.
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (auth.Principal, error)
}

// authenticate parses and verifies the Authorization header, storing the
// Principal for the RBAC checks downstream.
func authenticate(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		if verifier == nil {
			return
		}
		header := c.GetHeader("Authorization")
		rawToken, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "missing bearer token"})
			return
		}
		principal, err := verifier.Verify(c.Request.Context(), rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "invalid token"})
			return
		}
		c.Set(principalKey, principal)
	}
}

// requireRole is the RBAC gate: 403 when the authenticated caller lacks
// the role. A no-op while authentication is disabled.
func requireRole(verifier TokenVerifier, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if verifier == nil {
			return
		}
		principal := c.MustGet(principalKey).(auth.Principal)
		if !principal.HasRole(role) {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "requires role " + role})
			return
		}
	}
}
