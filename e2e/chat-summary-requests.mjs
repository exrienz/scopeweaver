import assert from "node:assert/strict";

// Pinned Norma v0.4.3 compaction/summarizer.go; identity must not depend on focus.
const summarySystem = "You are a helpful AI assistant tasked with summarizing conversations.";

export function assertCurrentSummaryFocus(requests, {cursor, marker, focus}) {
  const attempts = requests.slice(cursor).map((raw, offset) => {
    const messages = JSON.parse(raw);
    assert.ok(Array.isArray(messages), "captured provider messages must be an array");
    return {index: cursor + offset, messages};
  }).filter(({messages}) =>
    messages.some(message => message.role === "system" && message.content === summarySystem) &&
    messages.some(message => message.role === "user" && typeof message.content === "string" && message.content.includes(marker))
  );
  const evidence = JSON.stringify({cursor, marker, requestCount: requests.length, summaryIndices: attempts.map(attempt => attempt.index)});
  assert.ok(attempts.length > 0, `current compaction summary request missing: ${evidence}`);
  assert.ok(attempts.every(({messages}) => messages.some(message =>
    message.role === "user" && message.content === `Compression focus: ${focus}`
  )), `focus missing from current summary request: ${evidence}`);
  return attempts;
}
