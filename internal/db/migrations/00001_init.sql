-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE todos (
  id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
  body text NOT NULL CHECK (length(trim(body)) > 0),
  done boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER todos_touch_updated_at
BEFORE UPDATE ON todos
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

CREATE TABLE app_events (
  id bigserial PRIMARY KEY,
  kind text NOT NULL,
  body text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO todos (body) VALUES
  ('Edit this starter kit'),
  ('Open two browser tabs and watch HTMX SSE swaps'),
  ('Run BUS=nats mise run dev and compare the bus implementation');

INSERT INTO app_events (kind, body) VALUES
  ('seed', 'Database initialized by Goose migration');

-- +goose Down
DROP TABLE IF EXISTS app_events;
DROP TABLE IF EXISTS todos;
DROP FUNCTION IF EXISTS touch_updated_at();
