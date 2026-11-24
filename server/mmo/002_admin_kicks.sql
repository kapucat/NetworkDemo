BEGIN;
CREATE SCHEMA IF NOT EXISTS mmo_admin;
CREATE TABLE IF NOT EXISTS mmo_admin.connection_operations (
 operation_id uuid PRIMARY KEY,
 request jsonb NOT NULL,
 status text NOT NULL CHECK(status IN ('Started','Completed','Changed','TargetsGone')),
 affected_count integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
REVOKE ALL ON SCHEMA mmo_admin FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA mmo_admin FROM PUBLIC;
COMMIT;
