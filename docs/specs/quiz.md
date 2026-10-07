# quiz

## Goal

`POST /api/quizzes` returns a **Quiz** of up to ten multiple-choice **Questions** about items of one level and section. Each question is anchored to its item ([ADR-0005](../adr/0005-generated-questions-are-anchored-and-dropped-never-repaired.md)), written by an AI provider chain, and never stored ([ADR-0004](../adr/0004-quizzes-are-ai-generated-and-never-stored.md)).

## Non-goals

- No stored quiz, answer, score, or progress. Nothing is written to Postgres.
- No answer submission, streaming, rate limiting, or authentication. A provider is never asked twice for one quiz.
- No provider other than `openrouter`, `opencode_zen`, and `mock`; the first two share one OpenAI-compatible chat completions adapter. No SDK dependency.
- No change to the dataset, the converter, the seeder, the migrations, or the lesson routes and their bodies.
- No repair of model output: a question that breaks a rule is dropped.

## Request

`POST /api/quizzes` with a JSON object body of at most 64 KiB. Unknown keys are rejected.

| Key | Required | Rule |
|---|---|---|
| `level` | yes | `n5`, `n4`, `n3`, `n2`, or `n1` |
| `section` | yes | `kanji`, `vocabulary`, or `grammar` |
| `lang` | yes | `en` or `id`; the language of meaning choices and explanations |
| `lesson_ids` | no | lesson UUIDs, each a lesson of `level` and `section`; absent or `[]` means the whole section |
| `types` | no | question types of `section`, below; absent or `[]` means all of them |
| `count` | no | 1 to 10, default 5 |

## Question types

A target is one (item, type) pair of the scope whose item qualifies for the type. Up to `count` distinct targets are sampled uniformly at random, and the provider is asked for one question per target.

| Section | Type | Item qualifies when | Prompt | Answer | Model writes |
|---|---|---|---|---|---|
| kanji | `meaning` | always | `character` | `meaning_<lang>` | distractors, explanation |
| kanji | `reading` | `examples` not empty | a word from `examples`, chosen by the model | its reading, by the model | prompt, answer, distractors, explanation |
| vocabulary | `meaning` | always | `word` | `meaning_<lang>` | distractors, explanation |
| vocabulary | `reading` | `word` has a kanji and `reading` is kana only | `word` | `reading` | distractors, explanation |
| vocabulary | `usage` | `word` has kana or a kanji, and none of `／`, `（`, `(`, `〜` | a sentence with `＿＿`, by the model | `word` | prompt, distractors, explanation |
| grammar | `meaning` | always | `pattern` | `meaning_<lang>` | distractors, explanation |
| grammar | `usage` | always | a sentence with `＿＿`, by the model | the filled form, by the model | prompt, answer, distractors, explanation |

Prompt and answer cells naming a column are written by the server from the item's row; the model's value for them is ignored.

### Rules

A target gets a question only when the model returns exactly one entry for its code and type, and that entry passes every rule of its type. Entries for anything other than a target are ignored.

- Exactly 3 distractors. Each is non-empty after trimming, and the 3 distractors and the answer are 4 distinct strings.
- The explanation is non-empty.
- `reading`: the answer and every distractor are kana only (hiragana, katakana, and `ー`).
- kanji `reading`: the prompt is one of the item's `examples`.
- `usage`: the prompt holds `＿＿` exactly once, and the answer and every distractor hold Japanese script.
- vocabulary `usage`: the prompt does not contain `word`.
- grammar `usage`: the answer is non-empty.

Every dropped entry is logged with its code, type, and the rule it broke.

## Response

200 with `{"questions": [Question], "dropped": n}`, where `dropped` is the number of targets without a question. A Question is `{"code", "type", "prompt", "choices", "answer", "explanation"}`: `choices` are the answer and the 3 distractors in random order, and `answer` is the index of the correct choice. Questions follow target order.

### Errors

