# GitHub Module

## Status

- **Status**: Implemented (ogen)
- **Date**: 2026-02-11
- **API Version**: 2022-11-28
- **Client**: ogen generated (`pkg/githubapi/`)

## Endpoint Catalog

### User

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `get_user` | GET | `/user` | (none) |
| `list_starred_repos` | GET | `/users/{username}/starred` | **username**, sort, direction, per_page, page |

### Repositories

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_repos` | GET | `/user/repos` | type, sort, direction, per_page, page |
| `get_repo` | GET | `/repos/{owner}/{repo}` | **owner**, **repo** |
| `list_branches` | GET | `/repos/{owner}/{repo}/branches` | **owner**, **repo**, per_page |
| `list_commits` | GET | `/repos/{owner}/{repo}/commits` | **owner**, **repo**, sha, per_page, page |
| `get_file_content` | GET | `/repos/{owner}/{repo}/contents/{path}` | **owner**, **repo**, **path**, ref |

### Issues

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_issues` | GET | `/repos/{owner}/{repo}/issues` | **owner**, **repo**, state, per_page, page |
| `get_issue` | GET | `/repos/{owner}/{repo}/issues/{issue_number}` | **owner**, **repo**, **issue_number** |
| `create_issue` | POST | `/repos/{owner}/{repo}/issues` | **owner**, **repo**, **title**, body, labels, assignees |
| `update_issue` | PATCH | `/repos/{owner}/{repo}/issues/{issue_number}` | **owner**, **repo**, **issue_number**, title, body, state, labels, assignees |
| `add_issue_comment` | POST | `/repos/{owner}/{repo}/issues/{issue_number}/comments` | **owner**, **repo**, **issue_number**, **body** |

### Pull Requests

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_prs` | GET | `/repos/{owner}/{repo}/pulls` | **owner**, **repo**, state, per_page, page |
| `get_pr` | GET | `/repos/{owner}/{repo}/pulls/{pull_number}` | **owner**, **repo**, **pr_number** |
| `create_pr` | POST | `/repos/{owner}/{repo}/pulls` | **owner**, **repo**, **title**, **head**, **base**, body, draft |
| `list_pr_files` | GET | `/repos/{owner}/{repo}/pulls/{pull_number}/files` | **owner**, **repo**, **pr_number**, per_page |

### Search

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `search_repos` | GET | `/search/repositories` | **query**, sort, per_page, page |
| `search_code` | GET | `/search/code` | **query**, per_page, page |
| `search_issues` | GET | `/search/issues` | **query**, sort, per_page, page |

### Actions

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_workflows` | GET | `/repos/{owner}/{repo}/actions/workflows` | **owner**, **repo**, per_page |
| `list_workflow_runs` | GET | `/repos/{owner}/{repo}/actions/runs` | **owner**, **repo**, workflow_id, status, per_page |
| | GET | `/repos/{owner}/{repo}/actions/workflows/{workflow_id}/runs` | (workflow_id 指定時に自動選択) |

### Composite Tools

| Tool | 構成 API | Params (required **bold**) |
|------|----------|---------------------------|
| `describe_user` | get_user, list_repos(5), list_starred_repos(5) | (none) |
| `describe_repo` | get_repo, get_file_content(README.md), list_branches(10), list_issues(open,10), list_prs(open,10) | **owner**, **repo** |
| `describe_pr` | get_pr, list_pr_files(30) | **owner**, **repo**, **pr_number** |

## Summary

- **Total**: 24 tools (GET: 16, POST: 3, PATCH: 1, Composite: 3 + list_workflow_runs の 2 endpoint)
- **ogen operations**: 22 (list_workflow_runs が 2 エンドポイントにマッピング)

## Response Schemas

subset spec で定義しているスキーマ:

| Schema | Used by |
|--------|---------|
| SimpleUser | get_user |
| Repository | list_repos, list_starred_repos, get_repo, search_repos |
| RepositoryOwner | Repository.owner, Commit.author |
| Branch | list_branches |
| Commit, CommitCommit, CommitCommitAuthor | list_commits |
| FileContent | get_file_content |
| Issue, IssueUser, Label | list_issues, get_issue, create_issue, update_issue, search_issues |
| IssueComment | add_issue_comment |
| PullRequest, PullRequestHead, PullRequestBase | list_prs, get_pr, create_pr |
| PullRequestFile | list_pr_files |
| SearchResultRepositories | search_repos |
| SearchResultCode, SearchResultCodeItemsItem | search_code |
| SearchResultIssues | search_issues |
| Workflow, WorkflowsResponse | list_workflows |
| WorkflowRun, WorkflowRunsResponse | list_workflow_runs |
| CreateIssueRequest | create_issue |
| UpdateIssueRequest | update_issue |
| CreateCommentRequest | add_issue_comment |
| CreatePRRequest | create_pr |

## Composite Tool Response Format

すべて Level 2 (Field Selection) で JSON を返す。各 API は goroutine で並行呼出。

### describe_user

```json
{
  "profile": { "login", "name", "bio", "public_repos", "followers", "following", "created_at" },
  "repos": [{ "full_name", "description", "language", "stargazers_count", "fork" }],
  "starred": [{ "full_name", "description", "language", "stargazers_count" }],
  "_note": "..."
}
```

### describe_repo

```json
{
  "repo": { "full_name", "description", "language", "visibility", "stargazers_count", "forks_count",
            "open_issues_count", "default_branch", "archived", "fork", "topics", "created", "last_push" },
  "readme": "(content text, max 2000 chars)",
  "branches": ["name", ...],
  "issues": [{ "number", "title", "author", "date", "labels" }],
  "prs": [{ "number", "title", "author", "draft", "date" }],
  "_note": "..."
}
```

### describe_pr

```json
{
  "pr": { "number", "title", "state", "draft", "merged", "created", "updated", "merged_at",
          "author", "head", "base", "body (max 3000 chars)" },
  "files": [{ "file", "status", "add", "del" }],
  "_note": "..."
}
```

## Notes

- `get_file_content` は base64 エンコードされたコンテンツを自動デコードする
- `list_workflow_runs` は `workflow_id` の有無で 2 つの ogen オペレーションに分岐する
- `list_repos` は認証ユーザー自身のリポジトリ、`list_starred_repos` は任意ユーザーのスター
