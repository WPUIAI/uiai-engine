import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import assert from 'node:assert/strict';

const [address, output, name, executable] = process.argv.slice(2);
const url = new URL(address);
assert.equal(url.protocol, 'http:');
assert.equal(url.hostname, '127.0.0.1', 'This runner only accepts local test servers');
assert.match(name, /^(generic|screenshot)-(en|es|ar)$/);
const profile = await mkdtemp(join(tmpdir(), 'epwa-browser-'));
const chrome = spawn(executable, ['--headless=new', '--no-sandbox', '--disable-dev-shm-usage', '--no-first-run', '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=0', '--user-data-dir=' + profile, 'about:blank'], { stdio: ['ignore', 'ignore', 'pipe'] });
let stderr = '', socket;
chrome.stderr.on('data', chunk => { stderr = (stderr + chunk).slice(-4000); });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
try {
  let port;
  for (let attempt = 0; attempt < 100 && !port; attempt++) {
    try { port = Number((await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0]); }
    catch (error) { if (error.code !== 'ENOENT') throw error; await pause(50); }
  }
  assert(port, 'Chromium startup failed: ' + stderr);
  const pages = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
  socket = new WebSocket(pages.find(page => page.type === 'page').webSocketDebuggerUrl);
  await new Promise((resolve, reject) => { socket.addEventListener('open', resolve, { once: true }); socket.addEventListener('error', reject, { once: true }); });
  const pending = new Map(); let sequence = 0;
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data), request = pending.get(message.id);
    if (!request) return;
    pending.delete(message.id); clearTimeout(request.timer);
    message.error ? request.reject(new Error(JSON.stringify(message.error))) : request.resolve(message.result);
  });
  const call = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++sequence, timer = setTimeout(() => { pending.delete(id); reject(new Error('CDP timeout: ' + method)); }, 10000);
    pending.set(id, { resolve, reject, timer }); socket.send(JSON.stringify({ id, method, params }));
  });
  const evaluate = async expression => {
    const response = await call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    assert(!response.exceptionDetails, JSON.stringify(response.exceptionDetails));
    return response.result.value;
  };
  await call('Page.enable');
  await call('Page.addScriptToEvaluateOnNewDocument', { source: "window.__proofErrors=[];addEventListener('error',e=>__proofErrors.push(e.message));addEventListener('unhandledrejection',e=>__proofErrors.push(String(e.reason)));" });
  const metrics = [];
  for (const [width, height] of [[1280, 900], [390, 844]]) {
    await call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false });
    await call('Page.navigate', { url: address });
    let ready = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      if (await evaluate("Boolean(document.getElementById('status')?.dataset.readyLabel)")) { ready = true; break; }
      await pause(50);
    }
    assert(ready, 'Record did not reach ready state');
    await evaluate('document.fonts.ready');
    const measured = await evaluate('({width:innerWidth,scrollWidth:document.documentElement.scrollWidth,lang:document.documentElement.lang,dir:document.documentElement.dir,errors:window.__proofErrors})');
    const screenshot = await call('Page.captureScreenshot', { format: 'png' });
    await writeFile(join(output, `${name}-${width}.png`), Buffer.from(screenshot.data, 'base64'));
    await writeFile(join(output, `${name}-${width}.html`), await evaluate('document.documentElement.outerHTML'));
    assert.equal(measured.width, width);
    assert(measured.scrollWidth <= width + 1, `Horizontal overflow: ${JSON.stringify(measured)}`);
    assert.equal(measured.lang, url.searchParams.get('lang'));
    assert.equal(measured.dir, measured.lang === 'ar' ? 'rtl' : 'ltr');
    assert.deepEqual(measured.errors, []);
    metrics.push(measured);
  }
  await writeFile(join(output, name + '.json'), JSON.stringify({ real_package_routes: true, metrics }, null, 2));
} finally {
  socket?.close();
  if (chrome.exitCode === null && chrome.signalCode === null) {
    const exited = once(chrome, 'exit'); chrome.kill('SIGTERM');
    await Promise.race([exited, pause(2000)]);
    if (chrome.exitCode === null && chrome.signalCode === null) { chrome.kill('SIGKILL'); await exited; }
  }
  await rm(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
}
