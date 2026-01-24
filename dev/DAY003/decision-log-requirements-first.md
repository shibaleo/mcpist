---
title: 意思決定ログ Requirements First アプローチ
aliases:
  - decision-log-requirements-first
  - requirements-first-decision
tags:
  - MCPist
  - decision-log
  - process
document-type:
  - decision-log
document-class: decision-log
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# 意思決定ログ: Requirements Doc を先に作成する

## 日付
2026-01-12

## 背景・悩んでいたこと

現在 `spec-MCPist.md` という Specification（仕様書）が存在する。
次のステップとして以下の3つの選択肢があり、どれに進むべきか迷っていた。

| 選択肢 | 内容 |
|--------|------|
| Design Doc | 実装詳細（パッケージ構成、型定義、エラーハンドリング等）を書く |
| Requirements Doc | ユーザーストーリー・受け入れ基準・非機能要件を書く |
| Detailed Spec | 現在のSpecをより詳細化する |

### 現状の整理

- **プロジェクト状況**: コードはまだないが、プロトタイプのコードを流用可能
- **課題**: 要件の抜け漏れが不安、Specの妥当性も検証したい

### 現在のSpecの性質

`spec-MCPist.md` は Specification であり:

- ✅ システム構成図とコンポーネントの責務定義
- ✅ メタツールのインターフェース仕様（入力/出力/JSON例）
- ✅ エンドポイント一覧とプロトコル仕様
- ✅ 設計原則・制約の明示

しかし以下は含まれていない:

- ❌ ユーザーストーリー（Requirements）
- ❌ 受け入れ基準（Requirements）
- ❌ Goパッケージ構成・型定義（Design Doc）
- ❌ テスト戦略（Design Doc）

---

## 決定

**Requirements Doc を先に作成する**

---

## 理由

| 観点 | 説明 |
|------|------|
| **抜け漏れ検証** | Specから逆算してRequirementsを書くと「なぜこの機能が必要か」が明確になり、穴が見つかる |
| **プロトタイプ流用の判断基準** | 要件が明確なら「この既存コードは使える/使えない」が判断しやすい |
| **Specの妥当性検証** | Requirements vs Spec を突き合わせると、過剰な機能や不足が見える |

### Design Doc を先に書くリスク

- 要件が曖昧なまま詳細設計に入ると、後で大きな手戻りになる
- 「プロトタイプのコードをどこまで使うか」の判断軸がないまま設計することになる

---

## 進め方

```
[現状のSpec] → [Requirements抽出] → [Gap分析] → [Spec改訂 or Design Doc]
```

### Requirements Doc に書くべきこと

1. **ユーザーストーリー**: 「〜として、〜したい、なぜなら〜」
2. **受け入れ基準**: 各機能の完了条件
3. **非機能要件**: パフォーマンス、セキュリティ、運用
4. **スコープ外**: 明示的にやらないこと
5. **前提条件・制約**: 外部依存、技術制約

---

## 次のアクション

- [x] 想定されるユーザのペルソナ分析
- [x] 現在のSpecからユーザーストーリーを抽出する
- [x] 各機能の受け入れ基準を定義する → `DAY3/requirements/req-list.md`
- [x] 非機能要件を洗い出す → `DAY3/requirements/req-nfr.md`
- [x] スコープ外を明示する → `DAY3/requirements/req-ofs.md`
- [x] Requirements vs Spec のGap分析を行う → `gap-analysis.md`
