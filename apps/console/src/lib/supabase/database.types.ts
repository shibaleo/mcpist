export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export interface Database {
  public: {
    Tables: Record<string, never>
    Views: Record<string, never>
    Functions: {
      get_user_context: {
        Args: {
          p_user_id: string
        }
        Returns: {
          account_status: string
          free_credits: number
          paid_credits: number
          enabled_modules: string[]
          enabled_tools: Json
          language: string
          module_descriptions: Json
        }[]
      }
      list_service_connections: {
        Args: Record<string, never>
        Returns: {
          id: string
          service: string
          created_at: string
          updated_at: string
        }[]
      }
      upsert_service_token: {
        Args: {
          p_service: string
          p_credentials: Json
        }
        Returns: void
      }
      delete_service_token: {
        Args: {
          p_service: string
        }
        Returns: {
          deleted: boolean
        }
      }
      get_user_role: {
        Args: Record<string, never>
        Returns: string
      }
      list_api_keys: {
        Args: Record<string, never>
        Returns: {
          id: string
          name: string
          key_prefix: string
          last_used_at: string | null
          expires_at: string | null
          created_at: string
          is_expired: boolean
        }[]
      }
      generate_api_key: {
        Args: {
          p_name: string
          p_expires_in_days?: number | null
        }
        Returns: {
          id: string
          name: string
          key: string
          key_prefix: string
          expires_at: string | null
        }
      }
      revoke_api_key: {
        Args: {
          p_key_id: string
        }
        Returns: void
      }
      get_service_token: {
        Args: {
          p_service: string
        }
        Returns: Json
      }
      list_oauth_consents: {
        Args: Record<string, never>
        Returns: {
          id: string
          client_id: string
          client_name: string | null
          scopes: string
          granted_at: string
        }[]
      }
      revoke_oauth_consent: {
        Args: {
          p_consent_id: string
        }
        Returns: {
          revoked: boolean
        }
      }
      list_all_oauth_consents: {
        Args: Record<string, never>
        Returns: {
          id: string
          user_id: string
          user_email: string | null
          client_id: string
          client_name: string | null
          scopes: string
          granted_at: string
        }[]
      }
      list_oauth_apps: {
        Args: Record<string, never>
        Returns: {
          provider: string
          redirect_uri: string
          enabled: boolean
          has_credentials: boolean
          client_id: string | null
          created_at: string
          updated_at: string
        }[]
      }
      upsert_oauth_app: {
        Args: {
          p_provider: string
          p_client_id: string
          p_client_secret: string
          p_redirect_uri: string
          p_enabled?: boolean
        }
        Returns: {
          success: boolean
          action: string
          provider: string
        }
      }
      delete_oauth_app: {
        Args: {
          p_provider: string
        }
        Returns: {
          success: boolean
          provider?: string
          error?: string
          message?: string
        }
      }
      get_oauth_app_credentials: {
        Args: {
          p_provider: string
        }
        Returns: {
          provider?: string
          client_id?: string
          client_secret?: string
          redirect_uri?: string
          error?: string
          message?: string
        }
      }
      get_my_tool_settings: {
        Args: {
          p_module_name?: string | null
        }
        Returns: {
          module_name: string
          tool_id: string
          enabled: boolean
        }[]
      }
      upsert_my_tool_settings: {
        Args: {
          p_module_name: string
          p_enabled_tools: string[]
          p_disabled_tools: string[]
        }
        Returns: Json
      }
      get_my_module_descriptions: {
        Args: Record<string, never>
        Returns: {
          module_name: string
          description: string
        }[]
      }
      upsert_my_module_description: {
        Args: {
          p_module_name: string
          p_description: string
        }
        Returns: Json
      }
      get_my_preferences: {
        Args: Record<string, never>
        Returns: {
          language: string
        }[]
      }
      upsert_my_preferences: {
        Args: {
          p_language?: string | null
        }
        Returns: Json
      }
      // Onboarding
      complete_onboarding: {
        Args: {
          p_user_id: string
          p_event_id: string
        }
        Returns: {
          success: boolean
          already_completed?: boolean
          credits_granted?: number
          status?: string
          error?: string
          message?: string
        }
      }
      // Stripe integration
      get_stripe_customer_id: {
        Args: {
          p_user_id: string
        }
        Returns: {
          stripe_customer_id: string | null
        }
      }
      link_stripe_customer: {
        Args: {
          p_user_id: string
          p_stripe_customer_id: string
        }
        Returns: {
          success: boolean
          stripe_customer_id?: string
          error?: string
        }
      }
      add_credits: {
        Args: {
          p_user_id: string
          p_amount: number
          p_credit_type: string  // 'free' or 'paid'
          p_event_id: string
        }
        Returns: {
          success: boolean
          credit_type?: string
          free_credits?: number
          paid_credits?: number
          added?: number
          error?: string
          message?: string
        }
      }
      get_user_by_stripe_customer: {
        Args: {
          p_stripe_customer_id: string
        }
        Returns: string | null
      }
    }
    Enums: Record<string, never>
    CompositeTypes: Record<string, never>
  }
}
