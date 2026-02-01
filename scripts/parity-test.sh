#!/usr/bin/env bash
# A12: Parity Test Suite — verify Go engine matches Bun API contract
# Compares response schemas between Bun (:3007) and Go (:7456)
set -euo pipefail

BUN="http://127.0.0.1:3007"
GO="http://127.0.0.1:7456"
PASS=0; FAIL=0; SKIP=0

# Colors
G='\033[0;32m'; R='\033[0;31m'; Y='\033[0;33m'; N='\033[0m'

# Get a test API key from config
API_KEY="${TEST_API_KEY:-test-key-parity}"

check() {
    local name="$1" method="$2" path="$3" auth="${4:-}" body="${5:-}"
    local bun_args=(-s -o /dev/null -w "%{http_code}" -m 5)
    local go_args=(-s -o /dev/null -w "%{http_code}" -m 5)
    
    if [[ "$method" == "POST" ]]; then
        bun_args+=(-X POST -H "Content-Type: application/json")
        go_args+=(-X POST -H "Content-Type: application/json")
        if [[ -n "$body" ]]; then
            bun_args+=(-d "$body")
            go_args+=(-d "$body")
        fi
    fi
    
    if [[ "$auth" == "key" ]]; then
        bun_args+=(-H "X-API-Key: $API_KEY")
        go_args+=(-H "X-API-Key: $API_KEY")
    fi

    local bun_code go_code
    bun_code=$(curl "${bun_args[@]}" "${BUN}${path}" 2>/dev/null || echo "000")
    go_code=$(curl "${go_args[@]}" "${GO}${path}" 2>/dev/null || echo "000")

    if [[ "$bun_code" == "000" ]]; then
        printf "${Y}SKIP${N} %-40s Bun down\n" "$name"
        ((SKIP++))
        return
    fi

    # For public endpoints, both should return same code
    # For auth endpoints, both should return 401 without valid key
    if [[ "$bun_code" == "$go_code" ]]; then
        printf "${G}PASS${N} %-40s Bun=%s Go=%s\n" "$name" "$bun_code" "$go_code"
        ((PASS++))
    else
        printf "${R}FAIL${N} %-40s Bun=%s Go=%s\n" "$name" "$bun_code" "$go_code"
        ((FAIL++))
    fi
}

check_json_keys() {
    local name="$1" path="$2"
    local bun_keys go_keys
    bun_keys=$(curl -s -m 5 "${BUN}${path}" 2>/dev/null | jq -r 'keys[]' 2>/dev/null | sort | tr '\n' ',')
    go_keys=$(curl -s -m 5 "${GO}${path}" 2>/dev/null | jq -r 'keys[]' 2>/dev/null | sort | tr '\n' ',')

    if [[ -z "$bun_keys" ]]; then
        printf "${Y}SKIP${N} %-40s Bun no JSON\n" "$name"
        ((SKIP++))
        return
    fi

    if [[ "$bun_keys" == "$go_keys" ]]; then
        printf "${G}PASS${N} %-40s keys match: %s\n" "$name" "$go_keys"
        ((PASS++))
    else
        printf "${R}FAIL${N} %-40s Bun=[%s] Go=[%s]\n" "$name" "$bun_keys" "$go_keys"
        ((FAIL++))
    fi
}

echo "═══════════════════════════════════════════════════════════"
echo "  PARITY TEST: Bun (${BUN}) vs Go (${GO})"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ── P0: Critical public endpoints ──
echo "── P0: Health & Public ──"
check "GET /"                    GET  "/"
check "GET /health"              GET  "/health"
check "GET /api/health"          GET  "/api/health"
check "GET /api/health/providers" GET "/api/health/providers"
check_json_keys "health JSON schema"  "/api/health"

echo ""
echo "── P0: Public route info ──"
check "GET /api/critique/models"       GET "/api/critique/models"
check "GET /api/ui-reverse/operations" GET "/api/ui-reverse/operations"
check "GET /api/copilot/health"        GET "/api/copilot/health"

echo ""
echo "── P1: Auth-gated (expect 401 without key) ──"
check "POST /api/critique (no auth)"      POST "/api/critique"      ""  '{"url":"https://example.com"}'
check "POST /api/ui-reverse (no auth)"    POST "/api/ui-reverse"    ""  '{"url":"https://example.com"}'
check "POST /api/section-detect (no auth)" POST "/api/section-detect" "" '{"html":"<div>test</div>"}'
check "POST /api/layout-compare (no auth)" POST "/api/layout-compare" "" '{"sourceUrl":"a","targetUrl":"b"}'
check "POST /api/style-enhance (no auth)"  POST "/api/style-enhance"  "" '{"css":"body{}"}'
check "POST /api/copilot/chat (no auth)"   POST "/api/copilot/chat"   "" '{"message":"hello"}'
check "POST /api/intake/analyze (no auth)" POST "/api/intake/analyze"  "" '{"data":"test"}'

echo ""
echo "── P1: Auth-gated with key ──"
check "POST /api/critique (key)"    POST "/api/critique"    key '{"url":"https://example.com"}'
check "POST /api/ui-reverse (key)"  POST "/api/ui-reverse"  key '{"url":"https://example.com","operation":"analyze"}'
check "POST /api/copilot/chat (key)" POST "/api/copilot/chat" key '{"message":"hello"}'

echo ""
echo "── P2: Extension & Memory ──"
check "GET /api/extension/rate-limits"  GET "/api/extension/rate-limits"
check "POST /api/extension/token"       POST "/api/extension/token" "" '{"extensionId":"test","licenseKey":"test-key"}'
check "GET /api/extension/verify"       GET "/api/extension/verify"
check "GET /api/memory/testuser"        GET "/api/memory/testuser"       key

echo ""
echo "── P2: Usage & Admin ──"
check "GET /api/usage/all (no auth)"     GET "/api/usage/all"
check "GET /api/admin/services (no auth)" GET "/api/admin/services"
check "GET /api/admin/resources (no auth)" GET "/api/admin/resources"

echo ""
echo "── P2: Training & Intelligence ──"
check "GET /api/training/jobs (no auth)"     GET "/api/training/jobs"
check "GET /api/training/evals (no auth)"    GET "/api/training/evals"
check "GET /api/intelligence/health"         GET "/api/intelligence/health"
check "POST /api/intelligence/search"        POST "/api/intelligence/search" key '{"query":"test"}'

echo ""
echo "── P2: Workflow ──"
check "GET /api/workflow/templates"    GET "/api/workflow/templates"
check "POST /api/workflow/execute"     POST "/api/workflow/execute" key '{"action":"test"}'

echo ""
echo "═══════════════════════════════════════════════════════════"
printf "  RESULTS: ${G}%d PASS${N} | ${R}%d FAIL${N} | ${Y}%d SKIP${N}\n" "$PASS" "$FAIL" "$SKIP"
echo "═══════════════════════════════════════════════════════════"

if [[ $FAIL -gt 0 ]]; then
    exit 1
fi
