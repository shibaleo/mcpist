---
title: ARD-008 認証基盤のアーキテクチャ決定
aliases:
  - ARD-008
  - auth-token-vault-architecture
tags:
  - MCPist
  - ARD
  - architecture-decision
document-type:
  - decision-record
document-class: ARD
created: 2026-01-15T00:00:00+09:00
updated: 2026-01-15T00:00:00+09:00
---
# ARD-008: 認証基盤のアーキテクチャ決定

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `accepted` |
| Decision | Supabase Auth + Supabase Vault + Edge Functionで自前実装 |
| Deciders | アーキテクト |
| Date | 2026-01-15 |

---

## コンテキスト

MCPistの認証・トークン管理基盤を構築するにあたり、以下の選択肢を検討した：

| 選択肢 | Auth Provider | Token Vault | User Profile DB |
|--------|--------------|-------------|-----------------|
| A | Clerk | Auth0 Token Vault | Supabase |
| B | Clerk | Scalekit | Supabase |
| C | Auth0 | Auth0 Token Vault | Supabase |
| D | **Supabase Auth** | **Supabase Vault** | **Supabase DB** |

### 要件

MCPistでは以下の認証方式に対応する必要がある：

1. **OAuth 2.1 Authorization Code Flow** - LLMクライアント（Claude Code等）からの認可
2. **Bearer Token（Long-lived Token）** - API直接呼び出し用の長期トークン
3. **外部サービスOAuth** - Google Calendar, Microsoft Todo, Notion, **PKMist**等への連携

特に重要な要件：
- **任意のOAuthサービスに対応可能であること**（PKMist等の自作サービスを含む）
- MCPサーバーを自前実装する意義を最大化すること

---

## 決定

**Supabase Auth + Supabase Vault + Edge Functionで自前実装する。**

### 採用構成

| 責務 | プロバイダー | 実装 |
|------|-------------|------|
| **Auth Provider** | Supabase Auth | JWT発行・検証、セッション管理 |
| **Token Vault** | Supabase Vault | AES-256-GCM暗号化、トークン保存 |
| **Token Refresh** | Edge Function | 自前実装（約200行） |
| **User Profile DB** | Supabase DB | RLS、Tool Sieve、設定管理 |

---

## 検討過程

### 初期検討: 分散アーキテクチャの魅力

当初、**Supabase一極集中を避ける**ことを目的に、以下の分散構成を検討した：

```
Clerk（Auth）+ Scalekit（Token Vault）+ Supabase（DB）
```

**期待したメリット:**
1. 単一障害点の回避
2. ベンダーロックイン軽減
3. 各責務に最適なサービス選択

### 問題1: user_id共有の複雑さ

異なるプロバイダー間でuser_idを共有するには追加の実装が必要：

| 方式 | 実装コスト | 障害パターン |
|------|----------|-------------|
| JWTカスタムクレーム | 中 | 同期ミス |
| Webhook連携 | 高 | 同期失敗、リカバリ |
| マッピングテーブル | 中 | 毎回のDB呼び出し |

**結論**: AuthとVaultを繋げる一般的なソリューションは「同一プロバイダー」である。分散構成はuser_id連携のMaintenance Liabilityを生む。

### 問題2: 外部Token Vaultの対応サービス制限

Scalekit、Auth0 Token Vault等は**事前定義されたサービスのみ**対応：

| サービス | 対応状況 |
|---------|---------|
| GitHub, Notion, Slack等 | ◎ 対応 |
| PKMist（自作OAuth） | × **永久に非対応** |
| 任意のOAuthサービス | × 非対応 |

**致命的問題**: MCPistがPKMist（自作OAuthサーバー）と連携できない。

```
MCPist ─────→ 外部Token Vault ─────→ PKMist
                    ↓
              対応サービスに
              PKMistは含まれない
```

### 問題3: Refresh実装コストの比較

