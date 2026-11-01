#!/usr/bin/env python3
"""Proposal consistency only: does NOT test any auth runtime or issue a grant."""
import json, pathlib, unittest
ROOT = pathlib.Path(__file__).resolve().parents[1]
CONTRACT = ROOT / 'docs/contracts/PRIVATE_AUTH_RESOLVER_ACCEPTANCE_v1.json'
DOC = ROOT / 'docs/proposals/PRIVATE_AUTH_RESOLVER_ADMISSION_PROPOSAL_v1.md'

class ProposalConsistency(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data = json.loads(CONTRACT.read_text())
        cls.doc = DOC.read_text()

    def test_proposal_cannot_advertise_implementation(self):
        d = self.data
        self.assertEqual(d['status'], 'proposed')
        self.assertEqual(d['authority_class'], 'advisory_external')
        self.assertIs(d['implementation_implied'], False)
        self.assertIs(d['runtime_implemented'], False)
        self.assertEqual(d['interface']['deployment_status'], 'not_deployed')
        self.assertIn('NOT implementation admission', self.doc)

    def test_issuer_is_not_consumer_or_test(self):
        d = self.data['authority']
        self.assertEqual(d['issuer_owner'], 'Focusa')
        self.assertEqual(d['consumer_owner'], 'UIAI Engine')
        self.assertIs(d['self_issue'], False)
        self.assertIs(d['raw_secret_in_model'], False)
        self.assertEqual(d['recovery_material'], 'unconditionally_forbidden')
        self.assertEqual(d['unknown_renewability'], 'reject_before_provider_access')

    def test_unknown_and_secret_inputs_rejected(self):
        d = self.data['interface']
        self.assertEqual(d['unknown_fields'], 'reject')
        forbidden = {'password','otp','cookie','js','selector','url','actor',
                     'resume_token','authenticated','grant_document'}
        self.assertEqual(set(d['begin_forbidden']), forbidden)
        self.assertTrue(set(d['begin_required']).isdisjoint(forbidden))
        self.assertEqual(d['actor_source'], 'authenticated_peer_and_owning_records')
        self.assertEqual(d['expected_account_source'], 'trusted_authority_binding')
        self.assertEqual(d['observed_account_source'], 'fresh_provider_account_verifier')

    def test_exact_owner_decisions_stay_open(self):
        self.assertEqual([d['id'] for d in self.data['decisions']], ['D1','D2','D3'])
        self.assertTrue(all(d['status'] == 'owner_decision_required'
                            for d in self.data['decisions']))
        self.assertIn('Spec 156 explicitly remains IR0', self.doc)

    def test_runtime_acceptance_not_falsely_claimed(self):
        cases = self.data['runtime_cases']
        self.assertEqual([c['id'] for c in cases], ['PA%02d'%i for i in range(1,16)])
        self.assertTrue(all(c['status'] == 'required_not_executed' for c in cases))
        self.assertTrue(all(c['required_assertion'] for c in cases))
        categories = {c['category'] for c in cases}
        self.assertTrue({'positive_real_browser','stale_replay','origin_change',
                         'account_mismatch','grant_rejection','no_secret_leak',
                         'capture_quarantine_race','authority_interoperability'} <= categories)

    def test_capture_and_resume_ordering_gates(self):
        gates = self.data['gates']
        self.assertLess(gates.index('capture_quarantine_before_retrieval'),
                        gates.index('fresh_authenticated_account_observation'))
        self.assertLess(gates.index('fresh_authenticated_account_observation'),
                        gates.index('atomic_recheck_receipt_and_resume'))
        self.assertIn('secret_cleanup_before_capture_release', gates)
        self.assertIn('Ordinary agent fill/eval/click remain denied', self.doc)
        self.assertIn('no broker socket is added by this PR', self.doc)

    def test_release_and_original_issues_stay_open(self):
        d = self.data['release']
        self.assertIs(d['parent_independent_review_required'], True)
        self.assertIs(d['production_deploy_authorized'], False)
        self.assertEqual(d['closes_issues'], [])
        self.assertEqual(d['website_acceptance'], 'separate_pending')
        self.assertIn('exact_owner_approved', d['implementation_gate'])

if __name__ == '__main__':
    unittest.main(verbosity=2)
