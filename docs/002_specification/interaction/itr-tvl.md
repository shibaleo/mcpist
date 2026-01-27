# Token Vault インタラクション仕様書（itr-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.1 |
| Note | Token Vault Interaction Specification |

---

## 概要

Token Vault（TVL）は、外部サービスのOAuthトークン・API KEYを安全に管理するデータストア。

主な責務：
- 外部サービスのOAuthトークン保存・取得
- トークンリフレッシュの自動実行
- API KEY認証の提供
- トークンの暗号化保存

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| API Gateway | TVL ← GWY | API KEY検証 | [dtl-itr-GWY-TVL.md](./dtl-itr-GWY-TVL.md) |
| User Console | TVL ← CON | トークン登録 | [dtl-itr-CON-TVL.md](./dtl-itr-CON-TVL.md) |
| Modules | TVL → MOD | トークン提供 | [dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md) |
| External Auth Server | TVL → EAS | トークンリフレッシュ | [dtl-itr-EAS-TVL.md](./dtl-itr-EAS-TVL.md) |

---

## サービス別トークン形式

| Service | long_term_token | oauth_token |
|---------|-----------------|-------------|
| notion | Internal Integration Token (`ntn_xxx`) | OAuth Access Token |
| google_calendar | - | OAuth Access Token |
| microsoft_todo | - | OAuth Access Token |

---

## TVLが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | AUS経由で認証 |
| Auth Server (AUS) | OAuth2.0はAUS担当 |
| Session Manager (SSM) | DST経由 |
| Auth Middleware (AMW) | GWY経由 |
| MCP Handler (HDL) | MOD経由 |
| Identity Provider (IDP) | SSM経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itf-tvl.md](../dtl-spc/itf-tvl.md) | Token Vault API仕様 |
| [itr-gwy.md](./itr-gwy.md) | API Gateway詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
