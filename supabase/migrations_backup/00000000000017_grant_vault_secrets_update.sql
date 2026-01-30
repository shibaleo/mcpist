-- Grant UPDATE permission on vault.secrets to postgres
-- Required for update_module_token SECURITY DEFINER function to update tokens
-- The function runs as postgres owner, but postgres lacks UPDATE permission on vault.secrets by default

GRANT UPDATE ON vault.secrets TO postgres;
