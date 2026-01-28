# CON - SSM インタラクション詳細（dtl-itr-CON-SSM）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-014 |
| Note | User Console - Session Manager Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | User Console (CON) |
| 連携先 | Session Manager (SSM) |
| 内容 | ソーシャルログイン |
| プロトコル | OAuth 2.0 / OpenID Connect |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 / OpenID Connect（SSM経由） |
| 用途 | ユーザーログイン |

CONはSSMを経由してIDPと通信する。CONとIDPの直接通信はない。

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
    CON->>User: ダッシュボード表示
```

### 対応プロバイダ

- Google
- Apple
- Microsoft
- GitHub

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-CON.md](./itr-CON.md) | User Console 詳細仕様 |
| [itr-SSM.md](./itr-SSM.md) | Session Manager 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
