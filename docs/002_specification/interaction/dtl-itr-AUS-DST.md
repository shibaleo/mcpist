# AUS - DST インタラクション詳細（dtl-itr-AUS-DST）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-023 |
| Note | Auth Server - Data Store Interaction Detail |

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
| トリガー | ユーザー登録・認証完了時 |
| 操作 | auth.usersテーブルのユーザーIDをアプリケーションテーブルへ同期 |

### メカニズム

Supabase Authがauth.usersにレコードを作成した際、PostgreSQLトリガーによりアプリケーション側のユーザーテーブルにレコードが自動生成される。

### データフロー

| 方向 | 内容 |
|------|------|
| AUS → DST | ユーザー登録時にトリガーでユーザーIDを同期 |
| AUS ← DST | （直接の逆方向通信なし） |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-AUS.md](./itr-AUS.md) | Auth Server 詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
