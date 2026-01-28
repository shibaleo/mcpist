# DST - TVL インタラクション詳細（dtl-itr-DST-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-026 |
| Note | Data Store - Token Vault Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Data Store (DST) |
| 連携先 | Token Vault (TVL) |
| 内容 | トークン管理のためのユーザー紐付け |
| プロトコル | Supabase内部（同一DB） |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | トークン保存・取得時 |
| 操作 | ユーザーIDを外部キーとしてトークンをユーザーに紐付け |

### メカニズム

Data Store（アプリケーションテーブル）とToken Vault（Supabase Vault）は同一Supabaseプロジェクト内に存在する。ユーザーIDを外部キーとして、トークンの所有者管理を行う。

### データフロー

| 方向 | 内容 |
|------|------|
| DST → TVL | ユーザーIDによるトークン所有者の特定 |
| DST ← TVL | トークン存在確認（接続状態の参照） |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
