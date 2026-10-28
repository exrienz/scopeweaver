import {request} from "playwright";
import {randomUUID} from "node:crypto";
import {assertCurrentSummaryFocus} from "./chat-summary-requests.mjs";
import {waitChatState} from "./chat-controls.mjs";
import { LOCALES, namespaceMessages } from "./locale-resources.mjs";
import {withLocaleQuery} from "../web/src/i18n/config.ts";
const labels = namespaceMessages("interface");
async function closeContext(page, label) {
 const button=page.getByRole("button",{name:label,exact:true});
 try { await button.click(); } catch(error) {
  try { console.error("context-close-diagnostic",await page.evaluate(()=>({animations:document.getAnimations().map(a=>({state:a.playState,time:a.currentTime,target:a.effect?.target?.outerHTML?.slice(0,500)})),drawer:document.querySelector("[data-vaul-drawer]")?.outerHTML.slice(0,1800),panel:document.querySelector("[data-chat-context-panel]")?.getBoundingClientRect().toJSON()}))); } catch(captureError) { console.error("context-close-capture-failure",String(captureError)); }
  throw error;
 }
}
const actionURL=(base,action)=>{const [p,q]=base.split("?");return `${p}/${action}${q?`?${q}`:""}`};
export async function runChatContextChecks({base,token,fixture,step,check,openPage}){
 const api=await request.newContext({baseURL:base,extraHTTPHeaders:{Authorization:`Bearer ${token}`}});
 for(const locale of LOCALES)for(const width of [1440,390])for(const surface of ["standalone","task"]){
  const caseName=`${locale}/${width}/${surface}`;
  if(process.env.CHAT_CONTEXT_CASE&&!process.env.CHAT_CONTEXT_CASE.split(",").map(value=>value.trim()).includes(caseName))continue;
  const caseMarker=`context-case-${randomUUID()}`;
  const t=labels[locale];let path,url;
  await fixture.setBudget(64);
  if(surface==="standalone"){const r=await api.post("/api/conversations",{data:{agent_key:"auto",title:"Context fixture"}});check(r.ok(),await r.text());const c=await r.json();path=`/api/conversations/${c.id}/controls`;url=`${base}/chat?c=${c.id}`}
  else{const r=await api.post("/api/tasks",{data:{description:"Context fixture",goal:"Keep fixture idle",seed_first_intent:false}});check(r.ok(),await r.text());const task=await r.json();check((await api.post(`/api/tasks/${task.id}/control`,{data:{action:"pause"}})).ok(),"pause failed");path=`/api/tasks/${task.id}/chat/controls?seg=0`;url=`${base}/function/tasks/detail?id=${task.id}`}
  const p=await openPage(locale,token);await p.setViewportSize({width,height:900});
  try{await step(`[${locale}/${width}/${surface}] context visibility, capacity switch and manual compaction`,async()=>{
   await p.goto(url);const input=p.locator("textarea[data-chat-draft]").first();await input.waitFor();await p.locator("[data-chat-context-trigger]").waitFor();
   const calls=await fixture.calls({operation:"context-command",boundary:"before"});await input.fill("/context");await p.getByRole("button",{name:t.m0073,exact:true}).click();await p.locator("[data-chat-context-panel]").waitFor();await p.getByText(t.chatContextUnknown,{exact:true}).last().waitFor();check(await p.locator("[data-chat-compact]").isDisabled(),"unobserved context allows compact");check(await fixture.calls({operation:"context-command",boundary:"after"})===calls,"/context called the provider");await closeContext(p,t.chatContextClose);
   for(let i=0;i<5;i++){await input.fill(`context turn ${i}: `+(i===0?`${caseMarker} `:"")+"preserve detail ".repeat(120));if(i===0)await fixture.holdNext();await p.getByRole("button",{name:t.m0073,exact:true}).click();
    if(i===0){await waitChatState(api,path,state=>!!state.active);await p.locator("[data-chat-context-trigger]").click();await p.locator("[data-chat-context-panel]").waitFor();if(process.env.CHAT_CONTEXT_MUTATION)await p.locator("[data-chat-compact]").evaluate(button=>button.disabled=false);check(await p.locator("[data-chat-compact]").isDisabled(),"busy Compact now is enabled");await closeContext(p,t.chatContextClose);await fixture.release()}
    await waitChatState(api,path,state=>!state.active&&state.latest_run_status==="completed");await p.getByRole("button",{name:t.chatEdit,exact:true}).nth(i).waitFor();
   }
   await p.locator("[data-chat-context-trigger]").click();await p.locator("[data-chat-context-panel]").waitFor();await p.getByText("fixture-64",{exact:true}).waitFor();await p.waitForFunction(()=>{const panel=document.querySelector("[data-chat-context-panel]");if(!panel)return false;const box=panel.getBoundingClientRect();return box.y>=0&&box.x>=0&&box.right<=innerWidth+1});const box=await p.locator("[data-chat-context-panel]").boundingBox();check(box&&box.y>=0&&box.x>=0&&box.x+box.width<=width+1,`context panel extends outside the viewport: ${JSON.stringify(box)}`);check(await p.getByLabel(t.chatContextKeep,{exact:true}).inputValue()==="2","default retention changed");
   const before=await (await api.get(actionURL(path,"context"))).json();check(before.window===64000&&before.measured_input===80,"provider input anchor missing");const count=await fixture.calls({operation:"capacity-switch",boundary:"before"});await fixture.setBudget(32);
   const switched=await (await api.get(actionURL(path,"context"))).json();check(switched.window===32000&&switched.measured_input===0&&switched.estimated,"stale anchor survived model switch");check(await fixture.calls({operation:"capacity-switch",boundary:"after"})===count,"capacity inspection called provider");await p.getByText("fixture-32",{exact:true}).waitFor();
   const state=await (await api.get(path)).json();const expectedFocus="Preserve URLs";await p.getByLabel(t.chatContextFocus,{exact:true}).fill(expectedFocus);
   const stripFocus=process.env.CHAT_CONTEXT_STRIP_FOCUS_CASE===caseName;
   const compactURL=withLocaleQuery(`${base}${actionURL(path,"compact")}`,locale);let focusMutation;
   const stripCurrentFocus=async route=>{
    const request=route.request();if(request.method()!=="POST")return route.continue();
    check(!focusMutation,"current compact POST mutated more than once");
    const original=request.postDataJSON();check(original.focus===expectedFocus,"focus mutation requires original expected focus");
    const {focus,...mutated}=original;focusMutation={case:caseName,marker:caseMarker,url:request.url(),original,mutated};
    console.log("context-focus-mutation",JSON.stringify(focusMutation));
    await route.continue({postData:JSON.stringify(mutated)});
   };
   if(stripFocus)await p.route(compactURL,stripCurrentFocus);
   const requestCursor=(await fixture.requests({operation:"manual-compact",boundary:"before",marker:caseMarker})).length;
   await p.locator("[data-chat-compact]").click();await p.getByText(t.chatContextSuccess,{exact:true}).waitFor();
   if(stripFocus){await p.unroute(compactURL,stripCurrentFocus);check(!!focusMutation,"designated compact POST was not focus-mutated")}
   const compacted=await (await api.get(actionURL(path,"context"))).json();check(Number(compacted.history_revision)===Number(state.history_revision)+1,"compact did not publish revision");check(compacted.summary&&compacted.used<switched.used,"compact did not reduce working context");check(compacted.categories.every(category=>category.estimated),"local categories presented as measured");check(compacted.categories.some(category=>category.name==="system")&&compacted.categories.some(category=>category.name==="conversation"),"manual compaction lost category attribution");check(compacted.before_tokens===switched.categories.reduce((sum,category)=>sum+category.tokens,0),"before/after mixes measured usage with estimates");const captured=await fixture.requests({operation:"manual-compact",boundary:"after",marker:caseMarker});
   console.log("context-summary-capture",JSON.stringify({case:caseName,marker:caseMarker,cursor:requestCursor,requestCount:captured.length,focusMutated:!!focusMutation,requests:captured.slice(requestCursor)}));
   const attempts=assertCurrentSummaryFocus(captured,{cursor:requestCursor,marker:caseMarker,focus:expectedFocus});
   console.log("context-summary-attempts",JSON.stringify({case:caseName,marker:caseMarker,cursor:requestCursor,indices:attempts.map(attempt=>attempt.index)}));
   if(locale==="en"&&surface==="standalone"&&process.env.SCOPEWEAVER_E2E_ARTIFACTS){await p.locator("[data-chat-context-panel]").evaluate(panel=>panel.scrollTop=0);await p.screenshot({path:`${process.env.SCOPEWEAVER_E2E_ARTIFACTS}/context-${width}.png`})}
   if(process.env.CHAT_CONTEXT_DEBUG)await p.evaluate(()=>{window.__contextDebug=[];document.addEventListener("focusin",e=>window.__contextDebug.push(["focus",e.target.tagName,e.target.getAttribute("aria-label")]),true);new MutationObserver(()=>{const t=document.querySelector("[data-chat-context-trigger]");window.__contextDebug.push(["state",t?.getAttribute("data-state"),!!document.querySelector("[data-chat-context-panel]")])}).observe(document.body,{subtree:true,attributes:true,attributeFilter:["data-state"],childList:true})});
   await closeContext(p,t.chatContextClose);const posted=[];p.on("request",r=>{if(r.method()==="POST"&&new URL(r.url()).pathname.includes("/controls/inputs"))posted.push(r.url())});await input.fill("/usage");await p.getByRole("button",{name:t.m0073,exact:true}).click();await p.locator("[data-chat-context-panel]").waitFor();check(posted.length===0,"/usage became a model prompt");await closeContext(p,t.chatContextClose);await input.fill("/compress Preserve tool details 1");await p.getByRole("button",{name:t.m0073,exact:true}).click();await p.getByText(t.chatContextSuccess,{exact:true}).waitFor();const compressedAgain=await (await api.get(actionURL(path,"context"))).json();check(Number(compressedAgain.history_revision)===Number(compacted.history_revision)+1,"/compress did not use the context endpoint");check(posted.length===0,"/compress became an admitted model prompt");
  })}finally{if(process.env.CHAT_CONTEXT_DEBUG)console.log("context-debug",JSON.stringify(await p.evaluate(()=>window.__contextDebug?.slice(-45))));await fixture.release();await p.context().close()}
  if(`${locale}/${width}/${surface}`===(process.env.CHAT_CONTEXT_READINESS_CASE??"zh-Hans/1440/task"))await step(`[${locale}/${width}/${surface}] context panel survives initial controls hydration`,async()=>{
   // Regression: opening /context before the first controls GET resolves must keep the
   // panel open through that response (the original reset treated first hydration as a
   // session switch, force-closing the panel so the later native close failed). The
   // initial GET is held by a route, so the panel is provably pre-hydration and the
   // released response is the real backend's. The guards below use synthetic route state
   // for client identity: a later resolved session change under the same binding must
   // still close the panel, and its late in-flight context response must never reach the
   // view. The next step switches requested task bindings while pinning the actual
   // mounted inspector trigger, so a remount cannot satisfy that separate oracle.
   const tracked=promise=>{promise.catch(()=>{});return promise};
   const within=(promise,ms,message)=>{const timeout=new Promise((_,reject)=>setTimeout(()=>reject(new Error(message)),ms));timeout.catch(()=>{});return Promise.race([promise,timeout])};
   const created=await api.post("/api/tasks",{data:{description:"Context readiness fixture",goal:"Keep fixture idle",seed_first_intent:false}});
   check(created.ok(),await created.text());const task=await created.json();
   check((await api.post(`/api/tasks/${task.id}/control`,{data:{action:"pause"}})).ok(),"readiness fixture pause failed");
   const controlsPath=`/api/tasks/${task.id}/chat/controls`;const url=`${base}/function/tasks/detail?id=${task.id}`;
   const p=await openPage(locale,token);await p.setViewportSize({width,height:900});
   let releaseFirst=()=>{},releaseOld=()=>{},releaseReopened=()=>{};let firstReleased=false;
   try{
    let first=true,armed=false,routed=0;let firstSeenResolve,sessionChangedResolve;
    const firstSeen=new Promise(resolve=>firstSeenResolve=resolve),sessionChanged=new Promise(resolve=>sessionChangedResolve=resolve);
    const firstGate=new Promise(resolve=>releaseFirst=resolve),oldGate=new Promise(resolve=>releaseOld=resolve),reopenGate=new Promise(resolve=>releaseReopened=resolve);
    await p.route(candidate=>candidate.pathname===controlsPath,async route=>{
     if(route.request().method()!=="GET"){await route.continue();return}
     routed++;
     if(first){first=false;firstSeenResolve();await firstGate;await route.continue();return}
     if(armed){const response=await route.fetch();check(response.ok(),"readiness controls fetch failed");const state=await response.json();state.session={...state.session,exploration_id:String(Number(state.session.exploration_id)+1)};await route.fulfill({response,json:state});sessionChangedResolve();return}
     await route.continue();
    });
    const hydration=tracked(p.waitForResponse(response=>new URL(response.url()).pathname===controlsPath&&response.request().method()==="GET"));
    const contextLoaded=tracked(p.waitForResponse(response=>new URL(response.url()).pathname===`${controlsPath}/context`&&response.request().method()==="GET"));
    await p.goto(url);await within(firstSeen,10000,"initial controls GET not observed");
    const input=p.locator("textarea[data-chat-draft]").first();await input.waitFor();await p.locator("[data-chat-context-trigger]").waitFor();
    const calls=await fixture.calls();
    await input.fill("/context");await p.getByRole("button",{name:t.m0073,exact:true}).click();
    await p.locator("[data-chat-context-panel]").waitFor();
    check(routed===1,"another controls request bypassed the held initial response");
    check(await p.getByRole("button",{name:t.chatContextClose,exact:true}).count()===1,"close button missing before the initial controls response");
    releaseFirst();firstReleased=true;check((await hydration).ok(),"initial controls response failed");await contextLoaded;
    check(await p.locator("[data-chat-context-panel]").count()===1,"initial controls hydration closed an already-open context panel");
    check(await fixture.calls()===calls,"readiness /context called the provider");
    await closeContext(p,t.chatContextClose);await p.locator("[data-chat-context-panel]").waitFor({state:"detached"});
    let contexts=0;let contextStartedResolve,staleDeliveredResolve;
    const contextStarted=new Promise(resolve=>contextStartedResolve=resolve),staleDelivered=new Promise(resolve=>staleDeliveredResolve=resolve);
    await p.route(candidate=>candidate.pathname===`${controlsPath}/context`,async route=>{
     contexts++;
     if(contexts===1){contextStartedResolve();await oldGate;await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({history_revision:"1",sequence:"1",profile_id:"1",model:"readiness-stale-context",compactor:"fixture",source:"measured",summary:"",updated_at:new Date(0).toISOString(),window:2,used:1,output_reserve:0,headroom:1,compression_count:0,estimated:false,categories:[]})});staleDeliveredResolve();return}
     await reopenGate;
     await route.continue();
    });
    await p.locator("[data-chat-context-trigger]").click();await p.locator("[data-chat-context-panel]").waitFor();await within(contextStarted,10000,"old-session context load not observed");
    await p.getByLabel(t.chatContextFocus,{exact:true}).fill("readiness old-session focus");await p.getByLabel(t.chatContextKeep,{exact:true}).fill("7");
    armed=true;await p.evaluate(()=>window.dispatchEvent(new Event("focus")));await within(sessionChanged,10000,"session-change fixture not delivered");
    await p.locator("[data-chat-context-panel]").waitFor({state:"detached"});
    releaseOld();await within(staleDelivered,10000,"late context response not delivered");
    await p.locator("[data-chat-context-trigger]").click();await p.locator("[data-chat-context-panel]").waitFor();
    check(await p.getByLabel(t.chatContextFocus,{exact:true}).inputValue()==="","session change retained previous focus");check(await p.getByLabel(t.chatContextKeep,{exact:true}).inputValue()==="2","session change retained previous keep count");
    check(await p.getByText("readiness-stale-context").count()===0,"late context response from the replaced session leaked into the panel");
    check(await p.getByText(t.chatContextUnknown,{exact:true}).last().isVisible(),"reopened panel lost its unobserved context view");
    releaseReopened();await closeContext(p,t.chatContextClose);await p.locator("[data-chat-context-panel]").waitFor({state:"detached"});
    check(await fixture.calls()===calls,"readiness guards called the provider");
   }finally{if(!firstReleased)releaseFirst();releaseOld();releaseReopened();await p.context().close()}
  });
  if(locale+"/"+width+"/"+surface===(process.env.CHAT_CONTEXT_BINDING_CASE??"zh-Hans/1440/task"))await step("["+locale+"/"+width+"/"+surface+"] requested binding resets mounted inspector and rejects late context",async()=>{
   const tracked=promise=>{promise.catch(()=>{});return promise};
   const within=(promise,ms,message)=>{const timeout=new Promise((_,reject)=>setTimeout(()=>reject(new Error(message)),ms));timeout.catch(()=>{});return Promise.race([promise,timeout])};
   const tasks=[];
   for(const name of ["A","B"]){
    const created=await api.post("/api/tasks",{data:{description:"Context binding fixture "+name,goal:"Keep fixture idle",seed_first_intent:false}});check(created.ok(),await created.text());const task=await created.json();
    const paused=await api.post("/api/tasks/"+task.id+"/control",{data:{action:"pause"}});check(paused.ok(),"binding fixture pause failed");check((await paused.json()).paused===true,"binding task pause not persisted");tasks.push(task);
   }
   const [a,b]=tasks;const aPath="/api/tasks/"+a.id+"/chat/controls",bPath="/api/tasks/"+b.id+"/chat/controls";
   const p=await openPage(locale,token);await p.setViewportSize({width,height:900});
   let releaseOld=()=>{},releaseControls=()=>{},releaseNew=()=>{};
   try{
    const firstState=tracked(p.waitForResponse(response=>new URL(response.url()).pathname===aPath&&response.request().method()==="GET"));
    const firstContext=tracked(p.waitForResponse(response=>new URL(response.url()).pathname===aPath+"/context"&&response.request().method()==="GET"));
    await p.goto(base+"/function/tasks/detail?id="+a.id);check((await firstState).ok(),"binding A controls failed");check((await firstContext).ok(),"binding A initial context failed");
    const calls=await fixture.calls();let oldSeenResolve,bSeenResolve,newSeenResolve,oldDeliveredResolve;
    const oldSeen=new Promise(resolve=>oldSeenResolve=resolve),bSeen=new Promise(resolve=>bSeenResolve=resolve),newSeen=new Promise(resolve=>newSeenResolve=resolve),oldDelivered=new Promise(resolve=>oldDeliveredResolve=resolve);
    const oldGate=new Promise(resolve=>releaseOld=resolve),controlGate=new Promise(resolve=>releaseControls=resolve),newGate=new Promise(resolve=>releaseNew=resolve);
    await p.route(candidate=>candidate.pathname===aPath+"/context",async route=>{
     oldSeenResolve();await oldGate;await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({history_revision:"1",sequence:"1",profile_id:"1",model:"requested-binding-stale-context",compactor:"fixture",source:"measured",summary:"",updated_at:new Date(0).toISOString(),window:2,used:1,output_reserve:0,headroom:1,compression_count:0,estimated:false,categories:[]})});oldDeliveredResolve();
    });
    await p.route(candidate=>candidate.pathname===bPath,async route=>{if(route.request().method()!=="GET"){await route.continue();return}bSeenResolve();await controlGate;await route.continue()});
    await p.route(candidate=>candidate.pathname===bPath+"/context",async route=>{newSeenResolve();await newGate;await route.continue()});
    await p.locator("[data-chat-context-trigger]").click();await p.locator("[data-chat-context-panel]").waitFor();await within(oldSeen,10000,"binding A held context not observed");
    await p.getByLabel(t.chatContextFocus,{exact:true}).fill("requested binding old focus");await p.getByLabel(t.chatContextKeep,{exact:true}).fill("7");
    await p.evaluate(()=>{window.__bindingTrigger=document.querySelector("[data-chat-context-trigger]");window.__bindingDocument=document});
    const bState=tracked(p.waitForResponse(response=>new URL(response.url()).pathname===bPath&&response.request().method()==="GET"));
    await p.evaluate(id=>{const url=new URL(location.href);url.searchParams.set("id",id);window.history.pushState(null,"",url)},String(b.id));
    await within(bSeen,10000,"requested binding B controls GET not observed");
    check(await p.evaluate(()=>window.__bindingDocument===document&&window.__bindingTrigger?.isConnected&&window.__bindingTrigger===document.querySelector("[data-chat-context-trigger]")),"requested switch remounted the inspector instead of changing its binding");
    check(new URL(p.url()).searchParams.get("id")===String(b.id),"requested task URL did not change");
    await p.locator("[data-chat-context-panel]").waitFor({state:"detached"});
    await p.locator("[data-chat-context-trigger]").click();await p.locator("[data-chat-context-panel]").waitFor();
    check(await p.getByLabel(t.chatContextFocus,{exact:true}).inputValue()==="","requested binding retained old focus");check(await p.getByLabel(t.chatContextKeep,{exact:true}).inputValue()==="2","requested binding retained old keep count");
    const oldResponse=tracked(p.waitForResponse(response=>new URL(response.url()).pathname===aPath+"/context"&&response.request().method()==="GET"));
    releaseOld();await within(oldDelivered,10000,"binding A stale response not delivered");await (await oldResponse).finished();
    await p.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
    check(await p.getByText("requested-binding-stale-context").count()===0,"late context crossed requested binding");
    check(await p.getByText(t.chatContextUnknown,{exact:true}).last().isVisible(),"requested binding did not retain unknown view while B controls were held");
    releaseControls();check((await bState).ok(),"binding B controls failed");await within(newSeen,10000,"binding B context load not observed");
    check(await p.locator("[data-chat-context-panel]").count()===1,"first hydration of binding B closed its opened panel");
    check(await p.evaluate(()=>window.__bindingTrigger===document.querySelector("[data-chat-context-trigger]")&&window.__bindingTrigger.isConnected),"inspector was remounted after binding B hydration");
    releaseNew();await closeContext(p,t.chatContextClose);await p.locator("[data-chat-context-panel]").waitFor({state:"detached"});
    check(await fixture.calls()===calls,"requested binding guard called provider");
    console.log("requested-binding-receipt",JSON.stringify({from_task:a.id,to_task:b.id,same_trigger:true,b_controls_held_during_reset:true,focus_reset:true,keep_reset:true,stale_context_absent:true,provider_call_delta:0}));
   }finally{releaseOld();releaseControls();releaseNew();await p.context().close()}
  });
 }
 await api.dispose();
}
export async function runChatRuntimeChecks({base,token,fixture,step,check,openPage}){
 const api=await request.newContext({baseURL:base,extraHTTPHeaders:{Authorization:`Bearer ${token}`}});
 for(const locale of LOCALES)for(const width of [1440,390])for(const surface of ["standalone","task"]){
  if(process.env.CHAT_RUNTIME_CASE&&`${locale}/${width}/${surface}`!==process.env.CHAT_RUNTIME_CASE)continue;
  const t=labels[locale];await fixture.resetTools();let path,url,otherPath;
  if(surface==="standalone"){const response=await api.post("/api/conversations",{data:{agent_key:"auto",title:"Steer fixture"}});check(response.ok(),await response.text());const c=await response.json();path=`/api/conversations/${c.id}/controls`;url=`${base}/chat?c=${c.id}`}
  else{const response=await api.post("/api/tasks",{data:{description:"Steering fixture",goal:"Keep idle",seed_first_intent:false}});check(response.ok(),await response.text());const task=await response.json();check((await api.post(`/api/tasks/${task.id}/control`,{data:{action:"pause"}})).ok(),"fixture pause failed");const segment=await api.post(`/api/exploration/main-session/new?task=${task.id}`);check(segment.ok(),await segment.text());const {seq}=await segment.json();path=`/api/tasks/${task.id}/chat/controls?seg=${seq}`;otherPath=`/api/tasks/${task.id}/chat/controls?seg=0`;url=`${base}/function/tasks/detail?id=${task.id}`}
  const p=await openPage(locale,token);await p.setViewportSize({width,height:900});
  try{await step(`[${locale}/${width}/${surface}] running tool steering and shared execution gate`,async()=>{
   await p.goto(url);const input=p.locator("textarea[data-chat-draft]").first();await input.fill("steer-fixture-trigger");await p.getByRole("button",{name:t.m0073,exact:true}).click();await waitChatState(api,path,state=>!!state.active);
   const deadline=Date.now()+15000;while(!(await fixture.stats()).tool_started){check(Date.now()<deadline,"fixture tool did not start");await new Promise(resolve=>setTimeout(resolve,50))}
   if(otherPath){const rejected=await api.post(actionURL(otherPath,"inputs"),{data:{client_request_id:crypto.randomUUID(),mode:"send",message:"must not overlap"}});check(rejected.status()===409,"another task segment entered the active gate");const queued=await api.post(actionURL(otherPath,"inputs"),{data:{client_request_id:crypto.randomUUID(),mode:"queue",message:"queued other segment"}});check(queued.ok(),await queued.text());const state=await (await api.get(path)).json();check(state.inputs.some(item=>item.session.main_seg===0&&item.status==="queued"),"task queue did not share segment inputs")}
   await input.fill("focus correction");await p.getByRole("button",{name:t.chatSteer,exact:true}).click();await p.locator("[data-steer-input]").filter({hasText:"focus correction"}).waitFor();await fixture.release();await waitChatState(api,path,state=>!state.active&&state.latest_run_status==="completed"&&!state.inputs.some(item=>item.status==="queued"));check((await fixture.stats()).marker_calls===0,"steer did not block the next tool");check((await fixture.requests()).some(text=>text.includes("focus correction")),"steer missing from next request");await p.getByText("Correction accepted",{exact:true}).waitFor();
   if(otherPath)check((await fixture.requests()).at(-1).includes("queued other segment"),"other segment's accepted input did not execute")
   await fixture.holdNext();await input.fill("finish before correction delivery");await p.getByRole("button",{name:t.m0073,exact:true}).click();await waitChatState(api,path,state=>!!state.active);await input.fill("late correction");await p.getByRole("button",{name:t.chatSteer,exact:true}).click();await p.locator("[data-steer-input]").filter({hasText:"late correction"}).waitFor();
   let paused=await (await api.get(path)).json();check((await api.post(actionURL(path,"pause"),{data:{expected_revision:paused.revision}})).ok(),"pause before undelivered correction failed");await fixture.release();
   const undelivered=await waitChatState(api,path,state=>!state.active&&state.inputs.some(item=>item.kind==="steer"&&item.status==="undelivered"));await p.getByText(t.chatSteerUndelivered,{exact:true}).waitFor();const beforeFollowup=await fixture.calls();await p.getByRole("button",{name:t.chatSteerFollowUp,exact:true}).click();await p.locator("[data-queue-input]").filter({hasText:"late correction"}).waitFor();
   const followup=await (await api.get(path)).json();check(followup.paused&&followup.inputs.filter(item=>item.prompt.text==="late correction"&&item.kind==="turn").length===1,"correction did not become exactly one paused follow-up");check(followup.inputs.find(item=>item.prompt.text==="late correction").session.main_seg===undelivered.session.main_seg,"follow-up changed its original segment");check(await fixture.calls()===beforeFollowup,"follow-up ignored queue pause");check(await p.getByText(t.chatSteerUndelivered,{exact:true}).count()===0,"converted steer warning remains");await p.getByRole("button",{name:t.chatResumeQueue,exact:true}).click();await waitChatState(api,path,state=>!state.active&&state.inputs.length===0);check((await fixture.requests()).at(-1).includes("late correction"),"follow-up text did not execute");

  })}finally{await fixture.release();await p.context().close()}
 }
 await api.dispose();
}