| 構成 | 自前実装範囲 | コード量 | 障害パターン |
|------|------------|---------|-------------|
| Supabase一極集中 | Refreshロジックのみ | 約200行 | refresh失敗のみ |
| 分散構成 | Mapping + Webhook + 同期 | 500行以上 | 同期失敗、ID不整合、Webhook障害 |

**結論**: 分散構成の複雑さは、Refresh自前実装のコストを大きく上回る。

---

## 自前実装の責任分析

### Design Liability（設計責任）

**結論: Design Liabilityは実質的に発生しない。**

セキュリティの核心機能を全てSupabase（業界標準サービス）に委譲しているため。

| コンポーネント | 核心機能 | 責任の所在 | 自前実装の範囲 |
|--------------|---------|-----------|--------------|
| OAuth Auth Server | JWT発行・検証 | **Supabase Auth** | UIとRPC制御フロー |
| Token Vault | 暗号化保存 | **Supabase Vault** | APIラッパー（RPC） |
| Token Refresh | 外部API呼び出し | **自前** | HTTPリクエスト |

#### 自前実装（Edge Function）の役割

Edge Functionは**RPCエントリーポイント**としてのみ機能：

```
クライアント → Edge Function（RPC） → Supabase Auth / Vault
                    ↑
              制御フローのみ
              セキュリティ機能なし
```

- JWT発行・検証 → Supabase Authが実行
- トークン暗号化 → Supabase Vault（AES-256-GCM）が実行
- アクセス制御 → Supabase RLS（Row Level Security）が実行

**Supabase Auth自体がIDaaSである**ため、Clerk/Auth0を使う場合と同等の設計責任。

### Maintenance Liability（保守責任）

以下の仕様変更への追従責任は自分にある：

| 仕様 | 変更主体 | 変更頻度 | 対応方針 |
|------|---------|---------|---------|
| OAuth 2.1 | IETF | 低（2.0→2.1に8年） | 成熟仕様、破壊的変更は稀 |
| MCP Authorization | Anthropic | 中 | IDaaSは非対応、自前実装必須 |
| 各外部サービスのRefresh仕様 | 各社 | 低〜中 | サービス追加時に実装 |

**受容理由:**
- OAuth 2.1は成熟した仕様（2.0: 2012年、2.1 Draft: 2020年）
- MCP仕様は自前実装でないと追従できない（IDaaSはMCP非対応）
- Refresh仕様は標準的なOAuth 2.0フローで大きな変更は稀
- dwhbiでの運用経験により、仕様変更の監視・対応体制は確立済み

### Security Liability（セキュリティ責任）

| 観点 | 責任の所在 | 根拠 |
|------|-----------|------|
| JWT署名・検証 | Supabase | Supabase Auth標準機能 |
| トークン暗号化 | Supabase | Supabase Vault（AES-256-GCM） |
| アクセス制御 | Supabase | RLS（Row Level Security） |
| 脆弱性対応 | Supabase | SOC 2 Type II準拠 |
| Refreshロジック | **自前** | 外部HTTPリクエストのみ |

自前実装部分（Refresh）は外部OAuthサーバーへのHTTPリクエストのみであり、セキュリティの核心部分ではない。

---

## 自前実装の柔軟性

### MCPサーバー自前実装の意義

MCPistの価値は**任意のモジュール・サービスを追加できること**：

| ユーザー要望 | 外部Token Vault | Supabase自前 |
|------------|:---------------:|:------------:|
| Notion連携 | ◎ | ◎ |
| PKMist連携（自作） | × 非対応 | **◎ 対応** |
| 社内システム連携 | × 非対応 | **◎ 対応** |
| マイナーSaaS連携 | △ 対応待ち | **◎ 即座に可能** |

**結論**: 約200行のRefresh実装と引き換えに、任意のOAuthサービスに対応できる柔軟性を得られる。

### PKMistとの連携

将来構築するPKMist（自作OAuthサーバー）との連携：

```
┌─────────────┐      ┌─────────────┐
│   MCPist    │ ──── │   PKMist    │
│             │      │ (自作OAuth) │
│ Supabase    │      └─────────────┘
│  Auth+Vault │
│ (refresh対応)│
└─────────────┘
```

