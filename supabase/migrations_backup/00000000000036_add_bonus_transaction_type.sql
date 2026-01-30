-- =============================================================================
-- Migration: Add 'bonus' to credit_transaction_type enum
-- Description: Add bonus type for free credit grants (signup bonus, etc.)
-- =============================================================================

ALTER TYPE mcpist.credit_transaction_type ADD VALUE IF NOT EXISTS 'bonus';
