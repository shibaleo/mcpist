/**
 * OpenAPI 3.1 spec for MCPist API Gateway.
 * GET /openapi.json で公開。
 *
 * ソース: openapi.yaml → scripts/generate-openapi-json.mjs → openapi.generated.json
 */

import spec from "./openapi.generated.json";

const specJson = JSON.stringify(spec);

export function handleOpenApiSpec(_request: Request): Response {
  return new Response(specJson, {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
    },
  });
}
