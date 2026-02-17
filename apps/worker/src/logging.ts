/**
 * Worker ログ — stdio (console.log) のみ。
 * Cloudflare Workers の observability.logs が有効なので、
 * Loki への送信は Go Server 側の observability.go で実装する。
 */

/** リクエストログ */
export function logRequest(
  _env: unknown,
  requestId: string,
  method: string,
  path: string,
  statusCode: number,
  durationMs: number,
  extra: Record<string, unknown> = {}
): void {
  console.log(JSON.stringify({
    type: "request",
    request_id: requestId,
    method,
    path,
    status_code: statusCode,
    duration_ms: durationMs,
    ...extra,
  }));
}

/** セキュリティイベントログ */
export function logSecurityEvent(
  _env: unknown,
  requestId: string,
  event: string,
  details: Record<string, unknown> = {}
): void {
  console.warn(JSON.stringify({
    type: "security",
    request_id: requestId,
    event,
    ...details,
  }));
}
