import { describe, test, expect } from "bun:test";
import { mkdtemp, stat, rm, writeFile, symlink } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createBridge, MAX_REQUEST_BYTES, workerEnvironment } from "../src/bridge.mjs";
import { aliasMap, nativeHistory, safeJSON, validateRequest, zeroPrice } from "../src/protocol.mjs";

import { assertNoUnmanagedConfig, unmanagedConfigPaths } from "../src/native-cli.mjs";

const token="test-bridge-token-with-at-least-32-characters";
const auth={authorization:`Bearer ${token}`,"content-type":"application/json"};
const request=(changes={})=>({request_id:crypto.randomUUID(),model:"fixture/model",system:["Exact host system", "Second segment"],messages:[{role:"user",content:[{type:"text",text:"host question"}]}],tools:[],max_tokens:64,thinking:"disabled",...changes});
const tool=(name="read/file")=>({name,description:"Host owned action",input_schema:{type:"object",properties:{n:{type:"number"},text:{type:"string"}},required:["n","text"],additionalProperties:false}});
const fetchBridge=(bridge,path,body,extra={})=>bridge.fetch(new Request(`http://127.0.0.1:4097${path}`,{method:body?"POST":"GET",headers:auth,...extra,...(body?{body:JSON.stringify(body)}:{})}));
async function fixture(mode="text",providerID="fixture",timeout=20000) {
  const requests=[],headers=[],hits=[];let canceled=false;
  const root=await mkdtemp(join(tmpdir(),"gh49-native-test-")),marker=join(root,"MUST_NOT_EXIST");
  const server=Bun.serve({hostname:"127.0.0.1",port:0,async fetch(req){
    hits.push(req.url);if(req.method!=="POST")return new Response("redirect reached",{status:500});
    const body=await req.json();requests.push(body);headers.push(Object.fromEntries(req.headers));
    if(mode==="redirect")return Response.redirect(`http://127.0.0.1:${server.port}/redirect-target`,302);
    if(mode==="quota")return Response.json({error:{code:"insufficient_quota",message:"secret billing details"}},{status:402});
    if(mode==="rate")return Response.json({error:{code:"rate_limit_exceeded",message:"transient"}},{status:429});
    if(mode==="context")return Response.json({error:{code:"context_length_exceeded",message:"private request details"}},{status:400});
    if(mode==="generic400")return Response.json({error:{code:"invalid_request_error",message:"context_length_exceeded quota"}},{status:400});
    if(mode==="free-tier")return Response.json({error:{name:"FreeTierError",data:{message:"private policy details"}}},{status:403});
    if(mode==="generic403")return Response.json({error:{code:"permission_denied",message:"FreeTierError quota"}},{status:403});
    if(mode==="failure"||mode==="retry")return new Response("untrusted secret provider error",{status:mode==="retry"?500:401});
    if(mode==="hang"||(mode==="selective"&&body.messages.some(m=>m.role==="user"&&m.content.includes("please hang"))))return new Response(new ReadableStream({cancel(){canceled=true}}),{headers:{"content-type":"text/event-stream"}});
    const finish=mode==="multi"&&body.messages.at(-1)?.content==="finish";
    const names=(body.tools??[]).filter(t=>t.function.description?.startsWith("Host tool: ")).map(t=>t.function.name);
    const calls=mode==="unknown"?[{id:"evil",name:"shell",raw:JSON.stringify({command:`touch ${marker}`})}]:mode==="unsafe"?[{id:"call1",name:names[0],raw:'{"n":9007199254740993,"text":"x"}'}]:mode==="badjson"?[{id:"call1",name:names[0],raw:'{"n":1,"text":"x",}'}]:mode==="schema"?[{id:"call1",name:names[0],raw:'{"n":"bad","text":"x"}'}]:(mode==="tools"||(mode==="multi"&&!finish))?[{id:"call1",name:names[0],raw:'{ "n": 1.0, "text": "한글" }'},{id:"call2",name:names[1],raw:'{"n":2,"text":"second"}'}]:[];
    const chunks=[{choices:[{index:0,delta:{role:"assistant",reasoning_content:"native thought"},finish_reason:null}]},{choices:[{index:0,delta:{content:mode==="oversize"?"x".repeat(8000001):"native answer"},finish_reason:null}]}];
    for(const [index,call] of calls.entries()) {
      chunks.push({choices:[{index:0,delta:{tool_calls:[{index,id:call.id,type:"function",function:{name:call.name,arguments:call.raw.slice(0,5)}}]},finish_reason:null}]});
      chunks.push({choices:[{index:0,delta:{tool_calls:[{index,function:{arguments:call.raw.slice(5)}}]},finish_reason:null}]});
    }
    chunks.push({choices:[{index:0,delta:{},finish_reason:calls.length?"tool_calls":"stop"}],usage:{prompt_tokens:101,completion_tokens:23,total_tokens:124,prompt_tokens_details:{cached_tokens:5,cache_write_tokens:3}}});
    if(mode==="gzip"){const sse=chunks.map(c=>`data: ${JSON.stringify({id:"fixture",object:"chat.completion.chunk",created:1,model:"model",...c})}\n\n`).join("")+"data: [DONE]\n\n";return new Response(Bun.gzipSync(sse),{headers:{"content-type":"text/event-stream","content-encoding":"gzip"}})}
    return new Response(new ReadableStream({async start(output){try{for(const c of chunks){output.enqueue(new TextEncoder().encode(`data: ${JSON.stringify({id:"fixture",object:"chat.completion.chunk",created:1,model:"model",...c})}\n\n`));await new Promise(r=>setTimeout(r,5))}output.enqueue(new TextEncoder().encode("data: [DONE]\n\n"));output.close()}catch{}},cancel(){canceled=true}}),{headers:{"content-type":"text/event-stream"}});
  }});
  const providerConfig={providers:{[providerID]:{name:"Fixture",package:"@opencode/ai/providers/openai-compatible",settings:{baseURL:`http://127.0.0.1:${server.port}/v1`,apiKey:"explicit-fixture-only"},models:{model:{name:"Fixture model",capabilities:{tools:true},limit:{context:200000,output:32768},compatibility:{maxTokensField:"max_tokens"}}}}}};
  const bridge=createBridge({token,providerConfig,timeout});
  return {bridge,requests,headers,hits,marker,get canceled(){return canceled},async close(){await bridge.close();server.stop(true);await rm(root,{recursive:true,force:true})}};
}

