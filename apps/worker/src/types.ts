/** Worker 環境変数バインディング */
export interface Env {
  // バックエンド設定 (Go Server)
  PRIMARY_API_URL: string;
  SECONDARY_API_URL: string;

  // Clerk 認証
  CLERK_JWKS_URL: string;

  // Go Server JWKS (JWT API key 検証用)
  API_SERVER_JWKS_URL: string;

  // Gateway Secret (Worker → Go Server)
  GATEWAY_SECRET: string;

  // Stripe Webhook
  STRIPE_WEBHOOK_SECRET: string;

  // Grafana Loki
  GRAFANA_LOKI_URL: string;
  GRAFANA_LOKI_USER: string;
  GRAFANA_LOKI_API_KEY: string;
  APP_ENV: string;
}

export interface AuthResult {
  userId: string;
  email?: string;
  type: "jwt" | "api_key";
}
