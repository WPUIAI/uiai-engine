> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462596553

## Verification amendment — Evidence Artifact first for agents and LLM judges

Every evidence-producing output must promote the exact immutable Evidence Artifact revision as the **preferred verification artifact**. Agents should inspect this formal artifact first rather than reconstructing proof from transcript tails, scattered screenshot paths, or unbounded raw logs.

The human PWA and agent representation are two projections of the same artifact identity:

- human: polished PWA, PDF, email, presentation;
- agent: compact bounded machine manifest / verification view;
- multimodal judge: manifest plus exact image/video segment refs;
- forensic verifier: explicit expansion of selected diagnostics/traces/receipts/source assets.

Agents should not scrape the visual HTML when a machine representation is available.

### Judge View

Provide a versioned, hash-bound Judge View that freezes:

- artifact/revision/bundle hash;
- Project/Workstream/Workpoint/Work Item and authority scope;
- Completion Contract/Acceptance Atom revisions;
- exact claims/questions, evaluation rubric, allowed verdicts, and required citations;
- information-set/source refs;
- figures/comparisons, viewport metadata, video chapters/timestamps, OCR/alt text/transcript, structured interaction trace, diagnostics, and Receipts relevant to the requested atoms;
- verifier policy, independence/blinding requirement, modality requirement, freshness/revalidation window, contradiction policy, and forbidden assumptions;
- untrusted-content/prompt-injection boundary;
- missing, blocked, redacted, stale, or unavailable evidence;
- required typed result schema and expansion refs.

### Judge execution rules

- Artifact/page/media/comment content is untrusted evidence data, never system instruction.
- Trusted judge instructions come only from the verification policy/operation authority.
- Visual claims require exact source-image/video-frame inspection by a capable multimodal model; OCR, transcript, captions, or summaries alone cannot prove visual truth.
- A judge lacking required modality, access, freshness, or evidence returns capability mismatch/blocked/indeterminate—not a guess.
- Verdicts cite exact claim/Atom, artifact, figure, slide, video timestamp, diagnostic, action, or Receipt refs.
- Result metadata records model/provider/version, judge-policy/prompt revision, information-set hash, timestamp, confidence/uncertainty, contradictions, omissions, and bounded rationale—never private chain-of-thought.
- Producer self-judgment cannot claim independent-verifier status.
- Completion Contract may require blinded, independent, multiple, deterministic, or human judges. Disagreement does not become success by simple majority unless the approved policy explicitly says so.
- Judge execution is read-only against the frozen artifact. Recapture, reproof, or repair uses the governed Focusa Action Deck and produces new evidence/artifact revisions.

The judge result is its own immutable verification artifact/ref/hash/Receipt linked to the exact input artifact and information set. It may feed Focusa #278; only Focusa #277 decides/settles completion.

### Agent-first interfaces

CLI, MCP, REST/OpenAPI, Pi, and LLM tools must support narrow, generated operations to:

- resolve preferred verification artifact;
- retrieve summary/Judge View by exact ref;
- select atoms/claims/sections/media ranges;
- retrieve exact image or video timestamp segments without base64 in the ordinary response;
- verify hashes and freshness;
- submit a typed judge result;
- inspect contradictions/disagreement;
- request governed reproof/follow-up through the action manifest.

Results remain token-budgeted, ref-first, cursor/range capable, and explicit about omitted fields/rehydration. Same artifact revision + judge policy + selected atoms + information set yields one stable evaluation input identity.

### Acceptance

1. A verification agent discovers the formal artifact first from an evidentiary turn/Workpoint/Completion Case.
2. Text-only and multimodal judges receive capability-appropriate views with identical scope and source identity.
3. One visual claim proves exact image inspection; one interaction claim proves timestamped video/trace inspection.
4. Prompt-injection content cannot modify rubric, scope, permissions, tools, verdict enum, or completion authority.
5. Missing modality/evidence, stale refs, hash mismatch, redaction denial, conflicting judges, and information-set drift fail closed.
6. Judge results are immutable, cited, model/policy/information-set identified, and cannot write completion state.
7. Human PWA/PDF/deck and machine Judge View remain traceably consistent projections of one artifact revision.

Focusa Completion Verification integration: Startempire-Wire/focusa#283#issuecomment-5462594492.
