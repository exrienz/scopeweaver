import { z } from "zod";
import { translator } from "./messages.js";

function validationError(language, t) {
  const english = z.locales.en().localeError;
  return issue => {
    // Preserve Zod's installed English wording without changing global settings.
    if (language === "en") return english(issue);
    switch (issue.code) {
      case "invalid_type": {
        const received = issue.input === null ? "null" : Array.isArray(issue.input) ? "array"
          : typeof issue.input === "number" && Number.isNaN(issue.input) ? "NaN" : typeof issue.input;
        return t("validation.invalid_type", { expected: issue.expected, received });
      }
      case "invalid_value":
        return t("validation.invalid_value", { values: issue.values.map(value => JSON.stringify(value)).join("|") });
      case "too_small":
      case "too_big": {
        const minimum = issue.code === "too_small";
        const length = issue.origin === "string" || issue.origin === "array";
        return t("validation." + issue.code + (length ? "_length" : ""), {
          type: issue.origin, comparison: (minimum ? ">" : "<") + (issue.inclusive ? "=" : ""),
          bound: minimum ? issue.minimum : issue.maximum,
          unit: length ? t(issue.origin === "string" ? "validation.unit.characters" : "validation.unit.items") : "",
        });
      }
      case "invalid_format":
        return t("validation.invalid_format", { pattern: issue.pattern });
      case "unrecognized_keys":
        return t(issue.keys.length === 1 ? "validation.unrecognized_key" : "validation.unrecognized_keys",
          { keys: issue.keys.map(key => JSON.stringify(key)).join(", ") });
      default:
        return t("validation.invalid_input");
    }
  };
}


const outputSchema = z.object({ data: z.unknown() });

export function createTools(client) {
  const t = translator(client.config.language);
  const error = validationError(client.config.language || "en", t);
  const id = z.string({ error }).regex(/^[1-9][0-9]*$/, { error }).max(19, { error }).describe(t("schema.id.description"));
  const page = z.number({ error }).int({ error }).min(1, { error }).max(100000, { error }).default(1);
  const limit = z.number({ error }).int({ error }).min(1, { error }).max(100, { error }).default(20);
  function tool(name, title, description, shape, readOnly, run) {
    return { name: `scopeweaver_${name}`, title, description,
      schema: z.strictObject(shape, { error }), outputSchema, run,
      annotations: { readOnlyHint: readOnly, destructiveHint: !readOnly,
        idempotentHint: readOnly || name === "control_task", openWorldHint: true } };
  }
  const read = (path, query, signal) => client.request(path, { query, signal });
  const tools = [
    tool("health", t("tools.health.title"), t("tools.health.description"), {}, true,
      (_, signal) => client.request("/api/health", { auth: false, signal })),
    tool("list_tasks", t("tools.list_tasks.title"), t("tools.list_tasks.description"), { page, limit }, true,
      async (args, signal) => {
        const result = await read("/api/tasks", {}, signal);
        if (!Array.isArray(result.tasks)) throw new Error(t("response.task_list"));
        return { tasks: result.tasks.slice((args.page - 1) * args.limit, args.page * args.limit),
          total: result.tasks.length, page: args.page, page_size: args.limit, active: result.active };
      }),
    tool("get_task", t("tools.get_task.title"), t("tools.get_task.description"), { task_id: id }, true,
      (args, signal) => read(`/api/tasks/${args.task_id}`, {}, signal)),
    tool("get_coverage", t("tools.get_coverage.title"), t("tools.get_coverage.description"), { task_id: id }, true,
      (args, signal) => read(`/api/tasks/${args.task_id}/coverage`, {}, signal)),
    tool("list_findings", t("tools.list_findings.title"), t("tools.list_findings.description"), {
      task_id: id.optional(), page, limit,
      severity: z.enum(["critical", "high", "medium", "low", "info"], { error }).optional(),
      status: z.enum(["pending", "in_progress", "confirmed", "resolved", "fixed", "false_positive", "ignored", "duplicate", "risk_accepted"], { error }).optional(),
      q: z.string({ error }).max(500, { error }).optional(),
    }, true, (args, signal) => read("/api/exploration/findings", args, signal)),
    tool("get_finding", t("tools.get_finding.title"), t("tools.get_finding.description"), { finding_id: id }, true,
      (args, signal) => read(`/api/exploration/findings/${args.finding_id}`, {}, signal)),
  ];
  if (client.config.allowWrites) {
    tools.push(
      tool("create_task", t("tools.create_task.title"), t("tools.create_task.description"), {
        name: z.string({ error }).max(200, { error }).optional(), description: z.string({ error }).trim().min(1, { error }).max(10000, { error }),
        goal: z.string({ error }).trim().min(1, { error }).max(10000, { error }), timeout_seconds: z.number({ error }).int({ error }).min(1, { error }).max(86400, { error }).default(600),
        company_ids: z.array(z.number({ error }).int({ error }).positive({ error }).max(Number.MAX_SAFE_INTEGER, { error }), { error }).max(32, { error }).optional(),
      }, false, (args, signal) => client.request("/api/tasks", { method: "POST", body: args, signal })),
      tool("control_task", t("tools.control_task.title"), t("tools.control_task.description"), {
        task_id: id, action: z.enum(["pause", "resume"], { error }),
      }, false, (args, signal) => client.request(`/api/tasks/${args.task_id}/control`, {
        method: "POST", body: { action: args.action }, signal,
      })),
    );
  }
  return tools;
}

export async function executeTool(tool, args, signal) {
  const params = tool.schema.parse(args);
  const data = await tool.run(params, signal);
  return { content: [{ type: "text", text: JSON.stringify({ data }) }], structuredContent: { data } };
}
