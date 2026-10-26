package routes

const browserModePrivateAuth = "private_auth_transaction"

// browserModeDecisionCard owns discovery guidance only, never execution grants.
// Registered tool names are checked against the existing tool registry; external
// systems and unimplemented adapters cannot become ready through this card.
func browserModeDecisionCard(tools []map[string]any) map[string]any {
	registered := map[string]bool{}
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok {
			registered[name] = true
		}
	}
	modes := []map[string]any{
		{"id": "source_read", "status": "registered_tools", "tools": []string{"source_to_markdown", "browser_read"}, "cost": "low_text_output_browser_work_may_still_be_required", "persistence": "one_shot_or_existing_owned_session", "security": "public_sources_or_explicitly_authorized_session", "best_use": "Read documents without unnecessary screenshots"},
		{"id": "headless_session", "status": "registered_tools", "tools": []string{"browser_open", "browser_close"}, "cost": "browser_session", "persistence": "session_idle_lifetime_not_persistent_desktop", "security": "isolated_context_does_not_grant_provider_auth", "best_use": "Keep one owned browser page across interactions"},
		{"id": "semantic_control", "status": "registered_tools", "tools": []string{"browser_snapshot", "browser_click", "browser_fill", "browser_press"}, "cost": "bounded_observations_and_actions", "persistence": "owned_session", "security": "fresh_refs_and_existing_action_authority_required", "best_use": "Reliable DOM interactions using snapshot refs"},
		{"id": "diagnostics", "status": "registered_tools", "tools": []string{"browser_diagnostics"}, "cost": "low_bounded_text", "persistence": "owned_session_diagnostics", "security": "page_and_network_content_remains_untrusted", "best_use": "Unexpected page, failed request, console exception or blank view"},
		{"id": "visual_proof", "status": "registered_tools", "tools": []string{"browser_screenshot"}, "cost": "image_and_browser_work", "persistence": "artifact_retention_and_delivery_policy", "security": "inspect_pixels_and_verify_actual_delivery_separately", "best_use": "Verify rendered appearance, not just DOM or HTTP success"},
		{"id": "fpv_observation", "status": "registered_tools", "tools": []string{"browser_fpv_share"}, "cost": "live_browser_and_view_stream", "persistence": "bounded_share_and_session_lifetime", "security": "view_token_is_not_control_or_credential_authority", "best_use": "Authorized human observation; takeover requires separate control contract"},
		{"id": "agent_computer", "status": "external_system", "tools": []string{}, "cost": "external_runtime", "persistence": "external_owner_policy", "security": "verify_private_runtime_identity_and_existing_grants", "best_use": "Authorized persistent desktop through its owning system, not this headless API"},
		{"id": "external_cdp_attach", "status": "not_first_class", "tools": []string{}, "cost": "not_admitted", "persistence": "external_browser_ownership", "security": "CDP_or_SSH_transport_never_grants_credential_or_control_use", "best_use": "Existing external Chromium; no first-class attach tool is advertised"},
		{"id": "tailnet_mesh", "status": "not_first_class", "tools": []string{}, "cost": "not_admitted", "persistence": "external_node_policy", "security": "node_access_is_not_browser_or_credential_authority", "best_use": "Remote node routing requires an existing approved adapter"},
		{"id": browserModePrivateAuth, "status": "not_first_class", "tools": []string{}, "cost": "authorization_and_private_runtime_required", "persistence": "bounded_private_context_and_explicit_cleanup", "security": "renewable_only_never_public_demo_never_raw_values_in_model", "best_use": "Approved broker or human authentication; not an implemented generic transaction tool"},
	}
	for _, mode := range modes {
		if mode["status"] == "registered_tools" {
			for _, name := range mode["tools"].([]string) {
				if !registered[name] {
					mode["status"] = "not_registered"
					break
				}
			}
		}
	}
	return map[string]any{
		"schema":             "uiai.browser_modes.v1",
		"availability_basis": "tool_registration_not_runtime_readiness_or_authorization",
		"grants_authority":   false, "performs_upstream_probes": false,
		"modes": modes,
		"state_routing": []map[string]string{
			{"when": "unexpected_or_blank_page", "mode": "diagnostics", "next": "Read bounded diagnostics before changing approach"},
			{"when": "rendered_appearance_required", "mode": "visual_proof", "next": "Capture and inspect pixels; verify delivery independently"},
			{"when": "auth_challenge_pending", "mode": browserModePrivateAuth, "next": "Keep paused; use only approved private renewable broker/human path, then fresh observation and authority"},
			{"when": "popup_or_external_target_not_attached", "mode": "external_cdp_attach", "next": "No first-class attach is advertised; do not invent a tool or lose opener semantics"},
			{"when": "credential_form_in_public_or_shared_profile", "mode": browserModePrivateAuth, "next": "Block credential injection; require approved private context"},
		},
		"recipes": []map[string]any{
			{"id": "read_act_debug_proof", "status": "existing_tools_require_runtime_admission", "steps": []string{"read bounded content", "snapshot current refs", "perform only authorized action", "diagnostics on failure", "capture and verify required evidence", "close owned session"}},
			{"id": "auth_challenge_resume", "status": "requires_external_authorization", "steps": []string{"pause on challenge", "verify private context and exact target", "approved renewable broker or human resolution", "fresh observation and authority before resuming", "record cleanup and unresolved risk"}},
			{"id": "persistent_attach_detach", "status": "not_first_class", "steps": []string{"verify external ownership and attach capability", "block when no approved adapter exists", "detach only owned connection without destroying external browser"}},
			{"id": "secret_injection", "status": "requires_external_authorization", "steps": []string{"exact renewable secret grant and bound private target", "broker injects without values in model or command arguments", "redacted verification and cleanup receipt", "unknown residue is not zero residue"}},
			{"id": "provider_queue_cleanup", "status": "requires_exact_provider_grant", "steps": []string{"verify existing approved renewable provider access", "inspect exact queued heads read-only", "mutate only explicitly authorized build IDs", "retain terminal provider receipts, not dispatch claims"}},
			{"id": "scrub_and_safe_reopen", "status": "full_context_scrub_not_first_class", "steps": []string{"identify exact owned private context and origins", "use only supported authorized teardown", "verify cookies and all required storage/targets separately", "do not reopen a public demo with unknown residue"}},
		},
		"security":                        map[string]any{"nonrenewable_automation": "forbidden", "recovery_code_automation": "forbidden", "unknown_renewability": "blocked", "public_demo_credentials": "forbidden", "raw_secret_values_in_model": "forbidden", "transport_is_authority": false},
		"privileged_receipt_posture":      "requirements_only_not_an_emitted_or_verified_receipt",
		"privileged_receipt_requirements": []string{"actor_ref", "node_ref", "browser_context_ref", "origin_refs", "challenge_class", "action_refs", "result", "cookie_storage_cleanup", "target_cleanup", "residual_risk"},
	}
}
