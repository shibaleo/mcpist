# モジュールインターフェース仕様書（itf-mod）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Module Interface Specification |

---

## 概要

本ドキュメントは、MCPistのモジュールが実装すべきインターフェースを定義する。

---

## モジュールインターフェース

各モジュールは以下の機能を実装する。

| 機能 | 説明 |
|------|------|
| Name | モジュール名を返却 |
| Description | モジュールの説明を返却 |
| APIVersion | APIバージョンを返却 |
| Tools | 利用可能なツール一覧を返却 |
| Resources | 利用可能なリソース一覧を返却 |
| Prompts | 利用可能なプロンプト一覧を返却 |
| ExecuteTool | ツールを実行し結果を返却 |

---

## スキーマ定義

### モジュールスキーマ

| フィールド | 型 | 説明 |
|-----------|-----|------|
| module | string | モジュール名 |
| description | string | モジュールの説明 |
| api_version | string | APIバージョン |
| tools | array | ツールスキーマの配列 |
| resources | array | リソーススキーマの配列 |
| prompts | array | プロンプトスキーマの配列 |

### ツールスキーマ

| フィールド | 型 | 説明 |
|-----------|-----|------|
| name | string | ツール名 |
| description | string | ツールの説明 |
| inputSchema | object | 入力パラメータのJSONスキーマ |

### リソーススキーマ

| フィールド | 型 | 説明 |
|-----------|-----|------|
| uri | string | リソースURI |
| name | string | リソース名 |
| description | string | リソースの説明 |
| mimeType | string | MIMEタイプ（オプション） |

### プロンプトスキーマ

| フィールド | 型 | 説明 |
|-----------|-----|------|
| name | string | プロンプト名 |
| description | string | プロンプトの説明 |
| arguments | array | 引数定義（オプション） |

---

## 実行結果

### ツール実行結果

| フィールド | 型 | 説明 |
|-----------|-----|------|
| content | array | コンテンツブロックの配列 |
| isError | boolean | エラーフラグ |

### コンテンツブロック

| フィールド | 型 | 説明 |
|-----------|-----|------|
| type | string | コンテンツタイプ（"text"） |
| text | string | テキスト内容 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-mod.md](../interaction/itr-mod.md) | Modules インタラクション仕様 |
| [dsn-err.md](../../003_design/dsn-err.md) | エラーハンドリング設計書 |
