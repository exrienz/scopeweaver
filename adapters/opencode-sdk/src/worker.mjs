import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fail, publicError } from "./protocol.mjs";
const abort=new AbortController();
// This process handles one call. Set policy before lazily importing either SDK.
const platformFetch=globalThis.fetch;
globalThis.fetch=(input,init)=>platformFetch(input,{...init,redirect:"error"});
process.on("SIGTERM",()=>abort.abort());process.on("SIGINT",()=>abort.abort());
const write=value=>process.stdout.write(JSON.stringify(value)+"\n");
let root;
try {
  const input=JSON.parse(await new Response(Bun.stdin.stream()).text());
  root=input.root??await mkdtemp(join(tmpdir(),"scopeweaver-opencode-"));
  const runtime=input.runtime??"native";let discover,complete;
  if(runtime==="native"){const backend=await import("./native.mjs");discover=backend.discoverModels;complete=backend.completeNative}
  else if(runtime==="legacy"){const backend=await import("./legacy.mjs");discover=backend.discoverLegacyModels;complete=backend.completeLegacy}
  else fail("unsupported_runtime");
  if(input.mode==="models")write({kind:"models",models:await discover({root,providerConfig:input.providerConfig})});
  else write({kind:"result",result:await complete(input.request,{root,rawHistory:input.rawHistory,providerConfig:input.providerConfig,signal:abort.signal,emit:event=>write({kind:"event",event})})});
} catch(e){write({kind:"error",...publicError(e)});process.exitCode=1;}
finally{if(root)await rm(root,{recursive:true,force:true});}
