import { SUPPORTED_LANGUAGES, translator } from "./messages.js";
export { SUPPORTED_LANGUAGES } from "./messages.js";

const MAX_RESPONSE_BYTES = 2 * 1024 * 1024;

export function configFromEnv(env = process.env) {
  const requestedLanguage = env.SCOPEWEAVER_LANGUAGE || "en";
  const t = translator(SUPPORTED_LANGUAGES.includes(requestedLanguage) ? requestedLanguage : "en");
  let baseURL;
  try { baseURL = new URL(env.SCOPEWEAVER_URL || "http://127.0.0.1:8787"); }
  catch { throw new TypeError(t("config.url_invalid")); }
  const loopback = ["localhost", "127.0.0.1", "[::1]"].includes(baseURL.hostname);
  if ((baseURL.protocol !== "https:" && !(baseURL.protocol === "http:" && loopback)) ||
      baseURL.username || baseURL.password || baseURL.search || baseURL.hash || baseURL.pathname !== "/") {
    throw new Error(t("config.url_origin"));
  }
  const language = requestedLanguage;
  if (!SUPPORTED_LANGUAGES.includes(language)) throw new Error(t("config.language", { languages: SUPPORTED_LANGUAGES.join(", ") }));
  const timeoutMs = Number(env.SCOPEWEAVER_TIMEOUT_MS || 30000);
  if (!Number.isInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 120000) {
    throw new Error(t("config.timeout"));
  }
  const token = env.SCOPEWEAVER_TOKEN || "";
  if (/[\r\n]/.test(token)) throw new Error(t("config.token"));
  if (env.SCOPEWEAVER_ALLOW_WRITES && !["true", "false"].includes(env.SCOPEWEAVER_ALLOW_WRITES)) {
    throw new Error(t("config.writes"));
  }
  return { baseURL, language, timeoutMs, token, password: env.SCOPEWEAVER_PASSWORD || "",
    allowWrites: env.SCOPEWEAVER_ALLOW_WRITES === "true" };
}

export class ScopeWeaverClient {
  constructor(config = configFromEnv()) {
    this.config = config;
    this.t = translator(config.language);
    this.token = config.token;
    this.login = null;
  }

  redact(message) {
    for (const secret of [this.token, this.config.token, this.config.password]) {
      if (secret) message = message.replaceAll(secret, "[redacted]");
    }
    return message;
  }

  async authenticate(signal) {
    if (this.token) return;
    if (!this.config.password) throw new Error(this.t("auth.required"));
    // Share a single login when an agent calls multiple read tools concurrently.
    if (!this.login) {
      this.login = this.request("/api/auth/login", {
        method: "POST", body: { username: "ARTEX", password: this.config.password }, auth: false, signal,
      }).then((result) => {
        if (typeof result.token !== "string" || !result.token) throw new Error(this.t("auth.token_missing"));
        this.token = result.token;
      }).finally(() => { this.login = null; });
    }
    await this.login;
  }

  async request(path, { method = "GET", body, query = {}, auth = true, signal } = {}) {
    if (!path.startsWith("/api/")) throw new Error(this.t("request.path"));
    if (auth) await this.authenticate(signal);
    const url = new URL(path, this.config.baseURL);
    if (url.origin !== this.config.baseURL.origin) throw new Error(this.t("request.origin"));
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) url.searchParams.set(key, String(value));
    }
    const timeout = AbortSignal.timeout(this.config.timeoutMs);
    const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
    try {
      const response = await fetch(url, {
        method, redirect: "error", signal: combined,
        headers: { "Accept": "application/json", "Accept-Language": this.config.language,
          ...(auth ? { "Authorization": `Bearer ${this.token}` } : {}),
          ...(body !== undefined ? { "Content-Type": "application/json" } : {}) },
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      });
      const reader = response.body.getReader();
      const chunks = [];
      let length = 0;
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          length += value.byteLength;
          if (length > MAX_RESPONSE_BYTES) throw new Error(this.t("response.oversized"));
          chunks.push(Buffer.from(value));
        }
      } finally { await reader.cancel(); }
      let result;
      try { result = JSON.parse(Buffer.concat(chunks).toString("utf8")); }
      catch { throw new Error(this.t("response.non_json", { status: response.status })); }
      if (!response.ok) {
        const hint = response.status === 401 ? this.t("response.http_hint") : "";
        const detail = typeof result?.error === "string" ? this.redact(result.error).slice(0, 500) : this.t("response.failed");
        throw new Error(this.t("response.http", { status: response.status, detail, hint }));
      }
      return result;
    } catch (error) {
      if (combined.aborted) throw new Error(signal?.aborted ? this.t("request.cancelled") : this.t("request.timeout"));
      throw new Error(this.redact(error.message));
    }
  }
}
