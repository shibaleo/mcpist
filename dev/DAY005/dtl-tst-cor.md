---
title: MCPist テスト仕様サブコア定義
aliases:
  - dtl-tst-cor
  - test-sub-core
tags:
  - MCPist
  - architecture
  - sub-core
  - DTL
document-type: detail
document-class: DTL
created: 2026-01-14T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist テスト仕様サブコア定義

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 (DAY5) |
| Note | spec-tst.mdからサブコア要件を抽出 |

---

## 概要

本ドキュメントは、spec-tst.md（テスト仕様書）の中で、コア機能（COR-xxx）を前提とした場合に複数の独立した根拠を持つ要件を「サブコア」として定義する。

**評価基準:**
- コア機能を前提とした場合に、2つ以上の独立した根拠を持つ
- そのコア機能が変わらない限り、変更されない

---

## サブコア要件

### TST-COR-001: TOON/JSONL単体テスト

**前提コア:**
- COR-008 (TOON形式)
- COR-001 (メタツール方式)
- COR-007 (Go採用)

**独立した根拠（3つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | TOON形式 → パーサー・フォーマッターの高カバレッジ（90%以上）が必要 | COR-008 |
| 2 | メタツール方式 → 変数参照解決 `${id.items[N].field}` のテストが必須 | COR-001 |
| 3 | Go採用 → go testによる単体テスト、testcontainersによる結合テスト | COR-007 |

**定義:**
```
単体テスト対象（高優先度）:
├─ internal/toon: TOON パーサー・フォーマッター（90%以上）
├─ internal/jsonl: JSONL パーサー、依存関係解決（90%以上）
├─ internal/variable: 変数参照解決（90%以上）
└─ internal/auth: JWT検証、JWKS取得（85%以上）

テストケース例:
├─ 正常系: 3件のレコードパース
├─ 正常系: 0件（空結果）
├─ 異常系: 件数不一致
├─ 異常系: フィールド数不一致
└─ 異常系: エスケープ処理

変数参照テスト:
├─ 正常系: items[0].title
├─ 異常系: 存在しないID
└─ 異常系: インデックス範囲外
```

**spec-tst.mdでの位置:** 2. 単体テスト

---

### TST-COR-002: Token Broker結合テスト

**前提コア:**
- COR-003 (サーバー側認証設計)
- COR-005 (RLS非依存認可)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | サーバー側認証 → MCPサーバー↔Token Broker間の通信テスト | COR-003 |
| 2 | RLS非依存 → Edge Function内のuser_idフィルタリングテスト | COR-005 |

**定義:**
```
Token Broker結合テスト:
├─ 有効トークン取得: user_id, module → access_token返却
├─ 期限切れトークンのリフレッシュ: user_id, module(expired) → 新access_token
├─ リフレッシュトークン失効: user_id, module(revoked) → 401 + 再認証要求
└─ 未連携モジュール: user_id, unknown_module → 404 + 連携案内

モック方法:
├─ Supabase Auth: テスト用JWT静的生成
├─ Supabase DB: testcontainers (PostgreSQL)
├─ Supabase Vault: モックEdge Function
└─ 外部API: httptest.Server
```

**spec-tst.mdでの位置:** 3. 結合テスト（Token Broker）

---

### TST-COR-003: Tool Sieve結合テスト

**前提コア:**
- COR-001 (メタツール方式)
- COR-003 (サーバー側認証設計)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | メタツール方式 → get_module_schema時の権限フィルタリングテスト | COR-001 |
| 2 | サーバー側認証 → ロール権限に基づくアクセス制御テスト | COR-003 |

**定義:**
```
Tool Sieve結合テスト:
├─ 許可されたツール: developer × github_list_issues → 許可
├─ 禁止されたツール: viewer × github_create_issue → 拒否
├─ モジュール無効化: admin × notion_*(broken) → 拒否+エラーメッセージ
└─ ロール未割当: (none) × any → 拒否

整合性検証:
├─ /api/profile/tools の結果
├─ get_module_schema の結果
└─ 上記2つは完全一致すること
```

**spec-tst.mdでの位置:** 3. 結合テスト（Tool Sieve）

---

### TST-COR-004: JSONL並列実行テスト

**前提コア:**
- COR-001 (メタツール方式)
- COR-007 (Go採用)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | メタツール方式 → batch処理の依存関係解決テスト | COR-001 |
| 2 | Go採用 → goroutineによる並列実行、sync.Mapでの結果共有テスト | COR-007 |

