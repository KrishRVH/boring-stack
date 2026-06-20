-- name: ListTodos :many
SELECT id, body, done, created_at, updated_at
FROM todos
ORDER BY created_at DESC;

-- name: CreateTodo :one
INSERT INTO todos (body)
VALUES (trim(sqlc.arg(body)))
RETURNING id, body, done, created_at, updated_at;

-- name: ToggleTodo :one
UPDATE todos
SET done = NOT done
WHERE id = sqlc.arg(id)
RETURNING id, body, done, created_at, updated_at;

-- name: DeleteTodo :one
DELETE FROM todos
WHERE id = sqlc.arg(id)
RETURNING id, body, done, created_at, updated_at;

-- name: CountTodos :one
SELECT
  count(*)::bigint AS total,
  count(*) FILTER (WHERE done)::bigint AS done
FROM todos;

-- name: InsertEvent :one
INSERT INTO app_events (kind, body)
VALUES (sqlc.arg(kind), sqlc.arg(body))
RETURNING id, kind, body, created_at;

-- name: ListEvents :many
SELECT id, kind, body, created_at
FROM app_events
ORDER BY id DESC
LIMIT sqlc.arg(limit_rows)::int;
