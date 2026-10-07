# data-model

## Goal

The curated N5, N4, and N3 source lists are converted into a committed JSON dataset and seeded into Postgres in the data model below, with every row checked against its level's curriculum TOC.

## Non-goals

- No HTTP API, quiz, AI provider, or authentication. Later specs own them.
- nihongo-path and `~/Downloads/<level>` are only read. Nothing writes to them.
- No PDF, scan, progress log, or project-instruction file enters the repository. The snapshot holds the twelve markdown files only.
- The Minna no Nihongo supplemental grammar (`gm01`…, found only in the N5 and N4 TOCs) is not imported.
- Indonesian Learner Notes, Week Review Focus tables, and the TOC's Minna no Nihongo lesson column are not imported.
- No verification-status fields. Every row is human-curated, and AI output is never stored ([ADR-0004](../adr/0004-quizzes-are-ai-generated-and-never-stored.md)).
- The JSON in `data/<level>/` is never edited by hand ([ADR-0001](../adr/0001-source-lists-generate-the-dataset.md)).

## Data model

Every table has `created_at` and `updated_at` (`timestamptz NOT NULL DEFAULT now()`). Every id is a UUIDv5 ([ADR-0002](../adr/0002-deterministic-uuidv5-ids.md)) in the namespace `uuid.NewSHA1(uuid.NameSpaceURL, "https://github.com/yeremi777/nihongo-foundation")`, named as shown per table. Text marked optional is `NULL` when the source cell is empty or `—`; arrays are `'{}'` in that case.

### lesson

| Column | Type | Rule |
|---|---|---|
| `id` | `uuid` PK | name `<level>:lesson:<section>:<week>:<day>` |
| `level` | `text` | `CHECK IN ('n5','n4','n3','n2','n1')` |
| `section` | `text` | `CHECK IN ('kanji','vocabulary','grammar')` |
| `week`, `day` | `int` | `> 0` |
| `title`, `title_en`, `title_id` | `text NOT NULL` | Japanese, English, Indonesian lesson title |
| `week_title`, `week_title_en` | `text NOT NULL` | Japanese, English week title |
| `week_title_id` | `text` optional | Indonesian week title |

`UNIQUE (level, section, week, day)`.

### kanji, vocabulary, grammar

Shared columns, in every item table:

| Column | Type | Rule |
|---|---|---|
| `id` | `uuid` PK | name `<level>:<section>:<code>` |
| `level` | `text` | as `lesson.level`, equal to its lesson's level |
| `code` | `text NOT NULL` | `UNIQUE (level, code)` |
| `lesson_id` | `uuid NOT NULL` | FK `lesson(id)`, indexed; the lesson's section matches the table |
| `sequence` | `int NOT NULL` | 1-based row position within its lesson; `UNIQUE (lesson_id, sequence)` |
| `meaning_en`, `meaning_id` | `text NOT NULL` | |
| `sources` | `text[] NOT NULL` | non-empty, subset of `{soumatome, shinkanzen, minna-no-nihongo}` |

Per-table columns:

| Table | Column | Type |
|---|---|---|
| `kanji` | `character` | `text NOT NULL`, one character, `UNIQUE (level, character)` |
| | `onyomi`, `kunyomi`, `examples` | `text[] NOT NULL` |
| `vocabulary` | `word`, `reading`, `part_of_speech` | `text NOT NULL` |
| | `note` | `text` optional |
| `grammar` | `curriculum_code` | `text NOT NULL` |
| | `pattern` | `text NOT NULL` |
| | `reading`, `formula`, `note`, `example` | `text` optional; `example` is the Japanese sentence only |
| | `difficulty` | `text NOT NULL`, `CHECK IN ('easy','medium','hard')` |

### grammar_comparison, grammar_mistake

| Table | Columns | Id name |
|---|---|---|
| `grammar_comparison` | `id`, `lesson_id`, `sequence`, `pattern_a`, `pattern_b`, `difference`, `use_when`, `example` (all `text NOT NULL`) | `<level>:grammar_comparison:<week>:<day>:<sequence>` |
| `grammar_mistake` | `id`, `lesson_id`, `sequence`, `point`, `incorrect`, `correct`, `meaning_en`, `meaning_id` (all `text NOT NULL`) | `<level>:grammar_mistake:<week>:<day>:<sequence>` |

Both have `lesson_id` FK to a grammar lesson and `UNIQUE (lesson_id, sequence)`. Comparison patterns are kept as written, including the parenthetical after them.

## Conversion rules

- **Input:** `data/source/<level>/<level>_{kanji,vocabulary,grammar}_list.md` and `<level>_curriculum_toc.md`, copied from `~/Downloads/N3`, `N4`, `N5`.
- **Output:** `data/<level>/{lessons,kanji,vocabulary,grammar,grammar_comparisons,grammar_mistakes}.json`. Each file is an array of objects whose keys are the column names above, except the timestamps. Arrays are ordered by lesson week and day, then by `sequence`. Every run writes byte-identical output for the same input.
- **Weeks and lessons** come from the list headings `## Week N — <ja> (<paren>)` and `### Day N — <ja> (<paren>)`. `## Week N Review Focus` is not a week.
  - In grammar lists, `<paren>` holds exactly one ` / `, separating English and Indonesian.
  - In kanji and vocabulary lists, `<paren>` is English only. The Indonesian lesson title is the Indonesian column of the matching TOC day row. The Indonesian week title is the matching TOC week heading's text after ` / `, or `NULL` when the TOC has no heading for that week (N5 kanji and vocabulary).
