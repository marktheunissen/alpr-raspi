#  psql -h hostname -U postgres

CREATE TABLE events (
    id serial PRIMARY KEY,
    time timestamptz NOT NULL,
    camera text NOT NULL,
    plate text NOT NULL,
    plate_image text,
    frame_image text,
    site text NOT NULL
);

CREATE INDEX idx_plates ON events(plate);

# Create user + db
CREATE USER metabase_app WITH PASSWORD '';
CREATE DATABASE metabase_app;
GRANT ALL PRIVILEGES ON DATABASE metabase_app to metabase_app;
