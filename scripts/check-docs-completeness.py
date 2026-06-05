#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
checks={
 'search': {
   'README.md':['/api/search','uiai_search','browser_search','scripts/uiai-open-result.sh'],
   'docs/SESSION_API.md':['/api/search','uiai-search:<provider>:<query-hash>:<rank>'],
   'docs/PUBLIC_API_PARITY_MATRIX.md':['/api/search','uiai_search','browser_search','uiai-search:<provider>:<query-hash>:<rank>','scripts/uiai-open-result.sh'],
   'docs/ENDPOINT_AUTH_MATRIX.md':['/api/search','loopback-public remote-auth'],
   'scripts/smoke-agent-integrations.sh':['/api/search','browser_search'],
   'scripts/smoke-open-result.sh':['uiai-open-result.sh','open result smoke ok'],
 },
 'browser_session': {
   'README.md':['browser sessions/actions/diagnostics'],
   'docs/SESSION_API.md':['/api/session','uiai-browser:session=','browser_diagnostics'],
   'docs/AGENT_UX_COOKBOOK.md':['uiai_browser_open','uiai_browser_read','uiai_browser_diagnostics'],
   'docs/PUBLIC_API_PARITY_MATRIX.md':['/api/session','Full agent parity for core browser workflows'],
   'docs/ENDPOINT_AUTH_MATRIX.md':['/api/session','loopback-public remote-auth'],
 },
 'diagnostics_errors': {
   'README.md':['/api/errors','uiai_errors','browser_diagnostics'],
   'docs/SESSION_API.md':['/api/errors','uiai-diagnostics:','uiai-error:'],
   'docs/AGENT_UX_COOKBOOK.md':['Diagnostics-first debugging','uiai_errors'],
   'docs/PUBLIC_API_PARITY_MATRIX.md':['/api/errors','uiai_errors'],
   'docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md':['Failed network diagnostics','Focusa evidence'],
   'scripts/smoke-render-diagnostics-artifact.sh':['diagnostics artifact render smoke ok'],
   '.pi/skills/uiai-browser-debug/SKILL.md':['scripts/render-diagnostics-artifact.py'],
 },
 'focusa_packet': {
   'README.md':['uiai.focusa_research_diagnostics_packet.v1','uiai_focusa_packet_build','/api/agent/research-packet'],
   'docs/SESSION_API.md':['Packet parity status','/api/agent/research-packet','uiai.focusa_research_diagnostics_packet.v1'],
   'docs/UIAI_FOR_AGENTS_QUICKSTART.md':['uiai_focusa_packet_compose','scripts/uiai research packet'],
   'docs/PUBLIC_API_PARITY_MATRIX.md':['/api/agent/research-packet','scripts/smoke-focusa-packet-ci.sh'],
   'docs/ENDPOINT_AUTH_MATRIX.md':['/api/agent/research-packet','loopback-public remote-auth'],
   'docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md':['packet endpoint smoke','focusa-packet-smoke.json'],
   'scripts/check-focusa-packet-drift.sh':['scripts/smoke-focusa-packet-ci.sh'],
   '.github/workflows/browser-reliability.yml':['Focusa packet endpoint smoke','scripts/smoke-focusa-packet-ci.sh'],
 },
 'cli_pi_mcp': {
   'docs/SESSION_API.md':['scripts/uiai','research packet','session diagnostics'],
   'docs/UIAI_FOR_AGENTS_QUICKSTART.md':['/uiai research <query>','/uiai proof <url>','/uiai diagnose <session_id>'],
   'docs/PUBLIC_API_PARITY_MATRIX.md':['scripts/uiai research packet','scripts/check-tool-parity.sh'],
   'scripts/check-tool-parity.sh':['research packet','/uiai research <query>'],
 },
 'repo_skills': {
   'README.md':['.pi/skills/uiai-agent/SKILL.md','.pi/skills/uiai-focusa-packet/SKILL.md','.pi/skills/uiai-release/SKILL.md','.pi/skills/uiai-mcp/SKILL.md','.pi/skills/uiai-remote-auth/SKILL.md','.pi/skills/uiai-docs-maintenance/SKILL.md','.pi/skills/uiai-ci-debug/SKILL.md','.pi/skills/uiai-browser-debug/SKILL.md'],
   'docs/UIAI_FOR_AGENTS_QUICKSTART.md':['/skill:uiai-agent','/skill:uiai-focusa-packet','/skill:uiai-release','/skill:uiai-mcp','/skill:uiai-remote-auth','/skill:uiai-docs-maintenance','/skill:uiai-ci-debug','/skill:uiai-browser-debug','scripts/install-pi-skills.sh --apply'],
   'scripts/smoke-pi-skills.sh':['uiai-docs-maintenance','uiai-remote-auth','pi skill smoke ok'],
 },
 'release_ci': {
   'docs/AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md':['scripts/check-tool-parity.sh','scripts/release-service-smoke.sh --check-only','scripts/ci-log-summary.py'],
   'docs/CI_FAILURE_DIAGNOSTICS_GUIDE.md':['scripts/ci-log-summary.py','scripts/release-service-smoke.sh --check-only'],
   'docs/RELEASE_DEPLOY_RUNBOOK.md':['scripts/ci-log-summary.py','scripts/release-service-smoke.sh --check-only'],
   '.pi/skills/uiai-release/SKILL.md':['scripts/release-service-smoke.sh --check-only'],
   '.pi/skills/uiai-ci-debug/SKILL.md':['scripts/ci-log-summary.py'],
 },
}
missing=[]
for surface,file_checks in checks.items():
    for rel, needles in file_checks.items():
        p=ROOT/rel
        if not p.exists():
            missing.append({'surface':surface,'file':rel,'needle':'<file missing>'}); continue
        text=p.read_text(errors='replace')
        for needle in needles:
            if needle not in text:
                missing.append({'surface':surface,'file':rel,'needle':needle})
report={'ok':not missing,'surfaces':len(checks),'checked_files':len({f for v in checks.values() for f in v}),'missing':missing}
print(json.dumps(report, indent=2))
if missing:
    raise SystemExit('docs completeness check failed')
