# dataset-quiz

## Goal

`POST /api/quizzes/dataset` returns a **Quiz** of up to ten multiple-choice **Questions** about items of one level and section, built from the dataset alone: no AI provider is called, and every prompt, answer, and explanation is written from the item's row ([ADR-0007](../adr/0007-quizzes-come-from-an-ai-provider-or-the-dataset.md)).

## Non-goals

- No change to `POST /api/quizzes`, its providers, its errors, or its spec ([quiz](quiz.md)). A failing AI quiz never falls back to a dataset quiz.
- No `usage` question: the dataset holds no blank sentences.
- No stored quiz, answer, score, or progress. Nothing is written to Postgres.
- No change to the dataset, the converter, the seeder, or the migrations.
- No answer submission, streaming, rate limiting, or authentication.

## Request

`POST /api/quizzes/dataset` with a JSON object body of at most 64 KiB. Unknown keys are rejected. The keys and their rules are those of [quiz](quiz.md#request), except `types`, which holds question types of the section from the table below.

## Question types

A target is one (item, type) pair of the scope whose item qualifies for the type. Up to `count` distinct targets are sampled uniformly at random, as in [quiz](quiz.md#question-types).

| Section | Type | Item qualifies when | Prompt | Answer |
|---|---|---|---|---|
| kanji | `meaning` | always | `character` | `meaning_<lang>` |
| kanji | `reading` | `onyomi` and `kunyomi` are not both empty | `character` | one of `onyomi` and `kunyomi`, chosen at random |
| vocabulary | `meaning` | always | `word` | `meaning_<lang>` |
| vocabulary | `reading` | `word` has a kanji and `reading` is kana only | `word` | `reading` |
| grammar | `meaning` | always | `pattern` | `meaning_<lang>` |

A kanji `reading` asks one reading of the character. The AI quiz asks the reading of an example word under the same type.

### Distractors

The pool is every row of the quiz's level and section, whatever the requested lessons, except the target's item. Three distractors are drawn from the pool in random order. A candidate is skipped when it:

- repeats the answer or an earlier distractor;
- for `meaning`, shares a term with the answer, where a term is the meaning split at `,`, `/`, and `;`, trimmed and lowercased;
- for `reading`, is not kana only (hiragana, katakana, and `ー`);
- for kanji `reading`, is one of the target kanji's own readings.

A kanji `reading` candidate is one reading of another kanji. A target without 3 distractors gets no question.

### Explanation

Written from the row, in no language: `<prompt> · <answer>`; for kanji `reading`, `<character> · 音 <onyomi, comma-separated> · 訓 <kunyomi, comma-separated>`, with an empty group left out.

## Response

200 with `{"questions": [Question], "dropped": n}` exactly as in [quiz](quiz.md#response): `dropped` counts the targets without a question, and a Question's `type` is `meaning` or `reading`. A quiz whose targets all lack 3 distractors answers 200 with no question and `dropped` set; the seeded sections are large enough that it never happens. The questions pass the same rules as an AI quiz's: 4 distinct non-empty choices, a non-empty explanation, and kana-only choices for `reading`.

### Errors

The 400 and 404 codes, messages, and order are those of [quiz](quiz.md#errors), and a type outside the table answers `invalid_type`. There is no 502, 503, or 504. The route answers with `AI_PROVIDERS` empty.

## Acceptance criteria

- AC-1: Handler tests answer each 400 code of the quiz error table for its case, `usage` as `invalid_type`, and 404 `items_not_found` for a scope with no target, such as vocabulary `reading` on a lesson whose words are all kana.
- AC-2: Target tests prove the dataset targets are only the qualifying pairs of the table, never repeat a pair, number `min(count, targets)`, and give a kanji `reading` target an answer that is one of the character's readings.
- AC-3: Distractor tests prove a question's 3 distractors are distinct, come from other rows of the level and section, share no meaning term with the answer, are kana only for `reading`, and are never a kanji's own reading; a target with fewer than 3 candidates is counted in `dropped`.
- AC-4: In every question, the prompt and answer are those of the table whatever the pool holds, `choices` has 4 distinct strings, `choices[answer]` is the answer, and the explanation is the one written above.
- AC-5: The route answers 200 with no provider configured, and `POST /api/quizzes` still answers 503 `quiz_unavailable` in that state.
- AC-6: Repository integration tests prove the pool is every row of the level and section whatever the requested lessons, and a lesson of another level or section is rejected.
- AC-7: Against the seeded `.env` database with `AI_PROVIDERS` empty, a dataset quiz for each section of n5 answers 200 with 5 questions and `dropped` 0, and a request with `lesson_ids` asks only about items of those lessons.
- AC-8: The route test passes with `POST /api/quizzes/dataset` in `docs/openapi.yaml` under the `quizzes` tag, whose description covers both quiz kinds, and with an error code enum listing every code above.
- AC-9: Code comments and `docs/openapi.yaml` descriptions are short, direct, passive statements of the point; judged by `rubric`.

## Verification

```bash
set -a; . ./.env; set +a; B=127.0.0.1:$APP_PORT
go vet ./... && go vet -tags integration ./...
go test ./...                                                                    # AC-1 to AC-5, AC-8
make test-integration                                                            # AC-6, on the local test_nihongo_foundation database, never the .env one
make api AI_PROVIDERS=                                                           # serves the dataset quiz with no provider
for s in kanji vocabulary grammar; do curl -s -X POST $B/api/quizzes/dataset -d '{"level":"n5","section":"'$s'","lang":"id"}' | jq '{n: (.questions | length), dropped}'; done   # AC-7, 5 and 0 each
curl -s -o /dev/null -w '%{http_code}\n' -X POST $B/api/quizzes -d '{"level":"n5","section":"kanji","lang":"id"}'   # AC-5, 503
id=$(jq -r '[.[] | select(.section == "kanji")][0].id' data/n5/lessons.json); curl -s -X POST $B/api/quizzes/dataset -d '{"level":"n5","section":"kanji","lang":"en","lesson_ids":["'$id'"]}' | jq -r '.questions[].code'   # AC-7, codes of that lesson only
rubric: comments and openapi descriptions of this branch                         # AC-9
```
