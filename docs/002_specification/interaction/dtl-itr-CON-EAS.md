# CON - EAS インタラクション詳細（dtl-itr-CON-EAS）

## ドキュメント管理情報

| 項目      | 値                                                      |
| ------- | ------------------------------------------------------ |
| Status  | `reviewed`                                             |
| Version | v2.0                                                   |
| Note    | User Console - External Auth Server Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | User Console (CON) |
| 連携先 | External Auth Server (EAS) |
| 内容 | 認可フロー |
| プロトコル | OAuth 2.0 / HTTPS |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 |
| 用途 | 外部サービスへのアクセス権限取得 |

### フロー

```mermaid
sequenceDiagram
    participant User as ユーザー
    participant CON as User Console
    participant EAS as External Auth Server
    participant TVL as Token Vault

    User->>CON: 連携開始（/connect/:service）
    CON->>EAS: 認可リクエスト（OAuth）
    EAS->>User: 認証/同意画面
    User->>EAS: 同意
    EAS-->>CON: 認可コード（redirect）
    CON->>EAS: トークン交換
    EAS-->>CON: access_token, refresh_token
    CON->>TVL: トークン保存
    TVL-->>CON: 保存完了
    CON->>User: 連携完了表示
```

### 主な対応サービス

- Notion
- Google Calendar
- Microsoft To Do

---

## 関連ドキュメント

| ドキュメント                             | 内容                        |
| ---------------------------------- | ------------------------- |
| [itr-CON.md](./itr-CON.md)         | User Console 詳細仕様         |
| [itr-EAS.md](./itr-EAS.md)         | External Auth Server 詳細仕様 |

