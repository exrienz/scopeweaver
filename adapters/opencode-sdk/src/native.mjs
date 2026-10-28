import { OpenCode } from "@opencode/sdk";
import { createNativeCLI, nativeRules } from "./native-cli.mjs";
import { Config } from "@opencode/core/config";
import { Global } from "@opencode/util/global";
import { mkdir } from "node:fs/promises";
import { BridgeError, fail, safeJSON, rawObject, aliasMap, nativeHistory, zeroPrice } from "./protocol.mjs";

const rules=nativeRules;
const abortError = () => new BridgeError("canceled",499);
const signature = native => {
  const found=[];
  const walk=(v,d=0)=>{if(!v||typeof v!=="object"||d>3)return;for(const [k,x]of Object.entries(v)){if(["signature","encrypted"].includes(k)&&typeof x==="string")found.push(x);else if(typeof x==="object")walk(x,d+1)}};
  walk(native);const unique=[...new Set(found)];if(unique.length>1)fail("protocol_error");return unique[0];
};
function usage(tokens) {
  const result={input_tokens:tokens?.input,output_tokens:tokens?.output,cache_read_tokens:tokens?.cache?.read??0,cache_write_tokens:tokens?.cache?.write??0};
  if(!Object.values(result).every(n=>Number.isSafeInteger(n)&&n>=0))fail("protocol_error");
  return result;
}
async function createHost(root, providerConfig, plugins=[], request) {
  const paths=Object.fromEntries(["home","data","cache","config","state","tmp","bin","log","repos"].map(k=>[k,`${root}/${k}`]));
  await mkdir(paths.config,{recursive:true});
  const providers=structuredClone(providerConfig?.providers??{});
  if(request) {
    const split=request.model.indexOf("/");const provider=request.model.slice(0,split),model=request.model.slice(split+1);
    providers[provider]??={};providers[provider].models??={};providers[provider].models[model]??={};
    const selected=providers[provider].models[model];
    selected.body={...(request.thinking||request.thinking_type?{thinking:{type:request.thinking||request.thinking_type}}:{}),...(request.reasoning_effort?{reasoning_effort:request.reasoning_effort}:{})};
    selected.compatibility={...selected.compatibility,maxTokensField:selected.compatibility?.maxTokensField??"max_tokens"};
  }
  const content=JSON.stringify({providers,plugins:[],update:"disable",share:"disabled",snapshots:false,warming:false,compaction:{auto:false},permissions:rules,default_agent:"capture",agents:{capture:{mode:"primary",system:"",permissions:rules}}});
  return OpenCode.create({app:{name:"sdk",version:"2.1.4",channel:"latest"},database:{path:":memory:"},models:{fetch:!providerConfig,snapshot:!providerConfig},config:{directory:paths.config,project:false,claude:false,content},fs:{filewatcher:false,fff:false},plugins}, {overrides:[Global.node.replace(Global.layerWith(paths)),Config.node.replace(Config.configured({global:false,system:false,project:false,claude:false,content}))]});
}
export async function discoverModels({root,providerConfig}) {
  let publicIDs;
  if(!providerConfig) {
    try {
      const response=await fetch("https://opencode.ai/inference/v1/models",{redirect:"error",signal:AbortSignal.timeout(10000)});
      if(!response.ok)throw Error();
      const catalog=safeJSON(await response.text(),"sdk_failure");
      if(!Array.isArray(catalog.data))throw Error();
      publicIDs=new Set(catalog.data.map(m=>m.id));
    } catch { fail("sdk_failure",502); }
  }
  const host=await createHost(root,providerConfig);
  try {
    const {data}=await host.model.list({location:{directory:`${root}/config`}});
    const models=data.filter(m=>providerConfig?Object.hasOwn(providerConfig.providers,m.providerID)&&Object.hasOwn(providerConfig.providers[m.providerID].models,m.id):m.providerID==="opencode"&&publicIDs.has(m.id)&&zeroPrice(m.cost))
      .map(m=>({id:`${m.providerID}/${m.id}`,name:m.name??m.id})).sort((a,b)=>a.id.localeCompare(b.id));
    if(!models.length)fail("sdk_failure",502);
    return models;
  } finally { await host.close(); }
}

