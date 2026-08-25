// H7 cloud profile consent UX — explicit opt-in, revocable, stored in UserSettings
export type Consent = { cloudProfile:boolean, ts:string };
export function grant(c:Consent){ return {...c, cloudProfile:true, ts:new Date().toISOString()}; }
export function revoke(c:Consent){ return {...c, cloudProfile:false, ts:new Date().toISOString()}; }
