/**
 * Loki Push API へのログ送信。
 * ctx.waitUntil() 経由で呼び出し、レスポンスをブロックしない。
 */

import type { Env } from "./types";

interface LokiPushRequest {
  streams: {
    stream: Record<string, string>;
    values: string[][];
  }[];
}

function pushToLoki(
  env: Env,
  labels: Record<string, string>,
  data: Record<string, unknown>
): Promise<void> {
  if (!env.GRAFANA_LOKI_URL || !env.GRAFANA_LOKI_USER || !env.GRAFANA_LOKI_API_KEY) {
    return Promise.resolve();
  }

  labels["app"] = env.APP_ENV || "mcpist-dev";
  labels["instance"] = "worker";
  labels["region"] = "cloudflare";

  const body: LokiPushRequest = {
    streams: [{
      stream: labels,
      values: [[String(Date.now() * 1_000_000), JSON.stringify(data)]],
    }],
  };

  const auth = btoa(`${env.GRAFANA_LOKI_USER}:${env.GRAFANA_LOKI_API_KEY}`);

  return fetch(`${env.GRAFANA_LOKI_URL}/loki/api/v1/push`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Basic ${auth}`,
    },
    body: JSON.stringify(body),
  }).then(resp => {
    if (!resp.ok) console.error(`Loki push failed: ${resp.status}`);
  }).catch(err => {
    console.error("Loki push error:", err);
  });
}

/** リクエストログを Loki に送信 */
export function pushRequestLog(
  env: Env,
  requestId: string,
  method: string,
  path: string,
  statusCode: number,
  durationMs: number,
  extra: Record<string, unknown> = {}
): Promise<void> {
  return pushToLoki(env, { type: "request", method }, {
    request_id: requestId,
    method,
    path,
    status_code: statusCode,
    duration_ms: durationMs,
    ...extra,
  });
}

/** セキュリティイベントを Loki に送信 */
export function pushSecurityEvent(
  env: Env,
  requestId: string,
  event: string,
  details: Record<string, unknown> = {}
): Promise<void> {
  return pushToLoki(env, { type: "security", level: "warn" }, {
    request_id: requestId,
    event,
    ...details,
  });
}
