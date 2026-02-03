# DAY023 計画

## 日付

2026-02-03

---

## 概要

Sprint-007 1日目。Airtable OAuth 2.0 対応と、ログイン時のコールバックエラー解消を行う。

---

## 本日のタスク

### Phase 1: ログインコールバックエラー解消（優先度：高）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D23-001 | エラー再現・原因調査 | auth/callback での具体的なエラー内容確認 | 未着手 |
| D23-002 | PKCE / Cookie 設定確認 | DAY022 で修正済みだが再発の可能性 | 未着手 |
| D23-003 | 修正・動作確認 | 本番環境でログイン成功を確認 | 未着手 |

### Phase 2: Airtable OAuth 2.0 対応（優先度：中）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D23-010 | Airtable OAuth App 登録 | https://airtable.com/create/oauth | 未着手 |
| D23-011 | authorize ルート作成 | OAuth 2.0 + PKCE | 未着手 |
| D23-012 | callback ルート作成 | トークン交換、Vault 保存 | 未着手 |
| D23-013 | oauth-apps.ts 更新 | Airtable プロバイダー追加 | 未着手 |
| D23-014 | services/page.tsx 更新 | authConfig を OAuth 方式に変更 | 未着手 |
| D23-015 | Go モジュール OAuth 対応 | GetModuleToken で OAuth トークン取得 | 未着手 |
| D23-016 | 動作確認 | list_bases, list_records 等 | 未着手 |

---

## Airtable OAuth 2.0 仕様

### 認証情報

| 項目 | 内容 |
|------|------|
| Authorization URL | `https://airtable.com/oauth2/v1/authorize` |
| Token URL | `https://airtable.com/oauth2/v1/token` |
| PKCE | **必須** (code_challenge_method: S256) |
| トークン有効期限 | 2ヶ月 |
| リフレッシュトークン | あり（2ヶ月有効、使用時に新しいペア発行） |

### スコープ

```
data.records:read
data.records:write
schema.bases:read
schema.bases:write
```

### 参考

- [Airtable OAuth Integration Guide](https://airtable.com/developers/web/guides/oauth-integrations)
- [Airtable OAuth Reference](https://airtable.com/developers/web/api/oauth-reference)

---

## 完了条件

- [ ] 本番環境でログインが正常に動作する
- [ ] Airtable OAuth 認証が動作する
- [ ] Airtable ツール（list_bases, list_records）が動作する

---

## 参考

- [day022-review.md](./day022-review.md) - PKCE / Cookie の学び
- [sprint007-plan.md](../sprint/sprint007-plan.md) - Sprint 007 計画
