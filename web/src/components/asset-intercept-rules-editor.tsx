"use client";

import { translate as swt } from "@/i18n/runtime";
import { useI18n } from "@/i18n";
import * as React from "react";

import { PlusIcon, Trash2Icon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import type { AssetInterceptKind, AssetInterceptRuleInput } from "@/lib/types";

// Use NativeSelect rather than shadcn Select inside Sheet drawers:
// body-portaled dropdown interactions can mistakenly close the drawer; native dropdowns avoid this.
export const ASSET_INTERCEPT_KIND_OPTIONS: {
  value: AssetInterceptKind;
  label: string;
  placeholder: string;
}[] = [
  { value: "exact_domain", get label() { return swt("interface.m0664"); }, placeholder: "example.gov.cn" },
  { value: "exact_ip", get label() { return swt("interface.m0665"); }, placeholder: "203.0.113.10" },
  { value: "exact_url", get label() { return swt("interface.m0666"); }, placeholder: "https://example.com/login" },
  { value: "fuzzy_domain", get label() { return swt("interface.m0667"); }, placeholder: ".gov.cn" },
  { value: "fuzzy_ip", get label() { return swt("interface.m0668"); }, placeholder: "203.0.113." },
  { value: "fuzzy_url", get label() { return swt("interface.m0669"); }, placeholder: "/admin" },
  { value: "cidr", get label() { return swt("interface.m0670"); }, placeholder: "192.168.0.0/16" },
];

// AssetInterceptRulesEditor is a controlled multiline editor for block/allow rules,
// types, match values, and notes. The parent owns persistence and submission timing.
export function AssetInterceptRulesEditor({
  value,
  onChange,
}: {
  value: AssetInterceptRuleInput[];
  onChange: (v: AssetInterceptRuleInput[]) => void;
}) {
  "use no memo";
  const { t: swt, locale: swLocale } = useI18n();

  function update(i: number, patch: Partial<AssetInterceptRuleInput>) {
    onChange(value.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  }
  function remove(i: number) {
    onChange(value.filter((_, idx) => idx !== i));
  }
  function add() {
    onChange([...value, { action: "block", kind: "fuzzy_domain", pattern: "", note: "", enabled: true }]);
  }
  return (
    <div className="grid gap-2">
      <p className="text-muted-foreground text-xs">{swt("interface.scopeBoundaryNote")}</p>
      <p className="text-muted-foreground text-xs">{swt("interface.assetRulesBoundaryNote")}</p>
      <p className="text-muted-foreground text-xs">{swt("interface.fuzzyDomainHelp")}</p>
      {value.map((r, i) => {
        const ph = ASSET_INTERCEPT_KIND_OPTIONS.find((o) => o.value === r.kind)?.placeholder ?? "";
        return (
          // biome-ignore lint/suspicious/noArrayIndexKey: controlled rows have no stable IDs, so index keys are appropriate
          <div key={i} className="flex items-center gap-2">
            <NativeSelect
              size="sm"
              className="w-[84px] shrink-0"
              value={r.action}
              onChange={(e) => update(i, { action: e.target.value as "block" | "allow" })}
            >
              <NativeSelectOption value="block">{swt("interface.m0673")}</NativeSelectOption>
              <NativeSelectOption value="allow">{swt("interface.m0617")}</NativeSelectOption>
            </NativeSelect>
            <NativeSelect
              size="sm"
              className="w-[120px] shrink-0"
              value={r.kind}
              onChange={(e) => update(i, { kind: e.target.value as AssetInterceptKind })}
            >
              {ASSET_INTERCEPT_KIND_OPTIONS.map((o) => (
                <NativeSelectOption key={o.value} value={o.value}>
                  {o.label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <Input
              className="flex-1"
              placeholder={ph}
              value={r.pattern}
              onChange={(e) => update(i, { pattern: e.target.value })}
            />
            <Input
              className="w-[120px] shrink-0"
              placeholder={swt("interface.m0674")}
              value={r.note}
              onChange={(e) => update(i, { note: e.target.value })}
            />
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="text-destructive hover:text-destructive size-8 shrink-0"
              onClick={() => remove(i)}
            >
              <Trash2Icon className="size-4" />
            </Button>
          </div>
        );
      })}
      <Button type="button" size="sm" variant="outline" className="w-fit" onClick={add}>
        <PlusIcon className="size-4" /> {swt("interface.m2197")}</Button>
    </div>
  );
}