外部Token Vaultでは**永久に**PKMistに対応できないが、自前実装なら即座に対応可能。

---

## 検討した代替案

### 代替案A: Clerk + Auth0 Token Vault

```
Clerk（認証）+ Auth0 Token Vault（トークン管理）+ Supabase（DB）
```

**検討結果: 不採用**

| 評価項目 | 評価 |
|---------|------|
| DX | ◎ Clerk Embedded Components優秀 |
| コスト | × Auth0 Token Vault価格未定 |
| 柔軟性 | × PKMist等カスタムサービス非対応 |
| user_id連携 | × Webhook/マッピング必要 |

### 代替案B: Clerk + Scalekit + Supabase

```
Clerk（認証）+ Scalekit（Token Vault）+ Supabase（DB）
```

**検討結果: 不採用**

| 評価項目 | 評価 |
|---------|------|
| 分散 | ◎ 3プロバイダー分散 |
| AIエージェント対応 | ◎ Scalekit MCP Auth対応 |
| 柔軟性 | × 対応サービス固定（20+のみ） |
| user_id連携 | × Webhook/マッピング必要 |

**致命的問題**: PKMist（自作OAuth）に対応できない。

### 代替案C: Auth0単独

```
Auth0（認証）+ Auth0 Token Vault（トークン管理）
```

**検討結果: 不採用**

| 評価項目 | 評価 |
|---------|------|
| 統一性 | ◎ 単一ベンダー |
| コスト | × 有料プラン必須、Token Vault価格未定 |
| 柔軟性 | × 対応サービス固定 |

### 代替案D: Supabase一極集中（採用）

```
Supabase Auth + Supabase Vault + Supabase DB + Edge Function
```

**検討結果: 採用**

| 評価項目 | 評価 |
|---------|------|
| 統一性 | ◎ 単一プラットフォーム |
| コスト | ◎ Free tierで完結 |
| 柔軟性 | ◎ **任意のOAuthサービス対応** |
| user_id連携 | ◎ 自動（同一プラットフォーム） |
| 実装実績 | ◎ dwhbiで稼働中 |

---

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────┐
│                        MCPist システム                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐    │
│  │                    Supabase                             │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │    │
│  │  │ Supabase Auth│  │ Edge Function│  │ Supabase DB  │  │    │
│  │  │              │  │              │  │              │  │    │
│  │  │ ・ソーシャル  │  │ ・OAuth Server│ │ ・Tool Sieve │  │    │
│  │  │   ログイン   │  │ ・Token Vault│  │ ・oauth_tokens│ │    │
│  │  │ ・JWT発行   │  │   API        │  │ ・mcp_tokens │  │    │
│  │  │ ・セッション │  │ ・リフレッシュ │  │              │  │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │    │
│  │                           │                │            │    │
│  │                           │ Supabase Vault │            │    │
│  │                           │  (AES-256-GCM) │            │    │
│  │                           └────────────────┘            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│         │ user_id                    │ user_id → token          │
│         ▼                            ▼                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    MCPサーバー (Koyeb)                    │   │
│  │                                                         │   │
│  │  認証MW → Tool Sieve → モジュールレジストリ → モジュール    │   │
│  │                                      │                  │   │
│  │                                      ▼                  │   │
│  │                              外部API (Google, PKMist等)  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 責務分担

| コンポーネント | 提供者 | 責務 |
|--------------|-------|------|
| **ユーザー認証** | Supabase Auth | ソーシャルログイン、JWT発行、セッション管理 |
| **OAuthサーバー** | Edge Function | Authorization Code Flow、トークン発行、Consent画面 |
| **Long-lived Token** | Edge Function + DB | トークン発行、ハッシュ検証 |
| **Token Vault** | Supabase Vault | 外部サービストークンの暗号化保存 |
| **Token Refresh** | Edge Function | 自前実装、外部OAuthサーバーへのリフレッシュ要求 |
| **Tool Sieve** | Supabase DB | ロール・権限管理、ツールフィルタリング |
| **MCPサーバー** | Koyeb | MCP Protocol処理、モジュール実行 |