export async function completeNative(request,{root,providerConfig,rawHistory={},signal,emit}) {
  // Validate against the original catalog, before a per-call model overlay can create a new model.
  if(providerConfig) {
    const split=request.model.indexOf("/");
    if(!Object.hasOwn(providerConfig.providers,request.model.slice(0,split))||!Object.hasOwn(providerConfig.providers[request.model.slice(0,split)].models,request.model.slice(split+1)))fail("unknown_model");
  } else if(!(await discoverModels({root:`${root}/catalog`,providerConfig})).some(m=>m.id===request.model))fail("unknown_model");
  const names=[...new Set([...request.tools.map(t=>t.name),...request.messages.flatMap(m=>m.content.filter(b=>b.type==="tool_use").map(b=>b.name))])];
  const forward=new Map(names.map((name,index)=>[name,`sw_tool_${index}`])),aliases={forward,reverse:new Map([...forward].map(([name,alias])=>[alias,name]))},history=nativeHistory(request,aliases);
  const exposed=new Set(request.tools.map(t=>aliases.forward.get(t.name)));
  const oldIDs=new Set(request.messages.flatMap(m=>m.content.filter(b=>b.type==="tool_use").map(b=>b.id)));
  const calls=new Map(),rawChunks=new Map(),captures=new Set(),blocks=new Map(),content=[];let outputSize=0;
  let host,session,consumer,error,ownedInterrupt=false,streamed=false,terminal=false,rawFinish,tokens,network=0,wake,quotaEvidence=false,contextEvidence=false,freeTierEvidence=false,eventsReady=false;
  const eventAbort=new AbortController();
  const notify=()=>{wake?.();wake=undefined};
  const reject=(code="protocol_error")=>{error??=new BridgeError(code,502);notify()};
  const aborted=()=>{reject("canceled");if(session)void host.session.interrupt({sessionID:session.id,resume:false}).catch(()=>{})};
  signal.addEventListener("abort",aborted,{once:true});
  const wait=async predicate=>{while(!predicate()){if(signal.aborted)throw abortError();if(error)throw error;await new Promise(resolve=>{wake=resolve;if(predicate()||error||signal.aborted)notify()})}if(error)throw error;};
  const onControl=event=>{
    if(event.type==="capture"){captures.add(event.id);notify()}
    else if(event.type==="request"){network++;if(network!==1)reject("second_model_request")}
    else if(event.type==="failure")reject(event.code);
    else if(event.type==="evidence"){
      if(event.code==="quota_exhausted")quotaEvidence=true;
      if(event.code==="context_length_exceeded")contextEvidence=true;
      if(event.code==="free_tier_restricted")freeTierEvidence=true;
    }
  };
  const consume=e=>{
    const d=e.data;if(!session||d?.sessionID!==session.id)return;
    try {
      if(e.type==="session.text.started"||e.type==="session.reasoning.started"){
        const type=e.type.includes("reasoning")?"thinking":"text",key=`${type}:${d.ordinal}`;
        if(blocks.has(key))fail("protocol_error");
        const block={type,[type==="thinking"?"thinking":"text"]:""};blocks.set(key,block);content.push(block);
      } else if(e.type==="session.text.delta"||e.type==="session.reasoning.delta") {
        outputSize+=d.delta.length;if(outputSize>8000000)fail("protocol_error");
        const type=e.type.includes("reasoning")?"thinking":"text",key=`${type}:${d.ordinal}`;
        if(!blocks.has(key)){const block={type,[type==="thinking"?"thinking":"text"]:""};blocks.set(key,block);content.push(block)}
        blocks.get(key)[type==="thinking"?"thinking":"text"]+=d.delta;
        emit({type:type==="thinking"?"thinking_delta":"text_delta",text:d.delta});
      } else if(e.type==="session.reasoning.ended") {
        const sig=signature(d.native);if(sig){const b=blocks.get(`thinking:${d.ordinal}`);if(!b)fail("protocol_error");b.signature=sig;emit({type:"thinking_signature",text:sig})}
      } else if(e.type==="session.tool.input.started") {
        if(!exposed.has(d.name)||calls.has(d.id)||!d.id)fail("protocol_error");
        let index=calls.size,id=`sdk_${request.request_id}_${index}`;
        while(oldIDs.has(id))id=`sdk_${request.request_id}_${++index}`;
        oldIDs.add(id);
        const call={type:"tool_use",id,name:aliases.reverse.get(d.name)};calls.set(d.id,call);content.push(call);
      } else if(e.type==="session.tool.input.delta") {
        if(!calls.has(d.id)||calls.get(d.id).raw!==undefined)fail("protocol_error");
        outputSize+=d.delta.length;if(outputSize>8000000)fail("protocol_error");
        rawChunks.set(d.id,(rawChunks.get(d.id)??"")+d.delta);
      } else if(e.type==="session.tool.input.ended") {
        const call=calls.get(d.id);if(!call||call.raw!==undefined)fail("protocol_error");
        if(rawChunks.has(d.id)&&rawChunks.get(d.id)!==d.text)fail("protocol_error");
        rawObject(d.text);call.raw=d.text;
      } else if(e.type==="session.tool.called") {if(d.executed!==false||!calls.has(d.id))fail("protocol_error");}
      else if(e.type==="session.tool.failed") {if(!ownedInterrupt||d.error?.type!=="aborted")fail("protocol_error");}
      else if(e.type==="session.step.streamed")streamed=true;
      else if(e.type==="session.step.ended"||e.type==="session.step.failed") {
        if(e.type.endsWith("failed")&&!ownedInterrupt)fail("sdk_failure");
        tokens=usage(d.tokens);rawFinish=d.rawFinish??d.finish;
      } else if(e.type==="session.execution.failed") {
        reject(d.error?.message==="SECOND_MODEL_REQUEST_BLOCKED"?"second_model_request":"sdk_failure");terminal=true;
      } else if(e.type==="session.execution.succeeded"||e.type==="session.execution.interrupted")terminal=true;
    } catch(x) { reject(x.code); }
    notify();
  };
  let result;
  try {
    if(signal.aborted)throw abortError();
    host=await createNativeCLI({root,providerConfig,request,history,rawHistory,signal,onControl,tools:request.tools.map(t=>({name:aliases.forward.get(t.name),description:`Host tool: ${JSON.stringify(t.name)}. Use this function when host instructions refer to that tool.\n\n${t.description}`,input:t.input_schema}))});
    const available=await host.model.list({location:{directory:host.directory}});
    const model=available.data.find(m=>`${m.providerID}/${m.id}`===request.model);
    if(!model||(!providerConfig&&(model.providerID!=="opencode"||!zeroPrice(model.cost))))fail("unknown_model");
    if(model.limit?.output&&request.max_tokens>model.limit.output) {
      // The requested upper bound is preserved when the model has a smaller hard limit.
      request={...request,max_tokens:model.limit.output};
    }
    await host.setRequest(request);
    const split=request.model.indexOf("/");
    consumer=(async()=>{try{for await(const e of host.event.subscribe({signal:eventAbort.signal})){if(e.type==="server.connected"){eventsReady=true;notify()}else consume(e)}}catch{if(!eventAbort.signal.aborted)reject("sdk_failure")}})();
    await wait(()=>eventsReady);
    session=await host.session.create({location:{directory:host.directory},agent:"capture",model:{providerID:request.model.slice(0,split),id:request.model.slice(split+1)},permissions:rules,title:"ScopeWeaver model request"});
    emit({type:"message_start"});
    await host.session.prompt({sessionID:session.id,text:"ScopeWeaver authoritative context is supplied by the host."});
    await wait(()=>streamed||terminal);
    if(calls.size) {
      await wait(()=>[...calls.entries()].every(([id,c])=>c.raw!==undefined&&captures.has(id))||terminal);
      if(terminal||!calls.size)fail("protocol_error");
      ownedInterrupt=true;
      await host.session.interrupt({sessionID:session.id,resume:false});
    }
    await wait(()=>terminal);
    if(!ownedInterrupt)await host.waitRunner();
    if(!tokens||network!==1)fail("protocol_error");
    const stop_reason=calls.size?"tool_use":["length","max_tokens","max-tokens"].includes(rawFinish)?"max_tokens":["stop","end_turn","end-turn"].includes(rawFinish)?"end_turn":undefined;
    if(!stop_reason)fail("protocol_error");
    result={message:{role:"assistant",content},stop_reason,usage:tokens};
  } catch(e) { error=e instanceof BridgeError?e:new BridgeError(signal.aborted?"canceled":"sdk_failure",502); }
  finally {
    if(session){try{await host.session.interrupt({sessionID:session.id,resume:false});await host.session.wait({sessionID:session.id});await host.session.remove({sessionID:session.id})}catch{error??=new BridgeError("sdk_failure",502)}}
    eventAbort.abort();if(consumer)await consumer;
    if(host){try{await host.close()}catch{error??=new BridgeError("sdk_failure",502)}}
    signal.removeEventListener("abort",aborted);
  }
  if(error){if(error.code!=="canceled"){if(quotaEvidence)throw new BridgeError("quota_exhausted",402);if(contextEvidence)throw new BridgeError("context_length_exceeded",400);if(freeTierEvidence)throw new BridgeError("free_tier_restricted",403)}throw error;}
  if(signal.aborted)throw abortError();
  // No tool request becomes visible to the host until native cleanup has completed.
  for(const call of calls.values()){emit({type:"tool_use_start",tool_id:call.id,tool_name:call.name});emit({type:"tool_input_delta",text:call.raw})}
  emit({type:"message_delta",stop_reason:result.stop_reason,usage:result.usage});emit({type:"message_stop"});
  return result;
}
