---
title: ADR-005 RLSに依存しない認可設計
aliases:
  - ADR-005
  - no-rls-dependency
tags:
  - MCPist
  - ADR
  - security
  - authorization
document-type:
  - ADR
document-class: ADR
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# ADR-005: RLSに依存しない認可設計

## ステータス

採用

## コンテキスト

Token Brokerは外部サービスのOAuthトークンを管理するコンポーネントで、以下の要件がある:

1. MCPサーバーからのリクエストに対して、適切なユーザーのトークンのみを返却
2. 不正なアクセスを防ぐ認可制御が必要
3. 将来的にローカルアプリとしてバンドルする可能性がある

Supabase上で実装する場合、Row Level Security (RLS) という強力なDB層のアクセス制御機能が利用可能だが、採用すべきか検討が必要。

## 検討した選択肢

### 選択肢1: Supabase RLS（DB層）でアクセス制御

Supabase PostgreSQLのRLS機能を使い、Vaultテーブルに対するポリシーを設定。

**メリット:**
- PostgreSQL標準機能
- 宣言的なセキュリティポリシー
- SQL層での強制、バイパス困難

**デメリット:**
- **Supabase PostgreSQL固有機能への依存** → ベンダーロックイン
- 将来のローカルアプリ化（SQLite使用時）でRLS未対応
- Edge Functionからのアクセス時、RLSポリシー評価のオーバーヘッド
- service_roleキー使用時にRLSバイパスが必要（設計の矛盾）

### 選択肢2: Edge Function内でアプリケーション層認可（採用）

Token Broker（Edge Function）内で、`user_id`によるフィルタリングを実装。

**メリット:**
- **DB実装に非依存** → PostgreSQL/SQLiteどちらでも動作
- ローカルアプリ化時にロジック再利用可能
- シンプルなクエリ: `SELECT * FROM tokens WHERE user_id = ? AND service = ?`
- パフォーマンス: RLSポリシー評価不要

**デメリット:**
- アプリケーションコードでの認可制御（実装ミスのリスク）
- Edge Function URLの秘匿が必要

## 決定

**選択肢2（Edge Function内での認可）を採用**

**責務の明確化:**
- **Token Brokerが認可の責務を持つ** - `user_id`によるフィルタリングは必須実装
- **RLSは保険として使用** - クラウドホスト版でのみ、多層防御の一環として追加

Token Brokerは以下のように実装:

```typescript
// Edge Function (Supabase Deno)
export async function getToken(userId: string, service: string) {
  // アプリケーション層でのフィルタリング（認可の責務はここ）
  const { data } = await supabase
    .from('oauth_tokens')
    .select('*')
    .eq('user_id', userId)  // ← 必須：Token Brokerの責務
    .eq('service', service)
    .single();

  return data;
}
```

**クラウドホスト版のマイグレーション（多層防御）:**

```sql
-- RLSポリシー（保険としての多層防御）
-- 注意: これは保険であり、認可の主責務はToken Brokerにある
ALTER TABLE oauth_tokens ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can only access their own tokens"
ON oauth_tokens
FOR ALL
USING (user_id = auth.uid());
```

**ローカルアプリ化時（SQLite使用）:**

```typescript
// ローカルアプリ版 Token Broker
export async function getToken(userId: string, service: string) {
  // 同じ認可ロジックでSQLite対応（RLSなし）
  const token = db.prepare(
    'SELECT * FROM oauth_tokens WHERE user_id = ? AND service = ?'
  ).get(userId, service);

  return token;
}
```

## 根拠

### 1. ポータビリティ優先

MCPistはセルフホスト前提のプロジェクト。将来的に以下の展開が想定される:

- Supabase版（現行）
- ローカルアプリ版（Electron/Tauri + SQLite）
- 他のホスティング環境への移植

RLSに依存すると、PostgreSQL以外への移植が困難になる。

### 2. SQLiteはRLS未対応

ローカルアプリ化の際、軽量なSQLiteを使用する予定だが、SQLiteにはRLS機能がない。アプリケーション層での認可であれば、DB層を変更しても同じロジックが再利用できる。

### 3. 信頼境界の設計

MCPサーバー（Koyeb）で既にJWT検証済み → `user_id`確定済み。Token Brokerは同一システム内の信頼されたコンポーネントとして動作。

### 4. セキュリティ担保

- Edge Function URLは環境変数で管理（公開されない）
- MCPサーバー → Token Broker間は内部通信
- `user_id`はMCPサーバーのJWT検証で確定済み（改ざん不可）

## 影響

### 実装上の注意

- **必須:** Token Broker内で`user_id`によるフィルタを漏れなく実装（これが認可の主責務）
- **推奨:** クラウドホスト版ではRLSポリシーを追加（多層防御の保険）
- Edge Function URLは秘密情報として扱う
- MCPサーバー側のJWT検証が信頼の起点

### 多層防御の考え方

```
【防御層1】MCPサーバー: JWT検証 → user_id確定
【防御層2】Token Broker: user_idフィルタ（主責務）← ここが認可の本体
【防御層3】RLS（クラウドのみ）: DB層での保険 ← 万が一の保険
```

防御層2が失敗してもRLSが防ぐが、**RLSに依存してはいけない**（ローカル版で動かないため）。

### 移植性

- PostgreSQL → SQLiteへの移行が容易
- 認可ロジックはToken Brokerに集約（DB非依存）
- RLSはクラウド版の付加機能として扱う
- 同一の認可ロジックを複数DB実装で共有可能

## 参照

- [spec-sys.md § 2.3 Token Broker](../spec-sys.md)
- [spec-dsn.md § 3. データモデル](../spec-dsn.md)
