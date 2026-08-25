// H3 first-run keychain bootstrap — uses secure-store LazyStore, no localStorage
import { saveSecret, loadSecret } from "../secure-store";
export async function bootstrapKeychain(seed:string){ await saveSecret("cockpit-bootstrap", seed); return await loadSecret("cockpit-bootstrap"); }
