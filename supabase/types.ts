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
  public: {
    Tables: {
      [_ in never]: never
    }
    Views: {
      [_ in never]: never
    }
    Functions: {
      activate_subscription: {
        Args: { p_event_id: string; p_plan_id: string; p_user_id: string }
        Returns: Json
      }
      complete_user_onboarding: {
        Args: { p_event_id: string; p_user_id: string }
        Returns: Json
      }
      delete_credential: {
        Args: { p_module: string; p_user_id: string }
        Returns: Json
      }
      delete_oauth_app: { Args: { p_provider: string }; Returns: Json }
      delete_prompt: {
        Args: { p_prompt_id: string; p_user_id: string }
        Returns: Json
      }
      generate_api_key: {
        Args: {
          p_display_name: string
          p_expires_at?: string
          p_user_id: string
        }
        Returns: Json
      }
      get_credential: {
        Args: { p_module: string; p_user_id: string }
        Returns: Json
      }
      get_module_config: {
        Args: { p_module_name?: string; p_user_id: string }
        Returns: {
          description: string
          enabled: boolean
          module_name: string
          tool_id: string
        }[]
      }
      get_oauth_app_credentials: { Args: { p_provider: string }; Returns: Json }
      get_prompt: {
        Args: { p_prompt_id: string; p_user_id: string }
        Returns: Json
      }
      get_prompts: {
        Args: {
          p_enabled_only?: boolean
          p_prompt_name?: string
          p_user_id: string
        }
        Returns: {
          content: string
          description: string
          enabled: boolean
          id: string
          name: string
        }[]
      }
      get_stripe_customer_id: { Args: { p_user_id: string }; Returns: string }
      get_usage: {
        Args: { p_end_date: string; p_start_date: string; p_user_id: string }
        Returns: Json
      }
      get_user_by_stripe_customer: {
        Args: { p_stripe_customer_id: string }
        Returns: string
      }
      get_user_context: {
        Args: { p_user_id: string }
        Returns: {
          account_status: string
          connected_count: number
          daily_limit: number
          daily_used: number
          display_name: string
          enabled_modules: string[]
          enabled_tools: Json
          language: string
          module_descriptions: Json
          plan_id: string
          role: string
          settings: Json
        }[]
      }
      link_stripe_customer: {
        Args: { p_stripe_customer_id: string; p_user_id: string }
        Returns: Json
      }
      list_all_oauth_consents: {
        Args: never
        Returns: {
          client_id: string
          client_name: string
          granted_at: string
          id: string
          scopes: string
          user_email: string
          user_id: string
        }[]
      }
      list_api_keys: {
        Args: { p_user_id: string }
        Returns: {
          display_name: string
          expires_at: string
          id: string
          key_prefix: string
          last_used_at: string
          revoked_at: string
        }[]
      }
      list_credentials: {
        Args: { p_user_id: string }
        Returns: {
          created_at: string
          module: string
          updated_at: string
        }[]
      }
      list_modules: {
        Args: never
        Returns: {
          created_at: string
          id: string
          name: string
          status: string
        }[]
      }
      list_modules_with_tools: {
        Args: never
        Returns: {
          descriptions: Json
          id: string
          name: string
          status: string
          tools: Json
        }[]
      }
      list_oauth_apps: { Args: never; Returns: Json }
      list_oauth_consents: {
        Args: { p_user_id: string }
        Returns: {
          client_id: string
          client_name: string
          granted_at: string
          id: string
          scopes: string
        }[]
      }
      list_prompts: {
        Args: { p_module_name?: string; p_user_id: string }
        Returns: {
          content: string
          created_at: string
          description: string
          enabled: boolean
          id: string
          module_name: string
          name: string
          updated_at: string
        }[]
      }
      lookup_user_by_key_hash: { Args: { p_key_hash: string }; Returns: Json }
      record_usage: {
        Args: {
          p_details: Json
          p_meta_tool: string
          p_request_id: string
          p_user_id: string
        }
        Returns: undefined
      }
      revoke_api_key: {
        Args: { p_key_id: string; p_user_id: string }
        Returns: Json
      }
      revoke_oauth_consent: {
        Args: { p_consent_id: string; p_user_id: string }
        Returns: Json
      }
      sync_modules: { Args: { p_modules: Json }; Returns: Json }
      update_settings: {
        Args: { p_settings: Json; p_user_id: string }
        Returns: Json
      }
      upsert_credential: {
        Args: { p_credentials: Json; p_module: string; p_user_id: string }
        Returns: Json
      }
      upsert_module_description: {
        Args: {
          p_description: string
          p_module_name: string
          p_user_id: string
        }
        Returns: Json
      }
      upsert_oauth_app: {
        Args: {
          p_client_id: string
          p_client_secret: string
          p_enabled?: boolean
          p_provider: string
          p_redirect_uri: string
        }
        Returns: Json
      }
      upsert_prompt: {
        Args: {
          p_content: string
          p_description?: string
          p_enabled?: boolean
          p_module_name?: string
          p_name: string
          p_prompt_id?: string
          p_user_id: string
        }
        Returns: Json
      }
      upsert_tool_settings: {
        Args: {
          p_disabled_tools: string[]
          p_enabled_tools: string[]
          p_module_name: string
          p_user_id: string
        }
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
  public: {
    Enums: {},
  },
} as const
