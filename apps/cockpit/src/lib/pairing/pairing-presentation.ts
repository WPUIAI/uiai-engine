import { parseCockpitPairingRoom } from "$lib/contracts/focusa-pairing";

export interface PairingPresentationV1 { schema:"focusa.pairing_presentation.v1"; room_id:string; display_code:string; pair_href:string; qr_payload:string; expires_at:string }
export function createPairingPresentation(value:unknown):PairingPresentationV1{
 const room=parseCockpitPairingRoom(value);if(!["created","awaiting_operator","awaiting_vps"].includes(room.status))throw new Error("pairing room is not presentable");
 if(!room.pair_url||!room.pair_code)throw new Error("pairing URL and code are required");
 const pair=new URL(room.pair_url),daemon=new URL(room.daemon_url);if(pair.origin!==daemon.origin)throw new Error("pairing URL origin mismatch");
 if(/token|secret|authorization/i.test(pair.search)||/token|secret|authorization/i.test(pair.pathname))throw new Error("pairing URL contains forbidden secret shape");
 const code=room.pair_code.toUpperCase();if(!/^[A-Z0-9-]{8,32}$/.test(code))throw new Error("pairing code is invalid");
 return{schema:"focusa.pairing_presentation.v1",room_id:room.room_id,display_code:code,pair_href:pair.toString(),qr_payload:pair.toString(),expires_at:room.expires_at};
}
