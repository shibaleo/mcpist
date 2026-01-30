# ツールID化 + モジュール説明（多言語）対応 実装計画

## 目的

- ユーザー設定の整合性のため、tool_name を安定ID（tool_id）に移行する
- モジュール説明をユーザー追記で拡張し、get_module_schema で返却する
- 多言語はコード側で提供し、言語フォールバックは en-US に統一する

---

## 前提

- 対応言語は **en-US / ja-JP のみ**（BCP47）
- ユーザー説明は **単一言語の自由入力**（多言語入力は扱わない）
- RPCはとりあえずは実装しない（tools-export + Console 側JSON参照で運用）

---

## 方針

- tool_id はサーバー側のコードで固定（推奨: {module}:{tool}, 例: notion:create_page）
- モジュール/ツールの説明は **コード側で多言語保持** 例:
```json
{
  "descriptions": {
    "en-US": "Default description in English.",
    "ja-JP": "デフォルトの日本語説明。"
  }
}
```
- Handler が user.preferences.language を見て言語を選択
- Module.description は **default + user追記**
- Tool.description は ユーザー編集不可として、ユーザーの言語設定に応じた固定文を返却。
- この固定文はユーザーコンソールのツール説明に反映される
- 未対応言語・エラー時は **en-US にフォールバック**

---

## 変更内容

### 1. データモデル
- tool_settings.tool_name → tool_id に置換（破壊的）
- module_settings.description TEXT を追加

### 2. Go Server
- modules.Tool に ID を追加
- Module.Description(lang string) を実装（多言語）
- Tool description も多言語に対応
- handleGetModuleSchema で description 結合
  - default = Description(lang)（なければ en-US）
  - user = module_settings.description（デフォルトでは空文字列）
  - 返却 = default + "\n\n" + user

### 3. tools-export
- tools.json の id は tool.ID
- tools.json / services.json は descriptions を多言語で出力
- Console は descriptions[lang] ?? descriptions["en-US"] を表示

### 4. Console UI
- モジュール説明の編集欄を追加
- 保存は (user_id, module_id) をキーにする

---

## 実装ステップ

1) DBマイグレーション（tool_id 置換 / module_settings.description 追加）
2) Tool構造体 + ツール定義更新（ID追加）
3) Module / Tool の多言語 description 実装
4) tools-export の出力変更（descriptions）
5) Console 参照キーの tool_id 移行
6) Handler で description 結合（追記）

---

## 影響範囲

- tool_settings 既存データは互換なし
- tools.json の schema/参照キー変更
- Console/Server の tool参照ロジック変更

---

## 未決事項

- tool_id 命名規則の確定（推奨: module:tool）
- description の最大長
