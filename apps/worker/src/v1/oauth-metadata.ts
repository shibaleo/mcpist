import type { Env } from "../types";

/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * MCP クライアントが認可サーバーを発見するために使用
 */
export function handleOAuthProtectedResourceMetadata(request: Request, env: Env): Response {
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;

  return new Response(JSON.stringify({
    resource: `${baseUrl}/v1/mcp`,
    authorization_servers: [`${baseUrl}/v1/mcp/.well-known/oauth-authorization-server`],
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
 * Clerk 認証のメタデータ
 */
export async function handleOAuthAuthorizationServerMetadata(env: Env): Promise<Response> {
  // Clerk doesn't provide a standard OpenID discovery endpoint in the same way,
  // so we return the JWKS URL for token verification.
  return new Response(JSON.stringify({
    jwks_uri: env.CLERK_JWKS_URL,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code"],
    scopes_supported: ["openid", "profile", "email"],
  }), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
    },
  });
}
