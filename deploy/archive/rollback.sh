#!/usr/bin/env bash
# Rollback: Go → Bun+PHP (30 seconds)
set -euo pipefail

echo "═══ ROLLBACK: Go → Bun+PHP ═══"

# 1. Restore tunnel config
if [[ -f /etc/cloudflared/wpuiai.yml.bak ]]; then
    cp /etc/cloudflared/wpuiai.yml.bak /etc/cloudflared/wpuiai.yml
    systemctl restart cloudflared-wpuiai 2>/dev/null || true
    echo "  ✅ Tunnel config restored"
fi

# 2. Stop Go engine
systemctl stop uiai-engine 2>/dev/null || true
systemctl disable uiai-engine 2>/dev/null || true
echo "  ✅ Go engine stopped"

# 3. Restart old services
cd /home/wpuiai/ai-api
if [[ -f bun/src/index.ts ]]; then
    su - wpuiai -c "cd /home/wpuiai/ai-api && bun run bun/src/index.ts &" 2>/dev/null
    echo "  ✅ Bun restarted"
fi

if [[ -f index.php ]]; then
    su - wpuiai -c "cd /home/wpuiai/ai-api && php -S 127.0.0.1:3006 index.php &" 2>/dev/null
    echo "  ✅ PHP restarted"
fi

# 4. Restart Browserless
docker start browserless 2>/dev/null && echo "  ✅ Browserless restarted" || echo "  ⚠️  No Browserless container"

echo ""
echo "═══ ROLLBACK COMPLETE ═══"