---

## 認証方式

### 1. OAuth 2.1 Authorization Code Flow

LLMクライアント（Claude Code, Cursor等）からの標準的なOAuth認可。

```
1. LLMクライアントが認可リクエスト
   GET /oauth/authorize?response_type=code&client_id=...&redirect_uri=...
2. ユーザーがログイン（Supabase Auth）
3. Consent画面で許可
4. 認可コードをコールバック
5. トークン交換
   POST /oauth/token (grant_type=authorization_code)
6. Access Token + Refresh Token発行
```

**準拠規格:**
- RFC 8414: OAuth Authorization Server Metadata
- RFC 9728: OAuth Protected Resource Metadata
- OAuth 2.1 Draft

### 2. Bearer Token（Long-lived Token）

API直接呼び出し用の長期トークン。管理UIから発行・管理。

```
1. ユーザーが管理UIでトークン発行
2. トークン（64文字hex）を表示（一度のみ）
3. SHA-256ハッシュをDBに保存
4. APIリクエスト時にBearer Tokenとして使用
   Authorization: Bearer <64-char-hex-token>
5. MCPサーバーがハッシュ照合で検証
```

### 3. 外部サービスOAuth（任意のサービス対応）

Google Calendar, Microsoft Todo, Notion, **PKMist**等への連携。

```
1. ユーザーが管理UIで「連携」をクリック
2. Edge Functionが認可URLを生成
3. 外部サービスでユーザーが同意
4. コールバックでトークン取得
5. Supabase Vault（暗号化）に保存
6. MCPサーバーからToken Vault APIで取得
7. 期限切れ時はEdge Functionがリフレッシュ
```

---

## 既存実装の流用

dwhbiプロジェクトから以下のコンポーネントを流用：

| コンポーネント | 流用元 | 説明 |
|--------------|--------|------|
| Supabaseクライアント | `/lib/supabase/*` | Client/Server/Service Role各種 |
| Token Vault | `/lib/vault.ts` | 12+サービス対応のトークン管理 |
| OAuth認可開始 | `/api/oauth/{service}/route.ts` | 認可URL生成 |
| OAuthコールバック | `/api/oauth/{service}/callback/route.ts` | トークン交換・保存 |
| MCP認証 | `/api/mcp/lib/auth.ts` | Bearer Token + Long-lived Token検証 |
| Consent画面 | `/app/auth/consent/page.tsx` | OAuth同意画面UI |
| MCP Token管理 | `/api/mcp/tokens/route.ts` | CRUD API |

---

## データモデル

### oauth_tokens（外部サービストークン）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | トークンID（PK） |
| user_id | UUID | ユーザーID（FK） |
| service | VARCHAR | サービス名（notion, google_calendar, pkmist等） |
| access_token | TEXT | アクセストークン（Vault暗号化） |
| refresh_token | TEXT | リフレッシュトークン（Vault暗号化） |
| expires_at | TIMESTAMP | 有効期限 |
| scopes | TEXT[] | 許可スコープ |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

### mcp_tokens（Long-lived Token）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | トークンID（PK） |
| user_id | UUID | ユーザーID（FK） |
| name | VARCHAR | トークン名（ユーザー識別用） |
| token_hash | TEXT | SHA-256ハッシュ |
| expires_at | TIMESTAMP | 有効期限（NULL=無期限） |
| last_used_at | TIMESTAMP | 最終使用日時 |
| revoked | BOOLEAN | 失効フラグ |
| created_at | TIMESTAMP | 作成日時 |

---

## セキュリティ

### 暗号化

| データ | 暗号化方式 | 管理者 |
|--------|----------|--------|
| 外部サービストークン | AES-256-GCM | Supabase Vault |
| Long-lived Token | SHA-256ハッシュ（保存時） | アプリケーション |
| セッション | JWT（RS256署名） | Supabase Auth |

### アクセス制御

