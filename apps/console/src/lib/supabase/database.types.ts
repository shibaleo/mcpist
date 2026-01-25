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
          disabled_tools: Json
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
    }
    Enums: Record<string, never>
    CompositeTypes: Record<string, never>
  }
}
