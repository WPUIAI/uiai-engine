export const AGENT_FIRST_BROWSER_SCHEMA_IDS = [
  "uiai.agent_result.v1",
  "uiai.agent_client_capability_profile.v1",
  "uiai.browser_observation.v1",
  "uiai.browser_action_request.v2",
  "uiai.focusa_browser_verification_request.v1",
  "uiai.focusa_browser_verification_result.v1",
  "uiai.browser_content_provenance.v1",
  "uiai.browser_action_influence_manifest.v1",
  "uiai.browser_execution_capsule.v1",
  "uiai.origin_tool_candidate.v1",
] as const;

export type AgentFirstBrowserSchemaId = (typeof AGENT_FIRST_BROWSER_SCHEMA_IDS)[number];

const schemaSet = new Set<string>(AGENT_FIRST_BROWSER_SCHEMA_IDS);

export function isAgentFirstBrowserSchemaId(value: string): value is AgentFirstBrowserSchemaId {
  return schemaSet.has(value);
}

export function assertAgentFirstBrowserSchemaId(value: string): AgentFirstBrowserSchemaId {
  if (!isAgentFirstBrowserSchemaId(value)) throw new Error(`Unsupported agent-first browser schema: ${value}`);
  return value;
}
