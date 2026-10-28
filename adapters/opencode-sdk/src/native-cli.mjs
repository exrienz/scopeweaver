import { OpenCode } from "@opencode/client";
import { spawn } from "node:child_process";
import { createRequire } from "node:module";
import { mkdir, readFile, writeFile, lstat } from "node:fs/promises";
import { dirname, join } from "node:path";
import { userInfo } from "node:os";
import { pathToFileURL } from "node:url";
import { BridgeError, fail } from "./protocol.mjs";

const require=createRequire(import.meta.url);
export const nativeRules=[{action:"*",resource:"*",effect:"ask"},{action:"host_capture",resource:"*",effect:"allow"}];
export function unmanagedConfigPaths(platform=process.platform,user=userInfo().username,env=process.env){
  if(platform==="darwin")return ["/Library/Application Support/opencode",`/Library/Managed Preferences/${user}/ai.opencode.managed.plist`,"/Library/Managed Preferences/ai.opencode.managed.plist"];
  return [platform==="win32"?join(env.ProgramData??"C:\\ProgramData","opencode"):"/etc/opencode"];
}
export async function assertNoUnmanagedConfig(paths=unmanagedConfigPaths()){
  for(const path of paths){
    try{await lstat(path);fail("configuration_error");}catch(error){if(error.code!=="ENOENT")throw error instanceof BridgeError?error:new BridgeError("configuration_error");}
  }
}
async function binaryPath(){
  if(!["darwin","linux"].includes(process.platform)||!["x64","arm64"].includes(process.arch))fail("unsupported_runtime");
  const platform=process.platform==="win32"?"windows":process.platform;
  const base=`@opencode/cli-${platform}-${process.arch}`;
  const names=process.platform==="linux"?(process.arch==="x64"?[`${base}-baseline`,`${base}-baseline-musl`,base,`${base}-musl`]:[base,`${base}-musl`]):process.arch==="x64"&&process.platform!=="win32"?[`${base}-baseline`,base]:[base];
  for(const name of names){
    try{
      const file=require.resolve(`${name}/package.json`),metadata=JSON.parse(await readFile(file,"utf8"));
      if(metadata.version!=="2.1.4")continue;
      const binary=join(dirname(file),"bin",process.platform==="win32"?"opencode.exe":"opencode");
      const valid=await new Promise(resolve=>{
        const child=spawn(binary,["--version"],{stdio:"ignore",windowsHide:true});
        const timer=setTimeout(()=>{child.kill("SIGKILL");resolve(false)},5000);
        child.on("error",()=>{clearTimeout(timer);resolve(false)});child.on("exit",code=>{clearTimeout(timer);resolve(code===0)});
      });
      if(valid)return binary;
    }catch{}
  }
  fail("unsupported_runtime");
}

