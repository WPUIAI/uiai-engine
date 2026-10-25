import {test} from "node:test";
import assert from "node:assert/strict";
import {createUIAIClient,modelProjection} from "./client.mjs";
import {registerUIAITools,actions,browserParameters} from "./tools.mjs";

function fixture(){
 const calls=[];let alive=true;
 const fetchImpl=async(url,options={})=>{
  calls.push({url,options});if(!alive)throw new Error("private-provider-path/secret");
  if(url.endsWith("/api/health/browser"))return Response.json({status:"healthy",active_pages:0});
  if(url.endsWith("/api/session")&&options.method==="POST")return Response.json({session:{id:"owned-session"}},{headers:{"X-UIAI-Created-Session-ID":"owned-session"}});
  if(options.method==="DELETE")return Response.json({status:"closed"});
  if(url.endsWith("/owned-session")&&options.method==="GET")return Response.json({id:"owned-session"});
  return Response.json({status:"ok",refs:{e1:{selector:"#input"}},text:"owned page"});
 };
 return{calls,fetchImpl,down(){alive=false}};
}

test("all required browser operations use UIAI and preserve session/ref continuity",async()=>{
 const f=fixture(),client=createUIAIClient({provider:"uiai",fallback:"none"},{fetchImpl:f.fetchImpl});
 const opened=await client.execute("open",{url:"https://example.com",label:"qa"});assert.equal(opened.data.session.id,"owned-session");
 const reused=await client.execute("open",{url:"https://example.com",label:"qa",reuse:true});assert.equal(reused.data.reused,true);
 for(const action of ["navigate","read","snapshot","click","type","fill","scroll","diagnostics"]){
  await client.execute(action,{session_id:"owned-session",selector:"@e1",text:"synthetic input"});
 }
 const click=f.calls.find(call=>call.url.endsWith("/click"));assert.equal(JSON.parse(click.options.body).selector,"@e1");
 await client.execute("close",{session_id:"owned-session"});assert.equal(client.ownedCount(),0);
 await assert.rejects(client.execute("read",{session_id:"owned-session"}),error=>error.code==="session_not_owned_by_this_conversation");
 assert.ok(f.calls.every(call=>call.url.startsWith("http://127.0.0.1:7456/")));
});

test("unavailable and effect-bearing failures remain failures with no fallback or retry",async()=>{
 const f=fixture(),client=createUIAIClient({fallback:"none"},{fetchImpl:f.fetchImpl});f.down();
 await assert.rejects(client.health(),error=>error.code==="uiai_unavailable"&&!error.message.includes("private-provider"));
 assert.equal(f.calls.length,1);
 assert.throws(()=>createUIAIClient({fallback:"chromium"}),error=>error.code==="uiai_only_required");
 const failed=createUIAIClient({}, {fetchImpl:async()=>Response.json({error_class:"pending_side_effect_unknown",retryable:false,details:{side_effects_may_have_occurred:true},message:"private-secret"},{status:409})});
 await assert.rejects(failed.health(),error=>error.code==="pending_side_effect_unknown"&&error.details.side_effects_may_have_occurred&&!error.message.includes("private-secret"));
});

test("auth is selected and resolved per request without model-visible secret values",async()=>{
 const f=fixture(),refs=[];
 const client=createUIAIClient({apiKey:{source:"env",provider:"default",id:"UIAI_KEY"},bearerToken:"unused"},{fetchImpl:f.fetchImpl,resolveSecret:async ref=>{refs.push(ref);return "SYNTHETIC_API_KEY";}});
 await client.health();await client.health();assert.equal(refs.length,2);
 assert.equal(f.calls[0].options.headers["X-API-Key"],"SYNTHETIC_API_KEY");assert.equal(f.calls[0].options.headers.Authorization,undefined);
 assert.equal(f.calls[0].options.redirect,"error");
 const projected=modelProjection({cookie:"canary",password:"canary",api_key:"canary",artifact_url:"https://private-evidence.example/secret",artifact_ref:"artifact:keep",nested:{resume_token:"canary"}});
 assert.equal(projected.artifact_ref,"artifact:keep");assert.ok(!JSON.stringify(projected).includes("canary"));assert.ok(!JSON.stringify(projected).includes("private-evidence"));
 assert.throws(()=>createUIAIClient({baseUrl:"http://remote.example"}),error=>error.code==="remote_engine_requires_https");
});

