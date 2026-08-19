// H1 multi-node CRDT conflict UX — last-write-wins with operator badge
export type ConflictKind = "concurrent_edit";
export function resolveConflict(a:{ts:string,v:string}, b:{ts:string,v:string}) { return a.ts >= b.ts ? a : b; }
export function conflictBadge(k:ConflictKind){ return k==="concurrent_edit" ? "Resolved: newest wins" : ""; }
