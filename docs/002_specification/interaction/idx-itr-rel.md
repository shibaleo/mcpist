# インタラクション関係ID一覧（idx-itr-rel）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v1.0 |
| Note | Interaction Relation ID Index |

---

## 概要

コンポーネント間のインタラクション関係に付与されたIDの一覧。

---

## ITR-REL ID一覧（21件）

| ID | 連携 | 内容 | 参照ドキュメント |
|----|------|------|------------------|
| ITR-REL-001 | CLO - GWY | MCP通信 | [itr-clo](./itr-clo.md), [itr-gwy](./itr-gwy.md) |
| ITR-REL-002 | CLO - AUS | OAuth認可 | [itr-clo](./itr-clo.md), [itr-aus](./itr-aus.md) |
| ITR-REL-003 | CLK - GWY | MCP通信 | [itr-clk](./itr-clk.md), [itr-gwy](./itr-gwy.md) |
| ITR-REL-004 | GWY - AUS | トークン検証 | [itr-gwy](./itr-gwy.md), [itr-aus](./itr-aus.md) |
| ITR-REL-005 | GWY - TVL | APIキー検証 | [itr-gwy](./itr-gwy.md), [itr-tvl](./itr-tvl.md) |
| ITR-REL-006 | GWY - AMW | リクエスト転送 | [itr-gwy](./itr-gwy.md), [itr-amw](./itr-amw.md) |
| ITR-REL-007 | AMW - HDL | 認証済みリクエスト | [itr-amw](./itr-amw.md), [itr-hdl](./itr-hdl.md) |
| ITR-REL-008 | HDL - DST | ユーザー設定取得 | [itr-hdl](./itr-hdl.md), [itr-dst](./itr-dst.md) |
| ITR-REL-009 | HDL - MOD | プリミティブ操作委譲 | [itr-hdl](./itr-hdl.md), [itr-mod](./itr-mod.md) |
| ITR-REL-010 | MOD - TVL | トークン取得 | [itr-mod](./itr-mod.md), [itr-tvl](./itr-tvl.md) |
| ITR-REL-011 | MOD - DST | クレジット消費 | [itr-mod](./itr-mod.md), [itr-dst](./itr-dst.md) |
| ITR-REL-012 | MOD - EXT | API呼び出し | [itr-mod](./itr-mod.md), [itr-ext](./itr-ext.md) |
| ITR-REL-013 | TVL - EAS | トークンリフレッシュ | [itr-tvl](./itr-tvl.md), [itr-eas](./itr-eas.md) |
| ITR-REL-014 | CON - SSM | ソーシャルログイン | [itr-con](./itr-con.md), [itr-ssm](./itr-ssm.md) |
| ITR-REL-015 | CON - TVL | トークン登録 | [itr-con](./itr-con.md), [itr-tvl](./itr-tvl.md) |
| ITR-REL-016 | CON - DST | ツール設定登録 | [itr-con](./itr-con.md), [itr-dst](./itr-dst.md) |
| ITR-REL-017 | CON - PSP | 決済 | [itr-con](./itr-con.md), [itr-psp](./itr-psp.md) |
| ITR-REL-018 | CON - EAS | 認可フロー | [itr-con](./itr-con.md), [itr-eas](./itr-eas.md) |
| ITR-REL-019 | SSM - IDP | ソーシャルログイン | [itr-ssm](./itr-ssm.md), [itr-idp](./itr-idp.md) |
| ITR-REL-020 | SSM - AUS | ユーザー認証連携 | [itr-ssm](./itr-ssm.md), [itr-aus](./itr-aus.md) |
| ITR-REL-021 | PSP - DST | 有料クレジット情報 | [itr-psp](./itr-psp.md), [itr-dst](./itr-dst.md) |

---

## カテゴリ別分類

### MCP通信フロー（001-009）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-001 | CLO - GWY | OAuth2.0クライアントからのMCP通信 |
| ITR-REL-002 | CLO - AUS | OAuth認可フロー |
| ITR-REL-003 | CLK - GWY | APIキークライアントからのMCP通信 |
| ITR-REL-004 | GWY - AUS | OAuthトークン検証 |
| ITR-REL-005 | GWY - TVL | APIキー検証 |
| ITR-REL-006 | GWY - AMW | 認証済みリクエスト転送 |
| ITR-REL-007 | AMW - HDL | MCPリクエスト処理委譲 |
| ITR-REL-008 | HDL - DST | ユーザー設定・権限取得 |
| ITR-REL-009 | HDL - MOD | モジュールへのプリミティブ操作委譲 |

### モジュール実行フロー（010-013）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-010 | MOD - TVL | 外部サービストークン取得 |
| ITR-REL-011 | MOD - DST | クレジット消費記録 |
| ITR-REL-012 | MOD - EXT | 外部サービスAPI呼び出し |
| ITR-REL-013 | TVL - EAS | OAuth2.0トークンリフレッシュ |

### ユーザーコンソールフロー（014-018）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-014 | CON - SSM | ユーザーログイン |
| ITR-REL-015 | CON - TVL | 外部サービストークン登録 |
| ITR-REL-016 | CON - DST | ユーザー設定管理 |
| ITR-REL-017 | CON - PSP | クレジット購入 |
| ITR-REL-018 | CON - EAS | 外部サービスOAuth認可 |

### 外部サービス連携（019-021）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-019 | SSM - IDP | ソーシャルログイン（Google, GitHub等） |
| ITR-REL-020 | SSM - AUS | OAuth認可時のユーザー認証連携 |
| ITR-REL-021 | PSP - DST | 決済完了Webhook |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
