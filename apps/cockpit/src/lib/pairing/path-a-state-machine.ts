import type { CockpitPairingRoomV1 } from "$lib/contracts/focusa-pairing";

export type PathAPhase = "idle"|"starting"|"awaiting_operator"|"awaiting_vps"|"completing"|"persisting"|"verifying"|"completed"|"expired"|"cancelled"|"failed";
export interface PathAState { phase:PathAPhase; attempt:number; room?:CockpitPairingRoomV1; error?:string; cleanup_required:boolean }
export type PathAEvent =
 | {type:"START"}|{type:"ROOM_CREATED";room:CockpitPairingRoomV1}
 | {type:"STATUS";status:CockpitPairingRoomV1["status"]}|{type:"COMPLETION_VALIDATED"}
 | {type:"CREDENTIAL_PERSISTED"}|{type:"PROFILE_VERIFIED"}|{type:"EXPIRE"}|{type:"CANCEL"}
 | {type:"FAIL";error:string}|{type:"RETRY"};

export const initialPathAState=():PathAState=>({phase:"idle",attempt:0,cleanup_required:false});
const terminal=(state:PathAState,phase:"completed"|"expired"|"cancelled"|"failed",error?:string):PathAState=>({...state,phase,error,cleanup_required:Boolean(state.room)});

export function reducePathA(state:Readonly<PathAState>,event:PathAEvent):PathAState{
 if(event.type==="CANCEL"&&! ["completed","expired","cancelled"].includes(state.phase))return terminal({...state},"cancelled");
 if(event.type==="EXPIRE"&&state.room)return terminal({...state},"expired");
 if(event.type==="FAIL"&&! ["completed","expired","cancelled"].includes(state.phase))return terminal({...state},"failed",event.error.slice(0,240));
 switch(state.phase){
  case"idle":if(event.type==="START")return{phase:"starting",attempt:1,cleanup_required:false};break;
  case"starting":if(event.type==="ROOM_CREATED"&&event.room.status==="created")return{...state,phase:"awaiting_operator",room:event.room};break;
  case"awaiting_operator":if(event.type==="STATUS"&&event.status==="awaiting_vps")return{...state,phase:"awaiting_vps"};break;
  case"awaiting_vps":if(event.type==="STATUS"&&event.status==="completed")return{...state,phase:"completing"};break;
  case"completing":if(event.type==="COMPLETION_VALIDATED")return{...state,phase:"persisting"};break;
  case"persisting":if(event.type==="CREDENTIAL_PERSISTED")return{...state,phase:"verifying"};break;
  case"verifying":if(event.type==="PROFILE_VERIFIED")return terminal({...state},"completed");break;
  case"failed":if(event.type==="RETRY")return{phase:"starting",attempt:state.attempt+1,cleanup_required:false};break;
 }
 throw new Error(`illegal Path A transition: ${state.phase} + ${event.type}`);
}
