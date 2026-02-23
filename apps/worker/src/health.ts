import type { Env } from "./types";
import { jsonResponse } from "./http";

type HealthCheckError =
  | "timeout"
  | "dns_failure"
  | "connection_refused"
  | "ssl_error"
  | "http_error"
  | "unknown";

interface HealthCheckResult {
  healthy: boolean;
  error?: HealthCheckError;
  statusCode?: number;
  latencyMs?: number;
}

export async function handleHealthCheck(env: Env): Promise<Response> {
  const result = await checkBackendHealth(env.SERVER_URL);

  return jsonResponse({
    status: "ok",
    backend: buildBackendInfo(result),
  }, 200);
}

export async function performScheduledHealthCheck(env: Env): Promise<void> {
  const result = await checkBackendHealth(env.SERVER_URL);
  console.log(`[Cron] Health check - Primary: ${result.healthy}`);
}

async function checkBackendHealth(url: string): Promise<HealthCheckResult> {
  const healthUrl = `${url}/health`;
  const startTime = Date.now();

  try {
    const response = await fetch(healthUrl, {
      method: "GET",
      signal: AbortSignal.timeout(5000),
    });
    const latencyMs = Date.now() - startTime;

    if (response.ok) {
      return { healthy: true, statusCode: response.status, latencyMs };
    }
    return { healthy: false, error: "http_error", statusCode: response.status, latencyMs };
  } catch (error) {
    const latencyMs = Date.now() - startTime;
    return { healthy: false, error: classifyError(error), latencyMs };
  }
}

export function classifyError(error: unknown): HealthCheckError {
  if (!(error instanceof Error)) return "unknown";

  const message = error.message.toLowerCase();
  const name = error.name;

  if (name === "TimeoutError" || message.includes("timeout") || message.includes("aborted")) {
    return "timeout";
  }
  if (message.includes("dns") || message.includes("enotfound") || message.includes("getaddrinfo") ||
      message.includes("name resolution") || message.includes("internal error")) {
    return "dns_failure";
  }
  if (message.includes("econnrefused") || message.includes("connection refused") ||
      message.includes("network connection lost")) {
    return "connection_refused";
  }
  if (message.includes("ssl handshake") || message.includes("tls handshake") ||
      message.includes("certificate expired") || message.includes("self signed certificate") ||
      message.includes("unable to verify")) {
    return "ssl_error";
  }
  return "unknown";
}

export function buildBackendInfo(result: HealthCheckResult) {
  const info: Record<string, unknown> = { healthy: result.healthy };
  if (result.error) info.error = result.error;
  if (result.statusCode) info.statusCode = result.statusCode;
  if (result.latencyMs !== undefined) info.latencyMs = result.latencyMs;
  return info;
}
