"use client";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { Gauge, Loader2, X } from "lucide-react";
import { useI18n } from "@/i18n";
import { useIsMobile } from "@/hooks/use-mobile";
import type { useChatControls } from "@/hooks/use-chat-controls";
import type { ChatContextView, ChatCompactOptions } from "@/lib/types";
import { formatChatContext, canCompactChatContext, chatCompactDefaults } from "@/lib/chat-context-view";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
  DrawerTrigger,
} from "@/components/ui/drawer";

type Controls = ReturnType<typeof useChatControls>;
export type ChatContextHandle = { open: () => void; compact: (options: ChatCompactOptions) => Promise<boolean> };
export const ChatContextInspector = forwardRef<ChatContextHandle, { controls: Controls | null }>(
  function ChatContextInspector({ controls }, ref) {
    const { t, locale } = useI18n();
    const mobile = useIsMobile();
    const [open, setOpen] = useState(false);
    const [view, setView] = useState<ChatContextView | null>(null);
    const [focus, setFocus] = useState("");
    const [keep, setKeep] = useState(String(chatCompactDefaults.keep_recent));
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const [success, setSuccess] = useState(false);
    const working = useRef(false);
    const epoch = useRef(0);
    const alive = useRef(true);
    const panel = useRef<HTMLDivElement>(null);
    useEffect(() => {
      if (open && panel.current) panel.current.scrollTop = 0;
    }, [open]);
    const binding = controls?.key ?? "";
    const session = controls?.state?.session ? JSON.stringify(controls.state.session) : "";
    // Hydration bookkeeping: the first resolved session for a binding is initial
    // hydration, not a session switch. A panel opened before that response must survive
    // it, while later session changes under the same binding (task/exploration/segment
    // can move under one URL) and binding switches must keep resetting everything and
    // dropping late in-flight context responses.
    const hydrated = useRef({ binding: "", session: "" });
    useEffect(() => {
      alive.current = true;
      return () => {
        alive.current = false;
        epoch.current++;
      };
    }, []);
    useEffect(() => {
      const previous = hydrated.current;
      hydrated.current = { binding, session };
      // This binding has not resolved a session yet: the upcoming null -> resolved
      // transition is hydration, so leave whatever the user opened/typed untouched.
      if (previous.binding === binding && !previous.session) return;
      epoch.current++;
      setView(null);
      setError("");
      setSuccess(false);
      setFocus("");
      setKeep(String(chatCompactDefaults.keep_recent));
      setOpen(false);
    }, [binding, session]);
    async function load() {
      if (!controls?.state || working.current) return;
      const token = ++epoch.current;
      try {
        const next = await controls.context();
        if (alive.current && epoch.current === token) {
          setView(next);
          setError("");
        }
      } catch (error) {
        if (alive.current && epoch.current === token) setError((error as Error).message);
      }
    }
    useEffect(() => {
      void load();
    }, [controls?.state, locale]);
    function openFromComposer() {
      // Let the composer's pointer event finish before the popover installs
      // outside-click handling (including a previous closing layer).
      requestAnimationFrame(() => {
        if (alive.current) setOpen(true);
      });
    }
    async function compact(options: ChatCompactOptions) {
      openFromComposer();
      setFocus(options.focus);
      setKeep(String(options.keep_recent));
      if (!controls || working.current || !canCompactChatContext(controls.state, view)) {
        setError(t("interface.chatControlUnavailable"));
        return false;
      }
      working.current = true;
      setBusy(true);
      setError("");
      setSuccess(false);
      const token = ++epoch.current;
      try {
        const next = await controls.compact(options);
        if (alive.current && epoch.current === token) {
          setView(next);
          setSuccess(true);
        }
        return true;
      } catch (error) {
        if (alive.current && epoch.current === token) setError((error as Error).message);
        return false;
      } finally {
        working.current = false;
        if (alive.current) setBusy(false);
      }
    }
    useImperativeHandle(ref, () => ({
      open: () => {
        openFromComposer();
        void load();
      },
      compact,
    }));
    const formatted = formatChatContext(view, locale);
    const disabled =
      busy ||
      !canCompactChatContext(controls?.state ?? null, view) ||
      !Number.isSafeInteger(Number(keep)) ||
      Number(keep) < 1;
    const trigger = (
      <Button
        variant="ghost"
        size="sm"
        className="h-7 max-w-full min-w-0 gap-1.5 px-2 text-xs"
        data-chat-context-trigger
        title={t("interface.chatContext")}
      >
        <Gauge className="size-3.5 shrink-0" />
        <span className="truncate">
          {t("interface.chatContext")}: {formatted.usage}
        </span>
      </Button>
    );
    const header = (
      <div className="flex shrink-0 items-center justify-between gap-3 px-4 pt-3 pb-2">
        <h2 className="font-medium">{t("interface.chatContext")}</h2>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={t("interface.chatContextClose")}
          onClick={() => setOpen(false)}
        >
          <X className="size-4" />
        </Button>
      </div>
    );
    const body = (
      <div className="min-h-0 space-y-4 overflow-y-auto px-4 pb-4" ref={panel} data-chat-context-panel>
        <p className="text-lg tabular-nums" data-chat-context-usage>
          {formatted.usage}
        </p>
        <p className="text-xs text-muted-foreground">{t("interface.chatContextUsageHelp")}</p>
        <dl className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-x-3 gap-y-2 text-xs">
          <dt>{t("interface.chatContextModel")}</dt>
          <dd className="break-words">{view?.model || "—"}</dd>
          <dt>{t("interface.chatContextHeadroom")}</dt>
          <dd>{formatted.headroom}</dd>
          <dt>{t("interface.chatContextReserve")}</dt>
          <dd>
            {view
              ? `${view.output_reserve.toLocaleString()}${view.reserve_estimated ? ` · ${t("interface.chatContextEstimated")}` : ""}`
              : "—"}
          </dd>
          <dt>{t("interface.chatContextInput")}</dt>
          <dd>{view?.measured_input ? view.measured_input.toLocaleString() : "—"}</dd>
          <dt>{t("interface.chatContextCompactor")}</dt>
          <dd>{view?.compactor || "—"}</dd>
          <dt>{t("interface.chatContextCompressions")}</dt>
          <dd>{view?.compression_count ?? 0}</dd>
          <dt>{t("interface.chatContextUpdated")}</dt>
          <dd>
            {view?.updated_at && Date.parse(view.updated_at) > 0
              ? new Date(view.updated_at).toLocaleString(locale)
              : "—"}
          </dd>
          {!!view?.before_tokens && (
            <>
              <dt>{t("interface.chatContextBeforeAfter")}</dt>
              <dd>
                {view.before_tokens.toLocaleString()} / {view.after_tokens?.toLocaleString() ?? "—"}
              </dd>
            </>
          )}
        </dl>
        {!!formatted.categories.length && (
          <table className="w-full text-xs">
            <caption className="mb-2 text-left font-medium">{t("interface.chatContextCategories")}</caption>
            <tbody>
              {formatted.categories.map((row) => (
                <tr key={row.name} className="border-b">
                  <th scope="row" className="py-1 text-left font-normal">
                    {row.name}
                  </th>
                  <td className="text-right tabular-nums">{row.label}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <details className="text-xs">
          <summary>{t("interface.chatContextSummary")}</summary>
          <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words font-sans">
            {view?.summary || t("interface.chatContextNoSummary")}
          </pre>
        </details>
        <form
          className="space-y-3 border-t pt-3"
          onSubmit={(event) => {
            event.preventDefault();
            if (!disabled) void compact({ focus, keep_recent: Number(keep) });
          }}
        >
          <label className="block space-y-1 text-xs">
            <span>{t("interface.chatContextFocus")}</span>
            <Textarea
              value={focus}
              onChange={(event) => setFocus(event.target.value)}
              disabled={busy}
              maxLength={4000}
              rows={2}
            />
          </label>
          <label className="block space-y-1 text-xs">
            <span>{t("interface.chatContextKeep")}</span>
            <Input
              type="number"
              min={1}
              step={1}
              value={keep}
              onChange={(event) => setKeep(event.target.value)}
              disabled={busy}
            />
          </label>
          <p className="text-xs text-muted-foreground">{t("interface.chatContextHelp")}</p>
          <Button type="submit" size="sm" disabled={disabled} data-chat-compact>
            {busy && <Loader2 className="size-3 animate-spin" />}
            {busy ? t("interface.chatContextCompacting") : t("interface.chatContextCompact")}
          </Button>
          {error && (
            <p role="alert" className="text-xs text-destructive">
              {error}
            </p>
          )}
          {success && (
            <p role="status" className="text-xs text-green-600">
              {t("interface.chatContextSuccess")}
            </p>
          )}
        </form>
      </div>
    );
    const changeOpen = (value: boolean) => {
      setOpen(value);
      if (value) void load();
    };
    return (
      <div className="min-w-0 border-t px-2 py-1">
        {mobile ? (
          <Drawer open={open} onOpenChange={changeOpen}>
            <DrawerTrigger asChild>{trigger}</DrawerTrigger>
            <DrawerContent>
              <DrawerHeader className="sr-only">
                <DrawerTitle>{t("interface.chatContext")}</DrawerTitle>
                <DrawerDescription>{t("interface.chatContextUsageHelp")}</DrawerDescription>
              </DrawerHeader>
              {header}
              {body}
            </DrawerContent>
          </Drawer>
        ) : (
          <Popover open={open} onOpenChange={changeOpen}>
            <PopoverTrigger asChild>{trigger}</PopoverTrigger>
            <PopoverContent
              align="end"
              side="top"
              className="max-h-[min(80vh,var(--radix-popover-content-available-height,80vh))] w-[min(28rem,calc(100vw-1rem))] gap-0 overflow-hidden p-0"
              onCloseAutoFocus={(event) => event.preventDefault()}
              aria-label={t("interface.chatContext")}
            >
              {header}
              {body}
            </PopoverContent>
          </Popover>
        )}
      </div>
    );
  },
);
