# kanji-coverage

## Goal

`POST /api/kanji/coverage` takes a Japanese text and answers its **Kanji coverage**: each kanji of the text with its **Kanji** item, and the share of the text's kanji a learner who finished each **Level** has studied.

## Non-goals

- No word splitting, readings, or vocabulary and grammar matching. Only kanji are counted, and no dependency is added.
- No "studied up to a week or lesson" scope. Coverage is per **Level** only.
- No stored text, result, or progress. Nothing is written to Postgres.
- No change to the other item, lesson, level, and quiz routes or their bodies.
- No change to the dataset, the converter, the seeder, or the migrations. No index is added.
- No CORS, authentication, or rate limiting.

## Request

`POST /api/kanji/coverage` with a JSON object body of at most 64 KiB. Unknown keys are rejected.

| Key | Required | Rule |
|---|---|---|
| `text` | yes | trimmed of whitespace, then 1 to 10000 characters |

## Response

200 with `{"total", "levels": [Level coverage], "kanji": [Text kanji]}`, the summary before the detail.

- A kanji is a character of the Han script in Unicode category Lo, so `々` and `〇` are not kanji. `total` counts every kanji in the text, repeats included.
- A Text kanji is `{"character", "count", "item"}`, one per distinct kanji, in order of first appearance. `count` is its occurrences in the text. `item` is the **Kanji** row whose `character` is it, exactly as `GET /api/kanji/{level}/{code}` returns it, or `null` when no **Level** has it. A kanji held by several **Levels** takes the row of the lowest, from `n5`.
- A Level coverage is `{"level", "known", "known_percent"}`, one per **Level** holding kanji, from `n5`. `known` counts the occurrences of kanji whose `item` is of that **Level** or one below it. `known_percent` is `known * 100 / total` rounded down, so it is 100 only when every kanji is known, and 100 when `total` is 0.

### Errors

| Status | Code | Message |
|---|---|---|
| 400 | `invalid_body` | `Body must be a JSON object with known keys.` |
| 400 | `invalid_text` | `Text must be 1 to 10000 characters.` |

Validation runs in the table's order and answers the first failure. A store error answers 500 `internal_error`, as in [api](api.md).

## Acceptance criteria

- AC-1: Handler tests answer 400 `invalid_body` for a non-object body, an unknown key, trailing data, and a body over 64 KiB, and 400 `invalid_text` for a missing, blank, and 10001-character `text`; a 10000-character `text` answers 200.
- AC-2: A handler test answers `今日は日曜日。降る々` with `total` 6; Text kanji `今`, `日` (count 3), `曜`, `降` in that order, `item` `null` for a kanji the store lacks; and Level coverage whose `known` adds each lower **Level** and whose `known_percent` rounds down.
- AC-3: A handler test answers a text without kanji with `total` 0, `kanji` `[]`, and `known_percent` 100 at every **Level**.
- AC-4: A handler test forces a store error and gets 500 `internal_error` with no detail from the error in the body.
- AC-5: Repository integration tests prove the store returns, for the given characters, exactly the rows of `data/` holding them, from `n5` and then by **Code**, and the **Levels** that have kanji: `n5`, `n4`, `n3`.
- AC-6: The route test passes with `POST /api/kanji/coverage` in `docs/openapi.yaml`, whose error code enum lists `invalid_text`.
- AC-7: The existing item, lesson, level, and quiz tests pass unchanged.
- AC-8: Code comments and `docs/openapi.yaml` descriptions are short, direct, passive statements of the point, in the style of the existing ones; judged by `rubric`.

## Verification

```bash
B=127.0.0.1:8080                                                                  # APP_PORT from .env
go vet ./... && go vet -tags integration ./...
go test ./...                                                                     # AC-1 to AC-4, AC-6, AC-7
make test-integration                                                             # AC-5, AC-7, on the local test_nihongo_foundation database, never the .env one
make api                                                                          # demo: serves the coverage route
curl -s -X POST $B/api/kanji/coverage -d '{"text":"昨日、友だちと駅の近くのレストランで晩ごはんを食べました。"}' | jq '{total, levels, kanji: [.kanji[] | {character, count, level: .item.level}]}'
curl -s -X POST $B/api/kanji/coverage -d '{"text":"   "}'; curl -s -X POST $B/api/kanji/coverage -d '{"txt":"日"}'
rubric: comments and openapi descriptions of this branch                         # AC-8
```
