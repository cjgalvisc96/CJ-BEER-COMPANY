package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/auth"
)

const issuer = "http://localhost:8180/realms/brewup"

// idp is a minimal in-test identity provider: an RSA key, a JWKS endpoint,
// and a token signer — everything an OIDC resource server verifies against.
type idp struct {
	key    *rsa.PrivateKey
	signer jose.Signer
	server *httptest.Server
}

func newIDP(t *testing.T) *idp {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithHeader("kid", "test-key"),
	)
	require.NoError(t, err)

	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: key.Public(), KeyID: "test-key", Algorithm: "RS256", Use: "sig",
	}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(server.Close)
	return &idp{key: key, signer: signer, server: server}
}

// tokenWith signs a token with the given claims merged over sane defaults.
func (i *idp) tokenWith(t *testing.T, overrides map[string]any) string {
	t.Helper()
	claims := map[string]any{
		"iss":             issuer,
		"aud":             "brewup-api",
		"sub":             "user-1",
		"exp":             time.Now().Add(time.Hour).Unix(),
		"iat":             time.Now().Unix(),
		"realm_access":    map[string]any{"roles": []string{"viewer"}},
		"resource_access": map[string]any{"brewup-api": map[string]any{"roles": []string{"sales-manager"}}},
	}
	for key, value := range overrides {
		claims[key] = value
	}
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	jws, err := i.signer.Sign(payload)
	require.NoError(t, err)
	raw, err := jws.CompactSerialize()
	require.NoError(t, err)
	return raw
}

func newVerifier(t *testing.T, idp *idp) *auth.OIDCVerifier {
	t.Helper()
	return auth.NewOIDCVerifier(context.Background(), issuer, idp.server.URL, "brewup-api")
}

func TestVerifyAcceptsValidTokenAndMergesRoles(t *testing.T) {
	provider := newIDP(t)
	verifier := newVerifier(t, provider)

	principal, err := verifier.Verify(context.Background(), provider.tokenWith(t, nil))

	require.NoError(t, err)
	assert.Equal(t, "user-1", principal.Subject)
	assert.True(t, principal.HasRole("viewer"), "realm role")
	assert.True(t, principal.HasRole("sales-manager"), "client role")
	assert.False(t, principal.HasRole("warehouse-operator"))
}

func TestVerifyRejectsBadTokens(t *testing.T) {
	provider := newIDP(t)
	verifier := newVerifier(t, provider)
	ctx := context.Background()

	cases := map[string]string{
		"garbage":        "not-a-jwt",
		"wrong issuer":   provider.tokenWith(t, map[string]any{"iss": "https://evil.example"}),
		"wrong audience": provider.tokenWith(t, map[string]any{"aud": "other-api"}),
		"expired": provider.tokenWith(t, map[string]any{
			"exp": time.Now().Add(-time.Hour).Unix(),
		}),
	}
	for name, token := range cases {
		_, err := verifier.Verify(ctx, token)
		assert.Error(t, err, name)
	}

	// A token signed by a DIFFERENT key (forged) is rejected too.
	forger := newIDP(t)
	_, err := verifier.Verify(ctx, forger.tokenWith(t, nil))
	assert.Error(t, err, "forged signature")
}

func TestVerifyRejectsMalformedRoleClaims(t *testing.T) {
	provider := newIDP(t)
	verifier := newVerifier(t, provider)

	_, err := verifier.Verify(context.Background(),
		provider.tokenWith(t, map[string]any{"realm_access": "not-an-object"}))

	assert.ErrorContains(t, err, "parse token claims")
}

func TestVerifyToleratesMissingRoleClaims(t *testing.T) {
	provider := newIDP(t)
	verifier := newVerifier(t, provider)

	principal, err := verifier.Verify(context.Background(), provider.tokenWith(t, map[string]any{
		"realm_access":    map[string]any{},
		"resource_access": map[string]any{},
	}))

	require.NoError(t, err)
	assert.False(t, principal.HasRole("viewer"), "no roles granted")
}
