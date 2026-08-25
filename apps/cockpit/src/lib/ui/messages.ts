export const cockpitMessages = {
  "shell.workspace": "Workspace",
  "shell.find_do": "Find or do",
  "shell.search_placeholder": "Search workspace or capability…",
  "shell.activity": "Activity",
  "shell.recovery_only": "Recovery-only mode",
  "shell.manage_entitlement": "Manage entitlement",
  "sidebar.commands_opened": "Commands opened for {workspace}.",
  "sidebar.moved": "{workspace} moved within {group}.",
} as const;

export type CockpitMessageKey = keyof typeof cockpitMessages;

export function message(key: CockpitMessageKey, variables: Record<string, string> = {}): string {
  return Object.entries(variables).reduce(
    (result, [name, value]) => result.replaceAll(`{${name}}`, value),
    cockpitMessages[key] as string,
  );
}

export function directionForLocale(locale: string): "ltr" | "rtl" {
  return /^(ar|fa|he|ur)(-|$)/i.test(locale) ? "rtl" : "ltr";
}
