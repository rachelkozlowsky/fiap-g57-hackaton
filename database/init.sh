#!/bin/sh
set -e

# Create Users and Databases
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE USER auth_user WITH PASSWORD 'auth123456';
    CREATE USER video_user WITH PASSWORD 'video123456';
    CREATE USER processing_user WITH PASSWORD 'processing123456';
    CREATE USER notification_user WITH PASSWORD 'notification123456';
    CREATE USER status_user WITH PASSWORD 'status123456';

    CREATE DATABASE auth_db OWNER auth_user;
    CREATE DATABASE video_db OWNER video_user;
    CREATE DATABASE processing_db OWNER processing_user;
    CREATE DATABASE notification_db OWNER notification_user;
    CREATE DATABASE status_db OWNER status_user;
EOSQL

# Apply Schemas
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "auth_db" -f /docker-entrypoint-initdb.d/init-scripts/auth-schema.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "video_db" -f /docker-entrypoint-initdb.d/init-scripts/video-schema.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "processing_db" -f /docker-entrypoint-initdb.d/init-scripts/processing-schema.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "notification_db" -f /docker-entrypoint-initdb.d/init-scripts/notification-schema.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "status_db" -f /docker-entrypoint-initdb.d/init-scripts/status-schema.sql

# Grant Permissions
for db in auth_db video_db processing_db notification_db status_db; do
    user="${db%_db}_user"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" <<-EOSQL
        GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $user;
        GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $user;
        GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $user;
EOSQL
done