- **Codes:**
  - A vocabulary or grammar code is the list's `Item ID`.
  - A kanji code is the TOC day code of its lesson plus its 3-digit sequence, for example `k101-001`.
  - `curriculum_code` is the grammar code without its `-<letter>` suffix.
- **Sources:**
  - Kanji and vocabulary are `{soumatome}`.
  - Grammar decodes the TOC `Src` letter of its curriculum code:
    - N5 and N4: `S` → `{soumatome}`, `M` → `{minna-no-nihongo}`, `B` → `{soumatome, minna-no-nihongo}`
    - N3: `S` → `{soumatome}`, `K` → `{shinkanzen}`, `B` → `{soumatome, shinkanzen}`
  - `difficulty` comes from the same TOC row.
- **Part of speech:** trimmed; `／`, `/`, and `;` become ` / `; ASCII letters lowercased; `suru-verb` → `する-verb`, `i-adjective` → `い-adjective`, `na-adjective` and `na-adj` → `な-adjective`; ` · 自動詞` and ` · 他動詞` kept as written.
- **Readings and examples:** kanji `On`, `Kun`, and `Examples` cells split on `、` and `,`; `—` becomes an empty array.
- **Comparison notes and mistake cards:**
  - Comparison notes are the rows of a lesson's `Grammar Comparison Notes` table.
  - Mistake cards are its `grammar-mistake-card` blocks, read from the `Point`, `❌ Incorrect`, `✅ Correct`, `EN`, and `ID` lines.
  - Both are numbered by `sequence` within the lesson.
- **The conversion fails and writes nothing** when any of these holds:
  - a list day has no TOC day, or a TOC day has no list day (kanji and vocabulary);
  - a vocabulary code's `v<week><day>` prefix differs from its lesson;
  - a grammar curriculum code is missing from the TOC, or its `Src` letter is not defined for the level;
  - a code or kanji character repeats within a level;
  - a required cell is empty;
  - a grammar heading's parenthetical does not hold exactly one ` / `.

## Acceptance criteria

- AC-1: `data/source/` holds exactly the twelve markdown files, each byte-identical to its counterpart in `~/Downloads/<level>`.
- AC-2: Conversion produces these row counts:

  | Level | lessons | kanji | vocabulary | grammar | grammar_comparisons | grammar_mistakes |
  |---|---|---|---|---|---|---|
  | n5 | 32 | 108 | 266 | 77 | 90 | 67 |
  | n4 | 63 | 198 | 515 | 136 | 109 | 87 |
  | n3 | 114 | 336 | 1210 | 203 | 184 | 138 |

- AC-3: Running the conversion twice in a row leaves `data/` with no diff.
- AC-4: Unit tests prove each failure case under Conversion rules stops the conversion with an error naming the file and the offending row, and leaves the existing JSON untouched.
- AC-5: Unit tests prove the part-of-speech, source-decoding, heading-parsing, kanji-code, and UUIDv5 rules on fixed inputs, including `Noun; suru-verb` → `noun / する-verb`, `Na-adj/Noun` → `な-adjective / noun`, and N5 `B` versus N3 `B`.
- AC-6: No `part_of_speech` in `data/*/vocabulary.json` contains an uppercase ASCII letter, `／`, `suru`, `i-adjective`, or `na-adj`.
- AC-7: Migrations create the six tables with the columns, checks, foreign keys, and unique constraints above. A full down migration removes them, and `up` after `down` succeeds.
- AC-8: Seeding an empty database loads every JSON row. The table counts match AC-2 summed over the levels: lesson 209, kanji 642, vocabulary 1991, grammar 416, grammar_comparison 383, grammar_mistake 292.
- AC-9: Seeding a second time keeps the same ids and counts. Seeding after a row is removed from the JSON deletes that row. Each seed runs in one transaction, so a failed seed changes nothing.
- AC-10: Every `kanji`, `vocabulary`, and `grammar` row's `lesson_id` points to a lesson of the same level and matching section.

## Verification

```bash
test "$(find data/source -type f | wc -l)" -eq 12                                 # AC-1
for f in data/source/*/*.md; do l=$(basename "$(dirname "$f")" | tr n N); cmp "$f" ~/Downloads/$l/$(basename "$f"); done   # AC-1
go vet ./...
go test ./...                                                                    # AC-4, AC-5
make convert && make convert && git diff --exit-code -- data/                   # AC-2, AC-3
grep -E '"part_of_speech": "[^"]*([A-Z]|／|suru|i-adjective|na-adj)' data/*/vocabulary.json; test $? -eq 1   # AC-6
make up && make down && make up                                                  # AC-7, run by the user
make seed && make seed                                                           # AC-8, AC-9, run by the user
go test -tags integration ./internal/seed/...                                    # AC-9 removed-row and rollback cases, run by the user against a disposable database
DB_DSN="host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USERNAME password=$DB_PASSWORD sslmode=$DB_SSLMODE"   # values from .env
psql "$DB_DSN" -c "SELECT 'lesson', count(*) FROM lesson UNION ALL SELECT 'kanji', count(*) FROM kanji UNION ALL SELECT 'vocabulary', count(*) FROM vocabulary UNION ALL SELECT 'grammar', count(*) FROM grammar UNION ALL SELECT 'grammar_comparison', count(*) FROM grammar_comparison UNION ALL SELECT 'grammar_mistake', count(*) FROM grammar_mistake"   # AC-8
psql "$DB_DSN" -c "SELECT count(*) FROM kanji k JOIN lesson l ON l.id = k.lesson_id WHERE l.section <> 'kanji' OR l.level <> k.level"   # AC-10 returns 0, likewise vocabulary and grammar
```
