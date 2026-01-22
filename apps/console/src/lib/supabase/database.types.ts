export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export type Database = {
  mcpist: {
    Tables: {
      api_keys: {
        Row: {
          created_at: string | null
          expires_at: string | null
          id: string
          key_hash: string
          key_prefix: string
          last_used_at: string | null
          name: string
          revoked_at: string | null
          scopes: string[] | null
          service: string
          user_id: string
        }
        Insert: {
          created_at?: string | null
          expires_at?: string | null
          id?: string
          key_hash: string
          key_prefix: string
          last_used_at?: string | null
          name: string
          revoked_at?: string | null
          scopes?: string[] | null
          service?: string
          user_id: string
        }
        Update: {
          created_at?: string | null
          expires_at?: string | null
          id?: string
          key_hash?: string
          key_prefix?: string
          last_used_at?: string | null
          name?: string
          revoked_at?: string | null
          scopes?: string[] | null
          service?: string
          user_id?: string
        }
        Relationships: []
      }
      credit_transactions: {
        Row: {
          amount: number
          balance_after: number
          created_at: string
          description: string | null
          id: string
          reference_id: string | null
          transaction_type: string
          user_id: string
        }
        Insert: {
          amount: number
          balance_after: number
          created_at?: string
          description?: string | null
          id?: string
          reference_id?: string | null
          transaction_type: string
          user_id: string
        }
        Update: {
          amount?: number
          balance_after?: number
          created_at?: string
          description?: string | null
          id?: string
          reference_id?: string | null
          transaction_type?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "credit_transactions_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      credits: {
        Row: {
          balance: number
          created_at: string
          id: string
          updated_at: string
          user_id: string
        }
        Insert: {
          balance?: number
          created_at?: string
          id?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          balance?: number
          created_at?: string
          id?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "credits_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: true
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      mcp_tokens: {
        Row: {
          created_at: string
          expires_at: string | null
          id: string
          is_revoked: boolean
          last_used_at: string | null
          name: string
          token_hash: string
          user_id: string
        }
        Insert: {
          created_at?: string
          expires_at?: string | null
          id?: string
          is_revoked?: boolean
          last_used_at?: string | null
          name: string
          token_hash: string
          user_id: string
        }
        Update: {
          created_at?: string
          expires_at?: string | null
          id?: string
          is_revoked?: boolean
          last_used_at?: string | null
          name?: string
          token_hash?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "mcp_tokens_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      modules: {
        Row: {
          created_at: string
          description: string | null
          display_name: string
          id: string
          is_active: boolean
          name: string
          oauth_provider: string | null
          requires_oauth: boolean
          updated_at: string
        }
        Insert: {
          created_at?: string
          description?: string | null
          display_name: string
          id?: string
          is_active?: boolean
          name: string
          oauth_provider?: string | null
          requires_oauth?: boolean
          updated_at?: string
        }
        Update: {
          created_at?: string
          description?: string | null
          display_name?: string
          id?: string
          is_active?: boolean
          name?: string
          oauth_provider?: string | null
          requires_oauth?: boolean
          updated_at?: string
        }
        Relationships: []
      }
      oauth_authorization_codes: {
        Row: {
          client_id: string
          code: string
          code_challenge: string
          code_challenge_method: string
          created_at: string
          expires_at: string
          redirect_uri: string
          scope: string | null
          state: string | null
          used_at: string | null
          user_id: string
        }
        Insert: {
          client_id: string
          code: string
          code_challenge: string
          code_challenge_method?: string
          created_at?: string
          expires_at: string
          redirect_uri: string
          scope?: string | null
          state?: string | null
          used_at?: string | null
          user_id: string
        }
        Update: {
          client_id?: string
          code?: string
          code_challenge?: string
          code_challenge_method?: string
          created_at?: string
          expires_at?: string
          redirect_uri?: string
          scope?: string | null
          state?: string | null
          used_at?: string | null
          user_id?: string
        }
        Relationships: []
      }
      oauth_authorization_requests: {
        Row: {
          client_id: string
          code_challenge: string
          code_challenge_method: string
          created_at: string
          expires_at: string
          id: string
          redirect_uri: string
          scope: string
          state: string | null
          status: string
          user_id: string | null
        }
        Insert: {
          client_id: string
          code_challenge: string
          code_challenge_method?: string
          created_at?: string
          expires_at: string
          id: string
          redirect_uri: string
          scope?: string
          state?: string | null
          status?: string
          user_id?: string | null
        }
        Update: {
          client_id?: string
          code_challenge?: string
          code_challenge_method?: string
          created_at?: string
          expires_at?: string
          id?: string
          redirect_uri?: string
          scope?: string
          state?: string | null
          status?: string
          user_id?: string | null
        }
        Relationships: []
      }
      oauth_refresh_tokens: {
        Row: {
          client_id: string
          created_at: string
          expires_at: string
          id: string
          scope: string | null
          token: string
          used: boolean
          user_id: string
        }
        Insert: {
          client_id: string
          created_at?: string
          expires_at: string
          id?: string
          scope?: string | null
          token: string
          used?: boolean
          user_id: string
        }
        Update: {
          client_id?: string
          created_at?: string
          expires_at?: string
          id?: string
          scope?: string | null
          token?: string
          used?: boolean
          user_id?: string
        }
        Relationships: []
      }
      oauth_token_history: {
        Row: {
          access_token_secret_id: string | null
          created_at: string
          created_by_ip: string | null
          expired_at: string | null
          expired_by_ip: string | null
          expired_reason: string | null
          id: string
          refresh_token_secret_id: string | null
          service: string
          token_type: string | null
          user_id: string
        }
        Insert: {
          access_token_secret_id?: string | null
          created_at?: string
          created_by_ip?: string | null
          expired_at?: string | null
          expired_by_ip?: string | null
          expired_reason?: string | null
          id?: string
          refresh_token_secret_id?: string | null
          service: string
          token_type?: string | null
          user_id: string
        }
        Update: {
          access_token_secret_id?: string | null
          created_at?: string
          created_by_ip?: string | null
          expired_at?: string | null
          expired_by_ip?: string | null
          expired_reason?: string | null
          id?: string
          refresh_token_secret_id?: string | null
          service?: string
          token_type?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "oauth_token_history_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      oauth_tokens: {
        Row: {
          access_token_secret_id: string | null
          created_at: string
          expires_at: string | null
          id: string
          refresh_token_secret_id: string | null
          scope: string | null
          service: string
          token_type: string | null
          updated_at: string
          user_id: string
        }
        Insert: {
          access_token_secret_id?: string | null
          created_at?: string
          expires_at?: string | null
          id?: string
          refresh_token_secret_id?: string | null
          scope?: string | null
          service: string
          token_type?: string | null
          updated_at?: string
          user_id: string
        }
        Update: {
          access_token_secret_id?: string | null
          created_at?: string
          expires_at?: string | null
          id?: string
          refresh_token_secret_id?: string | null
          scope?: string | null
          service?: string
          token_type?: string | null
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "oauth_tokens_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      plans: {
        Row: {
          created_at: string
          credit_enabled: boolean
          display_name: string
          id: string
          is_active: boolean
          name: string
          quota_monthly: number | null
          rate_limit_burst: number
          rate_limit_rpm: number
          updated_at: string
        }
        Insert: {
          created_at?: string
          credit_enabled?: boolean
          display_name: string
          id?: string
          is_active?: boolean
          name: string
          quota_monthly?: number | null
          rate_limit_burst?: number
          rate_limit_rpm?: number
          updated_at?: string
        }
        Update: {
          created_at?: string
          credit_enabled?: boolean
          display_name?: string
          id?: string
          is_active?: boolean
          name?: string
          quota_monthly?: number | null
          rate_limit_burst?: number
          rate_limit_rpm?: number
          updated_at?: string
        }
        Relationships: []
      }
      processed_webhook_events: {
        Row: {
          event_id: string
          event_type: string
          id: string
          processed_at: string
        }
        Insert: {
          event_id: string
          event_type: string
          id?: string
          processed_at?: string
        }
        Update: {
          event_id?: string
          event_type?: string
          id?: string
          processed_at?: string
        }
        Relationships: []
      }
      subscriptions: {
        Row: {
          canceled_at: string | null
          created_at: string
          current_period_end: string | null
          current_period_start: string | null
          id: string
          plan_id: string
          psp_customer_id: string | null
          psp_subscription_id: string | null
          status: string
          updated_at: string
          user_id: string
        }
        Insert: {
          canceled_at?: string | null
          created_at?: string
          current_period_end?: string | null
          current_period_start?: string | null
          id?: string
          plan_id: string
          psp_customer_id?: string | null
          psp_subscription_id?: string | null
          status?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          canceled_at?: string | null
          created_at?: string
          current_period_end?: string | null
          current_period_start?: string | null
          id?: string
          plan_id?: string
          psp_customer_id?: string | null
          psp_subscription_id?: string | null
          status?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "subscriptions_plan_id_fkey"
            columns: ["plan_id"]
            isOneToOne: false
            referencedRelation: "plans"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "subscriptions_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: true
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      tool_costs: {
        Row: {
          created_at: string
          credit_cost: number
          id: string
          module_id: string
          tool_name: string
          updated_at: string
        }
        Insert: {
          created_at?: string
          credit_cost?: number
          id?: string
          module_id: string
          tool_name: string
          updated_at?: string
        }
        Update: {
          created_at?: string
          credit_cost?: number
          id?: string
          module_id?: string
          tool_name?: string
          updated_at?: string
        }
        Relationships: [
          {
            foreignKeyName: "tool_costs_module_id_fkey"
            columns: ["module_id"]
            isOneToOne: false
            referencedRelation: "modules"
            referencedColumns: ["id"]
          },
        ]
      }
      usage: {
        Row: {
          created_at: string
          id: string
          period_start: string
          request_count: number
          updated_at: string
          user_id: string
        }
        Insert: {
          created_at?: string
          id?: string
          period_start: string
          request_count?: number
          updated_at?: string
          user_id: string
        }
        Update: {
          created_at?: string
          id?: string
          period_start?: string
          request_count?: number
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "usage_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      user_module_preferences: {
        Row: {
          created_at: string
          id: string
          is_enabled: boolean
          module_id: string
          settings: Json | null
          updated_at: string
          user_id: string
        }
        Insert: {
          created_at?: string
          id?: string
          is_enabled?: boolean
          module_id: string
          settings?: Json | null
          updated_at?: string
          user_id: string
        }
        Update: {
          created_at?: string
          id?: string
          is_enabled?: boolean
          module_id?: string
          settings?: Json | null
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "user_module_preferences_module_id_fkey"
            columns: ["module_id"]
            isOneToOne: false
            referencedRelation: "modules"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "user_module_preferences_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      users: {
        Row: {
          avatar_url: string | null
          created_at: string
          display_name: string | null
          id: string
          role: string
          status: string
          updated_at: string
        }
        Insert: {
          avatar_url?: string | null
          created_at?: string
          display_name?: string | null
          id: string
          role?: string
          status?: string
          updated_at?: string
        }
        Update: {
          avatar_url?: string | null
          created_at?: string
          display_name?: string | null
          id?: string
          role?: string
          status?: string
          updated_at?: string
        }
        Relationships: []
      }
    }
    Views: {
      [_ in never]: never
    }
    Functions: {
      approve_oauth_authorization: {
        Args: { p_authorization_id: string; p_user_id: string }
        Returns: {
          code: string
          redirect_uri: string
          state: string
        }[]
      }
      cleanup_expired_oauth_authorization_requests: {
        Args: never
        Returns: number
      }
      cleanup_expired_oauth_codes: { Args: never; Returns: number }
      cleanup_expired_oauth_refresh_tokens: { Args: never; Returns: number }
      consume_oauth_code: {
        Args: { p_code: string }
        Returns: {
          client_id: string
          code: string
          code_challenge: string
          code_challenge_method: string
          expires_at: string
          redirect_uri: string
          scope: string
          state: string
          user_id: string
        }[]
      }
      consume_oauth_refresh_token: {
        Args: { p_token: string }
        Returns: {
          client_id: string
          expires_at: string
          scope: string
          token: string
          user_id: string
        }[]
      }
      deduct_credits: {
        Args: {
          p_amount: number
          p_description?: string
          p_reference_id?: string
          p_user_id: string
        }
        Returns: number
      }
      delete_oauth_token: { Args: { p_service: string }; Returns: boolean }
      deny_oauth_authorization: {
        Args: { p_authorization_id: string }
        Returns: {
          redirect_uri: string
          state: string
        }[]
      }
      get_masked_api_key: { Args: { p_service: string }; Returns: string }
      get_my_oauth_connections: {
        Args: never
        Returns: {
          created_at: string
          expires_at: string
          id: string
          is_expired: boolean
          scope: string
          service: string
          token_type: string
          updated_at: string
        }[]
      }
      get_my_role: { Args: never; Returns: string }
      get_my_token_history: {
        Args: { p_service?: string }
        Returns: {
          created_at: string
          expired_at: string
          expired_reason: string
          id: string
          service: string
          token_type: string
        }[]
      }
      get_oauth_authorization_request: {
        Args: { p_id: string }
        Returns: {
          client_id: string
          code_challenge: string
          code_challenge_method: string
          created_at: string
          expires_at: string
          id: string
          redirect_uri: string
          scope: string
          state: string
          status: string
        }[]
      }
      get_service_token: {
        Args: { p_service: string; p_user_id: string }
        Returns: {
          long_term_token: string
          oauth_token: string
        }[]
      }
      get_tool_cost: {
        Args: { p_module_name: string; p_tool_name: string }
        Returns: number
      }
      get_user_entitlement: {
        Args: { p_user_id: string }
        Returns: {
          credit_balance: number
          credit_enabled: boolean
          enabled_modules: string[]
          plan_name: string
          quota_monthly: number
          rate_limit_burst: number
          rate_limit_rpm: number
          usage_current_month: number
          user_status: string
        }[]
      }
      increment_usage: { Args: { p_user_id: string }; Returns: number }
      revoke_oauth_refresh_tokens: {
        Args: { p_client_id?: string; p_user_id: string }
        Returns: number
      }
      store_oauth_authorization_request: {
        Args: {
          p_client_id: string
          p_code_challenge: string
          p_code_challenge_method: string
          p_expires_at: string
          p_id: string
          p_redirect_uri: string
          p_scope: string
          p_state: string
        }
        Returns: undefined
      }
      store_oauth_code: {
        Args: {
          p_client_id: string
          p_code: string
          p_code_challenge: string
          p_code_challenge_method: string
          p_expires_at: string
          p_redirect_uri: string
          p_scope: string
          p_state: string
          p_user_id: string
        }
        Returns: undefined
      }
      store_oauth_refresh_token: {
        Args: {
          p_client_id: string
          p_expires_at: string
          p_scope: string
          p_token: string
          p_user_id: string
        }
        Returns: undefined
      }
      upsert_oauth_token: {
        Args: {
          p_access_token: string
          p_expires_at?: string
          p_refresh_token?: string
          p_scope?: string
          p_service: string
          p_token_type?: string
        }
        Returns: string
      }
      validate_api_key: {
        Args: { p_api_key: string; p_service: string }
        Returns: {
          user_id: string
        }[]
      }
    }
    Enums: {
      [_ in never]: never
    }
    CompositeTypes: {
      [_ in never]: never
    }
  }
  public: {
    Tables: {
      [_ in never]: never
    }
    Views: {
      [_ in never]: never
    }
    Functions: {
      approve_oauth_authorization: {
        Args: { p_authorization_id: string; p_user_id: string }
        Returns: {
          code: string
          redirect_uri: string
          state: string
        }[]
      }
      consume_oauth_code: {
        Args: { p_code: string }
        Returns: {
          client_id: string
          code: string
          code_challenge: string
          code_challenge_method: string
          expires_at: string
          redirect_uri: string
          scope: string
          state: string
          user_id: string
        }[]
      }
      consume_oauth_refresh_token: {
        Args: { p_token: string }
        Returns: {
          client_id: string
          expires_at: string
          scope: string
          token: string
          user_id: string
        }[]
      }
      deduct_credits: {
        Args: {
          p_amount: number
          p_description?: string
          p_reference_id?: string
          p_user_id: string
        }
        Returns: number
      }
      delete_oauth_token: { Args: { p_service: string }; Returns: boolean }
      deny_oauth_authorization: {
        Args: { p_authorization_id: string }
        Returns: {
          redirect_uri: string
          state: string
        }[]
      }
      generate_api_key: {
        Args: { p_expires_in_days?: number; p_name: string }
        Returns: Json
      }
      get_masked_api_key: { Args: { p_service: string }; Returns: string }
      get_my_oauth_connections: {
        Args: never
        Returns: {
          created_at: string
          expires_at: string
          id: string
          is_expired: boolean
          scope: string
          service: string
          token_type: string
          updated_at: string
        }[]
      }
      get_my_role: { Args: never; Returns: string }
      get_my_token_history: {
        Args: { p_service?: string }
        Returns: {
          created_at: string
          expired_at: string
          expired_reason: string
          id: string
          service: string
          token_type: string
        }[]
      }
      get_oauth_authorization_request: {
        Args: { p_id: string }
        Returns: {
          client_id: string
          code_challenge: string
          code_challenge_method: string
          created_at: string
          expires_at: string
          id: string
          redirect_uri: string
          scope: string
          state: string
          status: string
        }[]
      }
      get_service_token: {
        Args: { p_service: string; p_user_id: string }
        Returns: {
          long_term_token: string
          oauth_token: string
        }[]
      }
      get_tool_cost: {
        Args: { p_module_name: string; p_tool_name: string }
        Returns: number
      }
      get_user_entitlement: {
        Args: { p_user_id: string }
        Returns: {
          credit_balance: number
          credit_enabled: boolean
          enabled_modules: string[]
          plan_name: string
          quota_monthly: number
          rate_limit_burst: number
          rate_limit_rpm: number
          usage_current_month: number
          user_status: string
        }[]
      }
      increment_usage: { Args: { p_user_id: string }; Returns: number }
      list_api_keys: {
        Args: never
        Returns: {
          created_at: string
          expires_at: string
          id: string
          is_expired: boolean
          key_prefix: string
          last_used_at: string
          name: string
        }[]
      }
      revoke_api_key: { Args: { p_key_id: string }; Returns: Json }
      revoke_oauth_refresh_tokens: {
        Args: { p_client_id?: string; p_user_id: string }
        Returns: number
      }
      store_oauth_authorization_request: {
        Args: {
          p_client_id: string
          p_code_challenge: string
          p_code_challenge_method: string
          p_expires_at: string
          p_id: string
          p_redirect_uri: string
          p_scope: string
          p_state: string
        }
        Returns: undefined
      }
      store_oauth_code: {
        Args: {
          p_client_id: string
          p_code: string
          p_code_challenge: string
          p_code_challenge_method: string
          p_expires_at: string
          p_redirect_uri: string
          p_scope: string
          p_state: string
          p_user_id: string
        }
        Returns: undefined
      }
      store_oauth_refresh_token: {
        Args: {
          p_client_id: string
          p_expires_at: string
          p_scope: string
          p_token: string
          p_user_id: string
        }
        Returns: undefined
      }
      upsert_oauth_token: {
        Args: {
          p_access_token: string
          p_expires_at?: string
          p_refresh_token?: string
          p_scope?: string
          p_service: string
          p_token_type?: string
        }
        Returns: string
      }
      validate_api_key:
        | {
            Args: { p_api_key: string; p_service: string }
            Returns: {
              user_id: string
            }[]
          }
        | { Args: { p_key: string }; Returns: Json }
    }
    Enums: {
      [_ in never]: never
    }
    CompositeTypes: {
      [_ in never]: never
    }
  }
}

type DatabaseWithoutInternals = Omit<Database, "__InternalSupabase">

type DefaultSchema = DatabaseWithoutInternals[Extract<keyof Database, "public">]

export type Tables<
  DefaultSchemaTableNameOrOptions extends
    | keyof (DefaultSchema["Tables"] & DefaultSchema["Views"])
    | { schema: keyof DatabaseWithoutInternals },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof (DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"] &
        DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Views"])
    : never = never,
> = DefaultSchemaTableNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? (DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"] &
      DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Views"])[TableName] extends {
      Row: infer R
    }
    ? R
    : never
  : DefaultSchemaTableNameOrOptions extends keyof (DefaultSchema["Tables"] &
        DefaultSchema["Views"])
    ? (DefaultSchema["Tables"] &
        DefaultSchema["Views"])[DefaultSchemaTableNameOrOptions] extends {
        Row: infer R
      }
      ? R
      : never
    : never

export type TablesInsert<
  DefaultSchemaTableNameOrOptions extends
    | keyof DefaultSchema["Tables"]
    | { schema: keyof DatabaseWithoutInternals },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = DefaultSchemaTableNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Insert: infer I
    }
    ? I
    : never
  : DefaultSchemaTableNameOrOptions extends keyof DefaultSchema["Tables"]
    ? DefaultSchema["Tables"][DefaultSchemaTableNameOrOptions] extends {
        Insert: infer I
      }
      ? I
      : never
    : never

export type TablesUpdate<
  DefaultSchemaTableNameOrOptions extends
    | keyof DefaultSchema["Tables"]
    | { schema: keyof DatabaseWithoutInternals },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = DefaultSchemaTableNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Update: infer U
    }
    ? U
    : never
  : DefaultSchemaTableNameOrOptions extends keyof DefaultSchema["Tables"]
    ? DefaultSchema["Tables"][DefaultSchemaTableNameOrOptions] extends {
        Update: infer U
      }
      ? U
      : never
    : never

export type Enums<
  DefaultSchemaEnumNameOrOptions extends
    | keyof DefaultSchema["Enums"]
    | { schema: keyof DatabaseWithoutInternals },
  EnumName extends DefaultSchemaEnumNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[DefaultSchemaEnumNameOrOptions["schema"]]["Enums"]
    : never = never,
> = DefaultSchemaEnumNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[DefaultSchemaEnumNameOrOptions["schema"]]["Enums"][EnumName]
  : DefaultSchemaEnumNameOrOptions extends keyof DefaultSchema["Enums"]
    ? DefaultSchema["Enums"][DefaultSchemaEnumNameOrOptions]
    : never

export type CompositeTypes<
  PublicCompositeTypeNameOrOptions extends
    | keyof DefaultSchema["CompositeTypes"]
    | { schema: keyof DatabaseWithoutInternals },
  CompositeTypeName extends PublicCompositeTypeNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"]
    : never = never,
> = PublicCompositeTypeNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"][CompositeTypeName]
  : PublicCompositeTypeNameOrOptions extends keyof DefaultSchema["CompositeTypes"]
    ? DefaultSchema["CompositeTypes"][PublicCompositeTypeNameOrOptions]
    : never

export const Constants = {
  mcpist: {
    Enums: {},
  },
  public: {
    Enums: {},
  },
} as const