test("screenshots require ready evidence and explicit bounded origin authorization",async()=>{
 const f=fixture();let imageCalls=0;
 const data={format:"png",artifact_ref:"artifact:fixture",artifact_url:"https://evidence.example/record/",epwa_delivery:{schema:"uiai.epwa_delivery.v1",state:"ready",artifact:{artifact_ref:"artifact:fixture"},epwa:{record_url:"https://evidence.example/record/",portable_url:"https://evidence.example/record/portable.zip"}}};
 const bytes=Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLbtAAAAABJRU5ErkJggg==","base64");
 const fetchImpl=async(url,options)=>{
  if(url.startsWith("https://evidence.example/")){imageCalls++;assert.equal(options.headers,undefined);assert.equal(options.redirect,"error");return new Response(bytes,{headers:{"Content-Type":"image/png"}});}
  if(url.endsWith("/screenshot"))return Response.json(data);
  return f.fetchImpl(url,options);
 };
 const client=createUIAIClient({privateAttachments:true,evidenceOrigins:["https://evidence.example"]},{fetchImpl});
 await client.execute("open",{url:"https://example.com"});
 const captured=await client.execute("screenshot",{session_id:"owned-session"});assert.equal(captured.image.type,"image");assert.equal(captured.image.data,bytes.toString("base64"));assert.equal(imageCalls,1);
 const denied=createUIAIClient({privateAttachments:true},{fetchImpl});await denied.execute("open",{url:"https://example.com"});
 await assert.rejects(denied.execute("screenshot",{session_id:"owned-session"}),error=>error.code==="attachment_origin_not_authorized");assert.equal(imageCalls,1);
});

test("real tool factories fail closed on native browser policy and isolate conversation UUIDs",async()=>{
 const factories=new Map(),f=fixture(),prior=globalThis.fetch;globalThis.fetch=f.fetchImpl;
 try{
  const config={browser:{enabled:false},tools:{deny:["browser"]}};
  registerUIAITools({id:"uiai-engine",config,pluginConfig:{},registerTool(factory,options){factories.set(options.names[0],factory);}});
  const tool=factories.get("uiai_browser")({agentId:"agent",sessionId:"conversation-one"});
  const opened=await tool.execute("1",{action:"open",url:"https://example.com"});assert.equal(opened.isError,undefined);
  const other=factories.get("uiai_browser")({agentId:"agent",sessionId:"conversation-two"});
  const denied=await other.execute("2",{action:"read",session_id:"owned-session"});assert.equal(denied.isError,true);
  assert.match(denied.content[0].text,/session_not_owned_by_this_conversation/);
  config.browser.enabled=true;
  const unavailable=await tool.execute("3",{action:"read",session_id:"owned-session"});assert.equal(unavailable.isError,true);assert.match(unavailable.content[0].text,/native_browser_must_be_disabled/);
  assert.deepEqual(browserParameters.properties.action.enum,actions);
 }finally{globalThis.fetch=prior;}
});

test("queued responses cannot masquerade as settled work or trigger replay",async()=>{
 let calls=0;
 const client=createUIAIClient({}, {fetchImpl:async()=>{calls++;return Response.json({status:"queued",job_id:"owned-job"},{status:202});}});
 await assert.rejects(client.health(),error=>error.code==="uiai_call_unsettled"&&error.details.retryable===false);
 assert.equal(calls,1);
});

test("explicit scope is forwarded as metadata, never synthesized",async()=>{
 const f=fixture();const client=createUIAIClient({}, {fetchImpl:f.fetchImpl});
 await client.execute("open",{url:"https://example.com",focusa_scope:{project_ref:"project:provided"}});
 assert.deepEqual(JSON.parse(f.calls[0].options.body).focusa_scope,{project_ref:"project:provided"});
 assert.ok(!JSON.stringify(f.calls[0].options.body).includes("workset"));
 const data=modelProjection({recordUrl:"https://private",storagePath:"/secret/path",credentialHandle:"private",artifact_ref:"artifact:keep"});
 assert.deepEqual(data,{artifact_ref:"artifact:keep"});
});
