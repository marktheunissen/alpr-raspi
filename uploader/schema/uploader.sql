
# Run this as the LPR user otherwise perms will be wrong.
# psql -h localhost -U lpr -d lpr
CREATE TABLE last_seen (
    plate text PRIMARY KEY,
    time timestamptz NOT NULL
);
