import type { CardManifest } from "../contracts/card-manifest";
import type { CockpitWorkpointResume } from "../contracts/workpoint-resume";
import type { WorkspaceManifestEntry } from "./sidebar-manifest";

export type CommandIndexKind = "resume" | "workspace" | "registered_object" | "capability" | "system";

export interface CommandIndexEntry {
  id: string;
  kind: CommandIndexKind;
  label: string;
  hint: string;
  href: string;
  search_text: string;
}

export interface FooterDestination {
  id: string;
  label: string;
  route: string;
}

function entry(input: Omit<CommandIndexEntry, "search_text">, aliases: string[] = []): CommandIndexEntry {
  return {
    ...input,
    search_text: [input.label, input.hint, input.kind, ...aliases].join(" ").toLowerCase(),
  };
}

function hrefBelongsToWorkspace(href: string, workspace: WorkspaceManifestEntry): boolean {
  if (workspace.route === "/") return href === "/" || href.startsWith("/?") || href.startsWith("/#");
  return href === workspace.route || href.startsWith(`${workspace.route}/`) || href.startsWith(`${workspace.route}?`) || href.startsWith(`${workspace.route}#`);
}

export function buildCommandIndex(
  workspaces: readonly WorkspaceManifestEntry[],
  cards: readonly CardManifest[],
  footer: readonly FooterDestination[],
  resume: CockpitWorkpointResume | null,
): CommandIndexEntry[] {
  const commands: CommandIndexEntry[] = [];

  if (resume?.status === "resumable") {
    const targetWorkspace = workspaces.find((workspace) => workspace.id === resume.target.workspace_id);
    if (targetWorkspace && hrefBelongsToWorkspace(resume.target.href, targetWorkspace)) {
      commands.push(entry({
        id: `resume:${resume.workpoint_id}`,
        kind: "resume",
        label: `Resume ${resume.label}`,
        hint: `Workpoint · ${resume.target.workspace_id}`,
        href: resume.target.href,
      }, [resume.workpoint_id, resume.target.object_ref || ""]));
    }
  }

  for (const workspace of workspaces) {
    if (workspace.state === "planned") continue;
    commands.push(entry({
      id: `workspace:${workspace.id}`,
      kind: "workspace",
      label: `Open ${workspace.label}`,
      hint: `${workspace.group} · ${workspace.state}`,
      href: workspace.route,
    }, [workspace.id, workspace.description, ...workspace.subsections]));
  }

  for (const card of cards) {
    commands.push(entry({
      id: `object:${card.card_id}`,
      kind: "registered_object",
      label: `Inspect ${card.label}`,
      hint: `Registered object · ${card.product_surface}`,
      href: `/capabilities?object=${encodeURIComponent(card.card_id)}`,
    }, [card.card_id, card.contract_ref || "", card.normative_source]));
  }

  const capabilities = new Set(cards.flatMap((card) => card.capabilities));
  for (const capability of [...capabilities].sort()) {
    commands.push(entry({
      id: `capability:${capability}`,
      kind: "capability",
      label: `Inspect ${capability}`,
      hint: "Registered capability",
      href: `/capabilities?capability=${encodeURIComponent(capability)}`,
    }));
  }

  for (const destination of footer) {
    commands.push(entry({
      id: `system:${destination.id}`,
      kind: "system",
      label: `Open ${destination.label}`,
      hint: "System",
      href: destination.route,
    }, [destination.id]));
  }

  return commands;
}

export function filterCommandIndex(commands: readonly CommandIndexEntry[], query: string): CommandIndexEntry[] {
  const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (!terms.length) return [...commands];
  return commands.filter((command) => terms.every((term) => command.search_text.includes(term)));
}
