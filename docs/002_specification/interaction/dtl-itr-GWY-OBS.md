# GWY - OBS インタラクション詳細（dtl-itr-GWY-OBS）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-027 |
| Note | API Gateway - Observability Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Observability (OBS) |
| 内容 | HTTPリクエストログ送信 |
| プロトコル | HTTP（Loki Push API） |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | 全HTTPリクエスト処理時 |
| 操作 | リクエスト/レスポンスログの送信 |

### 送信するログ情報

| フィールド | 説明 |
|-----------|------|
| method | HTTPメソッド |
| path | リクエストパス |
| status | HTTPステータスコード |
| duration | レスポンス時間 |
| user_id | 認証済みユーザーID（認証成功時） |
| auth_type | 認証方式（jwt / api_key） |
| request_id | リクエスト追跡用ID |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-GWY.md](./itr-GWY.md) | API Gateway 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
