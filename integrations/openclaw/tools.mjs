import { createHash } from "node:crypto";
import { createUIAIClient, modelProjection } from "./client.mjs";

export const actions=["open","navigate","read","snapshot","click","type","fill","scroll","screenshot","diagnostics","close"];
const string={type:"string",maxLength:8192};
export const browserParameters={type:"object",additionalProperties:false,required:["action"],properties:{
 action:{type:"string",enum:actions},focusa_scope:{type:"object",maxProperties:16},session_id:{type:"string",maxLength:128},url:string,selector:string,text:string,
 label:{type:"string",maxLength:64},reuse:{type:"boolean"},width:{type:"integer",minimum:1,maximum:4096},height:{type:"integer",minimum:1,maximum:4096},
 x:{type:"integer"},y:{type:"integer"},deltaX:{type:"integer"},deltaY:{type:"integer"},max_chars:{type:"integer",minimum:1,maximum:30000},
 format:{type:"string",enum:["text","markdown","jpeg","png"]},mode:{type:"string",enum:["main_content","full"]},quality:{type:"integer",minimum:1,maximum:100},fullPage:{type:"boolean"},compact:{type:"boolean"},interactive:{type:"boolean"},include_links:{type:"boolean"},
}};
const result=(data,image)=>{
 const projected=modelProjection(data);let text=JSON.stringify(projected);
 if(text.length>30000)text=JSON.stringify({status:"bounded_result",truncated:true,next_action:"read a narrower selector or lower max_chars",session:projected.session?.id});
 return {content:[{type:"text",text},...(image?[image]:[])],details:projected};
};
const failure=error=>({isError:true,content:[{type:"text",text:JSON.stringify({status:"blocked",provider:"uiai",fallback:"none",error_class:/^[a-z][a-z0-9_]{0,63}$/.test(error?.code||"")?error.code:"uiai_unavailable",retryable:false,...modelProjection(error?.details||{})})}],details:{provider:"uiai",fallback:"none",operation_succeeded:false}});

export function registerUIAITools(api,{resolveSecret}={}){
 const scopes=new Map();
 function factory(context,health=false){
  return {
   name:health?"uiai_health":"uiai_browser",label:health?"UIAI health":"UIAI browser",
   description:health?"Read UIAI capacity/health only; this is not authenticated-page or business-task proof.":"Use UIAI only for scoped browser work; preserve session_id and snapshot refs, close owned sessions, and inspect uncertain effects before retry. No managed Chromium, desktop or node-browser fallback.",
   parameters:health?{type:"object",additionalProperties:false,properties:{}}:browserParameters,
   async execute(_id,parameters,signal){
    try{
     const runtimeConfig=context.getRuntimeConfig?.()||context.runtimeConfig||api.config;
     const config=runtimeConfig.plugins?.entries?.[api.id]?.config||api.pluginConfig||{};
     if(runtimeConfig.browser?.enabled!==false||!runtimeConfig.tools?.deny?.includes("browser"))throw Object.assign(new Error(),{code:"native_browser_must_be_disabled"});
     const identity=context.sessionId;
     if(!health&&(!identity||!context.agentId))throw Object.assign(new Error(),{code:"trusted_conversation_required"});
     const key=JSON.stringify([context.agentId,identity,config.baseUrl||"http://127.0.0.1:7456"]);
     if(!scopes.has(key)){
      if(scopes.size>=1000)throw Object.assign(new Error(),{code:"conversation_binding_limit"});
      scopes.set(key,{owned:new Set(),labels:new Map()});
     }
     const actorRef="openclaw:"+createHash("sha256").update(key).digest("hex").slice(0,24);
     const client=createUIAIClient({...config,actorRef},{state:scopes.get(key),resolveSecret:value=>resolveSecret?resolveSecret(value,runtimeConfig):value});
     if(health)return result(await client.health(signal));
     const {action,...args}=parameters;
     const {data,image}=await client.execute(action,args,signal);
     return result(data,image);
    }catch(error){return failure(error);}
   },
  };
 }
 api.registerTool(context=>factory(context),{names:["uiai_browser"],optional:true});
 api.registerTool(context=>factory(context,true),{names:["uiai_health"],optional:true});
}
