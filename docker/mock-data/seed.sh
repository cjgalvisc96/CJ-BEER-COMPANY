#!/bin/sh
# Demo-data seeder: authenticates against Keycloak (when configured),
# declares production orders as the warehouse operator, and places a sales
# order as the sales manager — demonstrating the RBAC split. Without
# KEYCLOAK_URL it runs unauthenticated (auth-disabled dev mode).
set -eu

API_URL="${API_URL:-http://localhost:8080}"
KEYCLOAK_URL="${KEYCLOAK_URL:-}"

echo "seed: waiting for $API_URL ..."
for _ in $(seq 1 60); do
  if curl -sf "$API_URL/healthz" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -sf "$API_URL/healthz" >/dev/null || { echo "seed: API never became healthy"; exit 1; }

token() { # token <username> — password grant against the brewup realm
  curl -sf "$KEYCLOAK_URL/realms/brewup/protocol/openid-connect/token" \
    -d grant_type=password -d client_id=brewup-api \
    -d "username=$1" -d password=brewup \
    | sed 's/.*"access_token":"\([^"]*\)".*/\1/'
}

OPERATOR_TOKEN=""
MANAGER_TOKEN=""
if [ -n "$KEYCLOAK_URL" ]; then
  echo "seed: fetching tokens from $KEYCLOAK_URL ..."
  for _ in $(seq 1 60); do
    OPERATOR_TOKEN=$(token operator || true)
    [ -n "$OPERATOR_TOKEN" ] && break
    sleep 2
  done
  [ -n "$OPERATOR_TOKEN" ] || { echo "seed: could not obtain operator token"; exit 1; }
  MANAGER_TOKEN=$(token manager)
fi

IPA_ID="11111111-1111-1111-1111-111111111111"
LAGER_ID="22222222-2222-2222-2222-222222222222"
STOUT_ID="33333333-3333-3333-3333-333333333333"

produce() { # produce <beer_id> <beer_name> <liters> — as warehouse operator
  curl -sf "$API_URL/v1/warehouses/availability" \
    -H 'Content-Type: application/json' \
    ${OPERATOR_TOKEN:+-H "Authorization: Bearer $OPERATOR_TOKEN"} \
    -d "{\"beer_id\":\"$1\",\"beer_name\":\"$2\",\"quantity\":{\"value\":$3,\"unit_of_measure\":\"Lt\"}}" \
    >/dev/null
}

produce "$IPA_ID" "BrewUp IPA" 120
produce "$LAGER_ID" "CJ Golden Lager" 200
produce "$STOUT_ID" "CJ Midnight Stout" 60

sleep 1 # let the availability projections settle

# The sales order is placed by the sales manager.
curl -sf "$API_URL/v1/sales" \
  -H 'Content-Type: application/json' \
  ${MANAGER_TOKEN:+-H "Authorization: Bearer $MANAGER_TOKEN"} \
  -d "{
    \"sales_order_number\": \"20260705-0001\",
    \"customer_name\": \"Bar La Cerveceria\",
    \"rows\": [
      {\"beer_id\":\"$IPA_ID\",\"beer_name\":\"BrewUp IPA\",
       \"quantity\":{\"value\":24,\"unit_of_measure\":\"Lt\"},
       \"price\":{\"value\":5,\"currency\":\"EUR\"}},
      {\"beer_id\":\"$LAGER_ID\",\"beer_name\":\"CJ Golden Lager\",
       \"quantity\":{\"value\":48,\"unit_of_measure\":\"Lt\"},
       \"price\":{\"value\":4,\"currency\":\"EUR\"}}
    ]
  }" >/dev/null

echo "seed: done — try GET $API_URL/v1/sales (with a viewer token) "