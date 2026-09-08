import { readFileSync } from 'node:fs';
import { runInContext, createContext } from 'node:vm';
import { test } from 'node:test';
import assert from 'node:assert/strict';

const catalogue = readFileSync(new URL('./assets/locale.js', import.meta.url), 'utf8');
const renderer = readFileSync(new URL('./assets/generic-record.js', import.meta.url), 'utf8');
function harness(language) {
  const nodes = new Map(), validity = new Map();
  const byId = id => {
    if (!nodes.has(id)) nodes.set(id, { replaceChildren(...children) { this.children = children; }, focus() {} });
    return nodes.get(id);
  };
  const context = createContext({
    window: {}, URL, Intl,
    location: { href: `https://fixture.example/?lang=${language}` }, navigator: { language: 'en' },
    document: { documentElement: {}, body: { dataset: {} }, querySelectorAll: () => [], getElementById: () => null },
    byId, text: (node, value) => { node.textContent = value; },
    fact: (label, value) => ({ label, value }), datum: (label, value) => ({ label, value }),
    setValidity: (key, label, state) => validity.set(key, { label, state }),
    setReadyStatus: (node, label) => { node.textContent = label; },
    renderLineage: scope => { byId('scope').value = scope; }, renderTimeline: () => {}, workItemInspectData: () => [],
    safeRef: ref => ref, validSHA256: value => /^[a-f0-9]{64}$/.test(value || ''),
    formatTime: value => String(value), formatBytes: value => String(value),
  });
  runInContext(catalogue, context);
  runInContext('const locale = window.EvidenceLocale; const tr = (key, values) => locale.t(key, values);', context);
  runInContext(renderer, context);
  return { context, byId, validity, t: context.window.EvidenceLocale.t, render: context.window.renderGenericEvidenceRecord };
}

for (const [language, expected] of [['en', 'Payload SHA-256'], ['es', 'SHA-256 del contenido'], ['ar', 'SHA-256 للمحتوى']]) {
  test(`${language}: portable record uses the shared catalogue and preserves authority boundaries`, async () => {
    const h = harness(language);
    for (const [, key] of renderer.matchAll(/tr\("([^"]+)"/g)) assert.notEqual(h.t(key), key, `Missing translation: ${key}`);
    await h.render({ schema: 'uiai.epwa_generic_artifact.v1', artifact_ref: 'artifact:fixture', revision: 1, asset_sha256: 'a'.repeat(64), scope: { project_ref: 'project:fixture' } });
    assert.equal(h.byId('inspect-grid').children[2].label, expected);
    assert.equal(h.byId('title').textContent, h.t('portable_artifact_record'));
    assert.equal(h.byId('status').textContent, h.t('artifact_loaded'));
    assert.equal(h.validity.get('completion').label, h.t('not_asserted'));
    assert.equal(h.validity.get('completion').state, 'not_determined');
    assert.equal(h.byId('scope').value.project_ref, 'project:fixture');
    assert.equal(h.context.document.documentElement.dir, language === 'ar' ? 'rtl' : 'ltr');
  });
  test(`${language}: immutable record localizes generated labels, not source evidence`, async () => {
    const h = harness(language);
    await h.render({ schema: 'uiai.evidence_artifact_manifest.v1', artifact_id: 'artifact:immutable', title: 'Original source title', summary: 'Original evidence content', scope: {}, assets: [{ asset_id: 'one', media_type: 'image/png', path: 'image.png', sha256: 'a'.repeat(64), width: 12, height: 34 }], claims: [] });
    assert.equal(h.byId('title').textContent, 'Original source title');
    assert.equal(h.byId('truth').textContent, 'Original evidence content');
    assert.equal(h.byId('facts').children[0].label, h.t('captured'));
    assert.equal(h.byId('inspect-grid').children[2].label, h.t('manifest_digest'));
    assert.equal(h.byId('limitations-copy').textContent, h.t('artifact_limitations'));
    assert.equal(h.validity.get('settlement').state, 'not_determined');
    assert.equal(h.byId('capture-label').textContent, `${new Intl.NumberFormat(language).format(12)} × ${new Intl.NumberFormat(language).format(34)} · image/png`);
    await assert.rejects(() => h.render({ schema: 'unsupported' }), { message: h.t('artifact_contract_invalid') });
  });
}
