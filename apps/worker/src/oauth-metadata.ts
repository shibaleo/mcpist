import type { Env } from "./types";

/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * MCP クライアントが認可サーバーを発見するために使用
 */
export function handleOAuthProtectedResourceMetadata(request: Request, env: Env): Response {
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;

  return new Response(JSON.stringify({
    resource: `${baseUrl}/mcp`,
    authorization_servers: [`${env.SUPABASE_URL}/auth/v1`],
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
 * Supabase Auth のメタデータをプロキシ
 */
export async function handleOAuthAuthorizationServerMetadata(env: Env): Promise<Response> {
  try {
    const response = await fetch(
      `${env.SUPABASE_URL}/auth/v1/.well-known/openid-configuration`
    );
    if (response.ok) {
      const metadata = await response.json();
      return new Response(JSON.stringify(metadata), {
        status: 200,
        headers: {
          "Content-Type": "application/json",
          "Cache-Control": "public, max-age=3600",
          "Access-Control-Allow-Origin": "*",
        },
      });
    }
  } catch {
    // Fall through to manual metadata
  }

  // Fallback: 手動構築
  return new Response(JSON.stringify({
    issuer: `${env.SUPABASE_URL}/auth/v1`,
    authorization_endpoint: `${env.SUPABASE_URL}/auth/v1/authorize`,
    token_endpoint: `${env.SUPABASE_URL}/auth/v1/token`,
    registration_endpoint: `${env.SUPABASE_URL}/auth/v1/oauth/register`,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code", "refresh_token"],
    code_challenge_methods_supported: ["S256"],
    token_endpoint_auth_methods_supported: ["none"],
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
