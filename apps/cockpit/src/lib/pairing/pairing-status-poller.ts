import{parseCockpitPairingRoom,type CockpitPairingRoomV1}from"$lib/contracts/focusa-pairing";
export interface PollReply{room:unknown;pressure?:boolean;retry_after_ms?:number}export type PollStatus=(roomId:string,signal:AbortSignal)=>Promise<PollReply>;export type PollSleep=(ms:number,signal:AbortSignal)=>Promise<void>;
const sleep:PollSleep=(ms,signal)=>new Promise((resolve,reject)=>{const id=setTimeout(resolve,ms);signal.addEventListener("abort",()=>{clearTimeout(id);reject(new DOMException("Aborted","AbortError"));},{once:true});});
export async function pollPairingRoom(roomId:string,fetchStatus:PollStatus,options:{signal:AbortSignal;sleep?:PollSleep;max_polls?:number}):Promise<CockpitPairingRoomV1>{
 if(!/^[A-Za-z0-9._~:-]{1,256}$/.test(roomId))throw new Error("room ID invalid");const wait=options.sleep??sleep,max=options.max_polls??200;if(max<1||max>400)throw new Error("poll bound invalid");let cadence=1500;
 for(let count=0;count<max;count++){if(options.signal.aborted)throw new DOMException("Aborted","AbortError");const reply=await fetchStatus(roomId,options.signal);const room=parseCockpitPairingRoom(reply.room);if(room.room_id!==roomId)throw new Error("pairing status room mismatch");if(["completed","expired","revoked","failed"].includes(room.status))return room;if(reply.pressure)cadence=Math.min(Math.max(reply.retry_after_ms??cadence*2,1500),10000);else cadence=1500;await wait(cadence,options.signal);}
 throw new Error("pairing status polling exhausted");
}