describe("bridge boundary",()=>{
  test("native CLI refuses unmanaged configuration entries, including broken symlinks",async()=>{
    const root=await mkdtemp(join(tmpdir(),"native-config-policy-"));try{
      expect(unmanagedConfigPaths("linux")).toEqual(["/etc/opencode"]);
      const absent=join(root,"absent");await assertNoUnmanagedConfig([absent]);
      const file=join(root,"managed");await writeFile(file,"{}");await expect(assertNoUnmanagedConfig([file])).rejects.toMatchObject({code:"configuration_error"});
      if(process.platform!=="win32"){const link=join(root,"broken");await symlink(absent,link);await expect(assertNoUnmanagedConfig([link])).rejects.toMatchObject({code:"configuration_error"})}
    }finally{await rm(root,{recursive:true,force:true})}
  });
  test("auth, browser origin, loopback, runtime and malformed input fail before worker",async()=>{
    let invoked=0;const bridge=createBridge({token,worker:async()=>{invoked++;return []}});
    expect((await bridge.fetch(new Request("http://127.0.0.1/v1/capabilities"))).status).toBe(401);
    expect((await fetchBridge(bridge,"/v1/capabilities",null,{headers:{...auth,origin:"null"}})).status).toBe(403);
    expect((await bridge.fetch(new Request("http://example.com/v1/capabilities",{headers:auth}))).status).toBe(403);
    expect((await fetchBridge(bridge,"/v1/capabilities?x=1")).status).toBe(403);
    expect((await fetchBridge(bridge,"/v1/complete",request({max_tokens:0}))).status).toBe(400);
    expect((await fetchBridge(bridge,"/v1/capabilities")).status).toBe(200);
    expect((await (await fetchBridge(bridge,"/v1/capabilities")).json()).runtime).toBe("native");
    expect((await bridge.fetch(new Request("http://127.0.0.1/v1/complete",{method:"POST",headers:{...auth,"content-length":String(MAX_REQUEST_BYTES+1)},body:"{}"}))).status).toBe(413);
    const legacy=createBridge({token,runtime:"legacy1",worker:async()=>{invoked++}});
    expect((await fetchBridge(legacy,"/v1/models")).status).toBe(400);
    expect(invoked).toBe(0);await bridge.close();await legacy.close();
  });
  test("unsafe history numbers and unsupported references fail closed",()=>{
    expect(()=>safeJSON('{"n":9007199254740993}')).toThrow();
    expect(()=>validateRequest(request({tools:[{...tool(),input_schema:{$ref:"https://example.com/schema"}}]}))).toThrow();
    expect(()=>validateRequest(request({messages:[{role:"user",content:[{type:"image"}]}]}))).toThrow();
  });
  test("unknown and empty pricing never imply free eligibility",()=>{
    for(const c of [undefined,[],[{}],[{input:0}],[{input:0,output:1}]])expect(zeroPrice(c)).toBe(false);
    expect(zeroPrice([{input:0,output:0}])).toBe(true);
  });
  test("worker environments never inherit cloud credentials, proxies or user config",()=>{
    const env=workerEnvironment("/tmp/owned-worker",{PATH:"/bin",LANG:"en_US.UTF-8",OPENCODE_API_KEY:"private",AWS_SECRET_ACCESS_KEY:"private",HOME:"/real-home",HTTP_PROXY:"http://proxy",BUN_OPTIONS:"--preload unwanted",NODE_OPTIONS:"--require unwanted",XDG_CONFIG_HOME:"/user-config"});
    expect(env.PATH).toBe("/bin");expect(env.LANG).toBe("en_US.UTF-8");expect(env.XDG_CONFIG_HOME).toBe("/tmp/owned-worker/xdg-config");
    for(const key of ["OPENCODE_API_KEY","AWS_SECRET_ACCESS_KEY","HOME","HTTP_PROXY","BUN_OPTIONS","NODE_OPTIONS"])expect(env[key]).toBeUndefined();
  });
  test("valid host names remain meaningful in tool-free history and aliases never collide",()=>{
    const input=request({tools:[tool("Bash"),tool("set_goals"),tool("sw_tool_0"),tool("read/file")]});const aliases=aliasMap(input);
    expect(aliases.forward.get("Bash")).toBe("Bash");expect(aliases.forward.get("set_goals")).toBe("set_goals");expect(aliases.forward.get("sw_tool_0")).toBe("sw_tool_0");expect(aliases.forward.get("read/file")).toBe("sw_tool_1");
  });
  test("a four-MiB host context fits the documented input boundary",async()=>{
    let length=0;const bridge=createBridge({token,worker:async input=>{length=input.request.system[0].length;return {message:{role:"assistant",content:[{type:"text",text:"ok"}]},stop_reason:"end_turn",usage:{input_tokens:0,output_tokens:0,cache_read_tokens:0,cache_write_tokens:0}}}});
    try{expect((await fetchBridge(bridge,"/v1/complete",request({system:["x".repeat(4*1024*1024)]}))).status).toBe(200);expect(length).toBe(4*1024*1024)}finally{await bridge.close()}
  });
  test("host tool correlation and roles stay exact in native history",()=>{
    const input=validateRequest(request({tools:[tool()],messages:[{role:"user",content:[{type:"text",text:"question"}]},{role:"assistant",content:[{type:"thinking",thinking:"old thought",signature:"old-signature"},{type:"tool_use",id:"old-id",name:"read/file",input:{n:1,text:"old"}}]},{role:"user",content:[{type:"tool_result",tool_use_id:"old-id",content:[{type:"text",text:"host result"}],is_error:true},{type:"text",text:"continue"}]}]}));
    const native=nativeHistory(input,aliasMap(input));
    expect(native[1].content).toEqual([{type:"tool-call",id:"old-id",name:"sw_tool_0",input:{n:1,text:"old"}}]);
    expect(native[2]).toEqual({role:"tool",content:[{type:"tool-result",id:"old-id",name:"sw_tool_0",result:{type:"error",value:"host result"}}]});
    expect(native[3]).toEqual({role:"user",content:[{type:"text",text:"continue"}]});
  });
});

