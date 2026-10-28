"use client";

import { translate as swt } from "@/i18n/runtime";
import { useI18n } from "@/i18n";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Textarea } from "@/components/ui/textarea";
import type { ParsedCompanyScopeText } from "@/lib/company-scope";
import type { CompanyScopeKind } from "@/lib/types";

const SCOPE_KIND_LABELS: Record<CompanyScopeKind, string> = {
  get domain() { return swt("interface.m0232"); },
  get host() { return swt("interface.m2883"); },
  ip: "IP",
  cidr: "CIDR",
  icp: "ICP",
  get keyword() { return swt("interface.m0599"); },
};

// Hover hints explain the breadth difference between the two name kinds.
const SCOPE_KIND_HINTS: Partial<Record<CompanyScopeKind, string>> = {
  get domain() { return swt("interface.m2885"); },
  get host() { return swt("interface.m2884"); },
};

export function ScopeTextEditor({
  id,
  value,
  onValueChange,
  parsed,
  label = swt("interface.m0227"),
  description = swt("interface.m2285"),
}: {
  id: string;
  value: string;
  onValueChange: (value: string) => void;
  parsed: ParsedCompanyScopeText;
  label?: string;
  description?: string;
}) {
  "use no memo";
  const { t: swt, locale: swLocale } = useI18n();

  const counts = React.useMemo(() => {
    const result = new Map<CompanyScopeKind, number>();
    for (const rule of parsed.rules) result.set(rule.kind, (result.get(rule.kind) ?? 0) + 1);
    return result;
  }, [swLocale, parsed.rules]);

  return (
    <Field data-invalid={parsed.errors.length > 0}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <FieldDescription>{description}</FieldDescription>
      <FieldDescription>{swt("interface.scopeBoundaryNote")}</FieldDescription>
      <Textarea
        id={id}
        rows={8}
        value={value}
        aria-invalid={parsed.errors.length > 0}
        placeholder={swt("interface.m2286")}
        className="min-h-36 resize-y font-mono text-sm"
        onChange={(event) => onValueChange(event.target.value)}
      />
      {parsed.rules.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 text-muted-foreground text-xs">
          <span>{swt("interface.m2287")}{" "}{parsed.rules.length} {swt("interface.m0328")}</span>
          {Object.entries(SCOPE_KIND_LABELS).map(([kind, kindLabel]) => {
            const count = counts.get(kind as CompanyScopeKind) ?? 0;
            return count > 0 ? (
              <Badge
                key={kind}
                variant="secondary"
                className="font-mono tabular-nums"
                title={SCOPE_KIND_HINTS[kind as CompanyScopeKind]}
              >
                {kindLabel} {count}
              </Badge>
            ) : null;
          })}
        </div>
      )}
      {parsed.errors.length > 0 && (
        <FieldError>
          {parsed.errors.slice(0, 5).map((item) => (
            <span key={`${item.line}-${item.error}`} className="block">
              {swt("interface.m0489")}{" "}{item.line} {swt("interface.m2288")}{item.error}
            </span>
          ))}
          {parsed.errors.length > 5 && <span className="block">{swt("interface.m2203")}{" "}{parsed.errors.length - 5} {swt("interface.m2289")}</span>}
        </FieldError>
      )}
    </Field>
  );
}
