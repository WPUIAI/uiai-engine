import { readEngineResponse } from "../pi/uiai/engine-response.mjs";
import { assertEvidenceDelivery } from "../pi/uiai/epwa-contract.mjs";

const routes={navigate:"navigate",read:"read",snapshot:"snapshot",click:"click",type:"type",fill:"fill",scroll:"scroll",screenshot:"screenshot",diagnostics:"diagnostics",close:""};
const hidden=/cookie|password|token|authorization|credential|api.?key|secret|storage.?state|headers|url|uri|path/i;
export function modelProjection(value,depth=0){
 if(depth>16)return "[bounded]";
 if(Array.isArray(value))return value.slice(0,100).map(item=>modelProjection(item,depth+1));
 if(value&&typeof value==="object")return Object.fromEntries(Object.entries(value).filter(([key])=>!hidden.test(key)).map(([key,item])=>[key,modelProjection(item,depth+1)]));
 return value;
}
function blocked(code,details={}){return Object.assign(new Error(code),{code,details});}

export function createUIAIClient(config,{fetchImpl=fetch,resolveSecret=async value=>value,state={owned:new Set(),labels:new Map()}}={}){
 if((config.provider??"uiai")!=="uiai"||(config.fallback??"none")!=="none")throw blocked("uiai_only_required");
 const base=new URL(config.baseUrl||"http://127.0.0.1:7456");
 if(!["http:","https:"].includes(base.protocol)||base.username||base.password||base.search||base.hash)throw blocked("invalid_engine_origin");
 if(base.protocol!=="https:"&&!["127.0.0.1","localhost","[::1]"].includes(base.hostname))throw blocked("remote_engine_requires_https");
 const origin=base.href.replace(/\/$/,"");
 const owned=state.owned;
 async function headers(){
  const result={"Content-Type":"application/json"};
  const selected=config.apiKey??config.bearerToken;
  if(selected!==undefined){
   let value;try{value=await resolveSecret(selected);}catch{throw blocked("credential_resolution_failed");}
   if(typeof value!=="string"||!value||/[\r\n\0]/.test(value))throw blocked("credential_resolution_failed");
   if(config.apiKey!==undefined)result["X-API-Key"]=value;else result.Authorization=`Bearer ${value}`;
  }
  return result;
 }
 async function request(path,body,method="POST",signal,opening=false){
  const controller=new AbortController();
  const abort=()=>controller.abort();signal?.addEventListener("abort",abort,{once:true});
  if(signal?.aborted)controller.abort();
  const timeout=setTimeout(abort,Math.min(Math.max(config.timeoutMs||30000,100),60000));
  try{
   const response=await fetchImpl(origin+path,{method,redirect:"error",signal:controller.signal,headers:await headers(),body:body===undefined?undefined:JSON.stringify(body)});
   const data=await readEngineResponse(response,{opening,validate:value=>assertEvidenceDelivery(value,false),closeSession:async id=>{await request(`/api/session/${encodeURIComponent(id)}`,undefined,"DELETE");}});
   if(!response.ok)throw blocked(/^[a-z][a-z0-9_]{0,63}$/.test(data.error_class||"")?data.error_class:"uiai_request_failed",{
    http_status:response.status,error_id:/^[a-zA-Z0-9:_-]{1,128}$/.test(data.error_id||"")?data.error_id:undefined,
    side_effects_may_have_occurred:data.details?.side_effects_may_have_occurred??"unknown",
    retryable:data.retryable===true,recovery:"diagnostics_and_reobserve_before_any_authorized_retry",
   });
   if(["queued","pending","failed"].includes(data.status)||["queued","pending"].includes(data.state))throw blocked("uiai_call_unsettled",{retryable:false,recovery:"inspect_and_reconcile_without_replaying"});
   return data;
  }catch(error){
   if(error?.code&&/^[a-z][a-z0-9_]{0,63}$/.test(error.code))throw error;
   throw blocked(signal?.aborted?"operation_cancelled":"uiai_unavailable",{retryable:false,recovery:"restore_uiAI_no_browser_fallback"});
  }finally{clearTimeout(timeout);signal?.removeEventListener("abort",abort);}
 }
 async function attachment(data,signal){
  if(!config.privateAttachments)throw blocked("private_attachment_not_enabled");
  assertEvidenceDelivery(data,true);
  const record=new URL(data.epwa_delivery.epwa.record_url);
  if(!(config.evidenceOrigins||[]).includes(record.origin))throw blocked("attachment_origin_not_authorized");
  const extension=data.format==="png"?"png":"jpg";
  record.pathname=record.pathname.replace(/\/?$/, "/")+"screenshot."+extension;
  const bounded=signal?AbortSignal.any([signal,AbortSignal.timeout(15000)]):AbortSignal.timeout(15000);
  const response=await fetchImpl(record.href,{redirect:"error",signal:bounded});
  if(!response.ok)throw blocked("attachment_unavailable");
  const mime=response.headers.get("content-type")?.split(";")[0];
  if(!["image/jpeg","image/png"].includes(mime)||Number(response.headers.get("content-length")||0)>8*1024*1024)throw blocked("attachment_invalid");
  const reader=response.body.getReader();let size=0;const chunks=[];
  try{for(;;){const {done,value}=await reader.read();if(done)break;size+=value.length;if(size>8*1024*1024)throw blocked("attachment_too_large");chunks.push(value);}}
  finally{await reader.cancel();}
  const bytes=Buffer.concat(chunks);
  if(mime==="image/jpeg"?bytes[0]!==255||bytes[1]!==216:!bytes.subarray(0,8).equals(Buffer.from([137,80,78,71,13,10,26,10])))throw blocked("attachment_invalid");
  return {type:"image",mimeType:mime,data:bytes.toString("base64")};
 }
 return {
  async health(signal){return request("/api/health/browser",undefined,"GET",signal);},
  async execute(action,args={},signal){
   if(action==="open"){
    if(args.reuse&&args.label&&state.labels.has(args.label)){
      const prior=state.labels.get(args.label);
      if(prior.url!==args.url)throw blocked("label_bound_to_different_target");
      const session=await request(`/api/session/${encodeURIComponent(prior.id)}`,undefined,"GET",signal);
      return {data:{session,reused:true}};
    }
    if(owned.size>=10)throw blocked("owned_session_limit");
    const data=await request("/api/session",{url:args.url,width:args.width,height:args.height,actor_ref:config.actorRef,focusa_scope:args.focusa_scope},"POST",signal,true);
    owned.add(data.session.id);if(args.label)state.labels.set(args.label,{id:data.session.id,url:args.url});return {data};
   }
   const id=args.session_id;
   if(!owned.has(id))throw blocked("session_not_owned_by_this_conversation");
   if(!Object.hasOwn(routes,action))throw blocked("unsupported_action");
   const body={...args};delete body.session_id;
   const data=await request(`/api/session/${encodeURIComponent(id)}`+(routes[action]?"/"+routes[action]:""),action==="close"||action==="diagnostics"?undefined:body,action==="close"?"DELETE":action==="diagnostics"?"GET":"POST",signal);
   if(action==="close"){owned.delete(id);for(const [label,item] of state.labels){if(item.id===id)state.labels.delete(label);}}
   return {data,image:action==="screenshot"?await attachment(data,signal):undefined};
  },
  ownedCount:()=>owned.size,
 };
}
