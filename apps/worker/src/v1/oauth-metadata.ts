import type { Env } from "../types";

/** CLERK_JWKS_URL から issuer URL を導出 (/.well-known/jwks.json を除去) */
function clerkIssuerUrl(env: Env): string {
  return env.CLERK_JWKS_URL.replace(/\/\.well-known\/jwks\.json$/, "");
}

/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * MCP クライアントが認可サーバーを発見するために使用
 */
export function handleOAuthProtectedResourceMetadata(request: Request, env: Env): Response {
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;

  return new Response(JSON.stringify({
    resource: `${baseUrl}/v1/mcp`,
    authorization_servers: [clerkIssuerUrl(env)],
    scopes_supported: ["openid", "profile", "email"],
    bearer_methods_supported: ["header"],
  }), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
    },
  });
}

/**
 * OAuth Authorization Server Metadata (RFC 8414)
 * Clerk の /.well-known/oauth-authorization-server をプロキシ
 */
export async function handleOAuthAuthorizationServerMetadata(env: Env): Promise<Response> {
  const issuer = clerkIssuerUrl(env);
  const res = await fetch(`${issuer}/.well-known/oauth-authorization-server`);
  return new Response(res.body, {
    status: res.status,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
    },
  });
}
