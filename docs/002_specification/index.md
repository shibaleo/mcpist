# MCPist 仕様書インデックス（idx-spc）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Specification Index |

---

## 概要

MCPist DAY8 仕様書の一覧。

---

## 仕様書一覧

| ドキュメント                     | 名称          | 内容                  |
| -------------------------- | ----------- | ------------------- |
| [spc-sys.md](./spc-sys.md) | システム仕様書     | コンポーネント定義、アーキテクチャ   |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 | コンポーネント間のやり取り       |
| [spc-itf.md](./spc-itf.md) | インターフェース仕様書 | プロトコル、エンドポイント、データ形式 |
| [spc-tbl.md](./spc-tbl.md) | テーブル仕様書     | DBスキーマ、テーブル配置       |
| [spc-dsn.md](./spc-dsn.md) | 設計仕様書       | 設計原則、パターン           |
| [spc-inf.md](spc-inf.md)   | インフラ仕様書     | ホスティング、デプロイ構成       |
| [spc-sec.md](./spc-sec.md) | セキュリティ仕様書   | セキュリティ要件、対策         |
| [spc-tst.md](./spc-tst.md) | テスト仕様書      | テスト戦略、テストケース        |
| [spc-ops.md](./spc-ops.md) | 運用仕様書       | 運用原則、監視、障害対応        |
| [spc-dev.md](./spc-dev.md) | 開発計画書       | フェーズ計画、マイルストーン      |

---

## ドキュメント構造

```
spc-sys（システム）
    │
    ├── spc-itr（インタラクション）── 誰が誰と話すか
    │
    ├── spc-itf（インターフェース）── どう話すか
    │
    ├── spc-tbl（テーブル）── データをどう保存するか
    │
    └── spc-dsn（設計）── どう作るか
            │
            ├── spc-infra（インフラ）── どこで動かすか
            │
            ├── spc-sec（セキュリティ）── どう守るか
            │
            ├── spc-tst（テスト）── どう検証するか
            │
            ├── spc-ops（運用）── どう運用するか
            │
            └── spc-dev（開発計画）── どう進めるか
```

---

## 詳細仕様（dtl-spc/）

| ドキュメント | 内容 |
|-------------|------|
| [idx-ept.md](idx-ept.md) | エンドポイント一覧 |
| [dtl-spc-hdl.md](dtl-spc-hdl.md) | MCP Handler詳細仕様 |
| [dtl-spc-credit-model.md](dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
| [itf-tvl.md](itf-tvl.md) | Token Vault API仕様 |
| [itf-mod.md](itf-mod.md) | モジュールインターフェース仕様 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dsn-adt.md](dsn-adt.md) | 監査・請求・分析設計書 |
| [dsn-err.md](dsn-err.md) | エラーハンドリング設計書 |
