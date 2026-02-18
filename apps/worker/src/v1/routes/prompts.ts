/**
 * /v1/prompts 関連ルート
 *
 * GET    /prompts      → list_prompts / get_prompts (auth / gateway auth)
 * GET    /prompts/:id  → get_prompt (auth)
 * PUT    /prompts      → upsert_prompt (auth)
 * DELETE /prompts/:id  → delete_prompt (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate, authenticateGateway } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const prompts = new Hono<{ Bindings: Bindings }>();

// GET /prompts — list_prompts / get_prompts (auth or gateway auth)
prompts.get("/", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  const isGateway = authenticateGateway(c.req.raw, c.env);
  if (!auth && !isGateway) return jsonResponse({ error: "Unauthorized" }, 401);

  const userId = auth ? auth.userId : c.req.query("user_id");
  if (!userId) return jsonResponse({ error: "Missing user_id" }, 400);

  // Gateway auth (Go Server) uses get_prompts RPC with enabled_only/name filters
  if (isGateway) {
    const params: Record<string, unknown> = {
      p_user_id: userId,
      p_enabled_only: c.req.query("enabled_only") === "true",
    };
    const promptName = c.req.query("name");
    if (promptName) params.p_prompt_name = promptName;

    return forwardToPostgREST(c.env, "get_prompts", params);
  }

  // Console auth uses list_prompts RPC
  const params: Record<string, unknown> = { p_user_id: userId };
  const moduleName = c.req.query("module");
  if (moduleName) params.p_module_name = moduleName;

  return forwardToPostgREST(c.env, "list_prompts", params);
});

// GET /prompts/:id — get_prompt
prompts.get("/:id", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "get_prompt", {
    p_user_id: auth.userId,
    p_prompt_id: c.req.param("id"),
  });
});

// PUT /prompts — upsert_prompt
prompts.put("/", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{
    prompt_id?: string;
    name: string;
    content: string;
    module_name?: string;
    enabled: boolean;
    description?: string;
  }>();

  return forwardToPostgREST(c.env, "upsert_prompt", {
    p_user_id: auth.userId,
    ...(body.prompt_id && { p_prompt_id: body.prompt_id }),
    p_name: body.name,
    p_content: body.content,
    ...(body.module_name && { p_module_name: body.module_name }),
    p_enabled: body.enabled,
    ...(body.description && { p_description: body.description }),
  });
});

// DELETE /prompts/:id — delete_prompt
prompts.delete("/:id", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "delete_prompt", {
    p_user_id: auth.userId,
    p_prompt_id: c.req.param("id"),
  });
});

export { prompts };