describe("actual native 2.1.4 runtime",()=>{
  test("model list is inference-free; exact system/history/settings and cache usage survive",async()=>{
    const f=await fixture();try{
      const models=await (await fetchBridge(f.bridge,"/v1/models")).json();expect(models.models).toEqual([{id:"fixture/model",name:"Fixture model"}]);expect(f.requests).toHaveLength(0);
      const res=await fetchBridge(f.bridge,"/v1/complete",request({temperature:0.25,stop:["STOP"]}));const body=await res.json();
      expect(res.status).toBe(200);expect(body.stop_reason).toBe("end_turn");expect(body.usage).toEqual({input_tokens:93,output_tokens:23,cache_read_tokens:5,cache_write_tokens:3});
      expect(f.requests).toHaveLength(1);const wire=f.requests[0];expect(wire.max_tokens).toBe(64);expect(wire.max_completion_tokens).toBeUndefined();expect(wire.temperature).toBe(0.25);expect(wire.stop).toEqual(["STOP"]);expect(wire.thinking).toEqual({type:"disabled"});
      expect(wire.messages[0].role).toBe("system");expect(wire.messages[0].content.startsWith("Exact host system\n\nSecond segment\n")).toBe(true);expect(wire.messages[0].content).toContain("Here is some useful information about the environment");expect(wire.messages.slice(1)).toEqual([{role:"user",content:"host question"}]);expect(wire.tools.map(t=>t.function.name)).toContain("shell");expect(wire.tools.filter(t=>t.function.description?.startsWith("Host tool: "))).toEqual([]);
      expect(f.headers[0]["x-opencode-session"]).toBeUndefined();expect(f.headers[0].authorization).toBe("Bearer explicit-fixture-only");expect(f.headers[0]["x-scopeweaver-egress"]).toBeUndefined();expect(f.headers[0]["x-scopeweaver-upstream"]).toBeUndefined();
    }finally{await f.close()}
  });
  test("native egress forwards compressed SSE without a second content decoding",async()=>{
    const f=await fixture("gzip");try{
      const res=await fetchBridge(f.bridge,"/v1/complete",request()),body=await res.json();expect(res.status).toBe(200);expect(body.message.content.find(b=>b.type==="text").text).toBe("native answer");expect(f.requests).toHaveLength(1);
    }finally{await f.close()}
  });
  test("multiple capture tools retain names/order/raw arguments; NDJSON stops once after cleanup",async()=>{
    const f=await fixture("tools");try{
      const input=request({tools:[tool("read/file"),tool("other.tool")]}),res=await fetchBridge(f.bridge,"/v1/stream",input);
      const events=(await res.text()).trim().split("\n").map(JSON.parse);
      expect(events.find(e=>e.type==="error")).toBeUndefined();expect(events.filter(e=>e.type==="message_stop")).toHaveLength(1);
      expect(events.filter(e=>e.type==="tool_use_start").map(e=>[e.tool_id,e.tool_name])).toEqual([[`sdk_${input.request_id}_0`,"read/file"],[`sdk_${input.request_id}_1`,"other.tool"]]);
      expect(events.filter(e=>e.type==="tool_input_delta").map(e=>e.text)).toEqual(['{ "n": 1.0, "text": "한글" }','{"n":2,"text":"second"}']);
      expect(events.find(e=>e.type==="thinking_delta").text).toBe("native thought");expect(events.find(e=>e.type==="text_delta").text).toBe("native answer");expect(f.requests).toHaveLength(1);
      const hostTools=f.requests[0].tools.filter(t=>t.function.description?.startsWith("Host tool: "));expect(hostTools.map(t=>t.function.name)).toEqual(["sw_tool_0","sw_tool_1"]);
      expect(hostTools[0].function.description).toContain('Host tool: "read/file"');expect(hostTools[0].function.description.endsWith(tool().description)).toBe(true);
    }finally{await f.close()}
  });
  test("complete retains raw numeric argument representation",async()=>{
    const f=await fixture("tools");try{const res=await fetchBridge(f.bridge,"/v1/complete",request({tools:[tool(),tool("second")]}));const raw=await res.text();expect(res.status).toBe(200);expect(raw).toContain('"input":{ "n": 1.0, "text": "한글" }')}finally{await f.close()}
  });
  for(const mode of ["unknown","schema","badjson","failure","retry","redirect","oversize"])test(`${mode} fails without host tool delivery or native action`,async()=>{
    const f=await fixture(mode);try{
      const events=(await (await fetchBridge(f.bridge,"/v1/stream",request({tools:[tool()]}))).text()).trim().split("\n").map(JSON.parse);
      expect(events.some(e=>e.type==="error")).toBe(true);expect(events.some(e=>e.type==="tool_use_start"||e.type==="message_stop")).toBe(false);
      expect(f.requests).toHaveLength(1);expect(f.hits).toHaveLength(1);expect(await stat(f.marker).then(()=>true,()=>false)).toBe(false);expect(JSON.stringify(events)).not.toContain("untrusted secret provider error");
    }finally{await f.close()}
  });
  test("concurrent per-call settings and tool catalogs are isolated",async()=>{
    const f=await fixture();try{
      const responses=await Promise.all([fetchBridge(f.bridge,"/v1/complete",request({max_tokens:37,temperature:0.1})),fetchBridge(f.bridge,"/v1/complete",request({max_tokens:73,temperature:0.9,thinking:"enabled",reasoning_effort:"low",tools:[tool()]}))]);
      expect(responses.map(r=>r.status)).toEqual([200,200]);expect(f.requests.map(r=>r.max_tokens).sort((a,b)=>a-b)).toEqual([37,73]);expect(f.requests.find(r=>r.max_tokens===37).temperature).toBe(0.1);expect(f.requests.find(r=>r.max_tokens===73).temperature).toBe(0.9);expect(f.requests.find(r=>r.max_tokens===73).tools.filter(t=>t.function.description?.startsWith("Host tool: "))).toHaveLength(1);expect(f.requests.find(r=>r.max_tokens===37).tools.filter(t=>t.function.description?.startsWith("Host tool: "))).toHaveLength(0);expect(f.requests.find(r=>r.max_tokens===73).reasoning_effort).toBe("low");
    }finally{await f.close()}
  });
  test("DELETE aborts an active SDK worker; peer request remains usable",async()=>{
    const f=await fixture("selective");try{
      const input=request({messages:[{role:"user",content:[{type:"text",text:"please hang"}]}]}),res=await fetchBridge(f.bridge,"/v1/stream",input);const reader=res.body.getReader();await reader.read();
      expect((await fetchBridge(f.bridge,"/v1/complete",request())).status).toBe(200);
      expect((await fetchBridge(f.bridge,`/v1/requests/${input.request_id}`,null,{method:"DELETE"})).status).toBe(200);
      let output="";while(true){const x=await reader.read();if(x.done)break;output+=new TextDecoder().decode(x.value)}expect(output).toContain('"type":"error"');expect(output).not.toContain('"type":"message_stop"');
      expect((await fetchBridge(f.bridge,"/v1/capabilities")).status).toBe(200);
    }finally{await f.close()}
  });
  test("host tool-result history is injected without native conversation growth",async()=>{
    const f=await fixture();try{
      const req=request({tools:[tool()],messages:[{role:"user",content:[{type:"text",text:"first"}]},{role:"assistant",content:[{type:"tool_use",id:"prior",name:"read/file",input:{n:7,text:"old"}}]},{role:"user",content:[{type:"tool_result",tool_use_id:"prior",content:[{type:"text",text:"host result"}]},{type:"text",text:"next"}]}]});
      const res=await fetchBridge(f.bridge,"/v1/complete",req);expect(res.status).toBe(200);
      const wire=f.requests[0].messages;expect(wire).toHaveLength(5);expect(wire[2].tool_calls[0]).toEqual({id:"prior",type:"function",function:{name:"sw_tool_0",arguments:'{"n":7,"text":"old"}'}});expect(wire[3]).toEqual({role:"tool",tool_call_id:"prior",content:"host result"});expect(wire[4]).toEqual({role:"user",content:"next"});
    }finally{await f.close()}
  });
  test("unknown explicit model is rejected before inference",async()=>{
    const f=await fixture();try{expect((await fetchBridge(f.bridge,"/v1/complete",request({model:"fixture/missing"}))).status).toBe(400);expect(f.requests).toHaveLength(0)}finally{await f.close()}
  });
  test("HTTP response cancellation cleans an owned SDK worker",async()=>{
    const f=await fixture("hang");try{const res=await fetchBridge(f.bridge,"/v1/stream",request());const reader=res.body.getReader();await reader.read();await reader.cancel();await f.bridge.close()}finally{await f.close()}
  });
  test("gateway session header stays stable across fresh native sessions",async()=>{
    const f=await fixture("text","opencode-go");try{
      const a=request({model:"opencode-go/model",session_id:"host-session"}),b=request({model:"opencode-go/model",session_id:"host-session"});expect(a.request_id).not.toBe(b.request_id);
      expect((await fetchBridge(f.bridge,"/v1/complete",a)).status).toBe(200);expect((await fetchBridge(f.bridge,"/v1/complete",b)).status).toBe(200);
      expect(f.headers.map(h=>h["x-opencode-session"])).toEqual(["host-session","host-session"]);
      expect(f.headers.map(h=>h["user-agent"])).toEqual([f.headers[0]["user-agent"],f.headers[0]["user-agent"]]);
    }finally{await f.close()}
  });
  test("OpenCode provider preserves its genuine native session identity",async()=>{
    const f=await fixture("text","opencode");try{
      expect((await fetchBridge(f.bridge,"/v1/complete",request({model:"opencode/model",session_id:"host-session"}))).status).toBe(200);
      expect(f.headers[0]["x-opencode-session"]).not.toBe("host-session");expect(f.headers[0]["x-opencode-session"]).toStartWith("ses_");
      expect(f.headers[0]["user-agent"]).toBe("opencode/latest/2.1.4/cli");expect(f.headers[0]["x-opencode-client"]).toBe("cli");
    }finally{await f.close()}
  });
  test("64-bit tool IDs retain authoritative raw arguments in stream and complete",async()=>{
    const f=await fixture("unsafe");try{
      const response=await fetchBridge(f.bridge,"/v1/complete",request({tools:[tool()]}));expect(response.status).toBe(200);expect(await response.text()).toContain('"input":{"n":9007199254740993,"text":"x"}');
      const events=(await (await fetchBridge(f.bridge,"/v1/stream",request({tools:[tool()]}))).text()).trim().split("\n").map(JSON.parse);expect(events.find(e=>e.type==="tool_input_delta").text).toBe('{"n":9007199254740993,"text":"x"}');expect(events.filter(e=>e.type==="message_stop")).toHaveLength(1);
    }finally{await f.close()}
  });
  test("64-bit host history is restored exactly before the native HTTP request",async()=>{
    const f=await fixture();try{
      const input=request({tools:[tool()],messages:[{role:"assistant",content:[{type:"tool_use",id:"old64",name:"read/file",input:{n:0,text:"raw"}}]},{role:"user",content:[{type:"tool_result",tool_use_id:"old64",content:[{type:"text",text:"host result"}]}]}]});
      const raw=JSON.stringify(input).replace('"n":0','"n":9007199254740993');
      const res=await f.bridge.fetch(new Request("http://127.0.0.1:4097/v1/complete",{method:"POST",headers:auth,body:raw}));expect(res.status).toBe(200);expect(f.requests[0].messages[1].tool_calls[0].function.arguments).toBe('{"n":9007199254740993,"text":"raw"}');
    }finally{await f.close()}
  });
  test("local JSON Pointer refs, defs and unions work in actual native tool schemas",async()=>{
    const f=await fixture("tools");try{
      const first=tool();first.input_schema={$defs:{Number:{anyOf:[{type:"number"},{type:"string"}]}},type:"object",properties:{n:{$ref:"#/$defs/Number"},text:{type:"string"}},required:["n","text"],additionalProperties:false};
      const res=await fetchBridge(f.bridge,"/v1/complete",request({tools:[first,tool("other")]}));expect(res.status).toBe(200);expect(f.requests[0].tools.find(t=>t.function.name==="sw_tool_0").function.parameters).toEqual(first.input_schema);
    }finally{await f.close()}
  });
  test("structured quota exhaustion is distinct from transient rate limiting",async()=>{
    for(const mode of ["quota","rate"]){const f=await fixture(mode);try{
      const res=await fetchBridge(f.bridge,"/v1/complete",request()),body=await res.json();
      expect(body.error.code==="quota_exhausted").toBe(mode==="quota");if(mode==="quota")expect(res.status).toBe(402);
      expect(JSON.stringify(body)).not.toContain("secret billing details");expect(f.requests).toHaveLength(1);
    }finally{await f.close()}}
  });
  test("reused upstream call IDs remain unique across real host tool turns",async()=>{
    const f=await fixture("multi");try{
      const history=[{role:"user",content:[{type:"text",text:"host question"}]}],allIDs=[];
      for(const text of ["continue","finish"]){
        const input=request({messages:structuredClone(history),tools:[tool(),tool("other")]}),response=await fetchBridge(f.bridge,"/v1/complete",input),result=await response.json();expect(response.status).toBe(200);
        const calls=result.message.content.filter(b=>b.type==="tool_use");expect(calls).toHaveLength(2);allIDs.push(...calls.map(c=>c.id));
        history.push(result.message,{role:"user",content:[...calls.map(c=>({type:"tool_result",tool_use_id:c.id,content:[{type:"text",text:"host approved result"}]})),{type:"text",text}]});
      }
      expect(new Set(allIDs).size).toBe(4);
      const last=await fetchBridge(f.bridge,"/v1/complete",request({messages:history,tools:[tool(),tool("other")]}));expect(last.status).toBe(200);expect((await last.json()).stop_reason).toBe("end_turn");expect(f.requests).toHaveLength(3);
      expect(f.requests[2].messages.filter(m=>m.role==="tool").map(m=>m.tool_call_id)).toEqual(allIDs);
    }finally{await f.close()}
  });
  test("an explicit deadline is bounded independently of host cancellation",async()=>{
    const f=await fixture("hang","fixture",1000);try{const res=await fetchBridge(f.bridge,"/v1/complete",request());expect(res.status).toBe(504);expect((await res.json()).error.code).toBe("timeout")}finally{await f.close()}
  });
  test("structured context overflow stays distinct from quota and generic bad requests",async()=>{
    for(const mode of ["context","generic400"]){const f=await fixture(mode);try{
      const res=await fetchBridge(f.bridge,"/v1/complete",request()),body=await res.json();expect(body.error.code==="context_length_exceeded").toBe(mode==="context");expect(body.error.code).not.toBe("quota_exhausted");if(mode==="context")expect(res.status).toBe(400);expect(f.requests).toHaveLength(1);
    }finally{await f.close()}}
  });
  test("structured hosted free-tier restriction stays distinct from generic forbidden responses",async()=>{
    for(const mode of ["free-tier","generic403"]){const f=await fixture(mode);try{
      const res=await fetchBridge(f.bridge,"/v1/complete",request()),body=await res.json();expect(body.error.code==="free_tier_restricted").toBe(mode==="free-tier");expect(body.error.code).not.toBe("quota_exhausted");if(mode==="free-tier")expect(res.status).toBe(403);expect(f.requests).toHaveLength(1);expect(JSON.stringify(body)).not.toContain("private policy details");
    }finally{await f.close()}}
  });
});
