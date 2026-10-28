import assert from "node:assert/strict";
import {request} from "playwright";
import { LOCALES, namespaceMessages } from "./locale-resources.mjs";
const labels = namespaceMessages("interface");

export async function waitChatState(api,path,predicate,timeout=20000){const deadline=Date.now()+timeout;let state;while(Date.now()<deadline){const response=await api.get(path);assert.ok(response.ok(),await response.text());state=await response.json();if(predicate(state))return state;await new Promise(resolve=>setTimeout(resolve,75))}throw new Error(`chat state did not settle: ${JSON.stringify(state)}`)}

export async function runChatQueueChecks({base,token,step,check,openPage,recoverInput,fixture}){
 const api=await request.newContext({baseURL:base,extraHTTPHeaders:{Authorization:`Bearer ${token}`}});
 for(const locale of LOCALES)for(const width of [1440,390])for(const surface of ["standalone","task"]){
  if(process.env.CHAT_QUEUE_CASE && `${locale}/${width}/${surface}`!==process.env.CHAT_QUEUE_CASE)continue;
  const t=labels[locale];let path,url,task;
  if(surface==="standalone"){const r=await api.post("/api/conversations",{data:{agent_key:"auto",title:"Queue browser fixture"}});check(r.ok(),"conversation creation failed");const conv=await r.json();path=`/api/conversations/${conv.id}/controls`;url=`${base}/chat?c=${conv.id}`}
  else{const r=await api.post("/api/tasks",{data:{description:"Queue task fixture",goal:"Keep idle",seed_first_intent:false}});check(r.ok(),"queue task failed");task=await r.json();check((await api.post(`/api/tasks/${task.id}/control`,{data:{action:"pause"}})).ok(),"fixture pause failed");path=`/api/tasks/${task.id}/chat/controls?seg=0`;url=`${base}/function/tasks/detail?id=${task.id}`}
  const actionPath=(action)=>{const [p,q]=path.split("?");return `${p}/${action}${q?`?${q}`:""}`};
  const state=await (await api.get(path)).json();check((await api.post(actionPath("pause"),{data:{expected_revision:state.revision}})).ok(),"queue pause failed");
  const p=await openPage(locale,token);await p.setViewportSize({width,height:900});
  await step(`[${locale}/${width}/${surface}] queue edits, order, pause and request identity`,async()=>{
   await p.goto(url);const input=p.locator("textarea[data-chat-draft]").first();await input.waitFor();
   await p.getByRole("button",{name:t.chatResumeQueue,exact:true}).waitFor();
   const submit=async(text)=>{await input.fill(text);await p.getByRole("button",{name:t.chatQueue,exact:true}).click();await p.waitForFunction(()=>document.querySelector("textarea[data-chat-draft]")?.value==="")};
   await submit("first");
   await input.evaluate(element=>{const data=new DataTransfer();data.items.add(new File(["queued document"],"queue-report.txt",{type:"text/plain"}));element.dispatchEvent(new ClipboardEvent("paste",{bubbles:true,cancelable:true,clipboardData:data}))});
   await p.getByRole("button",{name:"queue-report.txt",exact:true}).waitFor();await p.waitForFunction(()=>!document.querySelector("[data-attachment-id] .animate-spin"));
   await submit("second");await submit("third");
   const rows=p.locator("[data-queue-input]");await rows.nth(2).waitFor();
   const beforePatch=await (await api.get(path)).json();const originalOrder=beforePatch.inputs.map(item=>item.id);
   for(const [index,attachments] of [[0,undefined],[2,null]]){
    const item=beforePatch.inputs[index];const data={expected_revision:item.revision,message:item.prompt.text};if(attachments!==undefined)data.attachments=attachments;
    const response=await api.patch(actionPath(`inputs/${item.id}`),{data});check(response.ok(),`text-only API edit failed: ${await response.text()}`);const edited=await response.json();
    check(Array.isArray(edited.prompt.attachments)&&edited.prompt.attachments.length===0,"text-only API edit returned absent attachment manifest");check(edited.revision===String(BigInt(item.revision)+1n)&&edited.ordinal===item.ordinal,"text-only API edit changed revision or ordinal incorrectly");
   }
   const afterPatch=await (await api.get(path)).json();check(afterPatch.revision===String(BigInt(beforePatch.revision)+2n)&&afterPatch.paused===beforePatch.paused,"API edits changed queue pause or gate revision incorrectly");check(JSON.stringify(afterPatch.inputs.map(item=>item.id))===JSON.stringify(originalOrder),"API edits reordered queue");check(JSON.stringify(afterPatch.inputs[1])===JSON.stringify(beforePatch.inputs[1]),"API edits changed attachment-bearing input");
   await p.reload();await rows.nth(2).waitFor();check(JSON.stringify(await rows.evaluateAll(elements=>elements.map(element=>element.getAttribute("data-queue-input"))))===JSON.stringify(originalOrder),"reload changed queue order");
   for(const index of [0,2]){await rows.nth(index).getByRole("button",{name:t.chatQueueEdit,exact:true}).click();const dialog=p.getByRole("dialog");await dialog.locator("textarea").waitFor();check(await dialog.locator("textarea").inputValue()===beforePatch.inputs[index].prompt.text,"text-only editor did not restore API prompt");check(await dialog.locator("[data-attachment-id]").count()===0,"text-only editor restored unexpected attachments");await dialog.getByRole("button",{name:t.chatCancelEdit,exact:true}).click();await dialog.waitFor({state:"hidden"})}
   await rows.nth(1).getByRole("button",{name:t.chatQueueEdit,exact:true}).click();await p.getByRole("dialog").getByRole("button",{name:"queue-report.txt",exact:true}).waitFor();await p.getByRole("dialog").getByRole("button",{name:t.chatQueueSave,exact:true}).click();await p.getByRole("dialog").waitFor({state:"hidden"});
   const attachmentEdit=await (await api.get(path)).json();check(JSON.stringify(attachmentEdit.inputs[1].prompt)===JSON.stringify(beforePatch.inputs[1].prompt),"queue editor changed attachment manifest");check(attachmentEdit.inputs[1].revision===String(BigInt(beforePatch.inputs[1].revision)+1n),"attachment edit did not advance revision once");
   await rows.nth(2).getByRole("button",{name:t.chatMoveUp,exact:true}).click();
   await p.waitForFunction(()=>document.querySelectorAll("[data-queue-input] summary")[1]?.textContent==="third");
   await rows.nth(0).getByRole("button",{name:t.chatQueueRemove,exact:true}).click();await p.waitForFunction(()=>document.querySelectorAll("[data-queue-input]").length===2);
   await input.fill("pre-edit draft");await rows.nth(0).getByRole("button",{name:t.chatQueueEdit,exact:true}).click();await p.getByRole("dialog").locator("textarea").fill("cancelled edit");await p.getByRole("button",{name:t.chatCancelEdit,exact:true}).click();check(await input.inputValue()==="pre-edit draft","cancel overwrote composer draft");
   await rows.nth(0).getByRole("button",{name:t.chatQueueEdit,exact:true}).click();await p.getByRole("dialog").locator("textarea").fill("third revised");await p.getByRole("button",{name:t.chatQueueSave,exact:true}).click();await p.waitForFunction(()=>document.querySelector("[data-queue-input] summary")?.textContent==="third revised");
   const clientIDs=[];let lose=true;
   await p.route("**/controls/inputs**",async route=>{if(route.request().method()!=="POST")return route.continue();clientIDs.push(route.request().postDataJSON().client_request_id);if(lose){lose=false;await route.fetch();await route.abort("failed")}else await route.continue()});
   await input.fill("lost response");await p.getByRole("button",{name:t.chatQueue,exact:true}).click();await rows.nth(2).waitFor();check(await input.inputValue()==="lost response","uncertain submission lost draft");await p.getByRole("button",{name:t.chatQueue,exact:true}).click();await p.waitForFunction(()=>document.querySelector("textarea[data-chat-draft]")?.value==="");check(clientIDs.length===2&&clientIDs[0]===clientIDs[1],"lost response changed client identity");check(await rows.count()===3,"duplicate queued input");
   const other=await openPage(locale,token);await other.goto(url);await other.locator("[data-queue-input]").nth(2).waitFor();check(await other.locator("[data-queue-input] summary").first().textContent()==="third revised","second window disagrees");await other.context().close();
   if(recoverInput){const inputID=await rows.last().getAttribute("data-queue-input");await recoverInput(inputID);await p.getByRole("button",{name:t.chatDiscardLost,exact:true}).waitFor();check(await p.getByRole("button",{name:t.chatResumeQueue,exact:true}).isDisabled(),"unknown run did not block Resume");await p.getByRole("button",{name:t.chatDiscardLost,exact:true}).click();await p.getByRole("button",{name:t.chatDiscardLost,exact:true}).waitFor({state:"hidden"});check(await p.getByRole("button",{name:t.chatResumeQueue,exact:true}).isEnabled(),"discard must leave Resume available");check(await p.getByRole("button",{name:t.chatDiscardLost,exact:true}).count()===0,"discard did not acknowledge unknown input");
    const expected=(await (await api.get(path)).json()).inputs.filter(item=>item.status==="queued").map(item=>item.prompt.text);const before=fixture?await fixture.calls():0;
    await p.getByRole("button",{name:t.chatResumeQueue,exact:true}).click();await waitChatState(api,path,state=>!state.active&&!state.inputs.some(item=>item.status==="queued"));
    if(fixture){const requests=(await fixture.requests()).slice(before);const positions=expected.map(text=>requests.findIndex(request=>request.includes(text)));check(positions.every((position,index)=>position>=0&&(index===0||position>positions[index-1])),"queued execution order changed");check(!requests.some(text=>text.includes("lost-prompt-sentinel")),"discarded prompt replayed")}
}

  });await p.context().close();
 }
 await api.dispose();
}


