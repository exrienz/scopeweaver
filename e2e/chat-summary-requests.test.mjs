import assert from "node:assert/strict";
import test from "node:test";
import {assertCurrentSummaryFocus} from "./chat-summary-requests.mjs";

const system = "You are a helpful AI assistant tasked with summarizing conversations.";
const marker = "context-case-current";
const focus = "Preserve URLs";
const instruction = {role: "user", content: `Compression focus: ${focus}`};
const summary = (messages = [instruction], caseMarker = marker) => JSON.stringify([
  {role: "system", content: system},
  {role: "user", content: `context turn 0: ${caseMarker}`},
  ...messages,
]);
const ordinary = JSON.stringify([{role: "user", content: "unrelated planner traffic"}]);
const verify = (requests, cursor = 0) => assertCurrentSummaryFocus(requests, {cursor, marker, focus});

test("current summary survives unrelated trailing provider traffic", () => {
  assert.deepEqual(verify([summary(), summary(), ordinary], 1).map(attempt => attempt.index), [1]);
});

test("cursor excludes older same-marker success when current focus is missing", () => {
  assert.throws(() => verify([summary(), summary([])], 1), /focus missing from current summary request/);
});

test("older same-marker summary alone cannot establish a current attempt", () => {
  assert.throws(() => verify([summary(), ordinary], 1), /current compaction summary request missing/);
});

test("missing summary and wrong conversation marker fail", () => {
  for (const requests of [[], [ordinary], [summary([instruction], "context-case-other")]]) {
    assert.throws(() => verify(requests), /current compaction summary request missing/);
  }
});

test("marked ordinary requests and non-system summary decoys are excluded", () => {
  const marked = {role: "user", content: marker};
  for (const messages of [
    [marked, instruction],
    [{role: "user", content: system}, marked, instruction],
    [{role: "system", content: `Quoted: ${system}`}, marked, instruction],
  ]) {
    assert.throws(() => verify([JSON.stringify(messages)]), /current compaction summary request missing/);
  }
});

test("missing focus, wrong role and substring-only focus decoys fail", () => {
  for (const messages of [
    [],
    [{role: "system", content: instruction.content}],
    [{role: "assistant", content: instruction.content}],
    [{role: "user", content: `Quoted: ${instruction.content}`}],
    [{role: "user", content: `${instruction.content} and details`}],
    [{role: "user", content: focus}],
  ]) {
    assert.throws(() => verify([summary(), summary(messages)], 1), /focus missing from current summary request/);
  }
});

test("every matching attempt needs exact focus, in either order", () => {
  for (const requests of [[summary(), summary([])], [summary([]), summary()]]) {
    assert.throws(() => verify(requests), /focus missing from current summary request/);
  }
});

test("multiple valid retries pass and unrelated marked traffic is ignored", () => {
  const markedOrdinary = JSON.stringify([{role: "user", content: marker}]);
  const attempts = verify([summary(), markedOrdinary, summary(), ordinary, summary()]);
  assert.deepEqual(attempts.map(attempt => attempt.index), [0, 2, 4]);
});
