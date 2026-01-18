export interface Env {
  ENVIRONMENT: string;
}

export default {
  async fetch(
    request: Request,
    env: Env,
    ctx: ExecutionContext
  ): Promise<Response> {
    const url = new URL(request.url);

    // Health check
    if (url.pathname === "/health") {
      return new Response("ok", { status: 200 });
    }

    // TODO: Implement API Gateway logic
    // - JWT verification
    // - Rate limiting
    // - Request forwarding to MCP Server

    return new Response("MCPist Gateway", { status: 200 });
  },
};
