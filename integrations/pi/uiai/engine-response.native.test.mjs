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

test("native capture and packet tools preserve complete caller scope without inventing refs",async()=>{
 const scope={project_ref:"synthetic-project",workstream_ref:"synthetic-stream",workset_ref:"synthetic-set",callgraph_ref:"synthetic-graph",workpoint_ref:"synthetic-point",work_item_ref:"synthetic-item",continuity_ref:"synthetic-continuity",work_items:[{work_item_ref:"synthetic-item",title:"Owned transport fixture"}]};
 const {buildResearchDiagnosticsPacket}=await import("../uiai-engine.ts");
 const packet=buildResearchDiagnosticsPacket({goal:"Owned scope fixture",focusa_scope:scope,responses:[]});
 assert.equal(packet.scope_status,"present");assert.deepEqual(packet.focusa_scope,scope);
 const received=[];
 const server=createServer((req,res)=>{
  let raw="";req.on("data",chunk=>raw+=chunk);req.on("end",()=>{
   received.push(JSON.parse(raw));res.writeHead(503,{"Content-Type":"application/json"});res.end(JSON.stringify({error:"owned fixture publication unavailable",error_class:"epwa_publication_failed"}));
  });
 });
 await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
 const prior=process.env.UIAI_ENGINE_URL;process.env.UIAI_ENGINE_URL=`http://127.0.0.1:${server.address().port}`;
 try{
  for(const [name,args] of [
   ["uiai_browser_open",{url:"https://example.test"}],
   ["uiai_browser_screenshot",{session_id:"owned-fixture"}],
   ["uiai_screenshot",{url:"https://example.test"}],
   ["uiai_focusa_packet_compose",{goal:"Owned fixture",responses:[]}],
  ]){
   const tool=tools.get(name);assert.ok(tool);assert.ok(tool.parameters.properties.focusa_scope,`${name} schema dropped scope`);
   const before=received.length;
   try{await tool.execute("scope",{...args,focusa_scope:scope});}catch(error){assert.ok(error instanceof Error)}
   assert.equal(received.length,before+1);assert.deepEqual(received.at(-1).focusa_scope,scope);
  }
 }finally{
  if(prior===undefined)delete process.env.UIAI_ENGINE_URL;else process.env.UIAI_ENGINE_URL=prior;
  await new Promise(resolve=>server.close(resolve));
 }
});

test("canonical installer ships every native adapter runtime dependency",async()=>{
 const {mkdtemp,readFile,rm,mkdir,copyFile}=await import('node:fs/promises');
 const {tmpdir}=await import('node:os');
 const {join}=await import('node:path');
 const {spawnSync}=await import('node:child_process');
 const {fileURLToPath}=await import('node:url');
 const home=await mkdtemp(join(tmpdir(),'uiai-owned-install-fixture-'));
 const dest=join(home,'extensions','uiai-engine.ts');
 try{
  const script=fileURLToPath(new URL('../../../scripts/install-agent-integrations.sh',import.meta.url));
  const result=spawnSync('bash',[script],{encoding:'utf8',env:{...process.env,HOME:home,UIAI_PI_EXTENSION_DEST:dest,UIAI_MCP_CONFIG_DEST:join(home,'mcp.json'),UIAI_MCP_BUNDLE_ROOT:join(home,'bundle'),UIAI_ENGINE_URL:'http://127.0.0.1:1',UIAI_API_KEY:'',UIAI_BEARER_TOKEN:'',DRY_RUN:'0'}});
  assert.equal(result.status,0,result.stderr);
  const config=JSON.parse(await readFile(join(home,'mcp.json'),'utf8'));
  assert.equal(config.mcpServers['uiai-browser'].args[0],join(home,'bundle','mcp','browser-session-mcp.mjs'));
  const bundled=spawnSync(process.execPath,['--test',fileURLToPath(new URL('../../../mcp/browser-session-mcp.test.mjs',import.meta.url))],{encoding:'utf8',env:{...process.env,UIAI_TEST_MCP_SCRIPT:config.mcpServers['uiai-browser'].args[0]}});
  assert.equal(bundled.status,0,bundled.stdout+bundled.stderr);
  const mcpOnly=spawnSync('bash',[script],{encoding:'utf8',env:{...process.env,HOME:home,UIAI_INSTALL_PI:'0',UIAI_PI_EXTENSION_DEST:join(home,'excluded-pi','extension.ts'),UIAI_MCP_BUNDLE_ROOT:join(home,'bundle-only'),UIAI_MCP_CONFIG_DEST:join(home,'mcp-only.json'),UIAI_API_KEY:'',UIAI_BEARER_TOKEN:'',DRY_RUN:'0'}});
  assert.equal(mcpOnly.status,0,mcpOnly.stderr);await assert.rejects(readFile(join(home,'excluded-pi','extension.ts')),{code:'ENOENT'});
  const actual=await readFile(join(home,'extensions','uiai','engine-response.mjs'));
  assert.deepEqual(actual,await readFile(new URL('./engine-response.mjs',import.meta.url)));
  const installed=(await import(pathToFileURL(dest).href)).default;
  const installedTools=new Map();installed({on(){},registerCommand(){},registerTool(tool){installedTools.set(tool.name,tool)}});
  assert.ok(installedTools.get('uiai_browser_open').parameters.properties.focusa_scope);
  assert.ok(installedTools.get('uiai_browser_screenshot'));
  assert.equal(installedTools.get('uiai_search').parameters.properties.providers.minItems,2);
  assert.equal(installedTools.get('uiai_search').parameters.properties.providers.maxItems,3);
  const broken=join(home,'broken-source');await mkdir(join(broken,'scripts'),{recursive:true});await mkdir(join(broken,'integrations','pi','uiai'),{recursive:true});
  await copyFile(script,join(broken,'scripts','install-agent-integrations.sh'));
  await copyFile(dest,join(broken,'integrations','pi','uiai-engine.ts'));
  await copyFile(join(home,'extensions','uiai','epwa-contract.mjs'),join(broken,'integrations','pi','uiai','epwa-contract.mjs'));
  const rejected=spawnSync('bash',[join(broken,'scripts','install-agent-integrations.sh')],{encoding:'utf8',env:{...process.env,HOME:home,UIAI_PI_EXTENSION_DEST:join(home,'must-not-install','extension.ts'),UIAI_MCP_CONFIG_DEST:join(home,'must-not-install','mcp.json'),UIAI_API_KEY:'',UIAI_BEARER_TOKEN:'',DRY_RUN:'0'}});
  assert.notEqual(rejected.status,0);assert.match(rejected.stderr,/missing Pi runtime dependency:.*engine-response\.mjs/);

 }finally{await rm(home,{recursive:true})}
});


