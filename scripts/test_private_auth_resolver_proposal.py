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
                        gates.index('external_durable_receipt_then_local_fenced_resume'))
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

    def test_decision_packet_requires_bounded_choices_and_admission_fields(self):
        packet = self.data['decision_packet']
        self.assertEqual(packet['status'], 'advisory_external')
        self.assertEqual(packet['runtime_label'], 'NO auth runtime')
        self.assertFalse(packet['implementation_admission'])
        self.assertFalse(packet['root_promotion'])
        self.assertFalse(packet['distributed_atomicity_claim'])
        self.assertEqual(packet['lineage'], ['WPUIAI/uiai-engine#291',
            'WPUIAI/uiai-engine#145', 'Startempire-Wire/focusa#299',
            'Startempire-Wire/focusa#294'])
        required = {'rooted_decision_provenance','scope_forbidden_effects',
            'human_component_owners','pinned_repos_revisions_files_symbols',
            'api_schemas_errors_states','issuer_trust_peer_mapping',
            'request_account_origin_context_bindings','time_revocation_crash_rules',
            'capture_egress_owner_acknowledgments','golden_executable_fixtures',
            'capacity_retention_cleanup_rollback_reconcile',
            'dependency_graph_and_IR5_receipts'}
        self.assertEqual(set(packet['required_IR5_fields']), required)
        self.assertEqual(len(packet['unknown_mandatory']), 6)
        for row, choice in zip(self.data['decisions'], ['A','A','B']):
            self.assertEqual([a['id'] for a in row['alternatives']], ['A','B','C'])
            self.assertEqual(row['recommended_alternative'], choice)
            self.assertEqual(row['promotion_owner'], 'Sir V3')
            self.assertIsNone(row['promotion_evidence'])
            self.assertTrue(row['source_seams'])
            self.assertGreater(len(row['security_tradeoffs']), 40)
            self.assertGreater(len(row['admission_prerequisites']), 40)
        self.assertIn('NOT distributed', self.doc)
        self.assertNotIn('authorized result atomically with challenge resume', self.doc)

    def test_ipc_candidate_explicit_limits_and_negative_case_inventory(self):
        ipc = self.data['ipc_candidate']
        self.assertEqual(ipc['status'], 'proposed_not_admitted')
        self.assertEqual((ipc['frame_max_bytes'], ipc['ref_max_bytes'],
                          ipc['outstanding_per_peer'], ipc['active_per_context'],
                          ipc['read_deadline_seconds'], ipc['dispatch_deadline_seconds']),
                         (16384, 256, 4, 1, 5, 30))
        self.assertEqual(ipc['status_cancel_body'], ['resolution_ref','lease_generation'])
        self.assertEqual(set(ipc['reject_cases']), {'unknown_field','duplicate_key',
            'invalid_utf8','trailing_data','unsupported_version','oversize_frame',
            'malformed_ref','unknown_principal','shared_uid_ambiguous','pid_reuse',
            'stale_launch_generation','foreign_status_cancel','resource_exhaustion'})
        self.assertEqual(ipc['timeout_after_possible_use'], 'reconcile_never_release_budget')
        self.assertIn('UID alone', self.doc)
        self.assertIn('not existing service limits', self.doc)

    def test_distributed_crash_windows_are_enumerated_not_executed(self):
        windows = self.data['distributed_windows']
        self.assertEqual([w['id'] for w in windows], ['W%d'%i for i in range(1,8)])
        self.assertEqual([w['window'] for w in windows], [
            'external_reserve_reply_or_local_intent_lost','intent_durable_delivery_unknown',
            'delivery_input_submit_or_response_lost','account_observed_consume_pending_ack_lost',
            'consume_durable_resume_not_durable','resume_durable_reply_event_lost',
            'revoke_expire_cancel_close_restart'])
        self.assertTrue(all(w['status'] == 'required_not_executed' for w in windows))
        self.assertEqual(windows[2]['required_recovery'], 'quarantine_possible_use_no_resubmit')
        self.assertEqual(windows[4]['required_recovery'],
                         'no_new_input_revalidate_then_local_compare_transition')
        for phrase in ['linearization point', 'max clock skew', 'propagation budget',
                       'no distributed atomicity claim', 'possible_use']:
            self.assertIn(phrase, self.doc)

    def test_capture_egress_owners_remain_unresolved_and_source_backed(self):
        rows = self.data['capture_egress_inventory']
        self.assertEqual([r['id'] for r in rows],
                         ['C1','C2','C3','C4','C5','C6','E1','E2','E3'])
        for row in rows:
            self.assertIsNone(row['human_owner'])
            self.assertFalse(row['fence_verified'])
            self.assertIn('| '+row['id']+' |', self.doc)
            for path in row['source_path'].split(';'):
                if path not in ('not_implemented','not_selected'):
                    self.assertTrue((ROOT/path).is_file(), path)
        by_id = {r['id']: r for r in rows}
        self.assertEqual(by_id['C2']['unresolved_contract'], 'producer_subscriber_buffer_drain')
        self.assertEqual(by_id['C3']['unresolved_contract'], 'async_recorder_stop_drain_late_callback')
        self.assertEqual(by_id['E2']['unresolved_contract'],
                         'hostile_same_origin_input_listeners_and_allowed_destination_exfiltration')
        for phrase in ['same-origin malicious script','fetch/XHR/beacon/WebSocket',
                       'service-worker','complete audit','UNKNOWN']:
            self.assertIn(phrase, self.doc)

    def test_public_resolve_race_enumeration_requires_all_interleavings(self):
        rows = self.data['public_resolve_cases']
        self.assertEqual([r['id'] for r in rows], ['R%d'%i for i in range(1,9)])
        self.assertEqual([r['interleaving'] for r in rows], [
            'resolve_before_reservation','resolve_between_reserve_and_quarantine',
            'resolve_during_retrieval_input_submit','resolve_after_observation_consume_pending',
            'duplicate_resolve_after_private_resume_nil_challenge',
            'resolve_vs_cancel_close_expiry_navigation_replacement_restart',
            'resolve_vs_inflight_GET_read_stream_share_queued_action',
            'foreign_principal_legacy_token_or_private_ref'])
        self.assertTrue(all(r['status'] == 'required_not_executed' for r in rows))
        for row in rows:
            self.assertIn('- '+row['id']+':', self.doc)
        self.assertEqual(rows[2]['required_assertion'],
                         'no_ordinary_mutation_capture_release_or_second_consumer')
        self.assertEqual(rows[6]['required_assertion'],
                         'no_buffered_late_capture_or_guard_TOCTOU_escape')
        self.assertIn('authChallenge != nil', self.doc)
        self.assertIn('R1-R8', self.data['runtime_cases'][8]['required_assertion'])

    def test_distribution_metadata_scope_is_not_release_claim(self):
        scope = self.data['scope']
        self.assertEqual(scope['manifest_allowed_fields'], ['source_commit','generated_at'])
        self.assertEqual(scope['manifest_tool'], 'scripts/stamp-uiai-version.py')
        self.assertFalse(scope['release_artifact_version_change'])
        self.assertEqual(set(scope['allowed_changed_files']), {
            str(DOC.relative_to(ROOT)), str(CONTRACT.relative_to(ROOT)),
            'scripts/test_private_auth_resolver_proposal.py','docs/distribution-manifest.json'})
        self.assertIn('supported distribution manifest stamp metadata', self.doc)
        self.assertIn('NOT installed compatibility or deployment proof', self.doc)

if __name__ == '__main__':
    unittest.main(verbosity=2)
