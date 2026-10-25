// Execute the real native adapter. Only framework UI/registration dependencies
// are mocked; its HTTP code and the shared response contract remain real.
import { test } from "node:test";
import assert from "node:assert/strict";
import { registerHooks, createRequire } from "node:module";
import { pathToFileURL } from "node:url";
import { createServer } from "node:http";
import { gzipSync } from "node:zlib";

const require=createRequire(import.meta.url);
const typebox=process.env.UIAI_TEST_TYPEBOX || require.resolve("typebox",{paths:[process.env.UIAI_TEST_NODE_DEP_ROOT || "/usr/lib/node_modules/openclaw"]});
const framework={
 "@earendil-works/pi-coding-agent":"export const keyHint=()=>'';",
 "@earendil-works/pi-tui":"export class Text {constructor(text){this.text=text}}",
};
registerHooks({resolve(specifier,context,next){
 if(framework[specifier])return{url:"data:text/javascript,"+encodeURIComponent(framework[specifier]),shortCircuit:true};
 if(specifier==="typebox")return{url:pathToFileURL(typebox).href,shortCircuit:true};
 return next(specifier,context);
}});
const extension=(await import("../uiai-engine.ts")).default;
const tools=new Map();
extension({on(){},registerCommand(){},registerTool(tool){tools.set(tool.name,tool)}});

test("native Pi open decodes gzip and cleans its session on malformed bytes",async()=>{
 let mode="gzip",closed=0;
 const server=createServer((req,res)=>{
  res.setHeader("Content-Type","application/json");
  if(req.method==="DELETE") {closed++;res.end(JSON.stringify({status:"closed"}));return;}
  res.setHeader("X-UIAI-Created-Session-ID","native-created");
  if(mode==="gzip")res.setHeader("Content-Encoding","gzip");
  res.end(gzipSync(JSON.stringify({session:{id:"native-created"}})));
 });
 await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
 const prior=process.env.UIAI_ENGINE_URL;process.env.UIAI_ENGINE_URL=`http://127.0.0.1:${server.address().port}`;
 try{
  const open=tools.get("uiai_browser_open");assert.ok(open);
  const result=await open.execute("gzip",{url:"https://example.com"});
  assert.equal(JSON.parse(result.content[0].text).session.id,"native-created");
  await tools.get("uiai_browser_close").execute("close",{session_id:"native-created"});
  assert.equal(closed,1);
  mode="malformed";
  await assert.rejects(open.execute("bad",{url:"https://example.com"}),error=>error.code==="invalid_engine_response"&&error.cleanup_state==="closed");
  assert.equal(closed,2);
 }finally{
  if(prior===undefined)delete process.env.UIAI_ENGINE_URL;else process.env.UIAI_ENGINE_URL=prior;
  await new Promise(resolve=>server.close(resolve));
 }
});
