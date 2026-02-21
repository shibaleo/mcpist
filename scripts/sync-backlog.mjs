#!/usr/bin/env node
/**
 * Jira → backlog.csv 同期スクリプト
 *
 * Jira REST API v3 から Issue を取得し、dev/workdir/backlog.csv に書き出す。
 *
 * 環境変数 (.env):
 *   JIRA_DOMAIN       - e.g. your-org.atlassian.net
 *   JIRA_EMAIL         - Atlassian account email
 *   JIRA_API_TOKEN     - Atlassian API token
 *
 * Usage:
 *   node scripts/sync-backlog.mjs                    # MCPIST プロジェクトのみ
 *   node scripts/sync-backlog.mjs --project DWHBI    # 別プロジェクト指定
 */

import { writeFileSync, readFileSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { config } from "dotenv";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT_DIR = resolve(__dirname, "..");
const BACKLOG_PATH = resolve(ROOT_DIR, "dev/workdir/backlog.csv");

// Load .env from project root
config({ path: resolve(ROOT_DIR, ".env") });

// Jira credentials
const JIRA_DOMAIN = process.env.JIRA_DOMAIN;
const JIRA_EMAIL = process.env.JIRA_EMAIL;
const JIRA_API_TOKEN = process.env.JIRA_API_TOKEN;

if (!JIRA_DOMAIN || !JIRA_EMAIL || !JIRA_API_TOKEN) {
  console.error("Error: JIRA_DOMAIN, JIRA_EMAIL, JIRA_API_TOKEN are required");
  process.exit(1);
}

const JIRA_BASE = `https://${JIRA_DOMAIN}/rest/api/3`;
const AUTH_HEADER = `Basic ${Buffer.from(`${JIRA_EMAIL}:${JIRA_API_TOKEN}`).toString("base64")}`;

// --- Jira REST API ---

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function jiraGet(path, retries = 4) {
  for (let attempt = 0; attempt <= retries; attempt++) {
    const resp = await fetch(`${JIRA_BASE}${path}`, {
      headers: {
        Authorization: AUTH_HEADER,
        Accept: "application/json",
      },
    });

    if (resp.status === 429) {
      const retryAfter = parseInt(resp.headers.get("Retry-After") || "5", 10);
      const delay = retryAfter * 1000;
      console.log(`  Rate limited. Waiting ${retryAfter}s... (attempt ${attempt + 1}/${retries + 1})`);
      await sleep(delay);
      continue;
    }

    if (!resp.ok) {
      throw new Error(`Jira API error: ${resp.status} ${await resp.text()}`);
    }
    return resp.json();
  }
  throw new Error("Jira API: max retries exceeded (429)");
}

async function fetchAllIssues(projectKey) {
  const issues = [];
  const jql = encodeURIComponent(`project = ${projectKey} ORDER BY key ASC`);
  const fields = "summary,status,priority,issuetype,labels,created,updated";
  let nextPageToken = null;

  while (true) {
    let url = `/search/jql?jql=${jql}&maxResults=100&fields=${fields}`;
    if (nextPageToken) {
      url += `&nextPageToken=${encodeURIComponent(nextPageToken)}`;
    }
    console.log(`  Fetching ${projectKey} issues...`);
    const data = await jiraGet(url);
    issues.push(...data.issues);
    console.log(`  Got ${data.issues.length} issues (total so far: ${issues.length})`);
    if (data.isLast) break;
    nextPageToken = data.nextPageToken;
  }
  return issues;
}

// --- Normalization ---

const STATUS_MAP = {
  "To Do": "todo",
  "In Progress": "in_progress",
  "Done": "done",
  "完了": "done",
  "進行中": "in_progress",
  "キャンセル済み": "cancelled",
};

function normalizeStatus(name) {
  return STATUS_MAP[name] || name.toLowerCase().replace(/\s+/g, "_");
}

const PRIORITY_MAP = {
  Highest: "critical",
  High: "high",
  Medium: "medium",
  Low: "low",
  Lowest: "lowest",
};

function normalizePriority(name) {
  return PRIORITY_MAP[name] || (name ? name.toLowerCase() : "medium");
}

const TYPE_MAP = {
  "タスク": "task",
  "エピック": "epic",
  "サブタスク": "subtask",
  "バグ": "bug",
  "ストーリー": "story",
};

function normalizeType(name) {
  return TYPE_MAP[name] || (name ? name.toLowerCase() : "task");
}

// --- CSV ---

const CSV_COLUMNS = [
  "id",
  "source",
  "source_id",
  "type",
  "priority",
  "status",
  "title",
  "labels",
  "created",
  "updated",
];

function escapeCSV(value) {
  if (value == null) return "";
  const str = String(value);
  if (str.includes(",") || str.includes('"') || str.includes("\n")) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

function issueToRow(issue) {
  const f = issue.fields;
  return {
    id: issue.key,
    source: "jira",
    source_id: issue.key,
    type: normalizeType(f.issuetype?.name || "Task"),
    priority: normalizePriority(f.priority?.name),
    status: normalizeStatus(f.status?.name || "To Do"),
    title: f.summary || "",
    labels: (f.labels || []).join(";"),
    created: (f.created || "").slice(0, 10),
    updated: (f.updated || "").slice(0, 10),
  };
}

function rowToCSVLine(row) {
  return CSV_COLUMNS.map((col) => escapeCSV(row[col])).join(",");
}

function parseExistingCSV(filePath) {
  if (!existsSync(filePath)) return [];
  const content = readFileSync(filePath, "utf-8");
  const lines = content.trim().split("\n");
  if (lines.length <= 1) return [];

  const header = lines[0].split(",");
  return lines.slice(1).map((line) => {
    const values = parseCSVLine(line);
    const row = {};
    header.forEach((col, i) => {
      row[col] = values[i] || "";
    });
    return row;
  });
}

function parseCSVLine(line) {
  const values = [];
  let current = "";
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (inQuotes) {
      if (ch === '"' && line[i + 1] === '"') {
        current += '"';
        i++;
      } else if (ch === '"') {
        inQuotes = false;
      } else {
        current += ch;
      }
    } else {
      if (ch === '"') {
        inQuotes = true;
      } else if (ch === ",") {
        values.push(current);
        current = "";
      } else {
        current += ch;
      }
    }
  }
  values.push(current);
  return values;
}

// --- Main ---

async function main() {
  const args = process.argv.slice(2);
  let projectKeys = ["MCPIST"];

  if (args.includes("--project")) {
    const idx = args.indexOf("--project");
    projectKeys = [args[idx + 1]];
  }

  console.log(`Syncing Jira issues from: ${projectKeys.join(", ")}\n`);

  const allIssues = [];
  for (const key of projectKeys) {
    const issues = await fetchAllIssues(key);
    allIssues.push(...issues);
    console.log(`  ${key}: ${issues.length} issues`);
  }

  // Merge: Jira rows update existing, deleted issues are marked as "deleted"
  const existing = parseExistingCSV(BACKLOG_PATH);
  const existingJiraRows = existing.filter((r) => r.source === "jira");
  const localRows = existing.filter((r) => r.source !== "jira");

  const jiraRows = allIssues.map(issueToRow);
  const jiraIds = new Set(jiraRows.map((r) => r.id));

  // Rows in CSV but not in Jira anymore → mark as deleted
  const deletedRows = existingJiraRows
    .filter((r) => !jiraIds.has(r.id) && r.status !== "deleted")
    .map((r) => ({ ...r, status: "deleted", updated: new Date().toISOString().slice(0, 10) }));

  // Already-deleted rows from previous syncs
  const previouslyDeleted = existingJiraRows.filter((r) => r.status === "deleted" && !jiraIds.has(r.id));

  const allRows = [...jiraRows, ...deletedRows, ...previouslyDeleted, ...localRows];

  const csvContent =
    [CSV_COLUMNS.join(","), ...allRows.map(rowToCSVLine)].join("\n") + "\n";

  writeFileSync(BACKLOG_PATH, csvContent, "utf-8");
  console.log(`\nWrote ${allRows.length} rows to dev/workdir/backlog.csv`);
  console.log(`  Jira: ${jiraRows.length}, Deleted: ${deletedRows.length + previouslyDeleted.length}, Local: ${localRows.length}`);
}

main().catch((err) => {
  console.error("Error:", err.message);
  process.exit(1);
});