| レイヤー | 方式 |
|---------|------|
| DB | RLS（Row Level Security）by user_id |
| API | JWT検証 + user_id照合 |
| Edge Function | Service Role Key（内部通信） |

---

## コスト比較

| 構成 | 月額コスト | 備考 |
|------|----------|------|
| Clerk + Scalekit | $25〜 | Clerk有料プラン |
| Clerk + Auth0 | $25〜 + α（未定） | Auth0 Token Vault価格未定 |
| Auth0単独 | $$$（有料プラン必須） | Enterprise向け価格 |
| **Supabase自前実装** | **$0** | Free tierで十分 |

---

## リスクと緩和策

### リスク1: 自前実装の運用負荷

- **状況**: トークンリフレッシュ失敗、認証エラー等の対応が必要
- **緩和策**:
  - dwhbiでの運用実績を活かしたエラーハンドリング
  - 監視・アラート設定（Grafana）
  - リフレッシュ失敗時のユーザー通知機能

### リスク2: スケーラビリティ

- **状況**: ユーザー増加時のパフォーマンス
- **緩和策**:
  - Supabase Edge Functionは自動スケール
  - 必要に応じてPro planへアップグレード可能

### リスク3: Supabase単一障害点

- **状況**: Supabase障害時に全機能停止
- **緩和策**:
  - Supabaseの稼働率は99.9%+
  - 分散構成の複雑さ（Webhook同期、ID不整合）と比較して許容可能
  - PostgreSQL標準のため、必要に応じて移行可能

---

## 結論

**Supabase Auth + Supabase Vault + Edge Functionで自前実装する。**

### 決定理由

1. **柔軟性**: 任意のOAuthサービス（PKMist含む）に対応可能
2. **user_id統一**: 同一プラットフォームで自動的にuser_id共有
3. **Design Liability最小化**: セキュリティ核心機能はSupabaseに委譲
4. **Maintenance Liability許容**: OAuth 2.1は成熟仕様、MCPは自前実装必須
5. **実装コスト**: Refresh約200行 vs 分散構成500行以上
6. **コスト**: Supabase Free tierで運用可能
7. **実績**: dwhbiで同等システムが稼働中

### 却下した分散構成の理由

1. **user_id連携の複雑さ**: Webhook/マッピングテーブルが必要
2. **対応サービスの制限**: 外部Token VaultはPKMist等に非対応
3. **MCPサーバー自前実装の意義喪失**: 柔軟なモジュール追加ができなくなる

---

## 影響を受けるドキュメント

| ドキュメント | 変更内容 |
|-------------|---------|
| [spec-ifc.md](spec-ifc.md) | 認証フロー、Token Vault APIの詳細化 |
| [spec-dsn.md](../DAY5/004-spec-dsn/spec-dsn.md) | データモデルは現行維持 |
| [spec-sys.md](../DAY5/002-spec-sys/spec-sys.md) | コンポーネント構成は現行維持 |
| [spec-inf.md](../DAY5/003-spec-inf/spec-inf.md) | Edge Function追加 |

---

## 参考資料

- [Supabase Auth Documentation](https://supabase.com/docs/guides/auth)
- [Supabase Vault](https://supabase.com/docs/guides/database/vault)
- [Supabase Edge Functions](https://supabase.com/docs/guides/functions)
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-07)
- [RFC 8414 - OAuth Authorization Server Metadata](https://www.rfc-editor.org/rfc/rfc8414)
- [RFC 9728 - OAuth Protected Resource Metadata](https://www.rfc-editor.org/rfc/rfc9728)
- [MCP Authorization Specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/authorization)
- [Scalekit Token Vault](https://www.scalekit.com/features/token-vault)
- [Auth0 Token Vault](https://auth0.com/docs/get-started/authentication-and-authorization-flow/token-vault)

---

## 変更履歴

| 日付 | 変更内容 |
|------|---------|
| 2026-01-15 | 初版作成。分散構成（Clerk + Scalekit + Supabase）を検討後、Supabase一極集中に決定。Design Liability、Maintenance Liability、柔軟性、user_id連携の観点から総合判断 |
