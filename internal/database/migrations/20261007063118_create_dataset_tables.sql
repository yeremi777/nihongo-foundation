-- +goose Up
CREATE TABLE lesson (
    id            uuid PRIMARY KEY,
    level         text NOT NULL CHECK (level IN ('n5', 'n4', 'n3', 'n2', 'n1')),
    section       text NOT NULL CHECK (section IN ('kanji', 'vocabulary', 'grammar')),
    week          int  NOT NULL CHECK (week > 0),
    day           int  NOT NULL CHECK (day > 0),
    title         text NOT NULL,
    title_en      text NOT NULL,
    title_id      text NOT NULL,
    week_title    text NOT NULL,
    week_title_en text NOT NULL,
    week_title_id text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (level, section, week, day)
);

CREATE TABLE kanji (
    id         uuid PRIMARY KEY,
    level      text NOT NULL CHECK (level IN ('n5', 'n4', 'n3', 'n2', 'n1')),
    code       text NOT NULL,
    lesson_id  uuid NOT NULL REFERENCES lesson (id),
    sequence   int  NOT NULL CHECK (sequence > 0),
    character  text NOT NULL CHECK (char_length(character) = 1),
    onyomi     text[] NOT NULL,
    kunyomi    text[] NOT NULL,
    examples   text[] NOT NULL,
    meaning_en text NOT NULL,
    meaning_id text NOT NULL,
    sources    text[] NOT NULL CHECK (
        cardinality(sources) > 0
        AND sources <@ ARRAY['soumatome', 'shinkanzen', 'minna-no-nihongo']::text[]
    ),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (level, code),
    UNIQUE (level, character),
    UNIQUE (lesson_id, sequence)
);

CREATE TABLE vocabulary (
    id             uuid PRIMARY KEY,
    level          text NOT NULL CHECK (level IN ('n5', 'n4', 'n3', 'n2', 'n1')),
    code           text NOT NULL,
    lesson_id      uuid NOT NULL REFERENCES lesson (id),
    sequence       int  NOT NULL CHECK (sequence > 0),
    word           text NOT NULL,
    reading        text NOT NULL,
    part_of_speech text NOT NULL,
    note           text,
    meaning_en     text NOT NULL,
    meaning_id     text NOT NULL,
    sources        text[] NOT NULL CHECK (
        cardinality(sources) > 0
        AND sources <@ ARRAY['soumatome', 'shinkanzen', 'minna-no-nihongo']::text[]
    ),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (level, code),
    UNIQUE (lesson_id, sequence)
);

CREATE TABLE grammar (
    id              uuid PRIMARY KEY,
    level           text NOT NULL CHECK (level IN ('n5', 'n4', 'n3', 'n2', 'n1')),
    code            text NOT NULL,
    lesson_id       uuid NOT NULL REFERENCES lesson (id),
    sequence        int  NOT NULL CHECK (sequence > 0),
    curriculum_code text NOT NULL,
    pattern         text NOT NULL,
    reading         text,
    formula         text,
    note            text,
    example         text,
    difficulty      text NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    meaning_en      text NOT NULL,
    meaning_id      text NOT NULL,
    sources         text[] NOT NULL CHECK (
        cardinality(sources) > 0
        AND sources <@ ARRAY['soumatome', 'shinkanzen', 'minna-no-nihongo']::text[]
    ),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (level, code),
    UNIQUE (lesson_id, sequence)
);

CREATE TABLE grammar_comparison (
    id         uuid PRIMARY KEY,
    lesson_id  uuid NOT NULL REFERENCES lesson (id),
    sequence   int  NOT NULL CHECK (sequence > 0),
    pattern_a  text NOT NULL,
    pattern_b  text NOT NULL,
    difference text NOT NULL,
    use_when   text NOT NULL,
    example    text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (lesson_id, sequence)
);

CREATE TABLE grammar_mistake (
    id         uuid PRIMARY KEY,
    lesson_id  uuid NOT NULL REFERENCES lesson (id),
    sequence   int  NOT NULL CHECK (sequence > 0),
    point      text NOT NULL,
    incorrect  text NOT NULL,
    correct    text NOT NULL,
    meaning_en text NOT NULL,
    meaning_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (lesson_id, sequence)
);

-- +goose Down
DROP TABLE grammar_mistake;
DROP TABLE grammar_comparison;
DROP TABLE grammar;
DROP TABLE vocabulary;
DROP TABLE kanji;
DROP TABLE lesson;
