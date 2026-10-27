// Owned synthetic HTTP/stdio fixture: no real scope or publication is asserted.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {spawn} from 'node:child_process';
import {createInterface} from 'node:readline';
import {readEngineResponse} from './engine-response.mjs';
import {assertEvidenceDelivery} from './epwa-contract.mjs';

const artifact='uiai-artifact:sha256:'+'a'.repeat(64);
const blocked={delivery_state:'blocked',artifact_ref:artifact,epwa_delivery:{schema:'uiai.epwa_delivery.v1',state:'blocked',artifact:{artifact_ref:artifact},recovery_ref:'reconcile:epwa-scope-required'},private_title:'synthetic-private-title',focusa_scope:{project_ref:'synthetic-private-project'}};
function checkEnvelope(envelope){
 assert.equal(envelope.schema,'uiai.epwa_delivery_error.v1');
 assert.equal(envelope.code,'epwa_delivery_unavailable');
 assert.equal(envelope.delivery_state,'blocked');
 assert.equal(envelope.artifact_ref,artifact);
 assert.equal(envelope.recovery_ref,'reconcile:epwa-scope-required');
 const body=JSON.stringify(envelope);
 for(const secret of ['synthetic-private-title','synthetic-private-project','synthetic-private-bytes','artifact_url','portable_url']) assert.ok(!body.includes(secret),secret);
}
test('shared browser delivery failure is a typed degraded envelope, not discarded or successful',async()=>{
 for(const body of [blocked,{...blocked,screenshot:'synthetic-private-bytes'}]){
  await assert.rejects(readEngineResponse(Response.json(body),{validate:data=>assertEvidenceDelivery(data,true)}),error=>{
   assert.equal(error.code,'epwa_delivery_unavailable');
   checkEnvelope(error.delivery_error);
   assert.deepEqual(JSON.parse(error.message),error.delivery_error);
   return true;
  });
 }
});
test('actual MCP screenshot failures preserve typed metadata and isError without raw content',async()=>{
 let raw=false;
 const server=createServer((req,res)=>{res.setHeader('Content-Type','application/json');res.end(JSON.stringify(raw?{...blocked,screenshot:'synthetic-private-bytes'}:blocked));});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 const child=spawn(process.execPath,[new URL('../../../mcp/browser-session-mcp.mjs',import.meta.url).pathname],{env:{...process.env,UIAI_ENGINE_URL:`http://127.0.0.1:${server.address().port}`},stdio:['pipe','pipe','pipe']});
 const lines=createInterface({input:child.stdout});
 try{
  let id=0;
  for(const name of ['browser_screenshot','screenshot']){
   for(raw of [false,true]){
    const requestID=++id;
    const reply=new Promise((resolve,reject)=>{
     const timer=setTimeout(()=>{lines.off('line',onLine);reject(new Error('MCP fixture timed out'));},5000);
     function onLine(line){const value=JSON.parse(line);if(value.id===requestID){clearTimeout(timer);lines.off('line',onLine);resolve(value);}}
     lines.on('line',onLine);
    });
    child.stdin.write(JSON.stringify({jsonrpc:'2.0',id:requestID,method:'tools/call',params:{name,arguments:name==='screenshot'?{url:'https://example.test'}:{session_id:'owned-fixture'}}})+'\n');
    const response=await reply;
    assert.equal(response.result.isError,true);
    checkEnvelope(response.result.structuredContent);
    assert.deepEqual(JSON.parse(response.result.content[0].text),response.result.structuredContent);
   }
  }
 }finally{lines.close();child.kill();await new Promise(resolve=>server.close(resolve));}
});