export async function runChatHistoryChecks({base,token,step,check,openPage,fixture}){
 const api=await request.newContext({baseURL:base,extraHTTPHeaders:{Authorization:`Bearer ${token}`}});
 for(const locale of LOCALES)for(const width of [1440,390])for(const surface of ["standalone","task"]){
  if(process.env.CHAT_HISTORY_CASE&&`${locale}/${width}/${surface}`!==process.env.CHAT_HISTORY_CASE)continue;
  const t=labels[locale];let parent,path,url,task;
  if(surface==="standalone"){const r=await api.post("/api/conversations",{data:{agent_key:"auto",title:"History browser fixture"}});check(r.ok(),"conversation fixture failed");parent=await r.json();path=`/api/conversations/${parent.id}/controls`;url=`${base}/chat?c=${parent.id}`}
  else{const r=await api.post("/api/tasks",{data:{description:"History task fixture",goal:"Keep idle",seed_first_intent:false}});check(r.ok(),"task fixture failed");task=await r.json();check((await api.post(`/api/tasks/${task.id}/control`,{data:{action:"pause"}})).ok(),"fixture pause failed");path=`/api/tasks/${task.id}/chat/controls?seg=0`;url=`${base}/function/tasks/detail?id=${task.id}`}
  const p=await openPage(locale,token);await p.setViewportSize({width,height:900});
  try{await step(`[${locale}/${width}/${surface}] retry, edit, branches and archived attempts`,async()=>{
   await p.goto(url);const input=p.locator("textarea[data-chat-draft]").first();await input.waitFor();await p.locator("[data-chat-queue]").waitFor();
   await fixture.holdNext();await input.fill("first prompt");await p.getByRole("button",{name:t.m0073,exact:true}).click();await p.locator("div").filter({hasText:/^first prompt$/}).last().waitFor();await fixture.release();
   await p.getByRole("button",{name:t.chatEdit,exact:true}).last().waitFor();
   await input.fill("second prompt");await input.evaluate(element=>{const data=new DataTransfer();data.items.add(new File(["document"],"report.txt",{type:"text/plain"}));element.dispatchEvent(new ClipboardEvent("paste",{bubbles:true,cancelable:true,clipboardData:data}))});
   await p.getByRole("button",{name:"report.txt",exact:true}).waitFor();await p.waitForFunction(()=>!document.querySelector("[data-attachment-id] .animate-spin"));
   await p.getByRole("button",{name:t.m0073,exact:true}).click();await p.locator("div").filter({hasText:/^second prompt$/}).last().waitFor();await p.getByRole("button",{name:t.chatEdit,exact:true}).nth(1).waitFor();
   await input.fill("unsent draft");await p.getByRole("button",{name:t.chatEdit,exact:true}).last().click();const dialog=p.getByRole("dialog");await dialog.locator("textarea").waitFor();check(await dialog.locator("textarea").inputValue()==="second prompt","edit did not restore original prompt");await dialog.getByRole("button",{name:"report.txt",exact:true}).waitFor();await dialog.getByRole("button",{name:t.chatCancelEdit,exact:true}).click();await dialog.waitFor({state:"hidden"});check(await input.inputValue()==="unsent draft","cancelled edit changed composer draft");
   const countBefore=surface==="standalone"?(await (await api.get("/api/conversations")).json()).conversations.length:0;
   const sourceState=await (await api.get(path)).json();const pendingID=fixture.seedPendingApproval?await fixture.seedPendingApproval(sourceState.session,sourceState.latest_run_id,sourceState.latest_turn_id):null;
   await p.getByRole("button",{name:t.chatRetry,exact:true}).last().click();await p.getByRole("dialog").getByText(t.chatRetryEffect,{exact:true}).waitFor();await p.getByRole("dialog").getByRole("button",{name:t.chatRetry,exact:true}).click();
   await p.waitForFunction(()=>!document.querySelector('[role="dialog"]'));await p.getByRole("button",{name:t.chatRetry,exact:true}).last().waitFor();
   if(surface==="standalone")check((await (await api.get("/api/conversations")).json()).conversations.length===countBefore,"retry created a rail entry");
   await p.getByRole("button",{name:t.chatPreviousAttempts,exact:true}).last().click();await p.getByRole("dialog").getByText(new RegExp(t.chatReadAttempt)).waitFor();await p.getByRole("dialog").locator("summary").first().click();await p.getByRole("dialog").getByText("second prompt",{exact:true}).waitFor();check(await p.getByRole("dialog").locator("[data-chat-actions]").count()===0,"archived attempt is editable");if(pendingID){await p.getByRole("dialog").getByText(`#${pendingID}`,{exact:true}).waitFor();check(await p.getByRole("dialog").getByRole("button",{name:t.m0617,exact:true}).count()===0,"archived approval still permits Allow");check(await p.getByRole("dialog").getByRole("button",{name:t.m2063,exact:true}).count()===0,"archived approval still permits Deny");await fixture.clearPendingApproval(pendingID)}await p.getByRole("dialog").press("Escape");
   const calls=await fixture.calls();await p.getByRole("button",{name:t.chatBranch,exact:true}).last().click();await p.getByRole("dialog").getByRole("textbox",{name:t.chatBranchTitle,exact:true}).fill("Explicit child");await p.getByRole("dialog").getByRole("button",{name:t.chatBranch,exact:true}).click();await p.getByRole("button",{name:t.chatParentConversation,exact:true}).waitFor();check(await fixture.calls()===calls,"explicit branch ran a model");
   if(surface==="standalone"){check(new URL(p.url()).searchParams.get("c")!==String(parent.id),"branch did not navigate to child");check((await api.delete(`/api/conversations/${parent.id}`)).ok(),"parent deletion failed");await p.getByText(t.chatParentUnavailable,{exact:true}).waitFor();await p.getByText("report.txt",{exact:true}).last().waitFor();const childID=new URL(p.url()).searchParams.get("c");const doc=await api.get(`/api/workspace/read?path=${encodeURIComponent(`sessions/conv-${childID}/uploads/report.txt`)}`);check(doc.ok(),"child document unavailable after parent deletion");check((await doc.json()).content==="document","child document bytes changed")}
   else await p.getByText(t.chatSharedTask,{exact:true}).first().waitFor();
   await p.getByRole("button",{name:t.chatEdit,exact:true}).first().click();await p.getByRole("dialog").locator("textarea").fill("earlier edited");await p.getByRole("dialog").getByText(t.chatEditEarlier,{exact:true}).waitFor();await p.getByRole("dialog").getByRole("button",{name:t.chatSaveAndRun,exact:true}).click();await p.getByText("earlier edited",{exact:true}).waitFor();await p.getByRole("button",{name:t.chatParentConversation,exact:true}).waitFor();
  })}finally{await fixture.release();await p.context().close()}
 }
 await api.dispose();
}
