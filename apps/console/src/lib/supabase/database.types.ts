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
        Returns: void
      }
    }
    Enums: Record<string, never>
    CompositeTypes: Record<string, never>
  }
}
