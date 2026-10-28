import { readFile } from "node:fs/promises";

// This local plugin is copied into the request's private config directory.
// Keep the real OpenCode tool registry and implementations; policy blocks execution.
const inputURL=new URL("./bridge-input.json",import.meta.url);
const input=JSON.parse(await readFile(inputURL,"utf8"));
const {request,history,tools,rawHistory,channel,egress}=input;
const exposed=new Set(tools.map(t=>t.name));
const notify=event=>process.stdout.write(JSON.stringify({_scopeweaver_native:channel,event})+"\n");
let network=0;

export default {id:"scopeweaver.host-capture",async setup(ctx){
  await ctx.agent.transform(editor=>{
    for(const agent of editor.list())if(agent.id!=="capture")editor.remove(agent.id);
    editor.default("capture");
  });
  await ctx.tool.transform(editor=>{
    const nativeNames=new Set(editor.list().map(t=>t.id));
    for(const tool of tools){
      if(nativeNames.has(tool.name)||tool.name==="execute")throw Error("HOST_TOOL_COLLISION");
      editor.add({...tool,options:{codemode:false,permission:"host_capture"},async execute(_,context){
        notify({type:"capture",id:context.id});
        return new Promise((_,reject)=>{
          const stop=()=>reject(Error("CAPTURE_INTERRUPTED"));
          if(context.signal.aborted)stop();else context.signal.addEventListener("abort",stop,{once:true});
        });
      }});
    }
  });
  await ctx.tool.hook("execute.before",event=>{
    if(!exposed.has(event.tool)){
      notify({type:"failure",code:"protocol_error"});
      throw Error("NATIVE_EXECUTION_DENIED");
    }
  });
  await ctx.session.hook("context",async event=>{
    // Host instructions use the public custom-agent system setting. Keep the
    // runtime's generated initial context; replace only conversation history.
    event.messages=structuredClone(history);
    event.system.push({type:"text",text:'ScopeWeaver tool policy: The host application owns tool execution. Use only direct function tools whose description starts with "Host tool:". All other OpenCode tools, including execute and native filesystem, shell, browser, and subagent tools, are disabled by policy. Do not call them. If no host tools are listed, respond without tools.'});
    const current=JSON.parse(await readFile(inputURL,"utf8"));
    event.options.maxTokens=current.request.max_tokens;
    if(request.temperature!==undefined)event.options.temperature=request.temperature;
    if(request.stop!==undefined)event.options.stop=[...request.stop];
    if(request.reasoning_effort)event.options.reasoningEffort=request.reasoning_effort;
  });
  await ctx.session.hook("http.request",async event=>{
    if(event.kind!=="primary"||++network!==1){
      notify({type:"failure",code:"second_model_request"});
      throw Error("SECOND_MODEL_REQUEST_BLOCKED");
    }
    const body=await event.request.clone().json();
    const names=(body.tools??[]).map(t=>t.function?.name??t.name);
    if(new Set(names).size!==names.length||[...exposed].some(name=>!names.includes(name)))throw Error("SDK_TOOL_CATALOG_CHANGED");
    if(Object.keys(rawHistory).length){
      const restored=new Set();
      for(const message of body.messages??[])for(const call of message.tool_calls??[])if(Object.hasOwn(rawHistory,call.id)){
        call.function.arguments=rawHistory[call.id];restored.add(call.id);
      }
      if(restored.size!==Object.keys(rawHistory).length)throw Error("SDK_HISTORY_FORMAT_UNSUPPORTED");
      const headers=new Headers(event.request.headers);headers.delete("content-length");
      event.request=new Request(event.request,{headers,body:JSON.stringify(body)});
    }
    if(event.model.providerID==="opencode-go")event.request.headers.set("x-opencode-session",request.session_id??request.request_id);
    else if(event.model.providerID!=="opencode")event.request.headers.delete("x-opencode-session");
    // The CLI's Effect transport drops Request.redirect. Route the unchanged
    // genuine request through the worker's one-use, redirect-blocking transport.
    const headers=new Headers(event.request.headers);
    headers.set("x-scopeweaver-egress",egress.token);
    headers.set("x-scopeweaver-upstream",event.request.url);
    event.request=new Request(egress.url,{method:"POST",headers,body:await event.request.clone().arrayBuffer(),redirect:"error"});
    notify({type:"request"});
  });
  await ctx.session.hook("http.response",async event=>{
    if(event.kind!=="primary"||![400,402,403,413,429].includes(event.response.status)||!event.response.headers.get("content-type")?.includes("application/json"))return;
    try{
      const text=await event.response.clone().text();if(text.length>65536)return;
      const data=JSON.parse(text),code=data.error?.code??data.error?.type??data.error?.name,status=event.response.status;
      const billing=["insufficient_quota","billing_hard_limit_reached","insufficient_balance","credit_balance_exhausted","credits_exhausted"];
      if([402,429].includes(status)&&(billing.includes(code)||(status===402&&code==="quota_exceeded")))notify({type:"evidence",code:"quota_exhausted"});
      if([400,413].includes(status)&&["context_length_exceeded","model_context_window_exceeded","context_window_exceeded","prompt_too_long"].includes(code))notify({type:"evidence",code:"context_length_exceeded"});
      if(status===403&&code==="FreeTierError")notify({type:"evidence",code:"free_tier_restricted"});
    }catch{}
  });
}};
