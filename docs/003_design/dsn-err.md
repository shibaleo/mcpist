# エラーハンドリング設計書（dsn-err）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Error Handling Design |

---

## 概要

本ドキュメントは、MCPistにおけるエラーハンドリングとリトライポリシーを定義する。

---

## エラーコード

### トークン関連

| コード | 説明 |
|--------|------|
| TOKEN_NOT_CONFIGURED | トークン未設定 |
| TOKEN_EXPIRED | トークン期限切れ（リフレッシュ失敗） |
| TOKEN_INVALID | トークン無効 |

### 外部API関連

| コード | 説明 |
|--------|------|
| EXTERNAL_API_ERROR | 外部API呼び出しエラー |
| RATE_LIMITED | レート制限 |
| TIMEOUT | タイムアウト |

### 認可関連

| コード | 説明 |
|--------|------|
| MODULE_NOT_ALLOWED | モジュールアクセス不可 |
| CREDIT_EXHAUSTED | クレジット残高不足 |

---

## エラーレスポンス形式

### 基本形式

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### トークン未設定

```json
{
  "success": false,
  "error": {
    "code": "TOKEN_NOT_CONFIGURED",
    "message": "Token not configured for service: notion"
  }
}
```

### 外部API呼び出しエラー

```json
{
  "success": false,
  "error": {
    "code": "EXTERNAL_API_ERROR",
    "message": "Notion API returned 401: Invalid token"
  }
}
```

### レート制限

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded for Notion API",
    "retry_after": 60
  }
}
```

---

## リトライポリシー

| 条件 | リトライ | 最大回数 | バックオフ |
|------|----------|----------|-----------|
| 5xx エラー | する | 3回 | 指数バックオフ |
| 429 Rate Limit | する | 3回 | Retry-Afterに従う |
| 4xx エラー | しない | - | - |
| タイムアウト | する | 2回 | 固定1秒 |

### 指数バックオフ

```
wait_time = base_delay * (2 ^ retry_count) + jitter
base_delay = 1秒
jitter = 0-500ms のランダム値
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-mod.md](../002_specification/interaction/itr-mod.md) | Modules インタラクション仕様 |
| [itf-mod.md](../002_specification/dtl-spc/itf-mod.md) | モジュールインターフェース仕様 |
