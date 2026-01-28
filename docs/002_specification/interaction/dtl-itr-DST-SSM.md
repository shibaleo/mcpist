# DST - SSM インタラクション詳細（dtl-itr-DST-SSM）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-025 |
| Note | Data Store - Session Manager Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Session Manager (SSM) |
| 連携先 | Data Store (DST) |
| 内容 | ユーザー情報の登録・参照 |
| プロトコル | Supabase Client SDK |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | ユーザーログイン・セッション確認時 |
| 操作 | ユーザー情報の登録・参照 |

### データフロー

| 操作 | 説明 |
|------|------|
| 登録 | ソーシャルログイン完了後、ユーザープロファイル情報をDSTに登録 |
| 参照 | セッション検証時、ユーザー情報を取得 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-SSM.md](./itr-SSM.md) | Session Manager 詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
