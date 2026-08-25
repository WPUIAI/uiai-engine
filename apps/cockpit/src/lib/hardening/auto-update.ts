// H6 Tauri auto-update — check via updater, stage silent, require operator confirm to apply
export type UpdateState="idle"|"available"|"downloaded"|"failed";
export function shouldAutoUpdate(s:UpdateState){ return s==="downloaded"; }
