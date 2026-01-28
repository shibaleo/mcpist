# AUS - DST インタラクション詳細（dtl-itr-AUS-DST）

## ドキュメント管理情報

| 項目      | 値                                           |
| ------- | ------------------------------------------- |
| Status  | `reviewed`                                  |
| Version | v2.0                                        |
| Note    | Auth Server - Data Store Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Auth Server (AUS) |
| 連携先 | Data Store (DST) |
| 内容 | ユーザーID共有 |
| プロトコル | Supabase内部（トリガー） |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | ユーザー登録完了時 |
| 方向 | AUS → DST（単方向） |
| 操作 | アプリケーションテーブルへの初期レコード生成 |

### 生成されるレコード

**users テーブル:**

| フィールド | 型 | 初期値 |
|-----------|------|--------|
| id | UUID | AUS のユーザーID |
| account_status | string | `active` |
| preferences | object | `{}` |

### 期待する振る舞い

- AUS でユーザーが作成されると、DST に users レコードが自動生成される
- 初期状態は account_status = `active`、preferences = `{}`
- credits / module_settings / tool_settings / api_keys 等はユーザー操作時にアプリケーション層で作成される（AUS トリガーの責務外）
- AUS ← DST の逆方向通信は存在しない

---

## 関連ドキュメント

| ドキュメント                     | 内容             |
| -------------------------- | -------------- |
| [itr-AUS.md](./itr-AUS.md) | Auth Server 仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 仕様  |

