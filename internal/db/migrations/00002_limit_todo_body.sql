-- +goose Up
ALTER TABLE todos
  ADD CONSTRAINT todos_body_trim_length_check
  CHECK (length(trim(body)) > 0 AND char_length(trim(body)) <= 280);

-- +goose Down
ALTER TABLE todos
  DROP CONSTRAINT IF EXISTS todos_body_trim_length_check;
