# Sprint 006 レビュー

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-006 |
| 期間 | 2026-01-28 〜 2026-02-03 (7日間) |
| マイルストーン | M5: Stripe連携・品質基盤・仕様整備 |
| 状態 | **完了** |

---

## 計画 vs 実績サマリ

| 項目 | 計画 | 実績 | 達成度 |
|------|------|------|--------|
| Phase 1: Stripe 連携 | 6タスク | 6タスク完了 | ✅ 100% |
| Phase 2: 仕様書更新 | 8タスク | 2タスク完了 | ⚠️ 25% |
| Phase 3: テスト基盤 | 5タスク | 0タスク | ❌ 0% |
| Phase 4: CI/CD 整備 | 3タスク | 0タスク | ❌ 0% |
| 新規モジュール実装 | 計画外 | 10モジュール | ⭐ 計画外成果 |

---

## Phase 別詳細

### Phase 1: Stripe 決済連携 ✅ 完了

| ID | タスク | 計画 | 実績 | 差異 |
|----|--------|------|------|------|
| S6-001 | Stripe 商品・価格設定 | Product/Price作成 | $0 Price作成 | 無料クレジット付与用に変更 |
| S6-002 | サインアップ時の無料クレジット付与 | free_credits=1000 | pre_active→100クレジット | オンボーディング完了後に付与に変更 |
| S6-003 | Checkout Session API | 決済ページ | $0 Checkout | Phase 1 は無料フロー |
| S6-004 | Webhook ハンドラ | paid_credits反映 | free_credits反映 | 無料フロー用に調整 |
| S6-005 | Billing ページ | モック→実データ | 完了 | Signup Bonus カード追加 |
| S6-006 | クレジット消費検証 | E2E | 完了 | Claude Web で動作確認 |

**計画との差異:**
- 当初計画: 有料決済（¥500/月）→ 実績: 無料クレジット付与（Phase 1）
- 理由: 有料決済は Phase 2 以降に延期、まず無料体験フローを優先

### Phase 2: 仕様書更新 ⚠️ 部分完了

| ID | タスク | 計画 | 実績 | 状態 |
|----|--------|------|------|------|
| S6-010 | Rate Limit記述更新 | spc-dsn.md | 完了 | ✅ |
| S6-011 | JWT `aud` チェック整理 | spc-itf.md | 未着手 | 次Sprint |
| S6-012 | MCP拡張エラーコード整理 | spc-itf.md | 未着手 | 次Sprint |
| S6-013 | Console API設計更新 | spc-itf.md | 未着手 | 次Sprint |
| S6-014 | PSP Webhook仕様整理 | spc-itf.md | 未着手 | 次Sprint |
| S6-016 | Observability設計書 | dsn-observability.md | 未着手 | 次Sprint |
| S6-017 | MCP Tool Annotations | spc-itf.md | 完了 | ✅ |
| - | dtl-itr-XXX-YYY 全25件レビュー | 全件reviewed | 完了 | ✅ |

**未達成の理由:**
- モジュール実装（計画外）に注力したため、仕様書整備の時間が不足
- 優先度の判断: 機能拡充 > ドキュメント整備

### Phase 3: テスト基盤構築 ❌ 未着手

| ID | タスク | 計画 | 実績 |
|----|--------|------|------|
| S6-020 | E2E テスト設計書 | tst-e2e.md | 未着手 |
| S6-021 | Go Server ユニットテスト | *_test.go | 未着手 |
| S6-022 | Batch 権限チェックテスト | handler_test.go | 未着手 |
| S6-023 | Console ビルドテスト | CI workflow | 未着手 |

**未達成の理由:**
- モジュール実装が優先され、テスト基盤は次Sprint へ延期

### Phase 4: CI/CD 整備 ❌ 未着手

| ID | タスク | 計画 | 実績 |
|----|--------|------|------|
| S6-030 | tools.json 検証 CI | GitHub Actions | 未着手 |
| S6-031 | Go lint + test CI | GitHub Actions | 未着手 |
| S6-032 | Console lint + build CI | GitHub Actions | 未着手 |

