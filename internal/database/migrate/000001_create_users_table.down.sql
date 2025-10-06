-- internal/database/migrate/000001_create_users_table.down.sql

DROP TRIGGER IF EXISTS set_timestamp ON users;

DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS trigger_set_timestamp();
