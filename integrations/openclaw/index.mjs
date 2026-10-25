import { definePluginEntry } from "openclaw/plugin-sdk/plugin-entry";
import { resolveSecretRefValues } from "openclaw/plugin-sdk/runtime-secret-resolution";
import { registerUIAITools } from "./tools.mjs";

export default definePluginEntry({
 id:"uiai-engine",name:"UIAI Engine",description:"UIAI-only browser tools with no alternate browser fallback",
 register(api){registerUIAITools(api,{resolveSecret:async(value,config)=>{
  if(typeof value==="string")return value;
  if(!value||typeof value!=="object")throw new Error("credential_resolution_failed");
  const values=await resolveSecretRefValues([value],{config});
  if(values.size!==1)throw new Error("credential_resolution_failed");
  return values.values().next().value;
 }});},
});
