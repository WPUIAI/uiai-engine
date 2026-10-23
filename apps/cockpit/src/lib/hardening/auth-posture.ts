// H4 AI API auth posture — bearer via keychain, never logged, redacted preview only
import { redactedPreview } from "../secure-store";
export function authHeader(token:string){ return `Bearer ${token}`; }
export function preview(t:string){ return redactedPreview(t); }
