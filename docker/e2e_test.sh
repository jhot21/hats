#!/usr/bin/env bash
# E2E test script for HATS Docker Compose stack.
# Brings up the stack, runs assertions, then tears it down.
#
# Usage: bash e2e_test.sh [--no-build]   (from the docker/ directory)

set -euo pipefail

COMPOSE="docker compose -f compose.test.yml"
HATS_URL="http://localhost:8888"
MOCKHA_URL="http://localhost:8123"
HATS_TOKEN="test-hats-token"
BUILD_FLAG="--build"
PASS=0
FAIL=0

if [[ "${1:-}" == "--no-build" ]]; then
    BUILD_FLAG=""
fi

# --- Cleanup ---
cleanup() {
    echo ""
    echo "--- Tearing down compose stack ---"
    $COMPOSE down -v 2>&1 | tail -3
    if [ "$FAIL" -gt 0 ]; then
        echo "E2E result: $PASS passed, $FAIL FAILED"
        exit 1
    fi
    echo "E2E result: $PASS passed, 0 failed"
}
trap cleanup EXIT

# --- Helpers ---
pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

wait_for_url() {
    local name="$1" url="$2" max="${3:-30}"
    echo -n "Waiting for $name to be ready..."
    for i in $(seq 1 "$max"); do
        if curl -sf --max-time 2 "$url" > /dev/null 2>&1; then
            echo " OK"
            return 0
        fi
        sleep 1
        echo -n "."
    done
    echo " TIMED OUT"
    return 1
}

poll_until() {
    local label="$1" expected="$2" max="${3:-30}"
    shift 3
    echo -n "Waiting for: $label..."
    for i in $(seq 1 "$max"); do
        local got
        got=$("$@" 2>/dev/null || true)
        if [ "$got" = "$expected" ]; then
            echo " OK"
            return 0
        fi
        sleep 1
        echo -n "."
    done
    echo " TIMED OUT (last value: '$got')"
    return 1
}

inject_state() {
    curl -sf -X POST "$MOCKHA_URL/inject/state" \
        -H "Content-Type: application/json" \
        -d "{\"entity_id\":\"$1\",\"old_state\":\"$2\",\"new_state\":\"$3\"}" > /dev/null
}

hats_state() {
    curl -sf "$HATS_URL/api/state/$1" \
        -H "Authorization: Bearer $HATS_TOKEN" 2>/dev/null || true
}

mockha_connections() {
    curl -sf "$MOCKHA_URL/stats" 2>/dev/null \
        | grep -o '"connections":[0-9]*' | grep -o '[0-9]*$' || echo 0
}

# --- Startup ---
echo "=== HATS E2E Tests ==="
echo ""
echo "--- Starting compose stack ---"
$COMPOSE up $BUILD_FLAG -d

wait_for_url "mockha" "$MOCKHA_URL/stats"
wait_for_url "hats"   "$HATS_URL/status"

echo -n "Waiting for HATS to connect to mockha..."
for i in $(seq 1 30); do
    if [ "$(mockha_connections)" -ge 1 ]; then echo " OK"; break; fi
    sleep 1; echo -n "."
    if [ "$i" -eq 30 ]; then echo " TIMED OUT"; exit 1; fi
done

# --- Tests ---
echo ""
echo "--- Running tests ---"
echo ""

# Test 1: state_changed event reaches HATS KV and is served by the API
echo "Test 1: state_changed routing (HA → NATS KV → HATS API)"
inject_state "sensor.e2e_temp" "off" "on"
if poll_until "sensor.e2e_temp = on" "on" 15 hats_state "sensor.e2e_temp"; then
    pass "state_changed event routed to HATS API"
else
    fail "state_changed event did NOT reach HATS API within 15s"
fi

# Test 2: state value is updated on a second change
echo ""
echo "Test 2: state update (second event overwrites first)"
inject_state "sensor.e2e_temp" "on" "42.5"
if poll_until "sensor.e2e_temp = 42.5" "42.5" 15 hats_state "sensor.e2e_temp"; then
    pass "state update reflected in HATS API"
else
    fail "state update did NOT reach HATS API within 15s"
fi

# Test 3: a different entity doesn't cross-contaminate
echo ""
echo "Test 3: entity isolation (separate entities have separate state)"
inject_state "sensor.e2e_other" "off" "distinct"
if poll_until "sensor.e2e_other = distinct" "distinct" 15 hats_state "sensor.e2e_other"; then
    pass "separate entity has independent state"
else
    fail "entity isolation broken"
fi
# Original entity should still have its old value
got=$(hats_state "sensor.e2e_temp")
if [ "$got" = "42.5" ]; then
    pass "original entity state unchanged by other entity event"
else
    fail "original entity state corrupted: expected '42.5', got '$got'"
fi

# Test 4: connection is alive (no unexpected reconnects)
echo ""
echo "Test 4: connection stability (no unexpected reconnects)"
conns=$(mockha_connections)
if [ "$conns" -eq 1 ]; then
    pass "exactly 1 connection established (no spurious reconnects)"
else
    fail "expected 1 connection, got $conns"
fi
