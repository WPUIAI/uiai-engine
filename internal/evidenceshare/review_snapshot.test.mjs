import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

// Execute the shipped rendering function, not a second implementation.
const source = readFileSync(new URL('./assets/app.js', import.meta.url), 'utf8');
const render = source.slice(source.indexOf('async function renderReviewCase()'), source.indexOf('async function submitReviewDecision('));

async function renderFixture({ live, snapshot }) {
  const nodes = new Map();
  const byId = id => {
    if (!nodes.has(id)) nodes.set(id, { dataset: {}, hidden: false, textContent: '', querySelectorAll: () => [] });
    return nodes.get(id);
  };
  const context = vm.createContext({
    byId, activeReviewCase: null, text: (node, value) => { node.textContent = value; },
    tr: key => key, reviewMessage: value => value || '',
    fetchJSON: async ref => {
      if (ref === './review' && live) return live;
      if (ref === './review.json' && snapshot) return snapshot;
      throw new Error('unavailable');
    },
  });
  await vm.runInContext(render + '\nrenderReviewCase()', context);
  return nodes;
}

for (const posture of ['accepted', 'rejected', 'pending']) {
  test(`offline ${posture} review stays stale and read-only`, async () => {
    const nodes = await renderFixture({ snapshot: { schema: 'uiai.review_state.v1', posture, case_ref: 'case:fixture', artifact_ref: 'artifact:fixture' } });
    assert.equal(nodes.get('review-panel').dataset.state, 'stale');
    assert.equal(nodes.get('review-decision-form').hidden, true);
    assert.match(nodes.get('review-posture').textContent, /offline_snapshot/);
    assert.match(nodes.get('review-truth').textContent, /review_offline/);
    assert.doesNotMatch(nodes.get('review-truth').textContent, /review_accepted/);
  });
}

test('live accepted review retains its live posture', async () => {
  const nodes = await renderFixture({ live: { review_case: { schema: 'uiai.review_case.v1', posture: 'accepted', review_requirement_refs: ['requirement:fixture'], reviewer_assignment_ref: 'assignment:fixture', reviewer_ref: 'reviewer:fixture' } } });
  assert.equal(nodes.get('review-panel').dataset.state, 'accepted');
  assert.equal(nodes.get('review-truth').textContent, 'review_accepted');
});

test('missing live and offline review fails closed', async () => {
  const nodes = await renderFixture({});
  assert.equal(nodes.get('review-panel').dataset.state, 'blocked');
  assert.equal(nodes.get('review-decision-form').hidden, true);
});
