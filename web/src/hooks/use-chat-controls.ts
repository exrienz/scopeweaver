"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { chatControlsBase, chatControlsURL } from "@/lib/chat-commands";
import { newChatID } from "@/lib/chat-attachment-state";
import { http, HTTPError } from "@/lib/api";
import type {
  ChatContextView,
  ChatCompactOptions,
  ChatAdmission,
  ChatControlState,
  ChatPrompt,
  ChatSessionRef,
  ChatHistoryChange,
  ChatHistoryResult,
  ChatAttempt,
  Activity,
} from "@/lib/types";

// Uncertain responses retain the request identity across component lifetimes.
const requests = new Map<string, string>();
export async function submitChatAdmission(
  ref: ChatSessionRef,
  mode: "send" | "queue" | "steer",
  prompt: ChatPrompt,
  targetRunID?: string,
) {
  const key = chatControlsBase(ref);
  const fingerprint = JSON.stringify([key, mode, prompt, targetRunID]);
  const clientID = requests.get(fingerprint) ?? newChatID();
  requests.set(fingerprint, clientID);
  try {
    const result = await http<ChatAdmission>(chatControlsURL(ref, "inputs"), {
      method: "POST",
      body: JSON.stringify({
        client_request_id: clientID,
        mode,
        message: prompt.text,
        attachments: prompt.attachments,
        target_run_id: targetRunID,
      }),
    });
    requests.delete(fingerprint);
    return result;
  } catch (error) {
    if (error instanceof HTTPError && error.status >= 400 && error.status < 500) requests.delete(fingerprint);
    throw error;
  }
}

export function useChatControls(ref: ChatSessionRef | null) {
  const key = ref ? chatControlsBase(ref) : "";
  const active = useRef(key);
  active.current = key;
  const generation = useRef(0);
  const [snapshot, setSnapshot] = useState<{ key: string; state: ChatControlState | null; error?: string }>({
    key: "",
    state: null,
  });
  const state = snapshot.key === key ? snapshot.state : null;
  const error = snapshot.key === key ? snapshot.error : undefined;
  const reload = useCallback(async () => {
    if (!ref) return;
    const request = ++generation.current;
    try {
      const next = await http<ChatControlState>(chatControlsURL(ref));
      if (active.current === key && generation.current === request) setSnapshot({ key, state: next });
    } catch (error) {
      if (active.current === key && generation.current === request)
        setSnapshot((current) => ({
          key,
          state: current.key === key ? current.state : null,
          error: (error as Error).message,
        }));
    }
  }, [key]);
  useEffect(() => {
    void reload();
    const focus = () => void reload();
    window.addEventListener("focus", focus);
    const timer = setInterval(
      () => {
        if (document.visibilityState === "visible") void reload();
      },
      state?.active || state?.inputs.some((a) => a.status === "queued" || a.status === "claimed") ? 1000 : 5000,
    );
    return () => {
      generation.current++;
      clearInterval(timer);
      window.removeEventListener("focus", focus);
    };
  }, [reload, !!state?.active, !!state?.inputs.some((a) => a.status === "queued" || a.status === "claimed")]);
  async function mutate<T>(path: string, method = "POST", body?: unknown, target = ref, query = ""): Promise<T> {
    if (!target) throw new Error("Chat session unavailable");
    try {
      return await http<T>(chatControlsURL(target, path, query), {
        method,
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      });
    } finally {
      await reload();
    }
  }
  async function submit(mode: "send" | "queue" | "steer", prompt: ChatPrompt, targetRunID?: string) {
    if (!ref) throw new Error("Chat session unavailable");
    try {
      return await submitChatAdmission(ref, mode, prompt, targetRunID);
    } finally {
      await reload();
    }
  }
  const targetFor = (id: string) => state?.inputs.find((item) => item.id === id)?.session ?? ref;
  return {
    // Stable requested binding (chatControlsBase of the current ref). Lets consumers
    // distinguish initial hydration of one binding from a real binding/session switch.
    key,
    context: () => http<ChatContextView>(chatControlsURL(ref!, "context")),
    compact: async (options: ChatCompactOptions) => {
      if (!ref) throw new Error("Chat session unavailable");
      const fingerprint = JSON.stringify([key, "compact", options]);
      const clientID = requests.get(fingerprint) ?? newChatID();
      requests.set(fingerprint, clientID);
      try {
        const result = await mutate<ChatContextView>("compact", "POST", {
          ...options,
          client_request_id: clientID,
          expected_revision: state?.revision,
        });
        requests.delete(fingerprint);
        return result;
      } catch (error) {
        if (error instanceof HTTPError && error.status >= 400 && error.status < 500) requests.delete(fingerprint);
        throw error;
      }
    },
    prompt: (turnID: string) => http<ChatPrompt>(chatControlsURL(ref!, `turns/${encodeURIComponent(turnID)}/prompt`)),
    retry: (clientID: string) =>
      mutate<ChatHistoryResult>("retry", "POST", { client_request_id: clientID, expected_revision: state?.revision }),
    edit: (change: ChatHistoryChange) => mutate<ChatHistoryResult>("edit", "POST", change),
    branch: (change: ChatHistoryChange) => mutate<ChatHistoryResult>("branch", "POST", change),
    attempts: (turnID: string) =>
      http<{ attempts: ChatAttempt[] }>(chatControlsURL(ref!, "attempts", `turn_id=${encodeURIComponent(turnID)}`)),
    attemptActivities: (runID: string, before = 0) =>
      http<{ items: Activity[]; has_more: boolean }>(
        chatControlsURL(ref!, `attempts/${encodeURIComponent(runID)}/activities`, `before=${before}`),
      ),
    state,
    error,
    reload,
    submit,
    updateInput: (id: string, expected: string, prompt: ChatPrompt) =>
      mutate<ChatAdmission>(
        `inputs/${encodeURIComponent(id)}`,
        "PATCH",
        { expected_revision: expected, message: prompt.text, attachments: prompt.attachments },
        targetFor(id),
      ),
    removeInput: (id: string, expected: string) =>
      mutate<void>(
        `inputs/${encodeURIComponent(id)}`,
        "DELETE",
        undefined,
        targetFor(id),
        `expected_revision=${encodeURIComponent(expected)}`,
      ),
    reorder: (expected: string, ids: string[]) =>
      mutate<void>("queue/order", "POST", { expected_revision: expected, input_ids: ids }),
    pause: () => mutate<ChatControlState>("pause", "POST", { expected_revision: state?.revision }),
    resume: () => mutate<ChatControlState>("resume", "POST", { expected_revision: state?.revision }),
    queueSteerFollowup: (id: string, expected: string) =>
      mutate<ChatAdmission>(
        `inputs/${encodeURIComponent(id)}/followup`,
        "POST",
        { expected_revision: expected },
        targetFor(id),
      ),
    discardUnknown: (id: string, expected: string) =>
      mutate<ChatControlState>(
        `inputs/${encodeURIComponent(id)}/discard`,
        "POST",
        { expected_revision: expected },
        targetFor(id),
      ),
  };
}