| Status | Code | Message | When |
|---|---|---|---|
| 400 | `invalid_body` | `Body must be a JSON object with known keys.` | not JSON, not an object, an unknown key, a wrong value type, or over 64 KiB |
| 400 | `invalid_level` | `Level must be n5, n4, n3, n2, or n1.` | |
| 400 | `invalid_section` | `Section must be kanji, vocabulary, or grammar.` | |
| 400 | `invalid_lang` | `Lang must be en or id.` | |
| 400 | `invalid_type` | `Every type must be a question type of the section.` | |
| 400 | `invalid_count` | `Count must be 1 to 10.` | |
| 400 | `invalid_lesson` | `Every lesson must be a lesson of the level and section.` | a lesson id that is not a UUID, names no lesson, or names one of another level or section |
| 404 | `items_not_found` | `No item matches the request.` | the scope has no target |
| 502 | `quiz_generation_failed` | `Quiz generation failed.` | the provider call fails, its reply is not the expected JSON, or no target gets a question |
| 503 | `quiz_unavailable` | `Quiz provider is not configured.` | no listed provider is usable; answered before the body is read |
| 504 | `quiz_generation_timeout` | `Quiz generation timed out.` | the provider chain outlasts `AI_TIMEOUT_SECONDS` |

Validation runs in the table's order and answers the first failure. A 502 and a 504 log the cause; their bodies never contain it.

## Providers

`AI_PROVIDERS` lists provider names, comma-separated, in fall-through order.

- `openrouter` and `opencode_zen` are asked through `POST <server URL>/chat/completions` with the model, a system and a user message, `response_format: json_object`, and `Authorization: Bearer <API key>`. OpenRouter is also sent `HTTP-Referer` and `X-OpenRouter-Title`. The reply's first choice content, with a markdown code fence around it removed, is read as `{"questions": [Draft]}`.
- A listed provider without its API key or model is skipped with a startup warning naming it; an API key starting with `<` is a placeholder and counts as missing. No usable provider leaves quizzes answering 503.
- A transport error, a 5xx, or a 402, 408, 409, 425, or 429 falls through to the next provider. Any other failure stops the chain, and the logged cause joins every provider's failure.
- `mock` runs alone: listed with another provider, it stops startup. It is deterministic, makes no network call, and gives every target a question that passes its rules, with fixed distractors and explanation.

The user message holds `lang` and, per target, its code, type, and the item's fields without its ids, sequence, sources, and meaning in the other language. The model is sent no other item.

## Configuration

Read once at startup, as in [api](api.md). An invalid value stops startup with an error naming the variable.

| Variable | Default | Meaning |
|---|---|---|
| `AI_PROVIDERS` | empty | `openrouter`, `opencode_zen`, `mock`, comma-separated, each at most once; empty leaves quizzes answering 503, and `.env.example` sets `mock` |
| `AI_TIMEOUT_SECONDS` | `20` | 1 to 20; bounds one quiz's whole provider chain, so a quiz in flight ends within the 25s shutdown drain |
| `OPENROUTER_API_KEY`, `OPENROUTER_MODEL`, `OPENROUTER_SERVER_URL` | model `openrouter/free`, URL `https://openrouter.ai/api/v1` | OpenRouter |
| `OPENROUTER_APP_TITLE`, `OPENROUTER_HTTP_REFERER` | `Nihongo Foundation`, `http://127.0.0.1:8080` | sent as `X-OpenRouter-Title` and `HTTP-Referer` |
| `OPENCODE_ZEN_API_KEY`, `OPENCODE_ZEN_MODEL`, `OPENCODE_ZEN_SERVER_URL` | URL `https://opencode.ai/zen/v1` | OpenCode Zen; a leading `opencode/` on the model is dropped |

A server URL of a listed provider must be an absolute `http` or `https` URL. The quiz route extends its own write deadline to `AI_TIMEOUT_SECONDS` plus 5s; every other route keeps the 10s write timeout.

## Acceptance criteria

