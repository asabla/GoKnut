-- PostgreSQL initialization script for GoKnut
-- This script runs when the Postgres container is first created

-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Grant permissions (already done by default, but explicit is good)
GRANT ALL PRIVILEGES ON DATABASE goknut TO goknut;

-- Note: The application will run migrations automatically on startup.
-- This file is for any additional PostgreSQL-specific initialization.