export async function createNativeCLI({root,providerConfig,request,history,tools,rawHistory,signal,onControl}){
  await assertNoUnmanagedConfig();
  const binary=await binaryPath(),channel=crypto.randomUUID(),password=crypto.randomUUID()+crypto.randomUUID();
  const paths=Object.fromEntries(["home","config","data","cache","state","tmp","work"].map(name=>[name,join(root,`cli-${name}`)]));
  for(const path of Object.values(paths))await mkdir(path,{recursive:true});
  const plugin=join(paths.config,"host-capture");await mkdir(plugin,{recursive:true});
  await writeFile(join(plugin,"package.json"),JSON.stringify({name:"scopeweaver-host-capture",type:"module",main:"index.mjs"}));
  await writeFile(join(plugin,"index.mjs"),await readFile(new URL("./native-plugin.mjs",import.meta.url)));
  const input={request,history,tools,rawHistory,channel},inputPath=join(plugin,"bridge-input.json");
  const setRequest=async next=>{input.request=next;await writeFile(inputPath,JSON.stringify(input),{mode:0o600})};
  await setRequest(request);
  const providers=structuredClone(providerConfig?.providers??{}),split=request.model.indexOf("/"),provider=request.model.slice(0,split),model=request.model.slice(split+1);
  providers[provider]??={};providers[provider].models??={};providers[provider].models[model]??={};
  const selected=providers[provider].models[model];
  selected.body={...(request.thinking||request.thinking_type?{thinking:{type:request.thinking||request.thinking_type}}:{}),...(request.reasoning_effort?{reasoning_effort:request.reasoning_effort}:{})};
  selected.compatibility={...selected.compatibility,maxTokensField:selected.compatibility?.maxTokensField??"max_tokens"};
  const config={providers,plugins:[pathToFileURL(plugin).href],default_agent:"capture",agents:{capture:{mode:"primary",system:request.system.join("\n\n"),permissions:nativeRules}},permissions:nativeRules,update:"disable",share:"disabled",snapshots:false,warming:false,compaction:{auto:false}};
  const configPath=join(paths.config,"opencode.json");await writeFile(configPath,JSON.stringify(config),{mode:0o600});
  const env={PATH:process.env.PATH??"/usr/bin:/bin",PWD:paths.work,HOME:paths.home,USERPROFILE:paths.home,TMPDIR:paths.tmp,OPENCODE_TEST_HOME:paths.home,XDG_CONFIG_HOME:paths.config,XDG_DATA_HOME:paths.data,XDG_CACHE_HOME:paths.cache,XDG_STATE_HOME:paths.state,OPENCODE_CONFIG_DIR:paths.config,OPENCODE_CONFIG:configPath,OPENCODE_CONFIG_PROJECT_DISABLE:"1",OPENCODE_DISABLE_CLAUDE_CODE:"1",OPENCODE_PASSWORD:password};
  for(const key of ["SystemRoot","WINDIR","LANG","LC_ALL","TZ"])if(process.env[key])env[key]=process.env[key];
  const children=new Set();let client,runner,runnerExit,egress,closed=false;
  const start=args=>{
    if(signal.aborted)throw new BridgeError("canceled",499);
    const child=spawn(binary,args,{cwd:paths.work,env,windowsHide:true,stdio:["pipe","pipe","pipe"]});children.add(child);
    child.stdin.on("error",()=>{});child.stderr.on("data",()=>{});child.once("close",()=>children.delete(child));return child;
  };
  const close=async()=>{
    if(closed)return;closed=true;signal.removeEventListener("abort",abort);
    egress?.stop(true);
    for(const child of children){child.stdin.end();child.kill("SIGTERM")}
    if(children.size)await Promise.race([Promise.all([...children].map(child=>new Promise(resolve=>child.once("close",resolve)))),new Promise(resolve=>setTimeout(resolve,200))]);
    const remaining=[...children],exited=Promise.all(remaining.map(child=>new Promise(resolve=>child.once("close",resolve))));
    for(const child of remaining)child.kill("SIGKILL");
    if(remaining.length){
      let timer;try{await Promise.race([exited,new Promise((_,reject)=>{timer=setTimeout(()=>reject(new BridgeError("sdk_failure",502)),1000)})])}finally{clearTimeout(timer)}
    }
  };
  const abort=()=>{if(client)void client.session.interrupt({sessionID:activeSession,resume:false}).catch(()=>{});for(const child of children)child.kill("SIGTERM")};
  let activeSession;
  signal.addEventListener("abort",abort,{once:true});
  try{
    const token=crypto.randomUUID()+crypto.randomUUID();let sent=false;
    // Raw tool history becomes JSON inside JSON; allow its encoded expansion
    // while the public bridge continues to enforce its 16 MiB input limit.
    egress=Bun.serve({hostname:"127.0.0.1",port:0,idleTimeout:0,maxRequestBodySize:64*1024*1024,async fetch(incoming){
      if(incoming.method!=="POST"||new URL(incoming.url).pathname!=="/model"||incoming.headers.has("origin")||incoming.headers.get("x-scopeweaver-egress")!==token||sent)return new Response(null,{status:403});
      sent=true;
      try{
        const target=new URL(incoming.headers.get("x-scopeweaver-upstream")??"");
        if(target.username||target.password||target.hash||!["http:","https:"].includes(target.protocol))throw Error();
        const expected=providerConfig?new URL(providerConfig.providers[provider].settings.baseURL).origin:"https://opencode.ai";
        if(target.origin!==expected)throw Error();
        const headers=new Headers(incoming.headers);
        for(const name of ["host","connection","content-length","x-scopeweaver-egress","x-scopeweaver-upstream"])headers.delete(name);
        const response=await fetch(target,{method:"POST",headers,body:await incoming.arrayBuffer(),redirect:"error",signal:AbortSignal.any([signal,incoming.signal])});
        // Fetch exposes decoded bytes but retains the upstream encoding header.
        // The downstream CLI must not attempt to decode those bytes again.
        const received=new Headers(response.headers);
        for(const name of ["content-encoding","content-length","transfer-encoding","connection"])received.delete(name);
        return new Response(response.body,{status:response.status,headers:received});
      }catch{return new Response(null,{status:502})}
    }});
    input.egress={url:`http://127.0.0.1:${egress.port}/model`,token};await setRequest(request);
    const server=start(["serve","--stdio","--hostname","127.0.0.1","--port","0"]);
    const url=await new Promise((resolve,reject)=>{
      let buffer="";const timer=setTimeout(()=>reject(new BridgeError("sdk_failure",502)),15000);
      const done=(error,value)=>{clearTimeout(timer);error?reject(error):resolve(value)};
      server.stdout.on("data",chunk=>{
        buffer+=chunk.toString();if(buffer.length>1000000){done(new BridgeError("protocol_error",502));server.kill("SIGTERM");return}
        let index;while((index=buffer.indexOf("\n"))>=0){
          const line=buffer.slice(0,index);buffer=buffer.slice(index+1);let value;try{value=JSON.parse(line)}catch{continue}
          if(value._scopeweaver_native===channel){try{onControl(value.event)}catch{server.kill("SIGTERM")}}
          else if(typeof value.url==="string"){
            const parsed=new URL(value.url);if(parsed.protocol!=="http:"||parsed.hostname!=="127.0.0.1"||parsed.username||parsed.password||parsed.pathname!=="/")done(new BridgeError("protocol_error",502));else done(undefined,value.url);
          }
        }
      });
      server.once("error",()=>done(new BridgeError("sdk_failure",502)));server.once("exit",()=>done(new BridgeError(signal.aborted?"canceled":"sdk_failure",502)));
    });
    client=OpenCode.make({baseUrl:url,headers:{Authorization:`Basic ${Buffer.from(`opencode:${password}`).toString("base64")}`}});
    if((await client.server.info()).version!=="2.1.4")fail("unsupported_runtime");
    return {...client,directory:paths.work,setRequest,close,session:{...client.session,async prompt({sessionID}){
      activeSession=sessionID;
      runner=start(["run","--server",url,"--session",sessionID,"--model",request.model,"--agent","capture","--format","json"]);
      runnerExit=new Promise(resolve=>{runner.once("error",()=>resolve(-1));runner.once("exit",resolve)});
      let size=0;runner.stdout.on("data",chunk=>{size+=chunk.length;if(size>20000000){onControl({type:"failure",code:"protocol_error"});runner.kill("SIGTERM")}});
      runner.stdin.end("ScopeWeaver authoritative conversation is supplied by the host plugin.");
    }},async waitRunner(){if(runnerExit&&(await runnerExit)!==0)fail("sdk_failure",502)}};
  }catch(error){await close();throw error}
}
