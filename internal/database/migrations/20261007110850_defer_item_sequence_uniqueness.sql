-- +goose Up
-- Vocabulary and grammar keep their id when their row moves within a lesson,
-- so a re-seed that swaps two rows needs sequence uniqueness checked at commit.
ALTER TABLE vocabulary
    DROP CONSTRAINT vocabulary_lesson_id_sequence_key,
    ADD CONSTRAINT vocabulary_lesson_id_sequence_key UNIQUE (lesson_id, sequence) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE grammar
    DROP CONSTRAINT grammar_lesson_id_sequence_key,
    ADD CONSTRAINT grammar_lesson_id_sequence_key UNIQUE (lesson_id, sequence) DEFERRABLE INITIALLY IMMEDIATE;

-- +goose Down
ALTER TABLE vocabulary
    DROP CONSTRAINT vocabulary_lesson_id_sequence_key,
    ADD CONSTRAINT vocabulary_lesson_id_sequence_key UNIQUE (lesson_id, sequence);

ALTER TABLE grammar
    DROP CONSTRAINT grammar_lesson_id_sequence_key,
    ADD CONSTRAINT grammar_lesson_id_sequence_key UNIQUE (lesson_id, sequence);
