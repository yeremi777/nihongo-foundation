-- +goose Up
ALTER TABLE grammar_mistake
    DROP COLUMN incorrect,
    DROP COLUMN correct,
    ADD COLUMN lines jsonb NOT NULL CHECK (
        CASE WHEN jsonb_typeof(lines) = 'array' THEN jsonb_array_length(lines) > 0 ELSE false END
    ),
    ALTER COLUMN meaning_en DROP NOT NULL;

CREATE TABLE grammar_expression (
    id         uuid PRIMARY KEY,
    lesson_id  uuid NOT NULL REFERENCES lesson (id),
    sequence   int  NOT NULL CHECK (sequence > 0),
    expression text NOT NULL,
    reading    text,
    meaning_en text NOT NULL,
    meaning_id text NOT NULL,
    note       text,
    example    text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (lesson_id, sequence)
);

-- +goose Down
DROP TABLE grammar_expression;

UPDATE grammar_mistake SET meaning_en = '' WHERE meaning_en IS NULL;

ALTER TABLE grammar_mistake
    DROP COLUMN lines,
    ADD COLUMN incorrect text NOT NULL DEFAULT '',
    ADD COLUMN correct text NOT NULL DEFAULT '',
    ALTER COLUMN meaning_en SET NOT NULL;

ALTER TABLE grammar_mistake
    ALTER COLUMN incorrect DROP DEFAULT,
    ALTER COLUMN correct DROP DEFAULT;