test("native search research schema and HTTP forwarding preserve caller routing",async()=>{
 const search=tools.get("uiai_search");assert.ok(search);
 assert.equal(search.parameters.properties.providers.minItems,2);assert.equal(search.parameters.properties.providers.maxItems,3);assert.equal(search.parameters.properties.providers.uniqueItems,true);
 assert.equal(Object.hasOwn(search.parameters.properties.provider,"default"),false);
 const received=[];
 const server=createServer((req,res)=>{let raw="";req.on("data",chunk=>raw+=chunk);req.on("end",()=>{received.push(JSON.parse(raw));res.writeHead(200,{"Content-Type":"application/json"});res.end(JSON.stringify({delivery_state:"ready",artifact_url:"https://evidence.example/e/fixture",portable_url:"https://evidence.example/e/fixture"}))})});
 await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));const prior=process.env.UIAI_ENGINE_URL;process.env.UIAI_ENGINE_URL=`http://127.0.0.1:${server.address().port}`;
 const scope={project_ref:"synthetic-project",workstream_ref:"synthetic-stream",workset_ref:"synthetic-set",callgraph_ref:"synthetic-graph",workpoint_ref:"synthetic-point",work_item_ref:"synthetic-item",continuity_ref:"synthetic-continuity",work_items:[{work_item_ref:"synthetic-item",title:"Owned transport fixture"}]};
 try{for(const params of [{query:"fixture"},{query:"fixture",provider:"tavily"},{query:"fixture",providers:["brave","wikipedia"]},{query:"fixture",provider:"brave",providers:["brave","wikipedia"]}]){
  await search.execute("research",{...params,focusa_scope:scope});const body=received.at(-1);assert.equal(body.provider,params.provider);assert.equal(Object.hasOwn(body,"provider"),Object.hasOwn(params,"provider"));assert.deepEqual(body.providers,params.providers);assert.equal(Object.hasOwn(body,"providers"),Object.hasOwn(params,"providers"));assert.deepEqual(body.focusa_scope,scope);
 }}finally{if(prior===undefined)delete process.env.UIAI_ENGINE_URL;else process.env.UIAI_ENGINE_URL=prior;await new Promise(resolve=>server.close(resolve))}
});


test("native search errors retain safe setup next action",async()=>{
 const server=createServer((req,res)=>{req.resume();res.writeHead(503,{"content-type":"application/json"});res.end(JSON.stringify({error:"search_provider_unavailable",message:"Existing credential reference missing",suggested_next_action:"connect_existing_credential_reference: configure search.providers.brave.api_key_env; inspect /api/search/providers",private_extra:"synthetic-private-extra-never-echo"}))});
 await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));const prior=process.env.UIAI_ENGINE_URL;process.env.UIAI_ENGINE_URL=`http://127.0.0.1:${server.address().port}`;
 try{await assert.rejects(()=>tools.get("uiai_search").execute("recovery",{query:"fixture",provider:"brave"}),err=>err.status===503 && err.message.includes("Next: connect_existing_credential_reference") && err.message.includes("search.providers.brave.api_key_env") && !err.message.includes("synthetic-private-extra-never-echo"))}
 finally{if(prior===undefined)delete process.env.UIAI_ENGINE_URL;else process.env.UIAI_ENGINE_URL=prior;await new Promise(resolve=>server.close(resolve))}
});
