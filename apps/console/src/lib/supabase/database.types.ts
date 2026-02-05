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
      add_user_credits: {
        Args: {
          p_amount: number
          p_credit_type: string
          p_event_id: string
          p_user_id: string
        }
        Returns: Json
      }
      complete_user_onboarding: {
        Args: { p_event_id: string; p_user_id: string }
        Returns: Json
      }
      consume_user_credits: {
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
      delete_my_credential: { Args: { p_module: string }; Returns: Json }
      delete_my_prompt: { Args: { p_prompt_id: string }; Returns: Json }
      delete_oauth_app: { Args: { p_provider: string }; Returns: Json }
      generate_my_api_key: {
        Args: { p_display_name: string; p_expires_at?: string }
        Returns: Json
      }
      get_my_module_descriptions: {
        Args: never
        Returns: {
          description: string
          module_name: string
        }[]
      }
      get_my_prompt: { Args: { p_prompt_id: string }; Returns: Json }
      get_my_role: { Args: never; Returns: Json }
      get_my_settings: { Args: never; Returns: Json }
      get_my_tool_settings: {
        Args: { p_module_name?: string }
        Returns: {
          enabled: boolean
          module_name: string
          tool_id: string
        }[]
      }
      get_my_usage: {
        Args: { p_end_date: string; p_start_date: string }
        Returns: Json
      }
      get_oauth_app_credentials: { Args: { p_provider: string }; Returns: Json }
      get_stripe_customer_id: { Args: { p_user_id: string }; Returns: string }
      get_user_by_stripe_customer: {
        Args: { p_stripe_customer_id: string }
        Returns: string
      }
      get_user_context: {
        Args: { p_user_id: string }
        Returns: {
          account_status: string
          enabled_modules: string[]
          enabled_tools: Json
          free_credits: number
          language: string
          module_descriptions: Json
          paid_credits: number
        }[]
      }
      get_user_credential: {
        Args: { p_module: string; p_user_id: string }
        Returns: Json
      }
      get_user_prompt_by_name: {
        Args: { p_prompt_name: string; p_user_id: string }
        Returns: {
          content: string
          description: string
          enabled: boolean
          id: string
          name: string
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
      list_modules: {
        Args: never
        Returns: {
          created_at: string
          id: string
          name: string
          status: string
        }[]
      }
      list_my_api_keys: {
        Args: never
        Returns: {
          display_name: string
          expires_at: string
          id: string
          key_prefix: string
          last_used_at: string
          revoked_at: string
        }[]
      }
      list_my_credentials: {
        Args: never
        Returns: {
          created_at: string
          module: string
          updated_at: string
        }[]
      }
      list_my_oauth_consents: {
        Args: never
        Returns: {
          client_id: string
          client_name: string
          granted_at: string
          id: string
          scopes: string
        }[]
      }
      list_my_prompts: {
        Args: { p_module_name?: string }
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
      list_oauth_apps: { Args: never; Returns: Json }
      list_user_prompts: {
        Args: { p_enabled_only?: boolean; p_user_id: string }
        Returns: {
          content: string
          description: string
          enabled: boolean
          id: string
          name: string
        }[]
      }
      lookup_user_by_key_hash: { Args: { p_key_hash: string }; Returns: Json }
      revoke_my_api_key: { Args: { p_key_id: string }; Returns: Json }
      revoke_my_oauth_consent: { Args: { p_consent_id: string }; Returns: Json }
      sync_modules: { Args: { p_modules: string[] }; Returns: Json }
      update_my_settings: { Args: { p_settings: Json }; Returns: Json }
      upsert_my_credential: {
        Args: { p_credentials: Json; p_module: string }
        Returns: Json
      }
      upsert_my_module_description: {
        Args: { p_description: string; p_module_name: string }
        Returns: Json
      }
      upsert_my_prompt:
        | {
            Args: {
              p_content: string
              p_enabled?: boolean
              p_module_name?: string
              p_name: string
              p_prompt_id?: string
            }
            Returns: Json
          }
        | {
            Args: {
              p_content: string
              p_description?: string
              p_enabled?: boolean
              p_module_name?: string
              p_name: string
              p_prompt_id?: string
            }
            Returns: Json
          }
      upsert_my_tool_settings: {
        Args: {
          p_disabled_tools: string[]
          p_enabled_tools: string[]
          p_module_name: string
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
      upsert_user_credential: {
        Args: { p_credentials: Json; p_module: string; p_user_id: string }
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
