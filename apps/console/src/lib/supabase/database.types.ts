export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export type Database = {
  // Allows to automatically instantiate createClient with right options
  // instead of createClient<Database, { PostgrestVersion: 'XX' }>(URL, KEY)
  __InternalSupabase: {
    PostgrestVersion: "14.1"
  }
  mcpist: {
    Tables: {
      api_keys: {
        Row: {
          created_at: string
          expires_at: string | null
          id: string
          key_hash: string
          key_prefix: string
          last_used_at: string | null
          name: string
          revoked_at: string | null
          user_id: string
        }
        Insert: {
          created_at?: string
          expires_at?: string | null
          id?: string
          key_hash: string
          key_prefix: string
          last_used_at?: string | null
          name: string
          revoked_at?: string | null
          user_id: string
        }
        Update: {
          created_at?: string
          expires_at?: string | null
          id?: string
          key_hash?: string
          key_prefix?: string
          last_used_at?: string | null
          name?: string
          revoked_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "api_keys_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      credit_transactions: {
        Row: {
          amount: number
          created_at: string
          credit_type: string | null
          id: string
          module: string | null
          request_id: string | null
          task_id: string | null
          tool: string | null
          type: Database["mcpist"]["Enums"]["credit_transaction_type"]
          user_id: string
        }
        Insert: {
          amount: number
          created_at?: string
          credit_type?: string | null
          id?: string
          module?: string | null
          request_id?: string | null
          task_id?: string | null
          tool?: string | null
          type: Database["mcpist"]["Enums"]["credit_transaction_type"]
          user_id: string
        }
        Update: {
          amount?: number
          created_at?: string
          credit_type?: string | null
          id?: string
          module?: string | null
          request_id?: string | null
          task_id?: string | null
          tool?: string | null
          type?: Database["mcpist"]["Enums"]["credit_transaction_type"]
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
          free_credits: number
          paid_credits: number
          updated_at: string
          user_id: string
        }
        Insert: {
          free_credits?: number
          paid_credits?: number
          updated_at?: string
          user_id: string
        }
        Update: {
          free_credits?: number
          paid_credits?: number
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
      module_settings: {
        Row: {
          created_at: string
          enabled: boolean
          module_id: string
          user_id: string
        }
        Insert: {
          created_at?: string
          enabled?: boolean
          module_id: string
          user_id: string
        }
        Update: {
          created_at?: string
          enabled?: boolean
          module_id?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "module_settings_module_id_fkey"
            columns: ["module_id"]
            isOneToOne: false
            referencedRelation: "modules"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "module_settings_user_id_fkey"
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
          id: string
          name: string
          status: Database["mcpist"]["Enums"]["module_status"]
        }
        Insert: {
          created_at?: string
          id?: string
          name: string
          status?: Database["mcpist"]["Enums"]["module_status"]
        }
        Update: {
          created_at?: string
          id?: string
          name?: string
          status?: Database["mcpist"]["Enums"]["module_status"]
        }
        Relationships: []
      }
      processed_webhook_events: {
        Row: {
          event_id: string
          processed_at: string
          user_id: string
        }
        Insert: {
          event_id: string
          processed_at?: string
          user_id: string
        }
        Update: {
          event_id?: string
          processed_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "processed_webhook_events_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      prompts: {
        Row: {
          content: string
          created_at: string
          id: string
          module_id: string | null
          name: string
          updated_at: string
          user_id: string
        }
        Insert: {
          content: string
          created_at?: string
          id?: string
          module_id?: string | null
          name: string
          updated_at?: string
          user_id: string
        }
        Update: {
          content?: string
          created_at?: string
          id?: string
          module_id?: string | null
          name?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "prompts_module_id_fkey"
            columns: ["module_id"]
            isOneToOne: false
            referencedRelation: "modules"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "prompts_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      service_tokens: {
        Row: {
          created_at: string
          credentials_secret_id: string
          id: string
          service: string
          updated_at: string
          user_id: string
        }
        Insert: {
          created_at?: string
          credentials_secret_id: string
          id?: string
          service: string
          updated_at?: string
          user_id: string
        }
        Update: {
          created_at?: string
          credentials_secret_id?: string
          id?: string
          service?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "service_tokens_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      tool_settings: {
        Row: {
          created_at: string
          enabled: boolean
          module_id: string
          tool_name: string
          user_id: string
        }
        Insert: {
          created_at?: string
          enabled?: boolean
          module_id: string
          tool_name: string
          user_id: string
        }
        Update: {
          created_at?: string
          enabled?: boolean
          module_id?: string
          tool_name?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "tool_settings_module_id_fkey"
            columns: ["module_id"]
            isOneToOne: false
            referencedRelation: "modules"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "tool_settings_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      users: {
        Row: {
          account_status: Database["mcpist"]["Enums"]["account_status"]
          created_at: string
          id: string
          preferences: Json | null
          updated_at: string
        }
        Insert: {
          account_status?: Database["mcpist"]["Enums"]["account_status"]
          created_at?: string
          id: string
          preferences?: Json | null
          updated_at?: string
        }
        Update: {
          account_status?: Database["mcpist"]["Enums"]["account_status"]
          created_at?: string
          id?: string
          preferences?: Json | null
          updated_at?: string
        }
        Relationships: []
      }
    }
    Views: {
      [_ in never]: never
    }
    Functions: {
      add_paid_credits: {
        Args: { p_amount: number; p_event_id: string; p_user_id: string }
        Returns: Json
      }
      consume_credit: {
        Args: {
          p_amount: number
          p_module: string
          p_request_id: string
          p_task_id?: string
          p_tool: string
          p_user_id: string
        }
        Returns: Json
      }
      get_module_token: {
        Args: { p_module: string; p_user_id: string }
        Returns: Json
      }
      get_user_context: {
        Args: { p_user_id: string }
        Returns: {
          account_status: string
          disabled_tools: Json
          enabled_modules: string[]
          free_credits: number
          paid_credits: number
        }[]
      }
      lookup_user_by_key_hash: { Args: { p_key_hash: string }; Returns: Json }
      reset_free_credits: { Args: never; Returns: Json }
    }
    Enums: {
      account_status: "active" | "suspended" | "disabled"
      credit_transaction_type: "consume" | "purchase" | "monthly_reset"
      module_status:
        | "active"
        | "coming_soon"
        | "maintenance"
        | "beta"
        | "deprecated"
        | "disabled"
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
      add_paid_credits: {
        Args: { p_amount: number; p_event_id: string; p_user_id: string }
        Returns: Json
      }
      consume_credit: {
        Args: {
          p_amount: number
          p_module: string
          p_request_id: string
          p_task_id?: string
          p_tool: string
          p_user_id: string
        }
        Returns: Json
      }
      delete_service_token: { Args: { p_service: string }; Returns: Json }
      generate_api_key: {
        Args: { p_expires_in_days?: number; p_name: string }
        Returns: Json
      }
      get_module_token: {
        Args: { p_module: string; p_user_id: string }
        Returns: Json
      }
      get_my_role: { Args: never; Returns: string }
      get_user_context: {
        Args: { p_user_id: string }
        Returns: {
          account_status: string
          disabled_tools: Json
          enabled_modules: string[]
          free_credits: number
          paid_credits: number
        }[]
      }
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
      list_service_connections: {
        Args: never
        Returns: {
          created_at: string
          id: string
          service: string
          updated_at: string
        }[]
      }
      lookup_user_by_key_hash: { Args: { p_key_hash: string }; Returns: Json }
      reset_free_credits: { Args: never; Returns: Json }
      revoke_api_key: { Args: { p_key_id: string }; Returns: Json }
      upsert_service_token: {
        Args: { p_credentials: Json; p_service: string }
        Returns: Json
      }
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
    Enums: {
      account_status: ["active", "suspended", "disabled"],
      credit_transaction_type: ["consume", "purchase", "monthly_reset"],
      module_status: [
        "active",
        "coming_soon",
        "maintenance",
        "beta",
        "deprecated",
        "disabled",
      ],
    },
  },
  public: {
    Enums: {},
  },
} as const
