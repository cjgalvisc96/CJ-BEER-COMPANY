# ADR-0010: Authentication (OIDC tokens + SSO) and RBAC with Keycloak

- **Status**: accepted
- **Date**: 2026-07-05

## Context

The API was open. Production needs authentication (token-based, with SSO
for humans) and authorization (role-based), and the whole thing must be
testable end to end locally with Docker.

## Decision

**Keycloak** as the identity provider, chosen over the alternatives:
Zitadel (solid, Go-native, but younger ecosystem), Ory (composable but
needs Hydra+Kratos+Keto for the same outcome), Dex (no user store or role
management), SaaS IdPs (not runnable locally). Keycloak gives OIDC tokens,
SSO, realm-role RBAC, and — decisive for local e2e — `start-dev
--import-realm`: the whole realm (roles, users, client, audience mapper)
lives in `docker/keycloak/realm.json` and boots reproducibly in compose.

**The app stays a standard OIDC resource server** (`platform/auth`): it
verifies RS256 bearer tokens against a JWKS (signature, issuer, expiry,
audience) via go-oidc and extracts realm + client roles into a `Principal`.
Nothing Keycloak-specific beyond the role-claim shape — swapping IdPs is
configuration.

**RBAC in the ubiquitous language**, enforced by REST middleware:

| Role | May |
|---|---|
| `viewer` | read every projection |
| `sales-manager` | place sales orders |
| `warehouse-operator` | declare production orders |

Test users: `manager`, `operator`, `barfly` (password `brewup`).

**Modes**: auth is on iff `AUTH_ISSUER` is set (the compose stack sets
it); empty keeps the API open for zero-dependency dev and the in-process
test suite. `AUTH_ISSUER` (the expected `iss`) and `AUTH_JWKS_URL` (where
keys are fetched) are separate settings, because locally the issuer is the
host-visible URL (`localhost:8180`, forced via `KC_HOSTNAME`) while the
API fetches keys over the container network.

**SSO**: the IdP's job. The client has the standard authorization-code
flow enabled; browser apps log in via Keycloak (which can federate Google/
SAML/LDAP upstream) and call the API with the resulting bearer token. The
password grant is enabled for the seeder, curl, and CI.

## Consequences

- `/healthz` and `/readyz` stay open; every `/v1` route authenticates, and
  writes are role-gated (401 without a valid token, 403 without the role).
- The verifier is unit-tested against an in-test IdP (local JWKS + tokens
  signed with a throwaway RSA key — valid/expired/forged/wrong-audience/
  malformed-claims all covered); RBAC middleware is tested against a fake
  verifier; the real Keycloak wire is proven by the compose smoke test,
  which asserts the 401/403 cases and runs the whole saga with per-role
  tokens.
- The seeder now demonstrates RBAC: production orders as `operator`,
  the sales order as `manager`.
- Domain, application, and modules are untouched — auth is entirely a
  presentation/platform concern, as the layering demands.
