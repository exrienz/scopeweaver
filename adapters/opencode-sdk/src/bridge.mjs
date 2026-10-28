import { spawn } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { timingSafeEqual } from "node:crypto";
import { StringDecoder } from "node:string_decoder";
import { readFileSync } from "node:fs";
import { BridgeError, fail, publicError, safeJSON, parseRequest, validateProviderConfig, serializeComplete } from "./protocol.mjs";

// Each worker owns its SDK state, storage, environment, and one model request.
export const MAX_REQUEST_BYTES=16*1024*1024;
export function workerEnvironment(root,source=process.env) {
  const env={PATH:source.PATH??"/usr/bin:/bin",TMPDIR:root,OPENCODE_TEST_HOME:join(root,"home"),XDG_CONFIG_HOME:join(root,"xdg-config"),XDG_DATA_HOME:join(root,"xdg-data"),XDG_CACHE_HOME:join(root,"xdg-cache"),XDG_STATE_HOME:join(root,"xdg-state")};
  for(const key of ["SystemRoot","WINDIR","LANG","LC_ALL","TZ"])if(source[key])env[key]=source[key];
  return env;
}
export async function executeWorker(input,{signal,onEvent=()=>{},timeout=0}={}) {
  const root=await mkdtemp(join(tmpdir(),"scopeweaver-opencode-worker-"));
  const env=workerEnvironment(root);
  let child,timer,killTimer,canceled=false,timedOut=false;
  try {
    return await new Promise((resolve,reject)=>{
      const grouped=process.platform!=="win32";
      child=spawn(process.execPath,["--no-env-file",fileURLToPath(new URL("./worker.mjs",import.meta.url))],{cwd:root,env,detached:grouped,stdio:["pipe","pipe","pipe"]});
      let buffer="",size=0,result,error,pendingStop;const decoder=new StringDecoder("utf8");
      const kill=signal=>{try{if(grouped&&child.pid)process.kill(-child.pid,signal);else child.kill(signal)}catch{}};
      const stop=()=>{kill("SIGTERM");killTimer??=setTimeout(()=>kill("SIGKILL"),1500)};
      const abort=()=>{canceled=true;stop()};signal?.addEventListener("abort",abort,{once:true});
      if(timeout>0)timer=setTimeout(()=>{timedOut=true;stop()},timeout);
      if(signal?.aborted)abort();
      child.stderr.on("data",()=>{}); // Provider errors may contain credentials; never forward stderr.
      child.stdout.on("data",chunk=>{
        size+=chunk.length;if(size>20000000){error=new BridgeError("protocol_error",502);stop();return}
        buffer+=decoder.write(chunk);
        let index;while((index=buffer.indexOf("\n"))>=0){
          const line=buffer.slice(0,index);buffer=buffer.slice(index+1);if(!line)continue;
          try {
            const value=safeJSON(line,"protocol_error");
            if(value.kind==="event"){
              if(value.event.type==="message_stop"){if(pendingStop)throw new BridgeError("protocol_error",502);pendingStop=value.event}
              else onEvent(value.event);
            }
            else if(value.kind==="result")result=value.result;
            else if(value.kind==="models")result=value.models;
            else if(value.kind==="error")error=new BridgeError(value.code,({unknown_model:400,unsupported_input:422,invalid_request:400,configuration_error:400,canceled:499,timeout:504,quota_exhausted:402,context_length_exceeded:400,free_tier_restricted:403})[value.code]??502);
            else throw new BridgeError("protocol_error",502);
          }catch(e){error=e;stop()}
        }
      });
      child.on("error",()=>{error=new BridgeError("sdk_failure",502)});
      child.on("close",code=>{
        signal?.removeEventListener("abort",abort);clearTimeout(timer);clearTimeout(killTimer);
        kill("SIGKILL"); // An SDK-owned subprocess must not outlive its request worker.
        if(timedOut)reject(new BridgeError("timeout",504));
        else if(canceled||signal?.aborted)reject(new BridgeError("canceled",499));
        else if(error)reject(error);
        else if(code!==0||result===undefined||(buffer+decoder.end()).trim()||(input.mode!=="models"&&!pendingStop))reject(new BridgeError("sdk_failure",502));
        else {try{if(pendingStop)onEvent(pendingStop);resolve(result)}catch{reject(new BridgeError("canceled",499))}}
      });
      child.stdin.on("error",()=>{});
      child.stdin.end(JSON.stringify({...input,root}));
    });
  } finally {clearTimeout(timer);clearTimeout(killTimer);await rm(root,{recursive:true,force:true})}
}

