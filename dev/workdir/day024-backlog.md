# DAY024 バックログ（やり残し）

## 日付

2026-02-05

---

## 未完了タスク

### 1. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-020 | itr-TVL.md 更新 | ❌ 未着手 | Credentials リファクタリング・ランニングバランス反映 |
| S7-021 | dtl-itr-CON-TVL.md 更新 | 一部着手 | git diff に変更あり（未コミット） |
| S7-022 | dtl-itr-MOD-TVL.md 更新 | 一部着手 | git diff に変更あり（未コミット） |
| S7-023〜026 | その他仕様書更新 | ❌ 未着手 | |

### 2. クレジットモデル仕様書更新

| タスク | 状態 | 備考 |
|--------|------|------|
| dtl-spc-credit-model.md をランニングバランス方式に更新 | ❌ 未着手 | credits テーブル廃止・running balance 移行を反映 |

### 3. 言語設定が MCP ツールスキーマに反映されない問題

| タスク | 状態 | 備考 |
|--------|------|------|
| 原因調査 | 🔍 調査済み・未解決 | DB は正しく `en-US` を返しているが、tools/list のメタツール description が日本語で返る |

**調査結果:**
- ユーザー (`1391d544`, shiba.dog.leo.private) の `settings` に `{"language": "en-US"}` が保存されている
- `get_user_context` RPC は `language: "en-US"` を正しく返す
- Go Server の `DynamicMetaTools()` は `lang == "ja-JP"` の場合のみ日本語を返す
- DB→Server の流れに問題は見つからなかった
- **可能性**: MCP クライアント接続時（initialize）のキャッシュ問題、またはサーバー再起動前の古い言語設定が残っていた可能性

**次のアクション:**
- MCPクライアントを再接続して `tools/list` の言語が正しいか再確認
- サーバーログで実際に渡されている `language` 値を確認

### 4. Grafana ダッシュボード改善

| タスク | 状態 | 備考 |
|--------|------|------|
| アラート設定 | ❌ 未着手 | エラーレート閾値のアラートルール作成 |
| パネル改善 | ❌ 未着手 | 必要に応じて追加パネル |

### 5. ステージ済み変更のコミット

| タスク | 状態 | 備考 |
|--------|------|------|
| Console UI 改善のコミット | ❌ 未コミット | 11ファイルがステージ済み |

**提案コミットメッセージ:**
```
feat(console): add ivory light theme, refine dark theme, and clean up UI

- Add ivory/cream light theme with proper visual hierarchy (background < card < form)
- Refine dark theme to Obsidian-like slightly brighter tones
- Centralize MCP server URL access via getMcpServerUrl() with error on missing env
- Remove OAuth consents section from connections page
- Fix dashboard card highlight flash during loading
- Shorten sidebar labels and make resize handle thinner
- Allow textarea resize for tool custom descriptions
```

### 6. 未コミットの仕様書変更

| ファイル | 状態 |
|----------|------|
| `docs/002_specification/interaction/dtl-itr-CON-TVL.md` | 変更あり・未ステージ |
| `docs/002_specification/interaction/dtl-itr-MOD-TVL.md` | 変更あり・未ステージ |
| `docs/002_specification/interaction/itr-TVL.md` | 変更あり・未ステージ |
| `docs/graph/grh-deployment.canvas` | 変更あり・未ステージ |
| `dev/workdir/day024-plan.md` | 変更あり・未ステージ |
| `dev/workdir/day024-worklog.md` | 変更あり・未ステージ |

---

## 優先度

| 優先度 | タスク |
|--------|--------|
| 高 | ステージ済み変更のコミット |
| 高 | 言語設定問題の再確認（再接続で解消するか） |
| 中 | 仕様書更新 (S7-020〜026) |
| 中 | クレジットモデル仕様書更新 |
| 低 | Grafana ダッシュボード改善 |
