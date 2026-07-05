#!/bin/sh
# Demo-data seeder: waits for the API, then creates beers, brews stock, and
# places a confirmed order — so a fresh `task docker:up` is explorable
# immediately. Idempotent-ish: duplicate beer names 409 and are skipped.
set -eu

API_URL="${API_URL:-http://localhost:8080}"

echo "seed: waiting for $API_URL ..."
for _ in $(seq 1 60); do
  if curl -sf "$API_URL/healthz" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -sf "$API_URL/healthz" >/dev/null || { echo "seed: API never became healthy"; exit 1; }

json_field() { # json_field <field>  — tiny extractor, avoids jq dependency
  sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p"
}

create_beer() { # create_beer <name> <style> <abv> <price_cents> <description>
  curl -s "$API_URL/api/v1/beers" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$1\",\"style\":\"$2\",\"abv\":$3,\"price_cents\":$4,\"currency\":\"USD\",\"description\":\"$5\"}" \
    | json_field id
}

brew() { # brew <beer_id> <units>
  batch=$(curl -s "$API_URL/api/v1/batches" \
    -H 'Content-Type: application/json' \
    -d "{\"beer_id\":\"$1\",\"units\":$2}" | json_field id)
  curl -s "$API_URL/api/v1/batches/$batch/complete" \
    -H 'Content-Type: application/json' \
    -d "{\"produced_units\":$2}" >/dev/null
}

LAGER=$(create_beer "CJ Golden Lager" lager 4.8 450 "Flagship easy-drinking lager")
IPA=$(create_beer "CJ Hop Bomb" ipa 6.5 550 "West-coast IPA, resinous and bitter")
STOUT=$(create_beer "CJ Midnight Stout" stout 8.0 600 "Imperial stout aged on cacao")

[ -n "$LAGER" ] && brew "$LAGER" 120
[ -n "$IPA" ] && brew "$IPA" 80
[ -n "$STOUT" ] && brew "$STOUT" 40

sleep 1 # let the batch_completed events replenish inventory

if [ -n "$LAGER" ] && [ -n "$IPA" ]; then
  curl -s "$API_URL/api/v1/orders" \
    -H 'Content-Type: application/json' \
    -d "{\"customer_name\":\"Bar La Cerveceria\",\"lines\":[{\"beer_id\":\"$LAGER\",\"units\":24},{\"beer_id\":\"$IPA\",\"units\":12}]}" \
    >/dev/null
fi

echo "seed: done — try GET $API_URL/api/v1/beers"
