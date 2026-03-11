#!/bin/bash



set -e







psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL



    CREATE USER auth_user WITH PASSWORD 'auth123456';

    CREATE DATABASE auth_db;

    GRANT ALL PRIVILEGES ON DATABASE auth_db TO auth_user;

    \c auth_db

    GRANT ALL ON SCHEMA public TO auth_user;



    CREATE USER video_user WITH PASSWORD 'video123456';

    CREATE DATABASE video_db;

    GRANT ALL PRIVILEGES ON DATABASE video_db TO video_user;

    \c video_db

    GRANT ALL ON SCHEMA public TO video_user;



    CREATE USER processing_user WITH PASSWORD 'processing123456';

    CREATE DATABASE processing_db;

    GRANT ALL PRIVILEGES ON DATABASE processing_db TO processing_user;

    \c processing_db

    GRANT ALL ON SCHEMA public TO processing_user;



    CREATE USER status_user WITH PASSWORD 'status123456';

    CREATE DATABASE status_db;

    GRANT ALL PRIVILEGES ON DATABASE status_db TO status_user;

    \c status_db

    GRANT ALL ON SCHEMA public TO status_user;



    CREATE USER notification_user WITH PASSWORD 'notification123456';

    CREATE DATABASE notification_db;

    GRANT ALL PRIVILEGES ON DATABASE notification_db TO notification_user;

    \c notification_db

    GRANT ALL ON SCHEMA public TO notification_user;



EOSQL