- AC-1: Handler tests answer each 400 code for its case, in the order of the error table, and 404 `items_not_found` for a scope with no target, such as vocabulary `reading` on a lesson whose words are all kana.
- AC-2: Target tests prove sampling yields only qualifying (item, type) pairs of the requested lessons and types, never repeats a pair, and returns `min(count, targets)` of them.
- AC-3: Handler tests prove a server-written prompt and answer come from the row whatever the model sent, and each rule under Rules drops an entry that breaks it, with `dropped` counting the targets left without a question.
- AC-4: In every question, `choices` has 4 distinct strings and `choices[answer]` is the answer.
- AC-5: A failing provider and one with no usable entry answer 502 `quiz_generation_failed`, a provider outlasting `AI_TIMEOUT_SECONDS` answers 504 `quiz_generation_timeout`, no usable provider answers 503 `quiz_unavailable`, and none of these bodies contains the cause.
- AC-6: The chat adapter, against an `httptest` server, posts to `<server URL>/chat/completions` with the model, both messages, `json_object`, the bearer header, and its extra headers; it parses a valid reply with or without a code fence, and errors on a non-2xx status, an empty choice, and content that is not the expected JSON, marking only transport errors, 5xx, and 402, 408, 409, 425, 429 as falling through.
- AC-13: A chain falls through to the next provider on a falling-through failure, stops on any other, and answers the first success; the API built from `AI_PROVIDERS` skips a listed provider without key or model and asks the next one.
- AC-7: A quiz whose provider takes longer than the server's write timeout but less than `AI_TIMEOUT_SECONDS` still answers 200.
- AC-8: Startup with `AI_PROVIDERS=openai`, `AI_PROVIDERS=openrouter,openrouter`, `AI_PROVIDERS=openrouter,mock`, `AI_PROVIDERS=openrouter OPENROUTER_SERVER_URL=localhost`, `AI_TIMEOUT_SECONDS=abc`, `AI_TIMEOUT_SECONDS=0`, or `AI_TIMEOUT_SECONDS=21` exits non-zero naming the variable.
- AC-9: Repository integration tests prove the scope query returns exactly the items of the requested level, section, and lessons from `data/`, and rejects a lesson of another level or section.
- AC-10: Against the seeded `.env` database with `AI_PROVIDERS=mock`, a quiz for each section of n5 answers 200 with 5 questions and `dropped` 0, and a request with `lesson_ids` asks only about items of those lessons.
- AC-11: The route test passes with `POST /api/quizzes` in `docs/openapi.yaml`, whose error code enum lists every code above.
- AC-12: Code comments and `docs/openapi.yaml` descriptions are short, direct, passive statements of the point; judged by `rubric`.

## Verification

```bash
set -a; . ./.env; set +a; B=127.0.0.1:$APP_PORT
go vet ./... && go vet -tags integration ./...
go test ./...                                                                    # AC-1 to AC-8, AC-11, AC-13
make test-integration                                                            # AC-9, on the local test_nihongo_foundation database, never the .env one
make api AI_PROVIDERS=mock                                                       # serves the quiz route with the mock provider
for s in kanji vocabulary grammar; do curl -s -X POST $B/api/quizzes -d '{"level":"n5","section":"'$s'","lang":"id"}' | jq '{n: (.questions | length), dropped}'; done   # AC-10, 5 and 0 each
id=$(jq -r '[.[] | select(.section == "kanji")][0].id' data/n5/lessons.json); curl -s -X POST $B/api/quizzes -d '{"level":"n5","section":"kanji","lang":"en","lesson_ids":["'$id'"]}' | jq -r '.questions[].code'   # AC-10, codes of that lesson only
for v in AI_PROVIDERS=openai AI_PROVIDERS=openrouter,openrouter AI_PROVIDERS=openrouter,mock AI_TIMEOUT_SECONDS=abc AI_TIMEOUT_SECONDS=0 AI_TIMEOUT_SECONDS=21; do make api $v; done; make api AI_PROVIDERS=openrouter OPENROUTER_SERVER_URL=localhost   # AC-8, each exits non-zero
rubric: comments and openapi descriptions of this branch                         # AC-12
```
