#!/usr/bin/env node
/**
 * openapi.yaml → openapi.generated.json 変換スクリプト
 *
 * Usage: node scripts/generate-openapi-json.mjs
 */
import { readFileSync, writeFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "yaml";

const __dirname = dirname(fileURLToPath(import.meta.url));
// Use Go Server spec as the public API spec (single source of truth)
const yamlPath = resolve(__dirname, "../../server/api/openapi/server-openapi.yaml");
const jsonPath = resolve(__dirname, "../src/openapi.generated.json");

const yamlContent = readFileSync(yamlPath, "utf-8");
const spec = parse(yamlContent);
writeFileSync(jsonPath, JSON.stringify(spec), "utf-8");

console.log("✓ openapi.generated.json written");
