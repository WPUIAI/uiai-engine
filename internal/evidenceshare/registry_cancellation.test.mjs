import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

// Exercise shipped request/control owners, not a duplicate paging algorithm.
const source = readFileSync(new URL('./assets/app.js', import.meta.url), 'utf8');
function productionFunction(name, optional = false) {
  const match = new RegExp(`^(?:async )?function ${name}\\(`, 'm').exec(source);
  if (!match && optional) return '';
  assert.ok(match, `missing production ${name}`);
  const end = source.indexOf('\n}', match.index);
  assert.ok(end > match.index);
  return source.slice(match.index, end + 2);
}
function fixture() {
  const nodes = new Map(), pending = [], timers = [], snapshots = [];
  const byId = id => {
    if (!nodes.has(id)) nodes.set(id, { handlers: {}, dataset: {}, hidden: false,
      addEventListener(type, callback) { this.handlers[type] = callback; },
      setAttribute(name, value) { this[name] = value; }, querySelectorAll: () => [],
    });
    return nodes.get(id);
  };
  const state = { project: 'project:fixture', query: 'old', status: '', type: '',
    requestID: 0, resourceProfile: 'normal', artifactCursor: '', workItemCursor: '',
    artifactPageCursor: '', workItemPageCursor: '', cursorHistory: [], artifacts: [], workItems: [],
  };
  const context = vm.createContext({
    registryState: state, byId, AbortController, URL, URLSearchParams,
    document: { addEventListener() {} }, addEventListener() {},
    registryTableWrap: () => ({ addEventListener() {}, scrollTop: 0, clientHeight: 100 }),
    registryAPI: 'https://fixture.invalid/registry', registryPageSize: 25,
    registryHistoryLimit: 32, registryRowHeight: 56,
    queryString: fields => new URLSearchParams(fields).toString(),
    fetchJSON: url => new Promise((resolve, reject) => pending.push({ url, resolve, reject })),
    tr: key => key, setStatus() {}, setReadyStatus() {}, renderRegistrySummary() {}, updateRegistryURL() {}, showRegistryUnavailable() {},
    registryRecordCount: () => state.artifacts.length + state.workItems.length,
    saveRegistrySnapshot: () => snapshots.push({ query: state.query, rows: state.artifacts.slice() }),
    location: { href: 'https://fixture.invalid/evidence/' },
    setTimeout: callback => { timers.push(callback); return timers.length; }, clearTimeout() {},
  });
  vm.runInContext(productionFunction('invalidateRegistryRequest', true) + '\n' +
    productionFunction('loadRegistry') + '\n' + productionFunction('wireRegistryControls') +
    '\nwireRegistryControls();', context);
  return { context, state, pending, timers, snapshots, byId };
}

test('query input aborts the prior request immediately, before debounce', () => {
  const f = fixture();
  const controller = new AbortController();
  f.state.abortController = controller;
  f.byId('registry-query').handlers.input({ target: { value: ' new ' } });
  assert.equal(controller.signal.aborted, true);
  assert.equal(f.state.query, 'new');
  assert.equal(f.timers.length, 1, 'search remains debounced');
});

test('clearing project invalidates an in-flight request even without a next load', async () => {
  const f = fixture();
  const controller = new AbortController();
  f.state.abortController = controller;
  f.byId('registry-project').handlers.change({ target: { value: '' } });
  assert.equal(controller.signal.aborted, true);
  assert.equal(f.pending.length, 0);
});

test('a completed old response cannot be cached under a newly typed query', async () => {
  const f = fixture();
  const oldLoad = vm.runInContext('loadRegistry()', f.context);
  assert.equal(f.pending.length, 3);
  f.byId('registry-query').handlers.input({ target: { value: 'new' } });
  // Deliberately ignore abort: delivery may already have completed upstream.
  for (const request of f.pending) request.resolve(request.url.includes('/artifacts?')
    ? { rows: [{ artifact_ref: 'artifact:old-query' }] }
    : request.url.includes('/work-items?') ? { work_items: [] } : { freshness: 'live' });
  await oldLoad;
  assert.equal(f.state.artifacts.length, 0, 'superseded rows never commit');
  assert.equal(f.snapshots.length, 0, 'old rows never acquire the new query identity');
});

test('a superseded non-abort network error cannot replace the new view', async () => {
  const f = fixture();
  const oldLoad = vm.runInContext('loadRegistry()', f.context);
  f.byId('registry-query').handlers.input({ target: { value: 'new' } });
  for (const request of f.pending) request.reject(new Error('old request unavailable'));
  await assert.doesNotReject(oldLoad);
});

test('a current request failure still rejects rather than becoming empty success', async () => {
  const f = fixture();
  const load = vm.runInContext('loadRegistry()', f.context);
  for (const request of f.pending) request.reject(new Error('current request unavailable'));
  await assert.rejects(load, /current request unavailable/);
  assert.equal(f.snapshots.length, 0);
});

test('a current successful response retains its matching cache identity', async () => {
  const f = fixture();
  const load = vm.runInContext('loadRegistry()', f.context);
  for (const request of f.pending) request.resolve(request.url.includes('/artifacts?')
    ? { rows: [{ artifact_ref: 'artifact:matching-query' }] }
    : request.url.includes('/work-items?') ? { work_items: [] } : { freshness: 'live' });
  await load;
  assert.equal(f.state.artifacts.length, 1);
  assert.equal(f.snapshots.length, 1);
  assert.equal(f.snapshots[0].query, 'old');
  assert.equal(f.snapshots[0].rows[0].artifact_ref, 'artifact:matching-query');
});
