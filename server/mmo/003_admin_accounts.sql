BEGIN;
CREATE TABLE IF NOT EXISTS mmo_admin.account_operations (
 operation_id uuid PRIMARY KEY,
 account_id bigint NOT NULL,
 account_backup jsonb NOT NULL,
 sessions_backup jsonb NOT NULL,
 status text NOT NULL CHECK(status IN ('Prepared','Completed','Changed','Connected','Expired','Cancelled')),
 reason text NOT NULL DEFAULT '',
 expires_at timestamptz NOT NULL DEFAULT (now() + interval '10 minutes'),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
REVOKE ALL ON mmo_admin.account_operations FROM PUBLIC;
COMMIT;
