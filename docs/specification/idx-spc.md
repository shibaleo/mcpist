# MCPist 仕様書インデックス（idx-spc）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | Specification Index |

---

## 概要

MCPist 仕様書の一覧。

---

## 仕様書一覧

| ドキュメント                     | 名称          | 内容                  |
| -------------------------- | ----------- | ------------------- |
| [spc-sys.md](./spc-sys.md) | システム仕様書     | コンポーネント定義、アーキテクチャ   |
| [spc-itr.md](./spc-itr.md) | インタラクション仕様書 | コンポーネント間のやり取り       |
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

### インタラクション仕様（interaction/itr-xxx.md）

各コンポーネントの連携詳細を記載。

| #   | コンポーネント                  | 略称  | ドキュメント                             | 備考           |     |
| --- | ------------------------ | --- | ---------------------------------- | ------------ | --- |
| 1   | MCP Client (OAuth2.0)    | CLO | [itr-clo.md](./interaction/itr-clo.md) | 実装範囲外        | ✅   |
| 2   | MCP Client (API KEY)     | CLK | [itr-clk.md](./interaction/itr-clk.md) | 実装範囲外        | ✅   |
| 3   | API Gateway              | GWY | [itr-gwy.md](./interaction/itr-gwy.md) |              | ✅   |
| 4   | Auth Server              | AUS | [itr-aus.md](./interaction/itr-aus.md) |              | ✅   |
| 5   | Session Manager          | SSM | [itr-ssm.md](./interaction/itr-ssm.md) |              | ✅   |
| 6   | Data Store               | DST | [itr-dst.md](./interaction/itr-dst.md) |              | ✅   |
| 7   | Token Vault              | TVL | [itr-tvl.md](./interaction/itr-tvl.md) |              | ✅   |
| 8   | MCP Server               | SRV | [itr-srv.md](./interaction/itr-srv.md) | 外部向け抽象化      | ✅   |
| 9   | Auth Middleware          | AMW | [itr-amw.md](./interaction/itr-amw.md) | MCP Server内部 | ✅   |
| 10  | MCP Handler              | HDL | [itr-hdl.md](./interaction/itr-hdl.md) | MCP Server内部 | ✅   |
| 11  | Modules                  | MOD | [itr-mod.md](./interaction/itr-mod.md) | MCP Server内部 | ✅   |
| 12  | User Console             | CON | [itr-con.md](./interaction/itr-con.md) |              | ✅   |
| 13  | Identity Provider        | IDP | [itr-idp.md](./interaction/itr-idp.md) | 実装範囲外        | ✅   |
| 14  | External Auth Server     | EAS | [itr-eas.md](./interaction/itr-eas.md) | 実装範囲外        | ✅   |
| 15  | External Service API     | EXT | [itr-ext.md](./interaction/itr-ext.md) | 実装範囲外        | ✅   |
| 16  | Payment Service Provider | PSP | [itr-psp.md](./interaction/itr-psp.md) | 実装範囲外        | ✅   |

### インターフェース仕様（dtl-spc/itf-xxx.md）

API仕様、データ構造のオントロジー定義を記載。

| ドキュメント | 内容 |
|-------------|------|
| [itf-tvl.md](./dtl-spc/itf-tvl.md) | Token Vault API仕様 |
| [itf-mod.md](./dtl-spc/itf-mod.md) | モジュールインターフェース仕様 |

### その他

| ドキュメント | 内容 |
|-------------|------|
| [idx-ept.md](./dtl-spc/idx-ept.md) | エンドポイント一覧 |
| [dtl-spc-credit-model.md](./dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dsn-adt.md](../design/dsn-adt.md) | 監査・請求・分析設計書 |
| [dsn-err.md](../design/dsn-err.md) | エラーハンドリング設計書 |
