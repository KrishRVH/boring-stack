-- +goose Up
ALTER TABLE todos
  DROP CONSTRAINT todos_body_trim_length_check,
  ADD CONSTRAINT todos_body_max_length_check
  CHECK (length(trim(body)) <= 280);

-- +goose Down
ALTER TABLE todos
  DROP CONSTRAINT todos_body_max_length_check,
  ADD CONSTRAINT todos_body_trim_length_check
  CHECK (length(trim(body)) > 0 AND char_length(trim(body)) <= 280);
