# AUS - SSM インタラクション詳細（dtl-itr-AUS-SSM）

## ドキュメント管理情報

| 項目      | 値                                                |
| ------- | ------------------------------------------------ |
| Status  | `reviewed`                                       |
| Version | v2.0                                             |
| Note    | Auth Server - Session Manager Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Auth Server (AUS) |
| 連携先 | Session Manager (SSM) |
| 内容 | 認可フローにおけるセッション確認・ユーザー認証 |
| プロトコル | Supabase Auth 内部処理（同一基盤） |

> **実装上の注記**: 現在 AUS と SSM は同一の Supabase Auth インスタンスで実装されており、以下の連携はプロセス内部で完結する。本仕様書では将来的な物理分離を想定し、論理的な責務境界とデータフローを記述する。

---

## 詳細

| 項目 | 内容 |
|------|------|
| 方向 | AUS ↔ SSM（双方向） |
| 用途 | `/authorize` リクエスト時のセッション確認およびユーザー認証 |

### AUS → SSM（セッション確認）

AUS が認可リクエストを受信した際、SSM に対してユーザーのログイン状態を問い合わせる。

| 項目 | 内容 |
|------|------|
| トリガー | `/authorize` リクエスト受信時 |
| 入力 | セッション識別子（Cookie） |
| 出力 | 認証済み / 未認証 |

### SSM → AUS（ユーザー情報返却）

SSM がログイン処理を完了した後、AUS に対してユーザー情報を返却する。

| フィールド | 型 | 説明 |
|-----------|------|------|
| user_id | string (UUID) | ユーザー識別子 |
| email | string | メールアドレス |
| user_metadata | object | プロバイダから取得したプロフィール情報 |

user_metadata に含まれる主なフィールド:

| フィールド | 型 | 説明 |
|-----------|------|------|
| full_name | string | 表示名（プロバイダにより `name` の場合あり） |
| avatar_url | string | プロフィール画像 URL |

### 認可フロー内の連携

1. AUS が `/authorize` を受信
2. AUS が SSM にセッション確認を依頼
3. 認証済み → 5 へ
4. 未認証 → SSM のログインフローにリダイレクト → ログイン完了後 AUS に戻る
5. AUS が同意画面を表示（UI は CON が提供）
6. ユーザーが同意 → AUS が認可コードを発行

ログイン処理の詳細（認証方式、IDP 連携等）は [itr-SSM.md](./itr-SSM.md) を参照。
同意画面の実装は [dtl-itr-CON-SSM.md](./dtl-itr-CON-SSM.md) を参照。

### 期待する振る舞い

- AUS は認可リクエスト受信時に SSM のセッション状態を確認し、未認証ユーザーに対してログインフローを開始する
- SSM がログインを完了すると、user_id・email・user_metadata を AUS に返却する
- AUS は SSM から受け取った user_id をもとに認可コードを発行する
- 同意画面の表示は CON が担当し、Supabase Auth の OAuth API（getAuthorizationDetails / approveAuthorization / denyAuthorization）を介して AUS と連携する
- AUS ↔ SSM 間のセッション識別は Cookie ベースで行われる

---

## 関連ドキュメント

| ドキュメント                                     | 内容                   |
| ------------------------------------------ | -------------------- |
| [itr-SSM.md](./itr-SSM.md)                 | Session Manager 仕様   |
| [itr-AUS.md](./itr-AUS.md)                 | Auth Server 仕様       |
| [dtl-itr-CON-SSM.md](./dtl-itr-CON-SSM.md) | CON→SSM 同意画面・認証フロー |

