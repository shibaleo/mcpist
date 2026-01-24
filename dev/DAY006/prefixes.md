# ID体系定義

## ID構造

```
[カテゴリ][-サブカテゴリ]?-[production].[sprint].[patch]
```

### バージョニング

| 要素 | 説明 |
|------|------|
| production | プロダクトバージョン |
| sprint | スプリント番号 |
| patch | パッチ番号 |

---

## 略称一覧

| id | name | prefix | 説明 |
|----|------|--------|------|
| 1 | products | PRD | プロダクト |
| 2 | personas | PSN | ペルソナ |
| 3 | user | USR | ユーザー|
| | story | STR | ストーリー |

| 4 | acceptance | ACP | 受け入れ |
| 4 | condition | CDN | 条件 |
| 5 | requirements | REQ | 要件 |
| 6 | specifications | SPC | 仕様 |
| 7 | systems | SYS | システム |
| 8 | infrastructures | INF | インフラ |
| 9 | designs | DSN | 設計 |
| 10 | tests | TST | テスト |
| 11 | operations | OPS | 運用 |
| 12 | functional | FNC | 機能 |
| 13 | non_functional | NFC | 非機能 |
| 14 | out_of_scope | OFS | スコープ外 |
| 15 | details | DTL | 詳細 |
| 16 | cores | COR | コア |
| 17 | manuals | MNL | マニュアル |
| 18 | interfaces | ITF | インターフェース |

---

## ID例

| ID | 意味 |
|----|------|
| REQ-1.1.1 | 要件 |
| REQ-FNC-1.1.2 | 機能要件 |
| REQ-NFC-1.1.1 | 非機能要件 |
| SPC-REQ-1.2.4 | 要求仕様 |
| SPC-DSN-1.2.3 | 設計仕様 |
| SPC-SYS-1.1.1 | システム仕様 |
| DSN-DTL-2.5.6 | 詳細設計 |
| TST-COR-1.1.1 | コアテスト |
| PSN-1.0.0 | ペルソナ |
| UST-1.1.0 | ユーザーストーリー |
| ACC-1.1.1 | 受け入れ条件 |
| OPS-MNL-1.0.0 | 運用マニュアル |
