import { test } from "node:test";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { gzipSync } from "node:zlib";
import { spawn } from "node:child_process";
import { createInterface } from "node:readline";
import { readEngineResponse } from "./engine-response.mjs";

for (const payload of ["not-json-private-canary", '{"0":31,"1":139}', '{"session":{}}']) {
  test("malformed open closes its header-bound session without echoing bytes",async()=>{
    const closed=[];
    const response=new Response(payload,{headers:{"X-UIAI-Created-Session-ID":"owned-created-id"}});
    await assert.rejects(readEngineResponse(response,{opening:true,closeSession:async id=>closed.push(id)}),error=>{
      assert.equal(error.code,"invalid_engine_response");assert.equal(error.cleanup_state,"closed");
      assert.ok(!error.message.includes(payload));return true;
    });
    assert.deepEqual(closed,["owned-created-id"]);
  });
}

test("delivery validation closes only the newly created session",async()=>{
  const closed=[];
  await assert.rejects(readEngineResponse(Response.json({session:{id:"created"}}),{
    opening:true,closeSession:async id=>closed.push(id),validate:()=>{throw Object.assign(new Error("EPWA delivery unavailable"),{code:"epwa_delivery_unavailable"});}
  }),error=>error.cleanup_state==="closed"&&error.code==="epwa_delivery_unavailable");
  assert.deepEqual(closed,["created"]);
});

test("identity mismatch never closes an unrelated body id",async()=>{
 const closed=[];
 await assert.rejects(readEngineResponse(Response.json({session:{id:"unrelated"}},{headers:{"X-UIAI-Created-Session-ID":"created"}}),{opening:true,closeSession:async id=>closed.push(id)}));
 assert.deepEqual(closed,["created"]);
});

test("unknown and failed cleanup are explicit, never a successful close claim",async()=>{
 await assert.rejects(readEngineResponse(new Response("malformed"),{opening:true}),error=>error.cleanup_state==="unknown_session");
 await assert.rejects(readEngineResponse(new Response("malformed",{headers:{"X-UIAI-Created-Session-ID":"created"}}),{opening:true,closeSession:async()=>{throw new Error("synthetic private error");}}),error=>error.cleanup_state==="failed"&&!error.message.includes("synthetic private"));
 const failed=await readEngineResponse(Response.json({error:"server rejected"},{status:503,headers:{"X-UIAI-Created-Session-ID":"created"}}),{opening:true,closeSession:async()=>{throw new Error("failure");}});
 assert.equal(failed.session_cleanup,"failed");
});

for(const compressed of [false,true]){
 test(`fetch decodes a ${compressed?"gzip":"plain"} envelope exactly once`,async()=>{
  const server=createServer((req,res)=>{
   res.setHeader("Content-Type","application/json");
   if(compressed)res.setHeader("Content-Encoding","gzip");
   const body=JSON.stringify({session:{id:"gzip-owned"}});res.end(compressed?gzipSync(body):body);
  });
  await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
  try{
   const data=await readEngineResponse(await fetch(`http://127.0.0.1:${server.address().port}/api/session`),{opening:true});
   assert.equal(data.session.id,"gzip-owned");
  }finally{await new Promise(resolve=>server.close(resolve));}
 });
}

test("actual MCP adapter rejects a mislabelled gzip body and disposes its created session",async()=>{
 let closed=0;
 const server=createServer((req,res)=>{
  res.setHeader("Content-Type","application/json");
  if(req.method==="DELETE"&&req.url==="/api/session/mcp-created"){
   closed++;res.end(JSON.stringify({status:"closed"}));return;
  }
  res.setHeader("X-UIAI-Created-Session-ID","mcp-created");
  res.end(gzipSync(JSON.stringify({session:{id:"mcp-created"}})));
 });
 await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
 const child=spawn(process.execPath,[new URL("../../../mcp/browser-session-mcp.mjs",import.meta.url).pathname],{env:{...process.env,UIAI_ENGINE_URL:`http://127.0.0.1:${server.address().port}`,UIAI_MCP_TIMEOUT_MS:"3000"},stdio:["pipe","pipe","pipe"]});
 let stderr="";child.stderr.on("data",data=>{stderr+=data;});
 const lines=createInterface({input:child.stdout});
 try{
  const reply=new Promise((resolve,reject)=>{
   const timer=setTimeout(()=>reject(new Error("MCP response timed out: "+stderr)),6000);
   lines.on("line",line=>{try{const result=JSON.parse(line);if(result.id===1){clearTimeout(timer);resolve(result);}}catch(error){clearTimeout(timer);reject(error);}});
  });
  child.stdin.write(JSON.stringify({jsonrpc:"2.0",id:1,method:"tools/call",params:{name:"browser_open",arguments:{url:"https://example.com"}}})+"\n");
  const result=await reply;
  assert.equal(result.result.isError,true);
  assert.match(result.result.content[0].text,/session_cleanup=closed/);
  assert.ok(!result.result.content[0].text.includes('"0":'));
  assert.equal(closed,1);
 }finally{lines.close();child.kill();await new Promise(resolve=>server.close(resolve));}
});
