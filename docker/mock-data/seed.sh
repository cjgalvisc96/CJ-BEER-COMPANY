#!/bin/sh
# Demo-data seeder: waits for the API, declares production orders to fill
# the warehouse, and creates a sales order — so a fresh `task docker:up`
# is explorable immediately.
set -eu

API_URL="${API_URL:-http://localhost:8080}"

echo "seed: waiting for $API_URL ..."
for _ in $(seq 1 60); do
  if curl -sf "$API_URL/healthz" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -sf "$API_URL/healthz" >/dev/null || { echo "seed: API never became healthy"; exit 1; }

IPA_ID="11111111-1111-1111-1111-111111111111"
LAGER_ID="22222222-2222-2222-2222-222222222222"
STOUT_ID="33333333-3333-3333-3333-333333333333"

produce() { # produce <beer_id> <beer_name> <liters>
  curl -s "$API_URL/v1/warehouses/availability" \
    -H 'Content-Type: application/json' \
    -d "{\"beer_id\":\"$1\",\"beer_name\":\"$2\",\"quantity\":{\"value\":$3,\"unit_of_measure\":\"Lt\"}}" \
    >/dev/null
}

produce "$IPA_ID" "BrewUp IPA" 120
produce "$LAGER_ID" "CJ Golden Lager" 200
produce "$STOUT_ID" "CJ Midnight Stout" 60

sleep 1 # let the availability projections settle

curl -s "$API_URL/v1/sales" \
  -H 'Content-Type: application/json' \
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

echo "seed: done — try GET $API_URL/v1/sales and GET $API_URL/v1/warehouses/availability"
