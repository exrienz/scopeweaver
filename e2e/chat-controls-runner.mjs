import assert from "node:assert/strict";
import {writeFile} from "node:fs/promises";
import {chromium,request} from "playwright";
import {runChatQueueChecks,runChatHistoryChecks,waitChatState} from "./chat-controls.mjs";
import {runChatPasteChecks} from "./chat-paste.mjs";
import {runChatContextChecks,runChatRuntimeChecks} from "./chat-context.mjs";
import {runChatDraftRetentionChecks} from "./chat-draft-retention.mjs";

const base=process.env.SCOPEWEAVER_E2E_URL;assert.ok(base?.startsWith("http://127.0.0.1:"),"loopback test listener required");
const api=await request.newContext({baseURL:base});
const init=await api.post("/api/auth/init",{data:{password:"chat-controls-pass-1234"}});assert.ok(init.ok(),await init.text());
const login=await api.post("/api/auth/login",{data:{username:"ARTEX",password:"chat-controls-pass-1234"}});assert.ok(login.ok(),await login.text());const {token}=await login.json();
const fixtureRequest=async(action,body)=>{const response=await api.post(`/__fixture/${action}`,{headers:{"X-Chat-Fixture":process.env.SCOPEWEAVER_E2E_FIXTURE},data:body??{}});assert.ok(response.ok(),await response.text());return response.json()};
let activeCase="setup",readSequence=0,countSequence=0;
const fixtureStats=(kind,metadata={})=>{
 let operation=kind,boundary="observation";
 if(activeCase.includes("queue edits, order")){operation="queue-resume";boundary=kind==="calls"?"before":"after"}
 else if(activeCase.includes("retry, edit, branches")&&kind==="calls"){operation="explicit-branch";boundary=++countSequence%2?"before":"after"}
 else if(activeCase.includes("running tool steering")&&kind==="calls"){operation="steer-follow-up";boundary=++countSequence%2?"before":"after"}
 return fixtureRequest("stats",{diagnostic:{case:activeCase,operation,boundary,read:++readSequence,...metadata}});
};
const fixture={resetTools:()=>fixtureRequest("reset-tools"),holdNext:()=>fixtureRequest("hold"),release:()=>fixtureRequest("release"),calls:async metadata=>(await fixtureStats("calls",metadata)).calls,requests:async metadata=>(await fixtureStats("requests",metadata)).requests,stats:metadata=>fixtureStats("stats",metadata),setBudget:window=>fixtureRequest("budget",{window}),seedPendingApproval:async(ref,run,turn)=>(await fixtureRequest("approval",{ref,run,turn})).id,clearPendingApproval:id=>fixtureRequest("clear-approval",{id})};
const browser=await chromium.launch();const errors=[];const pages=[];let failed=0;
const started=performance.now();
const elapsed=()=>Math.round(performance.now()-started);
const helpers={base,token,fixture,check:assert.ok,step:async(name,fn)=>{activeCase=name;readSequence=0;countSequence=0;const began=performance.now();console.log(`START ${name} (elapsed ${elapsed()}ms)`);try{await fn();console.log(`PASS ${name} (${Math.round(performance.now()-began)}ms; elapsed ${elapsed()}ms)`)}catch(error){failed++;console.error(`FAIL ${name} (${Math.round(performance.now()-began)}ms; elapsed ${elapsed()}ms): ${error.stack}`);// Capture while the failing step still owns its live page, before scenario finally closes it.
for(const [index,p] of pages.entries()){
 if(p.isClosed())continue;
 const prefix=`${process.env.SCOPEWEAVER_E2E_ARTIFACTS}/failure-${process.env.CHAT_E2E_SUITE||"standard"}-${failed}-${index}`;
 if(!process.env.SCOPEWEAVER_E2E_ARTIFACTS)continue;
 for(const [kind,capture] of [["screenshot",()=>p.screenshot({path:`${prefix}.png`,timeout:5000})],["dom",async()=>writeFile(`${prefix}.html`,await p.content(),{mode:0o600})],["state",async()=>writeFile(`${prefix}.json`,JSON.stringify({case:name,error:String(error.stack||error),url:p.url(),page_errors:errors}),{mode:0o600})]]){
  try{await capture()}catch(captureError){console.error("diagnostic-capture-failure",JSON.stringify({case:name,page:index,kind,error:String(captureError.stack||captureError)}))}
 }
}
if(process.env.CHAT_QUEUE_RED||process.env.CHAT_CONTEXT_MUTATION)throw error}},openPage:async(locale,jwt)=>{const context=await browser.newContext();context.setDefaultTimeout(15000);await context.addInitScript(([locale,jwt])=>{localStorage.setItem("scopeweaver.locale",locale);localStorage.setItem("artex_token",jwt)},[locale,jwt]);const p=await context.newPage();p.on("pageerror",error=>errors.push(error.message));pages.push(p);return p}};
export async function runChatControlsChecks({base,token,restart,fixture,step,check,openPage}){
 const options={base,token,fixture,step,check,openPage};const only=process.env.CHAT_E2E_SUITE;const standard=!only||only==="standard";
 if(!only||only==="draft")await runChatDraftRetentionChecks(options);
 if(standard||only==="queue")await runChatQueueChecks({...options,recoverInput:restart});
 if(standard||only==="history")await runChatHistoryChecks(options);
 if(standard||only==="context")await runChatContextChecks(options);
 if(standard||only==="runtime")await runChatRuntimeChecks(options);
 if(standard||only==="paste")await runChatPasteChecks(options);
}

try{await runChatControlsChecks({...helpers,restart:id=>fixtureRequest("recover",{input_id:id})});assert.deepEqual(errors,[],"uncaught browser errors");console.log(`Chat browser gate: ${failed===0?"PASS":"FAIL"}`)}finally{await fixture.release();await browser.close();await api.dispose()}
if(failed)process.exitCode=1;
