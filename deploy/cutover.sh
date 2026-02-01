#!/usr/bin/env bash
# A13: Cutover from Bun+PHP+Node to single Go binary
# Rollback: ./rollback.sh (30 seconds)
set -euo pipefail

echo "═══════════════════════════════════════════════"
echo "  UIAI Engine Cutover: Bun+PHP+Node → Go"
echo "═══════════════════════════════════════════════"
echo ""

# Pre-flight checks
echo "── Pre-flight ──"

# 1. Go binary exists and is built
if [[ ! -f /home/wpuiai/uiai-engine/uiai-engine ]]; then
    echo "ABORT: Binary not found. Run: cd /home/wpuiai/uiai-engine && go build -o uiai-engine ./cmd/uiai-engine/"
    exit 1
fi
echo "  ✅ Binary exists ($(ls -lh /home/wpuiai/uiai-engine/uiai-engine | awk '{print $5}'))"

# 2. Config exists
if [[ ! -f /home/wpuiai/uiai-engine/config.yaml ]]; then
    echo "ABORT: config.yaml not found"
    exit 1
fi
echo "  ✅ Config exists"

# 3. Current Bun is running (we need something to replace)
if pgrep -f "bun.*index.ts" >/dev/null 2>&1; then
    echo "  ✅ Bun server running (will stop)"
else
    echo "  ⚠️  Bun not running (clean install)"
fi

echo ""
echo "── Step 1: Install systemd service ──"
cp /home/wpuiai/uiai-engine/deploy/uiai-engine.service /etc/systemd/system/uiai-engine.service
systemctl daemon-reload
echo "  ✅ Service installed"

echo ""
echo "── Step 2: Start Go engine (port 7456) ──"
systemctl start uiai-engine
sleep 2

if curl -s -m 5 http://127.0.0.1:7456/api/health | grep -q "healthy"; then
    echo "  ✅ Go engine healthy on :7456"
else
    echo "  ❌ Go engine failed to start"
    journalctl -u uiai-engine --no-pager -n 20
    exit 1
fi

echo ""
echo "── Step 3: Update Cloudflare tunnel ──"
# Backup current config
cp /etc/cloudflared/wpuiai.yml /etc/cloudflared/wpuiai.yml.bak

# Point ai.wpuiai.com to Go engine
sed -i 's|http://127.0.0.1:3007|http://127.0.0.1:7456|g' /etc/cloudflared/wpuiai.yml
sed -i 's|http://127.0.0.1:3006|http://127.0.0.1:7456|g' /etc/cloudflared/wpuiai.yml

# Restart cloudflared
systemctl restart cloudflared-wpuiai 2>/dev/null || cloudflared tunnel run --config /etc/cloudflared/wpuiai.yml wpuiai &
sleep 2
echo "  ✅ Tunnel updated: ai.wpuiai.com → :7456"

echo ""
echo "── Step 4: Verify external access ──"
EXT_CODE=$(curl -s -o /dev/null -w "%{http_code}" -m 10 "https://ai.wpuiai.com/api/health" 2>/dev/null || echo "000")
if [[ "$EXT_CODE" == "200" ]]; then
    echo "  ✅ External health check passed"
else
    echo "  ⚠️  External check returned $EXT_CODE (may need DNS propagation)"
fi

echo ""
echo "── Step 5: Stop old services ──"
# Stop Bun
pkill -f "bun.*index.ts" 2>/dev/null && echo "  ✅ Bun stopped" || echo "  ⚠️  Bun was not running"

# Stop PHP screenshot server
pkill -f "php.*index.php" 2>/dev/null && echo "  ✅ PHP stopped" || echo "  ⚠️  PHP was not running"

# Stop Browserless
docker stop browserless 2>/dev/null && echo "  ✅ Browserless container stopped" || echo "  ⚠️  Browserless was not running"

# Stop Vision daemon
pkill -f "vision-daemon" 2>/dev/null && echo "  ✅ Vision daemon stopped" || echo "  ⚠️  Vision daemon was not running"

echo ""
echo "── Step 6: Enable on boot ──"
systemctl enable uiai-engine
echo "  ✅ Enabled for auto-start"

echo ""
echo "═══════════════════════════════════════════════"
echo "  CUTOVER COMPLETE"
echo ""
echo "  Old: 4 processes (~1.5GB RAM)"
echo "    - Bun (:3007), PHP (:3006), Browserless (:3005), Vision daemon"
echo ""
echo "  New: 1 process (~50MB RAM)"
echo "    - Go engine (:7456)"
echo ""
echo "  Rollback: ./rollback.sh"
echo "═══════════════════════════════════════════════"
