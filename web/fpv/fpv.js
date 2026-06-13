(() => {
  const token = document.body.dataset.token;
  const sessionId = document.body.dataset.session;
  let frames = 0, started = Date.now(), active = true, frameErrors = 0, frameMs = 250, frameTimer = null, eventFilter = 'all', streamMode = 'cdp_screencast', lastFrameAt = 0, streamUrl = `/m/${token}/stream.cdp.mjpg`, picking = false, picked = null, markerSeq = 0;
  let stream = [];
  const $ = (id) => document.getElementById(id);
  const text = (id, v) => { const el = $(id); if (el) el.textContent = v ?? '—'; };

  function initIcons() {
    if (window.lucide) window.lucide.createIcons();
    if (document.querySelector('svg.lucide')) return;
    const glyphs = {
      'radio-tower':'⌁','shield-check':'⛨','fingerprint':'◎','activity':'↯','heart-pulse':'♡','mouse-pointer-click':'⌖','git-branch':'⑂','sparkles':'✦','list-tree':'☷','lock-keyhole':'●','gauge':'↻','badge-check':'✓','globe-2':'◉','scan':'▣','timer':'⏱','eye':'👁','wifi':'⌁','terminal':'⌘','cloud-off':'⇣','server-crash':'⚠','network':'⇄','send':'➤','timer-reset':'⏱'
    };
    document.querySelectorAll('i[data-lucide]').forEach(i => {
      i.className = 'glyph-fallback';
      i.textContent = glyphs[i.dataset.lucide] || '•';
    });
  }
  function seedCells(id, count = 48) {
    const el = $(id); if (!el || el.dataset.ready) return;
    el.dataset.ready = '1';
    el.innerHTML = Array.from({ length: count }, (_, i) => `<i style="animation-delay:${(i % 18) * 18}ms"></i>`).join('');
  }
  function eventType(kind) {
    const k = kind.toLowerCase();
    if (k.includes('frame')) return 'frame';
    if (k.includes('git')) return 'git';
    if (k.includes('focusa')) return 'focusa';
    if (k.includes('operator') || k.includes('audit') || k.includes('control')) return 'action';
    return 'status';
  }
  function visibleEvents() { return eventFilter === 'all' ? stream : stream.filter(e => e.type === eventFilter); }
  function renderStream() {
    const el = $('stream'); if (!el) return;
    const rows = visibleEvents();
    el.innerHTML = rows.length ? rows.map(e => `<div class="event" data-marker="${e.markerId || ''}" title="${escapeHtml(e.msg)}"><div class="glyph">${e.glyph}</div><div><div class="event-title">${escapeHtml(e.kind)}</div><div class="event-meta">${e.t.toLocaleTimeString()} · ${escapeHtml(e.msg)}</div></div></div>`).join('') : '<div class="empty">No events in this filter.</div>';
    el.querySelectorAll('[data-marker]').forEach(row => { const id = row.dataset.marker; if (!id) return; row.addEventListener('mouseenter', () => highlightMarker(id)); row.addEventListener('mouseleave', () => clearMarker(id)); row.addEventListener('click', () => highlightMarker(id, 900)); });
  }
  function addStream(kind, msg, glyph = '↯', markerId = '') {
    stream.unshift({ t: new Date(), kind, msg, glyph, type: eventType(kind), markerId });
    stream = stream.slice(0, 30);
    renderStream();
  }
  function showActionMarker(label, x, y) {
    const overlay = $('overlay'); if (!overlay) return '';
    const id = `action-marker-${++markerSeq}`;
    const marker = document.createElement('span'); marker.id = id; marker.className = 'action-marker'; marker.textContent = label;
    if (Number.isFinite(x) && Number.isFinite(y)) { marker.style.left = `${x}px`; marker.style.top = `${y}px`; } else { marker.classList.add('no-coord'); }
    overlay.append(marker); setTimeout(() => marker.remove(), 6500); return id;
  }
  function highlightMarker(id, ms = 0) { const marker = document.getElementById(id); if (!marker) return; marker.classList.add('pulse'); if (ms) setTimeout(() => clearMarker(id), ms); }
  function clearMarker(id) { document.getElementById(id)?.classList.remove('pulse'); }
  function escapeHtml(s = '') { return String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c])); }
  async function fpvControl(action, payload = {}) {
    return fetch(`/m/${token}/control`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action, ...payload }) }).then(r => r.json());
  }
  async function sendMsg() {
    const message = $('msg').value.trim(); if (!message) return;
    const r = await fpvControl('message', { message });
    text('controlResult', r.ok ? 'Saved to audit' : `Rejected: ${r.error || 'not allowed'}`);
    addStream('Operator note', r.ok ? 'Saved to audit' : 'Rejected', '✎');
    $('msg').value = ''; await statusTick();
  }
  async function sendClick() {
    const selector = $('selector').value.trim(); if (!selector) return;
    const r = await fpvControl('click', { selector });
    text('controlResult', r.ok ? 'Click sent' : `Click rejected: ${r.error || 'failed'}`);
    addStream('Control click', selector, '⌖'); await statusTick();
  }
  async function sendFill() {
    const selector = $('selector').value.trim(), value = $('fillText').value; if (!selector) return;
    const r = await fpvControl('fill', { selector, text: value });
    text('controlResult', r.ok ? 'Fill sent' : `Fill rejected: ${r.error || 'failed'}`);
    addStream('Control fill', selector, '✍'); await statusTick();
  }
  async function sendPoint(action='selector_at') { if(!picked) return; const r=await fpvControl(action,{x:picked.x,y:picked.y}); if(r.audit?.selector) $('selector').value=r.audit.selector; text('controlResult', r.ok ? (action==='click_xy'?'Point clicked':'Selector picked') : 'Point action failed: '+(r.audit?.error||r.error||'failed')); addStream(action==='click_xy'?'Control point click':'Selector picked', `${picked.x},${picked.y} ${r.audit?.selector||''}`, '⌖'); await statusTick(); }
  async function sendKey(key) {
    const r = await fpvControl('press', { key });
    text('controlResult', r.ok ? `${key} sent` : `Key rejected: ${r.error || 'failed'}`);
    addStream('Control key', key, '⌨'); await statusTick();
  }
  function setQuality(q) {
    frameMs = q === 'smooth' ? 250 : q === 'balanced' ? 500 : 1000;
    document.querySelectorAll('.quality-switch button').forEach(b => b.classList.toggle('active', b.dataset.quality === q));
    if (streamMode === 'polling') { clearInterval(frameTimer); frameTimer = setInterval(frameTick, frameMs); }
    addStream('Stream quality', `${q} mode`, '⌁');
  }
  function renderSeismo() {
    const el = $('seismo'); if (!el) return; seedCells('seismo', 48);
    [...el.children].forEach((bar, i) => {
      const h = 12 + Math.abs(Math.sin((frames + i) * .42)) * 62 + (i % 7) * 2;
      bar.style.height = `${Math.min(72, h)}px`; bar.className = frameErrors ? 'warn' : '';
    });
  }
  function renderGlyph(id, level = 1, bad = false) {
    const el = $(id); if (!el) return; seedCells(id, 72);
    [...el.children].forEach((cell, i) => {
      const hot = ((i + frames) % 13) < level + 2;
      cell.className = bad && hot ? 'bad' : hot ? `l${1 + ((i + frames) % 4)}` : '';
    });
  }
  function sweepCards() { document.querySelectorAll('.metric').forEach((c, i) => { if ((frames + i) % 11 === 0) { c.classList.add('sweep'); setTimeout(() => c.classList.remove('sweep'), 1100); } }); }
  function metric(label, value, hint = '', icon = '') { return `<article class="metric"><label>${icon} ${escapeHtml(label)}</label><strong>${escapeHtml(value ?? '—')}</strong>${hint ? `<small>${escapeHtml(hint)}</small>` : ''}</article>`; }
  function renderContext(ctx = {}) {
    const p = ctx.project || {}, repo = $('repoGrid');
    if (repo) repo.innerHTML = metric('Project', p.name, `Root: ${p.root || 'unknown'}`, '⌘') + metric('Branch', p.branch, '', '⑂') + metric('Head', p.head, p.dirty ? 'Working tree modified' : 'Working tree clean', '●') + metric('Public host', 'fpv.wpuiai.com', 'Path-gated to /m/*', '◧');
    const tree = $('repoTree'); if (tree) tree.textContent = (ctx.tree || []).map(i => `├─ ${i.path}  ${i.active ? 'active' : ''}`).join('\n') || 'No tree context available';
    const f = ctx.focusa || {}, fg = $('focusaGrid'), pred = typeof f.prediction === 'object' ? `${f.prediction.status || 'linked'} ${f.prediction.continuity_id || ''}` : f.prediction;
    if (fg) fg.innerHTML = metric('Focusa status', f.status || 'unknown', f.degraded ? 'Graceful degraded state' : (f.source || 'scope linked'), '✦') + metric('Workpoint', f.workpoint || 'unavailable', f.project_root || '', '⌘') + metric('Next step', f.next_step, '', '➜') + metric('Evidence', (f.evidence || []).join(', '), '', '✓') + metric('Prediction', pred || 'unavailable', '', '◌') + metric('Drift guard', f.drift_guard, '', '⛨');
    (ctx.history || []).slice(0, 5).forEach(h => { const msg = `${h.ref}: ${h.title}`; if (!stream.some(e => e.msg === msg)) addStream('Git history', msg, '⌘'); });
  }

  function activateTab(btn) {
    if (!btn) return;
    document.querySelectorAll('.tab').forEach(b => { b.classList.toggle('active', b === btn); b.setAttribute('aria-selected', b === btn ? 'true' : 'false'); });
    document.querySelectorAll('.panel').forEach(p => p.classList.toggle('active', p.id === btn.dataset.tab));
  }
  function activateTabByDelta(delta) {
    const tabs = [...document.querySelectorAll('.tab')];
    if (!tabs.length) return;
    const current = Math.max(0, tabs.findIndex(t => t.classList.contains('active')));
    activateTab(tabs[(current + delta + tabs.length) % tabs.length]);
  }
  function bindMobileSwipe() {
    const inspector = document.querySelector('.inspector');
    if (!inspector) return;
    let startX = 0, startY = 0;
    inspector.addEventListener('touchstart', ev => { const t = ev.changedTouches[0]; startX = t.clientX; startY = t.clientY; }, { passive: true });
    inspector.addEventListener('touchend', ev => {
      const t = ev.changedTouches[0], dx = t.clientX - startX, dy = t.clientY - startY;
      if (Math.abs(dx) > 56 && Math.abs(dx) > Math.abs(dy) * 1.35) { activateTabByDelta(dx < 0 ? 1 : -1); addStream('Mobile gesture', 'Swipe changed inspector tab', '⇄'); }
    }, { passive: true });
  }
  function tone(id, good, bad = false) { const el = $(id); if (el) el.className = bad ? 'bad' : good ? 'ok' : 'warn'; }
  function render(st) {
    const mode = st.mode === 'control' ? 'Control enabled' : 'Read-only';
    text('mode', mode); text('modeCard', mode); text('address', st.url); text('pageUrl', st.url); text('pageTitle', st.title); text('viewportSize', `${st.width || '?'} × ${st.height || '?'}`); text('expires', new Date(st.expires_at).toLocaleString()); text('views', st.views); text('chromeMeta', st.title || 'FPV Live');
    if (st.transport?.stream_url) { streamUrl = st.transport.stream_url; streamMode = st.transport.primary || streamMode; } const d = st.diagnostics || {}, errTotal = (d.console_errors || 0) + (d.failed_requests || 0) + (d.http_4xx || 0) + (d.http_5xx || 0);
    renderGlyph('healthGrid', Math.min(8, errTotal || 1), (d.http_5xx || 0) > 0);
    text('consoleErrors', d.console_errors || 0); text('failedRequests', d.failed_requests || 0); text('httpErrors', `${d.http_4xx || 0} / ${d.http_5xx || 0}`); text('requests', d.requests || 0);
    tone('consoleErrors', (d.console_errors || 0) === 0); tone('failedRequests', (d.failed_requests || 0) === 0); tone('httpErrors', (d.http_4xx || 0) + (d.http_5xx || 0) === 0, (d.http_5xx || 0) > 0);
    renderContext(st.context || {});
    const audit = st.audit || [], last = audit[audit.length - 1];
    if (last) { const msg = `${last.action}: ${last.message || last.selector || last.key || last.error || ''}`; if (!stream[0] || stream[0].msg !== msg) addStream('Audit event', msg, '✓'); }
  }
  function startMJPEG() {
    streamMode = 'mjpeg';
    clearInterval(frameTimer);
    const img = $('shot');
    img.onload = () => {
      const now = Date.now();
      frames++;
      if (lastFrameAt) text('latencyMs', `${now - lastFrameAt} ms`);
      lastFrameAt = now;
      const fps = (frames / ((now - started) / 1000)).toFixed(1);
      text('fps', `${fps} FPS`); text('fpsNum', fps); text('transportMode', streamMode === 'cdp_screencast' ? 'CDP screencast' : 'MJPEG stream'); text('streamQuality', Number(fps) > 2 ? 'Excellent' : 'Healthy');
      renderSeismo(); renderGlyph('glyphGrid', Math.max(1, Math.min(8, Math.round(Number(fps) || 1)))); sweepCards();
      if (frames % 16 === 0) addStream('Frame stream', 'MJPEG frame received', '↻');
    };
    img.onerror = () => { frameErrors++; text('transportMode', 'Polling fallback'); addStream('Frame stream', 'MJPEG unavailable; using polling fallback', '⚠'); startPolling(); };
    img.src = `${streamUrl}?t=${Date.now()}`;
  }
  function startPolling() {
    streamMode = 'polling';
    clearInterval(frameTimer);
    frameTimer = setInterval(frameTick, frameMs);
    frameTick();
  }
  async function statusTick() {
    if (!active) return;
    try {
      const r = await fetch(`/m/${token}/status`, { cache: 'no-store' }); const st = await r.json();
      if (!r.ok || st.error) { active = false; text('mode', 'Session ended'); text('fps', 'Stopped'); text('streamQuality', 'Stopped'); text('transportMode', 'Session ended'); $('shot').removeAttribute('src'); addStream('Session ended', st.error || 'share unavailable', '⏱'); return; }
      render(st);
    } catch (_) { text('mode', 'Offline'); addStream('Status', 'metadata retrying', '⚠'); }
  }
  function frameTick() {
    if (!active) return;
    const img = new Image();
    img.onload = () => {
      frameErrors = 0; frames++; $('shot').style.opacity = .985; $('shot').src = img.src; requestAnimationFrame(() => $('shot').style.opacity = 1);
      const fps = (frames / ((Date.now() - started) / 1000)).toFixed(1);
      text('fps', `${fps} FPS`); text('fpsNum', fps); text('streamQuality', Number(fps) > 2 ? 'Excellent' : 'Healthy');
      renderSeismo(); renderGlyph('glyphGrid', Math.max(1, Math.min(8, Math.round(Number(fps) || 1)))); sweepCards();
      if (frames % 16 === 0) addStream('Frame refresh', 'Browser frame received', '↻');
    };
    img.onerror = () => { frameErrors++; text('fps', 'Retrying'); text('streamQuality', 'Retrying'); if (frameErrors > 8) { active = false; text('fps', 'Stopped'); text('streamQuality', 'Stopped'); addStream('Frame stream', 'Stopped after repeated frame errors', '⚠'); } };
    img.src = `/m/${token}/screenshot.jpg?t=${Date.now()}`;
  }
  function bind() {
    document.querySelectorAll('.tab').forEach(btn => btn.addEventListener('click', () => activateTab(btn))); bindMobileSwipe();
    document.querySelectorAll('.filters button').forEach(btn => btn.addEventListener('click', () => { eventFilter = btn.dataset.filter; document.querySelectorAll('.filters button').forEach(b => b.classList.toggle('active', b === btn)); renderStream(); }));
    document.querySelectorAll('.quality-switch button').forEach(btn => btn.addEventListener('click', () => setQuality(btn.dataset.quality)));
    $('noteBtn')?.addEventListener('click', sendMsg); $('clickBtn')?.addEventListener('click', sendClick); $('fillBtn')?.addEventListener('click', sendFill); $('pickBtn')?.addEventListener('click', () => { picking=true; $('overlay').classList.add('picking'); text('controlResult','Click the browser view to pick a target'); }); $('clickPointBtn')?.addEventListener('click', () => sendPoint('click_xy')); $('clearPickBtn')?.addEventListener('click', () => { picked=null; $('overlay').innerHTML=''; $('selector').value=''; }); $('overlay')?.addEventListener('click', (ev)=>{ if(!picking) return; const r=$('overlay').getBoundingClientRect(); picked={x:Math.round(ev.clientX-r.left), y:Math.round(ev.clientY-r.top)}; $('overlay').innerHTML=`<span class="target-dot" style="left:${picked.x}px;top:${picked.y}px"></span>`; picking=false; $('overlay').classList.remove('picking'); sendPoint('selector_at'); });
    document.querySelectorAll('[data-key]').forEach(btn => btn.addEventListener('click', () => sendKey(btn.dataset.key)));
  }
  bind(); initIcons(); addStream('Viewer connected', 'FPV cockpit online', '◉'); statusTick(); startMJPEG(); setInterval(statusTick, 1500);
})();
