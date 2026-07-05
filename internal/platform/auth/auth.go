// Package auth verifies OIDC bearer tokens and extracts the caller's
// roles. The app is a standard OIDC resource server: any IdP that issues
// RS256 JWTs with realm/client roles works (Keycloak locally and in the
// compose stack; swapping IdPs is configuration, not code).
package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Principal is the authenticated caller: who they are and what they may do.
type Principal struct {
	Subject string
	roles   map[string]struct{}
}

func NewPrincipal(subject string, roles []string) Principal {
	set := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		set[role] = struct{}{}
	}
	return Principal{Subject: subject, roles: set}
}

// HasRole answers the RBAC question.
func (p Principal) HasRole(role string) bool {
	_, ok := p.roles[role]
	return ok
}

// OIDCVerifier validates bearer tokens: signature against the IdP's JWKS,
// issuer, expiry, and audience. The JWKS URL is configured separately from
// the issuer so the token's `iss` can be a host-visible URL while the keys
// are fetched over the container network (the classic local-Docker split).
type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
	clientID string
}

func NewOIDCVerifier(ctx context.Context, issuer, jwksURL, clientID string) *OIDCVerifier {
	keySet := oidc.NewRemoteKeySet(ctx, jwksURL)
	return &OIDCVerifier{
		verifier: oidc.NewVerifier(issuer, keySet, &oidc.Config{ClientID: clientID}),
		clientID: clientID,
	}
}

// keycloakClaims is the consumer-driven slice of the token: realm roles
// plus this client's roles.
type keycloakClaims struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

// Verify checks the raw bearer token and returns the caller.
func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (Principal, error) {
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return Principal{}, fmt.Errorf("verify token: %w", err)
	}
	var claims keycloakClaims
	if err := token.Claims(&claims); err != nil {
		return Principal{}, fmt.Errorf("parse token claims: %w", err)
	}
	roles := claims.RealmAccess.Roles
	roles = append(roles, claims.ResourceAccess[v.clientID].Roles...)
	return NewPrincipal(token.Subject, roles), nil
}
