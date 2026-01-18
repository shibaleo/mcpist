-- auth.users 作成時に mcpist.users も自動作成するトリガー

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO mcpist.users (id, display_name, status, role)
  VALUES (
    NEW.id,
    COALESCE(NEW.raw_user_meta_data->>'name', NEW.email),
    'active',
    'user'
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION mcpist.handle_new_user();

COMMENT ON FUNCTION mcpist.handle_new_user() IS 'auth.users作成時にmcpist.usersにレコードを自動作成';
