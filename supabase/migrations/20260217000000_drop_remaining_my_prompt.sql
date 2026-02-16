-- Drop remaining upsert_my_prompt with 5-arg signature (TEXT, TEXT, TEXT, UUID, BOOLEAN)
-- This was missed in the previous migration because the DROP used a 6-arg signature.
DROP FUNCTION IF EXISTS public.upsert_my_prompt(TEXT, TEXT, TEXT, UUID, BOOLEAN);
DROP FUNCTION IF EXISTS mcpist.upsert_my_prompt(TEXT, TEXT, TEXT, UUID, BOOLEAN);
