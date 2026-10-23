export const COCKPIT_WORKPOINT_RESUME_SCHEMA = "uiai.cockpit_workpoint_resume.v1" as const;

export interface ResumeTarget {
  workspace_id: string;
  href: string;
  object_ref?: string;
}

export interface ResumeRecovery {
  message: string;
  href: string;
  action_label: string;
}

export type CockpitWorkpointResume =
  | {
      schema: typeof COCKPIT_WORKPOINT_RESUME_SCHEMA;
      source: "focusa_workpoint_resume";
      canonical: true;
      status: "resumable";
      workpoint_id: string;
      label: string;
      target: ResumeTarget;
      observed_at: string;
    }
  | {
      schema: typeof COCKPIT_WORKPOINT_RESUME_SCHEMA;
      source: "focusa_workpoint_resume";
      canonical: true;
      status: "blocked";
      recovery: ResumeRecovery;
      observed_at: string;
    }
  | {
      schema: typeof COCKPIT_WORKPOINT_RESUME_SCHEMA;
      source: "focusa_workpoint_resume";
      canonical: true;
      status: "unavailable";
      observed_at: string;
    };

function record(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`${label} must be an object`);
  return value as Record<string, unknown>;
}

function exactKeys(value: Record<string, unknown>, keys: readonly string[], label: string): void {
  const allowed = new Set(keys);
  const unsupported = Object.keys(value).filter((key) => !allowed.has(key));
  if (unsupported.length) throw new Error(`${label} contains unsupported fields: ${unsupported.join(", ")}`);
}

function text(value: Record<string, unknown>, key: string, label: string): string {
  const current = value[key];
  if (typeof current !== "string" || !current.trim()) throw new Error(`${label}.${key} is required`);
  return current;
}

function localHref(value: string, label: string): string {
  if (!value.startsWith("/") || value.startsWith("//") || value.includes("\\") || /[\u0000-\u001f]/.test(value)) {
    throw new Error(`${label} must be a local Cockpit path`);
  }
  return value;
}

function opaqueRef(value: string, label: string): string {
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/.test(value)) throw new Error(`${label} must be an opaque identifier`);
  return value;
}

function timestamp(value: string): string {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/.test(value) || Number.isNaN(Date.parse(value))) {
    throw new Error("observed_at must be RFC3339 UTC");
  }
  return value;
}

export function parseCockpitWorkpointResume(value: unknown): CockpitWorkpointResume {
  const item = record(value, "workpoint_resume");
  if (item.schema !== COCKPIT_WORKPOINT_RESUME_SCHEMA || item.source !== "focusa_workpoint_resume" || item.canonical !== true) {
    throw new Error("workpoint_resume must be a canonical Focusa-derived view model");
  }
  const status = text(item, "status", "workpoint_resume");
  const observedAt = timestamp(text(item, "observed_at", "workpoint_resume"));

  if (status === "resumable") {
    exactKeys(item, ["schema", "source", "canonical", "status", "workpoint_id", "label", "target", "observed_at"], "workpoint_resume");
    const target = record(item.target, "workpoint_resume.target");
    exactKeys(target, ["workspace_id", "href", "object_ref"], "workpoint_resume.target");
    const parsedTarget: ResumeTarget = {
      workspace_id: opaqueRef(text(target, "workspace_id", "workpoint_resume.target"), "workspace_id"),
      href: localHref(text(target, "href", "workpoint_resume.target"), "target.href"),
    };
    if (target.object_ref !== undefined) parsedTarget.object_ref = opaqueRef(text(target, "object_ref", "workpoint_resume.target"), "object_ref");
    return {
      schema: COCKPIT_WORKPOINT_RESUME_SCHEMA,
      source: "focusa_workpoint_resume",
      canonical: true,
      status,
      workpoint_id: opaqueRef(text(item, "workpoint_id", "workpoint_resume"), "workpoint_id"),
      label: text(item, "label", "workpoint_resume"),
      target: parsedTarget,
      observed_at: observedAt,
    };
  }

  if (status === "blocked") {
    exactKeys(item, ["schema", "source", "canonical", "status", "recovery", "observed_at"], "workpoint_resume");
    const recovery = record(item.recovery, "workpoint_resume.recovery");
    exactKeys(recovery, ["message", "href", "action_label"], "workpoint_resume.recovery");
    return {
      schema: COCKPIT_WORKPOINT_RESUME_SCHEMA,
      source: "focusa_workpoint_resume",
      canonical: true,
      status,
      recovery: {
        message: text(recovery, "message", "workpoint_resume.recovery"),
        href: localHref(text(recovery, "href", "workpoint_resume.recovery"), "recovery.href"),
        action_label: text(recovery, "action_label", "workpoint_resume.recovery"),
      },
      observed_at: observedAt,
    };
  }

  if (status === "unavailable") {
    exactKeys(item, ["schema", "source", "canonical", "status", "observed_at"], "workpoint_resume");
    return {
      schema: COCKPIT_WORKPOINT_RESUME_SCHEMA,
      source: "focusa_workpoint_resume",
      canonical: true,
      status,
      observed_at: observedAt,
    };
  }

  throw new Error(`unsupported workpoint resume status: ${status}`);
}

export function workpointResumeFromHost(): CockpitWorkpointResume | null {
  if (typeof window === "undefined") return null;
  const candidate = window.__UIAI_COCKPIT_CONTRACTS__?.workpoint_resume;
  if (candidate === undefined) return null;
  try {
    return parseCockpitWorkpointResume(candidate);
  } catch {
    return null;
  }
}
