export class BridgeError extends Error {
  constructor(code, status = 400) { super(code); this.code = code; this.status = status; }
}
export const fail = (code, status) => { throw new BridgeError(code, status); };
export const zeroPrice = cost => Array.isArray(cost)&&cost.length>0&&cost.every(c=>c?.input===0&&c?.output===0);
export const messages = Object.freeze({
  unauthorized: "Bridge authentication required.", forbidden: "Only non-browser loopback requests are accepted.",
  invalid_request: "Invalid completion request.", unsupported_input: "This input cannot be represented safely by the SDK.",
  unsupported_runtime: "Unsupported SDK/runtime pair or platform. Use a tested pinned runtime.",
  unknown_model: "The selected model is unavailable in this bridge.", duplicate_request: "This request ID is already active.",
  canceled: "The request was canceled.", timeout: "The SDK request timed out.",
  sdk_failure: "The SDK request failed. Check the selected model and provider configuration.",
  protocol_error: "The SDK response could not be represented safely.", configuration_error: "Invalid bridge provider configuration.",
  second_model_request: "An unexpected additional SDK model request was blocked.",
  quota_exhausted: "The upstream provider reported exhausted quota or billing credit.",
  context_length_exceeded: "The upstream model reported that the context length was exceeded.",
  free_tier_restricted: "The hosted free tier is restricted for this SDK runtime.",
  not_found: "Route not found.", body_too_large: "Request body exceeds the bridge limit.",
});
export const publicError = error => {
  const code = Object.hasOwn(messages, error?.code) ? error.code : "sdk_failure";
  return { code, message: messages[code] };
};
export function safeJSON(text, code = "invalid_request") {
  try {
    return JSON.parse(text, (_, value) => {
      if (typeof value === "number" && (!Number.isFinite(value) || (Number.isInteger(value) && !Number.isSafeInteger(value)))) throw Error();
      if (value && typeof value === "object" && !Array.isArray(value) && Object.hasOwn(value, "__proto__")) throw Error();
      return value;
    });
  } catch { fail(code); }
}
export function rawObject(text) {
  if(typeof text!=="string"||text.length>2000000)fail("protocol_error");
  let value;try{value=JSON.parse(text)}catch{fail("protocol_error")}
  if(!value||typeof value!=="object"||Array.isArray(value))fail("protocol_error");
  const errors=[],tree=parseTree(text,errors,{disallowComments:true,allowTrailingComma:false});
  if(!tree||errors.length)fail("protocol_error");
  const walk=node=>{if(node.type==="object"){const keys=new Set();for(const p of node.children??[]){const key=p.children[0].value;if(keys.has(key)||key==="__proto__")fail("protocol_error");keys.add(key)}}for(const c of node.children??[])walk(c)};walk(tree);
  return value;
}
export function parseRequest(text) {
  const errors=[],tree=parseTree(text,errors,{disallowComments:true,allowTrailingComma:false});
  if(!tree||errors.length)fail("invalid_request");
  const rawNodes=new Set(),rawHistory=Object.create(null);
  // Tool arguments are authoritative raw JSON. Their JS projection is never executed.
  let request;try{request=JSON.parse(text)}catch{fail("invalid_request")}
  for(const [i,m]of (Array.isArray(request.messages)?request.messages:[]).entries())for(const [j,b]of (Array.isArray(m?.content)?m.content:[]).entries())if(b?.type==="tool_use"){
    const node=findNodeAtLocation(tree,["messages",i,"content",j,"input"]);
    if(node){rawNodes.add(node);rawHistory[b.id]=text.slice(node.offset,node.offset+node.length)}
  }
  const walk=(node,raw=false)=>{
    raw=raw||rawNodes.has(node);
    if(node.type==="object"){
      const keys=new Set();for(const p of node.children??[]){const key=p.children[0].value;if(keys.has(key)||key==="__proto__")fail("invalid_request");keys.add(key)}
    }
    if(!raw&&node.type==="number"&&(!Number.isFinite(node.value)||(Number.isInteger(node.value)&&!Number.isSafeInteger(node.value))))fail("unsupported_input");
    for(const c of node.children??[])walk(c,raw);
  };walk(tree);
  return {request:validateRequest(request),rawHistory};
}
const object = x => x && typeof x === "object" && !Array.isArray(x);
const str = (x, max = 16*1024*1024) => typeof x === "string" && x.length <= max;
const knownKeys = (x, allowed) => Object.keys(x).every(k => allowed.includes(k));
export function validateRequest(x) {
  if (!object(x) || !knownKeys(x, ["request_id","session_id","model","system","messages","tools","max_tokens","temperature","stop","thinking","thinking_type","reasoning_effort"])) fail("invalid_request");
  if (!str(x.request_id, 128) || !/^[A-Za-z0-9_-]+$/.test(x.request_id) || !str(x.model, 512) || !/^[^/\s]+\/\S+$/.test(x.model)) fail("invalid_request");
  if(x.session_id!==undefined&&(!str(x.session_id,256)||!x.session_id||/[\r\n\x00-\x1f\x7f]/.test(x.session_id)))fail("invalid_request");
  if (!Array.isArray(x.system) || !x.system.every(s => str(s)) || !Array.isArray(x.messages) || !x.messages.length || x.messages.length > 1000 || !Array.isArray(x.tools) || x.tools.length > 128) fail("invalid_request");
  if (!Number.isSafeInteger(x.max_tokens) || x.max_tokens < 1 || x.max_tokens > 1000000) fail("invalid_request");
  if (x.temperature !== undefined && (typeof x.temperature !== "number" || !Number.isFinite(x.temperature) || x.temperature < 0 || x.temperature > 2)) fail("invalid_request");
  if (x.stop !== undefined && (!Array.isArray(x.stop) || x.stop.length > 16 || !x.stop.every(s => str(s, 4096) && s.length))) fail("invalid_request");
  for (const k of ["thinking", "thinking_type"]) if (x[k] !== undefined && !["", "enabled", "disabled"].includes(x[k])) fail("unsupported_input");
  if (x.reasoning_effort !== undefined && !["", "none", "minimal", "low", "medium", "high", "xhigh", "max"].includes(x.reasoning_effort)) fail("unsupported_input");
  const names = new Set();
  for (const t of x.tools) {
    if (!object(t) || !knownKeys(t,["name","description","input_schema"]) || !str(t.name, 512) || !t.name || names.has(t.name) || !str(t.description, 32000) || !object(t.input_schema)) fail("invalid_request");
    schemaCheck(t.input_schema); names.add(t.name);
  }
  const calls = new Map(), results = new Set();
  for (const m of x.messages) {
    if (!object(m) || !knownKeys(m,["role","content"]) || !["assistant","user"].includes(m.role) || !Array.isArray(m.content)) fail("invalid_request");
    for (const b of m.content) {
      if (!object(b)) fail("invalid_request");
      if (b.type === "text") { if (!knownKeys(b,["type","text"]) || !str(b.text)) fail("invalid_request"); }
      else if (b.type === "thinking") { if (m.role !== "assistant" || !knownKeys(b,["type","thinking","signature"]) || !str(b.thinking) || (b.signature !== undefined && !str(b.signature))) fail("invalid_request"); }
      else if (b.type === "tool_use") {
        if (m.role !== "assistant" || !knownKeys(b,["type","id","name","input"]) || !str(b.id,256) || !b.id || calls.has(b.id) || !str(b.name,512) || !b.name || !object(b.input)) fail("invalid_request");
        calls.set(b.id,b.name);
      } else if (b.type === "tool_result") {
        if (m.role !== "user" || !knownKeys(b,["type","tool_use_id","content","is_error"]) || !calls.has(b.tool_use_id) || results.has(b.tool_use_id) || !Array.isArray(b.content) || !b.content.every(c => object(c) && knownKeys(c,["type","text"]) && c.type === "text" && str(c.text)) || (b.is_error !== undefined && typeof b.is_error !== "boolean")) fail("unsupported_input");
        results.add(b.tool_use_id);
      } else fail("unsupported_input");
    }
  }
  if ([...calls.keys()].some(id => !results.has(id))) fail("unsupported_input");
  return x;
}
function schemaCheck(value, depth = 0, root = value) {
  if (depth > 32) fail("unsupported_input");
  if (Array.isArray(value)) { for (const v of value) schemaCheck(v, depth+1,root); }
  else if (object(value)) {
    if (Object.keys(value).length > 1000 || ["$dynamicRef", "$id"].some(k => Object.hasOwn(value,k))) fail("unsupported_input");
    if(Object.hasOwn(value,"$ref")) {
      if(typeof value.$ref!=="string"||(value.$ref!=="#"&&!value.$ref.startsWith("#/")))fail("unsupported_input");
      let target=root;
      for(const part of value.$ref==="#"?[]:value.$ref.slice(2).split("/")) {
        if(/~(?![01])/.test(part))fail("unsupported_input");
        const key=part.replaceAll("~1","/").replaceAll("~0","~");
        if(!target||typeof target!=="object"||!Object.hasOwn(target,key))fail("unsupported_input");target=target[key];
      }
    }
    for (const v of Object.values(value)) schemaCheck(v, depth+1,root);
  }
}
export function aliasMap(request) {
  const names = [...request.tools.map(t=>t.name), ...request.messages.flatMap(m=>m.content.filter(b=>b.type === "tool_use").map(b=>b.name))];
  const unique=[...new Set(names)],valid=name=>/^[A-Za-z0-9_-]{1,64}$/.test(name),reserved=new Set(unique.filter(valid)),forward=new Map();let index=0;
  for(const name of unique){let alias=name;if(!valid(name)){do{alias=`sw_tool_${index++}`}while(reserved.has(alias));reserved.add(alias)}forward.set(name,alias)}
  return { forward, reverse: new Map([...forward].map(([k,v])=>[v,k])) };
}
export function nativeHistory(request, aliases) {
  const names = new Map(); const out=[];
  const thinking = request.thinking || request.thinking_type;
  for (const m of request.messages) {
    let content=[];
    const flush = () => { if(content.length)out.push({role:m.role,content}); content=[]; };
    for (const b of m.content) {
      if(b.type === "text") content.push({type:"text",text:b.text});
      else if(b.type === "thinking") { if(thinking !== "disabled")content.push({type:"reasoning",text:b.thinking,...(b.signature?{encrypted:b.signature}:{})}); }
      else if(b.type === "tool_use") { const name=aliases.forward.get(b.name);names.set(b.id,name);content.push({type:"tool-call",id:b.id,name,input:b.input}); }
      else { flush();out.push({role:"tool",content:[{type:"tool-result",id:b.tool_use_id,name:names.get(b.tool_use_id),result:{type:b.is_error?"error":"text",value:b.content.map(t=>t.text).join("\n")}}]}); }
    }
    flush();
  }
  return out;
}
export function serializeComplete(result) {
  const blocks=result.message.content.map(b=>b.type === "tool_use"
    ? `{"type":"tool_use","id":${JSON.stringify(b.id)},"name":${JSON.stringify(b.name)},"input":${b.raw}}`
    : JSON.stringify(b));
  return `{"message":{"role":"assistant","content":[${blocks.join(",")}]},"stop_reason":${JSON.stringify(result.stop_reason)},"usage":${JSON.stringify(result.usage)}}`;
}
export function validateProviderConfig(config) {
  if (config === undefined) return undefined;
  if (!object(config) || !knownKeys(config,["providers"]) || !object(config.providers) || !Object.keys(config.providers).length) fail("configuration_error");
  for(const [id,p] of Object.entries(config.providers)) {
    if(!/^[A-Za-z0-9_-]+$/.test(id) || !object(p) || !knownKeys(p,["name","package","settings","models"]) || p.package !== "@opencode/ai/providers/openai-compatible" || !object(p.settings) || !knownKeys(p.settings,["baseURL","apiKey","timeout"]) || !object(p.models))fail("configuration_error");
    let url;try{url=new URL(p.settings.baseURL)}catch{fail("configuration_error")}
    if(!["http:","https:"].includes(url.protocol)||url.username||url.password||url.search||url.hash)fail("configuration_error");
    if(p.settings.apiKey!==undefined&&!str(p.settings.apiKey,8192))fail("configuration_error");
    for(const [name,m] of Object.entries(p.models)) {
      if(!name||!object(m)||!knownKeys(m,["name","limit","capabilities","compatibility"]))fail("configuration_error");
      if(m.limit!==undefined&&(!object(m.limit)||!knownKeys(m.limit,["context","input","output"])||!Object.values(m.limit).every(n=>Number.isSafeInteger(n)&&n>0)))fail("configuration_error");
      if(m.compatibility!==undefined&&(!object(m.compatibility)||!knownKeys(m.compatibility,["maxTokensField","reasoningField"])||(m.compatibility.maxTokensField!==undefined&&!["max_tokens","max_completion_tokens"].includes(m.compatibility.maxTokensField))))fail("configuration_error");
    }
  }
  return structuredClone(config);
}
import { parseTree, findNodeAtLocation } from "jsonc-parser";
