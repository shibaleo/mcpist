# GWY - TVL インタラクション詳細（dtl-itr-GWY-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-005 |
| Note | API Gateway - Token Vault Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Token Vault (TVL) |
| 内容 | APIキー検証 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| 方向 | GWY が TVL に API KEY ハッシュを問い合わせ |
| 用途 | MCP Client (API KEY) からのリクエスト検証 |

### セキュリティ設計

API KEYの平文をGWYのメモリに保持しないため、ハッシュ比較方式を採用する。

```
保存時: TVLにSHA256(api_key)を保存
検証時: GWYがSHA256(受信したapi_key)を計算し、TVLに問い合わせ
```

### 検証フロー

1. GWYがAPI KEYを受信（`Authorization: Bearer mpt_xxx`）
2. GWYがAPI KEYのSHA256ハッシュを計算（平文は即破棄）
3. キャッシュにハッシュ→user_idの対応があれば使用
4. キャッシュになければTVLにハッシュで検証リクエスト
5. TVLがハッシュに紐づくユーザーIDを返却
6. ハッシュ→user_idの対応をキャッシュ
7. 検証成功時：ユーザー情報をヘッダーに付与してAMWへ転送

### キャッシュ

- ハッシュ→user_idの対応をキャッシュ可能（ハッシュは平文ではないため）
- API KEY平文はキャッシュしない

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-gwy.md](./itr-gwy.md) | API Gateway 詳細仕様 |
| [itr-tvl.md](./itr-tvl.md) | Token Vault 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
