import type { APIErrorBody, BenchResponse, MutationEnvelope, SessionResponse } from "./types";

const tokenStorageKey = "countershape.studio.access.v1";
const tokenPattern = /^[A-Za-z0-9_-]{43}$/;

export class StudioAPIError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(status: number, body: APIErrorBody) {
    super(body.detail);
    this.name = "StudioAPIError";
    this.status = status;
    this.code = body.code;
  }
}

export class StudioClient {
  readonly #token: string;
  #csrf = "";

  private constructor(token: string) {
    this.#token = token;
  }

  static fromLaunchLocation(): StudioClient {
    const fragment = new URLSearchParams(window.location.hash.slice(1));
    const launched = fragment.get("access_token");
    if (launched !== null) {
      if (!tokenPattern.test(launched)) {
        window.history.replaceState(null, "", window.location.pathname + window.location.search);
        throw new Error("The studio launch credential is malformed.");
      }
      window.sessionStorage.setItem(tokenStorageKey, launched);
      window.history.replaceState(null, "", window.location.pathname + window.location.search);
    }
    const token = window.sessionStorage.getItem(tokenStorageKey) ?? "";
    if (!tokenPattern.test(token)) {
      throw new Error("This tab has no studio launch credential. Reopen it from the Countershape process.");
    }
    return new StudioClient(token);
  }

  async session(): Promise<SessionResponse> {
    const value = await this.#request<SessionResponse>("/api/v1/session");
    this.#csrf = value.csrf;
    return value;
  }

  bench(): Promise<BenchResponse> {
    return this.#request<BenchResponse>("/api/v1/bench/seed-cli-precedence");
  }

  async mutate(operation: "visit" | "propose" | "reveal" | "revise" | "finalize", body: MutationEnvelope): Promise<BenchResponse> {
    if (this.#csrf === "") {
      throw new Error("The mutation session has not been admitted.");
    }
    return this.#request<BenchResponse>(`/api/v1/bench/seed-cli-precedence/${operation}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": this.#csrf,
      },
      body: JSON.stringify(body),
    });
  }

  async #request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await fetch(path, {
      ...init,
      cache: "no-store",
      credentials: "omit",
      redirect: "error",
      headers: {
        ...init.headers,
        Authorization: `Bearer ${this.#token}`,
      },
    });
    const contentType = response.headers.get("Content-Type") ?? "";
    if (!contentType.startsWith("application/json")) {
      throw new Error("The studio returned an unexpected response type.");
    }
    const body = (await response.json()) as T | APIErrorBody;
    if (!response.ok) {
      const refusal = body as APIErrorBody;
      throw new StudioAPIError(response.status, {
        code: typeof refusal.code === "string" ? refusal.code : "STUDIO_RESPONSE_REFUSED",
        detail: typeof refusal.detail === "string" ? refusal.detail : "The studio refused this request.",
      });
    }
    return body as T;
  }
}
