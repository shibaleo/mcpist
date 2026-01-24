# ADR: Tool Sieveの命名再検討

## ステータス

**承認済み** - 2026-01-16

## コンテキスト

MCPサーバーの認可機能として「Tool Sieve」を設計した。当初の設計では、Sieveが以下の役割を担う予定だった：

1. ユーザーの許可ツール一覧をDBから取得・キャッシュ
2. ツール呼び出し時の許可チェック（ゲート）
3. スキーマ取得時のフィルタリング

しかし、責務分離の観点から設計を見直した結果、Sieveは「純粋なDBキャッシュ」に徹することになった。

これにより、「Sieve（ふるい）」という名前と実際の役割に乖離が生じた。

## 決定

### 命名の変更

| 旧名称 | 新名称 | 役割 |
|--------|--------|------|
| `Sieve` | `PermissionCache` | DBキャッシュ（許可ツール一覧を保持） |
| `SieveMiddleware` | - | 廃止（PermissionGate関数に置き換え） |
| - | `PermissionGate` | 関数：ツール呼び出しの許可/拒否判定 |
| - | `PermissionFilter` | 関数：スキーマのフィルタリング |

### 設計方針

- **PermissionCache**: 状態を持つためstruct
- **PermissionGate**: AuthMiddleware内およびHandler内で呼び出す関数
- **PermissionFilter**: Handler内で呼び出す関数

### パッケージ構成

```
internal/
  permission/
    permission.go    ← Cache, Gate, Filter（最初は1ファイル、必要に応じて分割）
  auth/
    middleware.go    ← permission.Gate()を呼び出し
  mcp/
    handler.go       ← permission.Filter(), permission.Gate()を呼び出し
```

**設計判断**: `utils`ではなく`permission`パッケージとして独立させる理由
- `permission.Gate(...)` の方が呼び出し時に意図が明確
- 将来的にロール判定など追加しやすい
- utilsは「どこにも属さない汎用関数」に限定した方がクリーン

## 検討した代替案

### 命名の代替案

| 案 | Cache名 | 評価 |
|----|---------|------|
| A | `Sieve` | × 役割と名前が乖離（ふるいではなくキャッシュ） |
| B | `AllowListCache` | △ 悪くないが冗長 |
| C | `GrantCache` | △ 「付与」のニュアンスは課金向き |
| D | `EntitlementCache` | △ エンタープライズ感が強い |
| E | `PermissionCache` | ○ 素直で分かりやすい |

### 構造の代替案

| 案 | 構造 | 評価 |
|----|------|------|
| A | 全て1つのstruct（Sieve）に集約 | × 責務が混在 |
| B | Cache/Gate/Filter全てstruct | △ 過剰な抽象化 |
| C | Cacheはstruct、Gate/Filterは関数 | ○ シンプルで適切 |

### パッケージ配置の代替案

| 案 | 配置 | 評価 |
|----|------|------|
| A | `internal/utils/permission.go` | △ utilsが肥大化しがち、責務が曖昧 |
| B | `internal/permission/permission.go` | ○ 独立パッケージで責務明確、拡張しやすい |

### Middleware分離の代替案

| 案 | 構造 | 評価 |
|----|------|------|
| A | AuthMiddleware + PermissionMiddleware（分離） | △ 不要な分離、複雑化 |
| B | AuthMiddleware内でPermissionGate呼び出し（統合） | ○ シンプル、依存関係が明確 |

### スーパーユーザー対応の代替案

| 案 | 方式 | 評価 |
|----|------|------|
| A | PermissionGateをスキップ | △ バイパス経路が存在、監査が複雑 |
| B | PermissionCacheが全ツールを返す | ○ 同じ経路、監査が容易 |

## 結果

### 採用した命名体系

```
Permission* プレフィックスで統一

PermissionCache  - 状態を持つキャッシュ（struct）
PermissionGate   - 許可/拒否の判定（関数）
PermissionFilter - フィルタリング（関数）
```

### メリット

1. **接頭辞の統一**: `Permission*` で関連性が明確
2. **役割が名前に表れている**:
   - Cache → データを保持
   - Gate → 通す/止める
   - Filter → 絞り込む
3. **可読性**: 新規メンバーでも理解しやすい
4. **適切な抽象度**: 状態を持つものだけstruct、ロジックは関数
5. **シンプルな構造**: Middleware層を統合し複雑度を低減
6. **安全なスーパーユーザー対応**: バイパスなしで同じ経路を通る

## 関連ドキュメント

- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システムの詳細設計
- [dsn-tool-sieve.md](./dsn-tool-sieve.md) - 認証・認可アーキテクチャ全体
- [ui-prototype-review.md](../DAY6/ui-prototype-review.md) - UIプロトタイプと権限の階層構造
