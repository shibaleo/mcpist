/** Worker 環境変数バインディング */
export interface Env {
  // KV Namespaces
  API_KEY_CACHE: KVNamespace;

  // バックエンド設定
  PRIMARY_API_URL: string;
  SECONDARY_API_URL: string;

  // Supabase Auth 設定
  SUPABASE_URL: string;
  SUPABASE_JWKS_URL: string;
  SUPABASE_PUBLISHABLE_KEY: string;

  // PostgREST 設定
  POSTGREST_URL: string;
  POSTGREST_API_KEY: string;

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
  type: "jwt" | "api_key";
}