**未達成の理由:**
- Phase 3 と同様、モジュール実装を優先

---

## 計画外成果: 10モジュール・133ツール追加 ⭐

Sprint計画には含まれていなかったが、大幅なモジュール拡張を実施。

### モジュール実装一覧

| 日 | モジュール | ツール数 | 認証方式 | 特記事項 |
|----|-----------|---------|----------|----------|
| DAY021 | Google Tasks | 9 | OAuth 2.0 | OAuth共有コールバック方式 |
| DAY021 | Microsoft To Do | 8 | OAuth 2.0 | mcpist-dev で実装 |
| DAY022 | Todoist | 8 | OAuth 2.0 | リフレッシュトークンなし |
| DAY022 | Trello | 17 | **OAuth 1.0a** | HMAC-SHA1署名、3-legged |
| DAY022 | GitHub | 20 | OAuth 2.0 / PAT | alternativeAuthパターン |
| DAY022 | Asana | 12 | OAuth 2.0 / PAT | 読み取り専用、FlexibleTime型導入 |
| DAY022 | Google Docs | 4 | OAuth 2.0 | Google OAuth統合 |
| DAY022 | Google Drive | 22 | OAuth 2.0 | Google OAuth統合 |
| DAY022 | Google Apps Script | 17 | OAuth 2.0 | script.scriptapp スコープ追加 |
| DAY022 | PostgreSQL | 7 | Connection String | SSRF対策、UUID変換 |
| - | Google Sheets | 28 | OAuth 2.0 | 既存モジュールの全ツールテスト |

### 数値サマリ

| 項目 | Sprint 005 終了時 | Sprint 006 終了時 | 増加 |
|------|------------------|------------------|------|
| モジュール数 | 8 | 18 | **+10** |
| ツール数 | 115 | 248 | **+133** |

---

## 日別進捗

### DAY017 (2026-01-28)
- dtl-itr-XXX-YYY 25件のレビュー開始（20件 reviewed）
- dtl-itr-IDP-SSM.md, dtl-itr-MOD-TVL.md レビュー完了

### DAY018 (2026-01-29)
- tool_id + 多言語対応（MCP Server, Console）
- dtl-itr レビュー完了（残り5件）
- E2E テスト完了

### DAY019 (2026-01-30)
- **Stripe Phase 1 完了**（S6-001〜006）
- オンボーディングフロー改善（tools step 削除、残高アラート追加）
- RPC設計・マイグレーション統合（36ファイル→9ファイル）
- RPC命名規則統一（`_my_`=Console, `_user_`=Router/API Server）

### DAY020 (2026-01-31)
- database.types.ts 再生成
- Console ビルド確認（RPC名変更後）
- Claude Web E2E テスト（Notion search + get_page_content）
- Liam ERD セットアップ

### DAY021 (2026-02-01)
- **Google Tasks モジュール実装**（9ツール）
- **Microsoft To Do モジュール実装**（8ツール）
- **prompts MCP 実装**（list/get、description/content分離）
- Console プロンプト管理UI（description追加、楽観的更新）
- Console テーマ改善（Liam ERD風ダークテーマ）
- /services ページ分離
- PKCE認証エラー修正

### DAY022 (2026-02-02〜03)
- **Todoist モジュール実装**（8ツール）
- **Trello モジュール実装**（17ツール、OAuth 1.0a）
- **GitHub OAuth 実装**（20ツール、alternativeAuth）
- **Asana モジュール実装**（12ツール、FlexibleTime型導入）
- **Google Docs モジュール実装**（4ツール）
- **Google Drive モジュール実装**（22ツール）
- **Google Sheets 全28ツールテスト完了**
- **Google Apps Script モジュール実装**（17ツール）
- **PostgreSQL モジュール実装**（7ツール、UUID変換対応）