const encoder=new TextEncoder();
const nativeVersionMatches=["@opencode/cli/package.json","@opencode/sdk","@opencode/client","@opencode/plugin","@opencode/ai","@opencode/core/config","@opencode/util/global"].every(id=>{
  try{
    let path=dirname(fileURLToPath(import.meta.resolve(id)));
    const name=id.split("/").slice(0,2).join("/");
    while(path!==dirname(path)){try{const p=JSON.parse(readFileSync(join(path,"package.json"),"utf8"));if(p.name===name)return p.version==="2.1.4"}catch{}path=dirname(path)}
    return false;
  }catch{return false}
});
const legacyVersionMatches=["@opencode-ai/sdk/v2","opencode-ai/package.json"].every(id=>{
  try{
    let path=dirname(fileURLToPath(import.meta.resolve(id)));
    const name=id.startsWith("@")?id.split("/").slice(0,2).join("/"):id.split("/")[0];
    while(path!==dirname(path)){try{const p=JSON.parse(readFileSync(join(path,"package.json"),"utf8"));if(p.name===name)return p.version==="1.19.4"}catch{}path=dirname(path)}
    return false;
  }catch{return false}
});
const json=(value,status=200)=>new Response(JSON.stringify(value),{status,headers:{"content-type":"application/json","cache-control":"no-store"}});
async function readBody(req) {
  if(Number(req.headers.get("content-length"))>MAX_REQUEST_BYTES)fail("body_too_large",413);
  const reader=req.body?.getReader();if(!reader)fail("invalid_request");
  const chunks=[];let size=0;
  try{while(true){const {done,value}=await reader.read();if(done)break;size+=value.byteLength;if(size>MAX_REQUEST_BYTES){await reader.cancel();fail("body_too_large",413)}chunks.push(value)}}finally{reader.releaseLock()}
  return new TextDecoder("utf-8",{fatal:true}).decode(Buffer.concat(chunks));
}
export function createBridge({token,providerConfig,timeout=0,runtime="native",worker=executeWorker}={}) {
  if(typeof token!=="string"||token.length<32||/[\s\r\n]/.test(token))throw new Error("SCOPEWEAVER_OPENCODE_TOKEN must contain at least 32 non-whitespace characters");
  if(!Number.isSafeInteger(timeout)||timeout<0||timeout>86400000)throw new Error("Bridge timeout must be 0..86400000 milliseconds");
  providerConfig=validateProviderConfig(providerConfig);
  const active=new Map();let closed=false;
  const authorize=req=>{
    const u=new URL(req.url);
    if(req.headers.has("origin")||!["127.0.0.1","localhost","[::1]"].includes(u.hostname)||u.username||u.password||u.search||u.hash)fail("forbidden",403);
    const supplied=req.headers.get("authorization")??"",expected=`Bearer ${token}`;
    if(Buffer.byteLength(supplied)!==Buffer.byteLength(expected)||!timingSafeEqual(Buffer.from(supplied),Buffer.from(expected)))fail("unauthorized",401);
    if(!((runtime==="native"&&nativeVersionMatches&&process.platform!=="win32")||(runtime==="legacy"&&legacyVersionMatches&&process.platform!=="win32")))fail("unsupported_runtime",400);
  };
  const controllerFor=req=>{
    const controller=new AbortController();const abort=()=>controller.abort();req.signal.addEventListener("abort",abort,{once:true});if(req.signal.aborted)abort();
    return {controller,dispose:()=>req.signal.removeEventListener("abort",abort)};
  };
  return {
    async fetch(req){
      try {
        authorize(req);if(closed)fail("canceled",503);
        const path=new URL(req.url).pathname;
        if(req.method==="GET"&&path==="/v1/capabilities")return json({protocol:"scopeweaver-opencode-sdk",version:1,sdk_version:runtime==="native"?"2.1.4":"1.19.4",runtime,tool_execution:"capture-only",streaming:true});
        if(req.method==="DELETE"&&/^\/v1\/requests\/[A-Za-z0-9_-]{1,128}$/.test(path)){active.get(path.slice("/v1/requests/".length))?.abort();return json({aborted:true})}
        if(req.method==="GET"&&path==="/v1/models"){
          const {controller,dispose}=controllerFor(req);const key=Symbol();active.set(key,controller);
          try{return json({models:await worker({mode:"models",runtime,providerConfig},{signal:controller.signal,timeout})})}finally{active.delete(key);dispose()}
        }
        if(req.method!=="POST"||!["/v1/complete","/v1/stream"].includes(path))fail("not_found",404);
        if(!req.headers.get("content-type")?.toLowerCase().startsWith("application/json"))fail("invalid_request",415);
        const {request,rawHistory}=parseRequest(await readBody(req));
        if(active.has(request.request_id))fail("duplicate_request",409);
        const {controller,dispose}=controllerFor(req);active.set(request.request_id,controller);
        const cleanup=()=>{active.delete(request.request_id);dispose()};
        if(path==="/v1/complete"){
          try{const result=await worker({mode:"complete",runtime,request,rawHistory,providerConfig},{signal:controller.signal,timeout});return new Response(serializeComplete(result),{headers:{"content-type":"application/json","cache-control":"no-store"}})}finally{cleanup()}
        }
        const stream=new ReadableStream({start(output){
          void worker({mode:"complete",runtime,request,rawHistory,providerConfig},{signal:controller.signal,timeout,onEvent:event=>output.enqueue(encoder.encode(JSON.stringify(event)+"\n"))})
            .then(()=>{output.close()},error=>{try{output.enqueue(encoder.encode(JSON.stringify({type:"error",...publicError(error)})+"\n"));output.close()}catch{}}).finally(cleanup);
        },cancel(){controller.abort()}});
        return new Response(stream,{headers:{"content-type":"application/x-ndjson","cache-control":"no-store"}});
      }catch(error){return json({error:publicError(error)},error instanceof BridgeError?error.status:400)}
    },
    async close(){closed=true;for(const c of active.values())c.abort();while(active.size)await new Promise(resolve=>setTimeout(resolve,20));},
  };
}
