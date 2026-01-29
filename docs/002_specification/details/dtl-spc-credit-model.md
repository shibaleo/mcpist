# クレジットモデル詳細仕様書（dtl-spc-credit-model）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Credit Model Detail Specification |

---

## 概要

本ドキュメントはMCPistのクレジットシステムの設計を記述する。

---

## 設計原則

1. **直感的なUX**: 課金 = クレジット積み上げ
2. **無料ユーザーの継続利用**: 少量利用なら永続的に利用可能
3. **非アクティブユーザーの冪等性**: 状態管理の運用コスト削減

---

## クレジット種別

| フィールド | 型 | 説明 |
|-----------|-----|------|
| free_credits | number | 無料クレジット（上限1000、毎月補充） |
| paid_credits | number | 有料クレジット（上限なし、積み上げ） |

**内部・ユーザー表示ともに2種類を分離して管理・表示する。**

---

## 動作ルール

### 新規登録時

```
free_credits = 1000
paid_credits = 0
```

### 月初処理（Cron）

```
free_credits = 1000
```

- 無料クレジットを1000に補充（リセットではなく「1000に戻す」）
- paid_creditsは変更なし
- 非アクティブユーザー: `free_credits = 1000` のまま → 冪等

### クレジット購入時

```
paid_credits += 購入額
```

- PSPからのWebhook（checkout.session.completed）で加算
- 上限なし、永続的に積み上げ可能

### クレジット消費時

```
if free_credits > 0:
    free_credits -= 1
else:
    paid_credits -= 1
```

- 無料クレジットを先に消費
- 無料が0になったら有料クレジットを消費
- 合計残高が0の場合はツール実行を拒否

---

## ユーザー表示

### 管理画面（CON）

無料枠と購入分を分離して表示する。

**表示例:**
```
クレジット残高
├ 無料枠: 800 / 1,000（毎月1日に回復）
└ 購入分: 500
```

**表示のメリット:**
- 無料枠が1000まで回復することが明確
- 購入分は別枠で積み上がることが明確
- 消費順序（無料→購入）を自然に理解できる

### MCP Server（HDL）

クレジット不足時のエラーレスポンス:

```json
{
  "error": {
    "code": -32001,
    "message": "Insufficient credits",
    "data": {
      "required": 1,
      "available": 0
    }
  }
}
```

---

## 冪等性の保証

| ユーザー状態 | 月初処理後の状態 |
|-------------|-----------------|
| 非アクティブ（無料） | `free_credits = 1000, paid_credits = 0` |
| 非アクティブ（課金済み） | `free_credits = 1000, paid_credits = N`（変化なし） |

非アクティブユーザーは毎月同じ処理を受けても状態が変わらない。

---

## Auto-recharge（オプション機能）

| フィールド | 型 | 説明 |
|-----------|-----|------|
| enabled | boolean | Auto-recharge有効/無効 |
| threshold | number | 自動購入をトリガーする残高閾値 |
| recharge_amount | number | 自動購入するクレジット数 |

クレジット消費時に合計残高（free + paid）がthresholdを下回ると、PSP経由で自動的にrecharge_amount分の有料クレジットを購入する。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-dst.md](../interaction/itr-dst.md) | Data Store詳細仕様 |
| [itr-psp.md](../interaction/itr-psp.md) | Payment Service Provider詳細仕様 |
| [itr-con.md](../interaction/itr-con.md) | User Console詳細仕様 |
| [itr-hdl.md](../interaction/itr-hdl.md) | MCP Handler詳細仕様 |
