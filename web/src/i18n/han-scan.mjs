// AST-aware scanner for Han (CJK ideograph) characters in runtime code.
// Comments are ignored; string literals, template literals, JSX text and
// regex literals are reported. Used by the i18n coverage test and by
// `node src/i18n/han-scan.mjs` for a manual audit.

import ts from "typescript";

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HAN = /\p{Script=Han}/u;

export const SRC_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

function walk(dir, out) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full, out);
    else if (/\.(tsx?|mjs|js)$/.test(entry.name)) out.push(full);
  }
  return out;
}

export function sourceFiles(root = SRC_ROOT) {
  return walk(root, []).sort();
}

export function scanFile(file) {
  return scanText(fs.readFileSync(file, "utf8"), file);
}

export function scanText(text, file = path.join(SRC_ROOT, "ordinary.tsx")) {
  let kind = ts.ScriptKind.JS;
  if (file.endsWith(".tsx")) kind = ts.ScriptKind.TSX;
  else if (file.endsWith(".ts")) kind = ts.ScriptKind.TS;
  const sf = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, kind);
  const hits = [];
  const visit = (node) => {
    if (
      ts.isStringLiteralLike(node) ||
      ts.isTemplateHead(node) ||
      ts.isTemplateMiddle(node) ||
      ts.isTemplateTail(node) ||
      ts.isJsxText(node) ||
      ts.isRegularExpressionLiteral(node)
    ) {
      const value = ts.isRegularExpressionLiteral(node) ? node.text : (node.text ?? node.getText(sf));
      if (HAN.test(value)) {
        // Only direct string values of the six typed CJK catalog objects
        // are translation resources. Other code in this file remains scanned.
        const property = node.parent;
        const declaration = property?.parent?.parent;
        if (
          ["zh-hans.ts", "zh-hant.ts", "ja.ts"].some(
            (name) => path.resolve(file) === path.join(SRC_ROOT, "i18n/messages", name),
          ) &&
          ts.isStringLiteral(node) &&
          property &&
          ts.isPropertyAssignment(property) &&
          property.initializer === node &&
          declaration &&
          ts.isVariableDeclaration(declaration) &&
          ["app", "common", "english", "interfaceMessages", "providers", "settings"].includes(
            declaration.name.getText(sf),
          )
        )
          return;
        const { line } = sf.getLineAndCharacterOfPosition(node.getStart(sf));
        hits.push({ line: line + 1, text: value.trim().slice(0, 120) });
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);
  return hits;
}

export function hasHanComment(file) {
  const text = fs.readFileSync(file, "utf8");
  let count = 0;
  const scanner = ts.createScanner(ts.ScriptTarget.Latest, false, ts.LanguageVariant.JSX, text);
  for (let tok = scanner.scan(); tok !== ts.SyntaxKind.EndOfFileToken; tok = scanner.scan()) {
    if (
      (tok === ts.SyntaxKind.SingleLineCommentTrivia || tok === ts.SyntaxKind.MultiLineCommentTrivia) &&
      HAN.test(scanner.getTokenText())
    )
      count++;
  }
  return count;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const verbose = process.argv.includes("-v");
  let total = 0;
  for (const file of sourceFiles()) {
    const hits = scanFile(file);
    if (!hits.length) continue;
    total += hits.length;
    console.log(`${path.relative(SRC_ROOT, file)}: ${hits.length}`);
    if (verbose) for (const h of hits) console.log(`  ${h.line}: ${h.text}`);
  }
  console.log(`total runtime Han literals: ${total}`);
}