---

## 技術的な学び・成果

### 1. OAuth 1.0a 対応
- Trello で OAuth 1.0a（HMAC-SHA1署名、3-legged フロー）を実装
- OAuth 2.0 との主な違い: 署名生成、状態管理、トークンシークレット

### 2. FlexibleTime 型の導入
- 問題: Console は ISO 文字列、Go は Unix タイムスタンプを期待
- 解決: `FlexibleTime` カスタム型で両形式をサポート
- 影響: asana, google_calendar, google_tasks, microsoft_todo, notion

### 3. pgx の UUID 型変換
- 問題: pgx は UUID を `[16]byte` で返す
- 解決: `convertValue` 関数で文字列形式 `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` に変換

### 4. Google OAuth 統合
- 複数の Google サービス（Calendar, Tasks, Drive, Docs, Sheets）を1回の認証で
- スコープを統合、"google" として1レコード保存
- ユーザー体験の改善（認証回数削減）

### 5. alternativeAuth パターン
- OAuth と API Key/PAT の両方をサポート
- 対応モジュール: Notion, GitHub, Asana
- UI で「または」で区切って両オプションを表示

### 6. RPC 命名規則の確立
- `_my_`: Console から呼び出される RPC（ユーザー自身のデータ）
- `_user_`: Router/API Server から呼び出される RPC（代理アクセス）

---

## 残課題（次Sprint へ引き継ぎ）

### 優先度: 高
| ID | タスク | 備考 |
|----|--------|------|
| BL-011 | JWT `aud` チェック要件整理 | 実装では明示チェックなし |
| BL-012 | MCP 拡張エラーコード整理 | JSON-RPC 標準コードのみに更新 |
| S6-020 | E2E テスト設計書作成 | tst-e2e.md |

### 優先度: 中
| ID | タスク | 備考 |
|----|--------|------|
| BL-013 | Console API 設計更新 | REST API → Supabase RPC 方式 |
| BL-014 | PSP Webhook 仕様整理 | Phase 1 実装に合わせて更新 |
| D19-005 | Observability 設計書作成 | dsn-observability.md |
| BL-090 | credentials JSON構造の乖離解消 | 仕様ではネスト、実装ではフラット |

### 優先度: 低
| ID | タスク | 備考 |
|----|--------|------|
| S6-030〜032 | CI/CD 整備 | GitHub Actions |
| S6-021〜023 | ユニットテスト | Go / Console |

→ 詳細は [sprint006-backlog.md](./sprint006-backlog.md) を参照

---

## 振り返り

### 良かった点
1. **大幅な機能拡充**: 10モジュール・133ツールの追加でサービス価値が向上
2. **OAuth 1.0a 対応**: Trello サポートでレガシーサービスにも対応可能に
3. **Stripe Phase 1 完了**: 課金基盤の第一歩が完成
4. **prompts MCP 実装**: MCP プリミティブ対応が進展

### 改善点
1. **仕様書整備の遅れ**: モジュール実装に注力しすぎて、ドキュメントが後回しに
2. **テスト基盤未着手**: E2E テスト設計が未完了、品質基盤が不十分
3. **計画 vs 実績の乖離**: 計画にないタスクが大量発生し、当初計画が形骸化

### 次Sprint への教訓
1. **モジュール実装を計画に含める**: 需要に応じて柔軟に対応する前提で計画
2. **仕様書更新を並行実施**: 実装と同時にドキュメントを更新する習慣化
3. **テスト基盤を優先**: 回帰テストなしでの開発継続はリスク

---

## 参考

- [sprint006.md](./sprint006.md) - Sprint 006 計画書・進捗ログ
- [sprint006-backlog.md](./sprint006-backlog.md) - 残課題一覧
- [day020-worklog.md](../workdir/day020-worklog.md)
- [day021-worklog.md](../workdir/day021-worklog.md)
- [day022-worklog.md](../workdir/day022-worklog.md)
