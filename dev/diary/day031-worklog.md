# DAY031 作業ログ

## 日付

2026-02-16

---

## 実施内容

### Console RPC 移行計画の策定

Console (Next.js) から呼ばれる全 RPC を棚卸し、PostgREST 移行に向けた設計を完了。

#### 分析

- Console から呼ばれる RPC 35 箇所を全数調査
- 必要性を分類: 必須 (16), RLS 代替可能 (13), 要検討 (4)
- 呼出コンテキストを精査: ブラウザ直接 / Server Action / Route Handler / Admin

#### 設計判断

| 判断 | 理由 |
|------|------|
| 全 DB アクセスを RPC に閉じ込め | スキーマ非公開を維持。Console/Server はテーブル名・カラム名を知らない |
| ブラウザ → PostgREST 直接通信を廃止 | Next.js Backend が中継。Go Server と同じパターンに統一 |
| `auth.uid()` を全廃、`p_user_id` パラメータに | PostgREST 単体で動作可能に (Supabase Auth 非依存) |
| `_my_` / `_user_` プレフィックスを除去 | `p_user_id` パラメータがあるので冗長。ただし検索対象が user の場合は残す |
| 4 RPC を統合、1 RPC を新設 | `get_my_role` + `get_my_settings` → `get_user_context` 拡張。`get_my_tool_settings` + `get_my_module_descriptions` → `get_module_config` 新設 |

#### 命名規則

- `_my_` 廃止 (auth.uid() がないので「自分の」は意味がない)
- `_user_` 廃止 (p_user_id パラメータで自明)
- 例外: `lookup_user_by_key_hash`, `get_user_context`, `get_user_by_stripe_customer` は検索対象が user なので残す
- admin 用は `_all_` で区別

#### 最終 RPC 数

| カテゴリ | 数 |
|---------|---|
| Server | 9 |
| Console | 20 |
| Stripe | 5 |
| **合計** | **34** |

### 成果物

| ファイル | 内容 |
|---------|------|
| `dev/workdir/day031-impl-console-rpc-migration.md` | 移行計画書 (Phase 0-3, 画面単位の移行手順) |
| `docs/graph/grh-rpc-design.canvas` | RPC 設計図を 9 RPC → 34 RPC に拡張 |

### Canvas 変更内容

- Summary ノード: 9 RPC → 34 RPC (Server/Console/Stripe の3セクション)
- Console Backend ノード追加 (Next.js Server Actions / Route Handlers)
- Console RPC グループ 8 個追加 (API Key / Credential / Prompt / Module Config / User / OAuth / Admin / Stripe)
- 共有 RPC へのエッジ追加 (get_user_context, upsert_credential, get_oauth_app_credentials)
- Server RPC 名を新名に更新 (get_prompts, get_credential, upsert_credential)
- テーブル 2 個追加 (oauth_consents, stripe_customers)

---

## コミット

なし (設計・計画のみ)