**定義:**
```
JSONL並列実行テスト:
├─ 独立タスク3件: afterなし×3 → 3件並列実行
├─ 依存チェーン: A→B→C → 順次実行
├─ 分岐依存: A→B, A→C → A完了後B,C並列
├─ 循環依存検出: A→B→A → パース時エラー
└─ 依存先失敗: A(fail)→B → Bスキップ、errorsに記録

実行エンジン検証:
├─ 並列実行の同時性
├─ 変数解決のタイミング
└─ エラー伝播の正確性
```

**spec-tst.mdでの位置:** 3.3 テストケース（JSONL並列実行）

---

### TST-COR-005: 認証セキュリティテスト

**前提コア:**
- COR-003 (サーバー側認証設計)
- COR-005 (RLS非依存認可)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | サーバー側認証 → JWT検証、権限昇格防止テスト | COR-003 |
| 2 | RLS非依存 → アプリケーション層でのIDOR防止テスト | COR-005 |

**定義:**
```
認証・認可テスト:
├─ JWT偽造: 不正署名のJWT → 401 Unauthorized
├─ JWT期限切れ: 期限切れJWT → 401 + リフレッシュ案内
├─ 権限昇格: userがadmin APIを呼び出し → 403 Forbidden
├─ IDOR: 他ユーザーのリソースアクセス → 403 or 404
└─ RLS回避: SQLインジェクションでRLS回避試行 → クエリ失敗

権限境界テスト:
├─ 自分のプロファイル編集: user×自分 → 許可
├─ 他人のプロファイル編集: user×他user → 拒否
├─ ロール作成: user → 拒否
├─ ロール作成: admin → 許可
└─ 自分のシステムロール変更: admin×自分 → 拒否（最後のadmin保護）
```

**spec-tst.mdでの位置:** 6. セキュリティテスト

---

### TST-COR-006: 管理UI E2Eテスト

**前提コア:**
- COR-006 (Next.js採用)
- COR-003 (サーバー側認証設計)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | Next.js採用 → Playwright E2Eテスト、SPA動作検証 | COR-006 |
| 2 | サーバー側認証 → admin/user権限分離のE2E検証 | COR-003 |

**定義:**
```
管理UI E2Eテスト:
├─ ログイン成功: メール+パスワード → ダッシュボード表示
├─ ログイン失敗: 許可リスト外 → エラーメッセージ表示
├─ ロール作成: /admin/roles → 新規作成 → 保存 → 一覧表示
├─ ユーザーロール割当: /admin/users → 選択 → 割当 → 更新反映
└─ OAuth連携: /oauth/connect → 認可 → コールバック → 連携済み表示

権限制限E2E:
├─ userがadminページにアクセス → リダイレクト or エラー
└─ adminが全ページにアクセス → 正常表示

テストアカウント:
├─ test-admin@mcpist.app: admin機能テスト
├─ test-user@mcpist.app: 一般機能テスト
└─ test-viewer@mcpist.app: 権限制限テスト
```

**spec-tst.mdでの位置:** 5. E2Eテスト

---

## 非サブコア要件（単一根拠）

以下は重要な要件だが、コア機能からの導出が単一であるためサブコアとしない。

| 要件 | 根拠数 | 理由 |
|------|--------|------|
| 負荷テスト（k6） | 1 | 運用品質からの単一導出 |
| 障害テスト（Chaos Engineering） | 1 | 運用品質からの単一導出 |
| OAuthフローテスト | 1 | 外部API連携からの単一導出 |
| ロールバックテスト | 1 | 運用安全性からの単一導出 |
| DBマイグレーションテスト | 1 | 運用安全性からの単一導出 |
| 外部API互換性テスト | 1 | 外部API連携からの単一導出 |
| 定期ヘルスチェック | 1 | 運用品質からの単一導出 |

---

## サブコアマトリックス

| ID | サブコア要件 | 前提コア | 根拠数 |
|----|--------------|----------|--------|
| TST-COR-001 | TOON/JSONL単体テスト | COR-008, COR-001, COR-007 | 3 |
| TST-COR-002 | Token Broker結合テスト | COR-003, COR-005 | 2 |
| TST-COR-003 | Tool Sieve結合テスト | COR-001, COR-003 | 2 |
| TST-COR-004 | JSONL並列実行テスト | COR-001, COR-007 | 2 |
| TST-COR-005 | 認証セキュリティテスト | COR-003, COR-005 | 2 |
| TST-COR-006 | 管理UI E2Eテスト | COR-006, COR-003 | 2 |

---

## 関連ドキュメント

- [dtl-core.md](dtl-core.md) - コア機能定義
- [spec-tst.md](../DAY4/spec-tst.md) - テスト仕様書
