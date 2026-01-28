# DST - GWY インタラクション詳細（dtl-itr-DST-GWY）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-024 |
| Note | Data Store - API Gateway Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Data Store (DST) |
| 内容 | APIキー検証 |
| プロトコル | Supabase RPC |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | API Key認証リクエスト受信時（KVキャッシュミス時） |
| 操作 | APIキーハッシュによるユーザー検索 |

### 検証フロー

1. API KeyをSHA-256でハッシュ化
2. KVキャッシュを照合（TTL: 24h / soft max-age: 1h）
3. キャッシュミス時、Supabase RPC `lookup_user_by_key_hash` を実行
4. 検証成功時、結果をKVキャッシュに書き込み

### RPC関数

| 関数 | 入力 | 出力 |
|------|------|------|
| `lookup_user_by_key_hash` | SHA-256ハッシュ値 | user_id, key_status |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-GWY.md](./itr-GWY.md) | API Gateway 詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
