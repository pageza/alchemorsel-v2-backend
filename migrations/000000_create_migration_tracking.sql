-- Create migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create function to check if migration has been applied
CREATE OR REPLACE FUNCTION migration_applied(migration_version VARCHAR)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM schema_migrations WHERE version = migration_version
    );
END;
$$ LANGUAGE plpgsql;

-- Create function to record migration
CREATE OR REPLACE FUNCTION record_migration(migration_version VARCHAR, migration_name VARCHAR)
RETURNS VOID AS $$
BEGIN
    INSERT INTO schema_migrations (version, name) VALUES (migration_version, migration_name);
END;
$$ LANGUAGE plpgsql;

-- Create function to remove migration record (for rollback)
CREATE OR REPLACE FUNCTION remove_migration(migration_version VARCHAR)
RETURNS VOID AS $$
BEGIN
    DELETE FROM schema_migrations WHERE version = migration_version;
END;
$$ LANGUAGE plpgsql; 