# DAY019 レビュー・設計メモ

## クレジット実装方針

### 決定事項

- **都度購入モデル**で進める（サブスクリプションは後回し）
- Stripe サンドボックスで **0円商品** を作成して購入フローを実装

### 理由

1. **安定供給できるか不明**
   - MCPサーバーの運用コストが読めない
   - API呼び出し頻度に応じたコスト変動
   - インフラ負荷の予測が困難

2. **UXの一貫性は維持したい**
   - ユーザーにはStripe Checkout経由で購入させる
   - テスト期間中は0円だが、本番では有料化予定
   - 購入→クレジット付与のフローを先に確立しておく

### 実装詳細

- Stripe Product: "MCPist Credits (100)"
- Stripe Price: $0.00
- Webhook: `checkout.session.completed` → `add_credits` RPC
- 冪等性: `processed_webhook_events` テーブルで event_id 重複チェック

### 将来の拡張

- 有料化時は Price ID を本番用に差し替え
- サブスクリプション対応は需要を見て判断
- 使用量ベースの課金も検討可能（Stripe Metered Billing）
