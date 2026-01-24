# IDP - SSM インタラクション詳細（dtl-itr-IDP-SSM）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-019 |
| Note | Identity Provider - Session Manager Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Session Manager (SSM) |
| 連携先 | Identity Provider (IDP) |
| 内容 | ソーシャルログイン |
| プロトコル | OAuth 2.0 / OpenID Connect |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 / OpenID Connect |
| 用途 | ソーシャルログインによるユーザー認証 |

### 対応プロバイダ

- Google
- Apple
- Microsoft
- GitHub

### フロー

```mermaid
sequenceDiagram
    participant User as ユーザー
    participant CON as User Console
    participant SSM as Session Manager
    participant IDP as Identity Provider

    User->>CON: ソーシャルログインボタンクリック
    CON->>SSM: 認証開始リクエスト
    SSM->>IDP: OAuth認可リクエスト
    IDP->>User: ログイン画面
    User->>IDP: 認証情報入力
    IDP-->>SSM: 認可コード
    SSM->>IDP: トークン交換
    IDP-->>SSM: ID Token, Access Token
    SSM->>SSM: ユーザー情報取得・作成
    SSM-->>CON: セッション確立
    CON-->>User: ログイン完了
```

### SSMの処理

1. IDPから受け取ったID Tokenを検証
2. ユーザー情報（email, name等）を抽出
3. 既存ユーザーか確認
4. 新規の場合はユーザーレコード作成
5. セッショントークンを発行

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-ssm.md](./itr-ssm.md) | Session Manager 詳細仕様 |
| [itr-idp.md](./itr-idp.md) | Identity Provider 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
