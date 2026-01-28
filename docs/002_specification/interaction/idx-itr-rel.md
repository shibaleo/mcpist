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

## ITR-REL ID一覧（25件）

| ID | 連携 | 内容 | 参照ドキュメント |
|----|------|------|------------------|
| ITR-REL-001 | CLO - GWY | MCP通信 | [itr-CLO](./itr-CLO.md), [itr-GWY](./itr-GWY.md) |
| ITR-REL-002 | CLO - AUS | OAuth認可 | [itr-CLO](./itr-CLO.md), [itr-AUS](./itr-AUS.md) |
| ITR-REL-003 | CLK - GWY | MCP通信 | [itr-CLK](./itr-CLK.md), [itr-GWY](./itr-GWY.md) |
| ITR-REL-004 | GWY - AUS | トークン検証 | [itr-GWY](./itr-GWY.md), [itr-AUS](./itr-AUS.md) |
| ITR-REL-006 | GWY - AMW | リクエスト転送 | [itr-GWY](./itr-GWY.md), [itr-AMW](./itr-AMW.md) |
| ITR-REL-007 | AMW - HDL | 認証済みリクエスト | [itr-AMW](./itr-AMW.md), [itr-HDL](./itr-HDL.md) |
| ITR-REL-008 | HDL - DST | ユーザー設定取得 | [itr-HDL](./itr-HDL.md), [itr-DST](./itr-DST.md) |
| ITR-REL-009 | HDL - MOD | プリミティブ操作委譲 | [itr-HDL](./itr-HDL.md), [itr-MOD](./itr-MOD.md) |
| ITR-REL-010 | MOD - TVL | トークン取得 | [itr-MOD](./itr-MOD.md), [itr-TVL](./itr-TVL.md) |
| ITR-REL-012 | MOD - EXT | API呼び出し | [itr-MOD](./itr-MOD.md), [itr-EXT](./itr-EXT.md) |
| ITR-REL-014 | CON - SSM | ソーシャルログイン | [itr-CON](./itr-CON.md), [itr-SSM](./itr-SSM.md) |
| ITR-REL-015 | CON - TVL | トークン登録 | [itr-CON](./itr-CON.md), [itr-TVL](./itr-TVL.md) |
| ITR-REL-016 | CON - DST | ツール設定登録 | [itr-CON](./itr-CON.md), [itr-DST](./itr-DST.md) |
| ITR-REL-017 | CON - PSP | 決済 | [itr-CON](./itr-CON.md), [itr-PSP](./itr-PSP.md) |
| ITR-REL-018 | CON - EAS | 認可フロー | [itr-CON](./itr-CON.md), [itr-EAS](./itr-EAS.md) |
| ITR-REL-019 | SSM - IDP | ソーシャルログイン | [itr-SSM](./itr-SSM.md), [itr-IDP](./itr-IDP.md) |
| ITR-REL-020 | SSM - AUS | ユーザー認証連携 | [itr-SSM](./itr-SSM.md), [itr-AUS](./itr-AUS.md) |
| ITR-REL-021 | PSP - DST | 有料クレジット情報 | [itr-PSP](./itr-PSP.md), [itr-DST](./itr-DST.md) |
| ITR-REL-022 | AMW - DST | ユーザーコンテキスト取得 | [itr-AMW](./itr-AMW.md), [itr-DST](./itr-DST.md) |
| ITR-REL-023 | AUS - DST | ユーザーID共有 | [itr-AUS](./itr-AUS.md), [itr-DST](./itr-DST.md) |
| ITR-REL-024 | GWY - DST | APIキー検証 | [itr-GWY](./itr-GWY.md), [itr-DST](./itr-DST.md) |
| ITR-REL-025 | SSM - DST | ユーザー情報登録・参照 | [itr-SSM](./itr-SSM.md), [itr-DST](./itr-DST.md) |
| ITR-REL-026 | DST - TVL | ユーザー紐付け | [itr-DST](./itr-DST.md), [itr-TVL](./itr-TVL.md) |
| ITR-REL-027 | GWY - OBS | HTTPリクエストログ | [itr-GWY](./itr-GWY.md) |
| ITR-REL-028 | HDL - OBS | ツール実行ログ | [itr-HDL](./itr-HDL.md) |

---

## カテゴリ別分類

### MCP通信フロー（001-009）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-001 | CLO - GWY | OAuth2.0クライアントからのMCP通信 |
| ITR-REL-002 | CLO - AUS | OAuth認可フロー |
| ITR-REL-003 | CLK - GWY | APIキークライアントからのMCP通信 |
| ITR-REL-004 | GWY - AUS | OAuthトークン検証 |
| ITR-REL-006 | GWY - AMW | 認証済みリクエスト転送 |
| ITR-REL-007 | AMW - HDL | MCPリクエスト処理委譲 |
| ITR-REL-008 | HDL - DST | ユーザー設定・権限取得 |
| ITR-REL-009 | HDL - MOD | モジュールへのプリミティブ操作委譲 |

### モジュール実行フロー（010-013）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-010 | MOD - TVL | 外部サービストークン取得 |
| ITR-REL-012 | MOD - EXT | 外部サービスAPI呼び出し |

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

### データストア・インフラ連携（022-028）

| ID | 連携 | 説明 |
|----|------|------|
| ITR-REL-022 | AMW - DST | ユーザーコンテキスト取得（アカウント状態・クレジット・モジュール設定） |
| ITR-REL-023 | AUS - DST | ユーザーID共有（PostgreSQLトリガー） |
| ITR-REL-024 | GWY - DST | APIキー検証（SHA-256ハッシュ照合） |
| ITR-REL-025 | SSM - DST | ユーザー情報の登録・参照 |
| ITR-REL-026 | DST - TVL | トークン管理のユーザー紐付け |
| ITR-REL-027 | GWY - OBS | HTTPリクエストログ送信 |
| ITR-REL-028 | HDL - OBS | ツール実行ログ・セキュリティイベント送信 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
