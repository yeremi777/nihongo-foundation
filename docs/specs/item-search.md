# item-search

## Goal

`GET /api/kanji`, `GET /api/vocabulary`, and `GET /api/grammar` search a section's **Items** with filters and pages, and `GET /api/<section>/{level}/{code}` returns one **Item** by its **Code** ([ADR-0006](../adr/0006-items-are-addressed-by-level-and-code.md)).

## Non-goals

- No search across sections. Each route reads one section.
- No `part_of_speech` or `lesson_id` filter, and no paging of `/api/lessons`.
- No change to the lesson and quiz routes or their bodies.
- No change to the dataset, the converter, the seeder, or the migrations. No index is added.
- No CORS, authentication, or rate limiting.

## Routes

| Method and path | 200 body | Errors |
|---|---|---|
| `GET /api/kanji` | Page of Kanji | 400 `invalid_level`, `invalid_week`, `invalid_source`, `invalid_query`, `invalid_page`, `invalid_size` |
| `GET /api/vocabulary` | Page of Vocabulary item | as `/api/kanji` |
| `GET /api/grammar` | Page of Grammar point | as `/api/kanji`, and `invalid_difficulty` after `invalid_source` |
| `GET /api/kanji/{level}/{code}` | Kanji | 400 `invalid_level`, 404 `item_not_found` |
| `GET /api/vocabulary/{level}/{code}` | Vocabulary item | as above |
| `GET /api/grammar/{level}/{code}` | Grammar point | as above |

An item is its table row, exactly as lesson detail returns it ([api](api.md)). A Page is `{"items": [item], "page", "size", "total"}`, where `total` counts every match and `items` holds the matches of the requested page. `total` and `items` are read in one snapshot.

### Query parameters

An absent or empty parameter is not applied. Unknown parameters are ignored, so `difficulty` on `/api/kanji` and `/api/vocabulary` has no effect.

| Parameter | Routes | Rule |
|---|---|---|
| `level` | all lists | `n5`, `n4`, `n3`, `n2`, or `n1` |
| `week` | all lists | a positive integer, given with `level`; the lesson's week |
| `source` | all lists | `soumatome`, `shinkanzen`, or `minna-no-nihongo`; one of the item's sources |
| `difficulty` | grammar | `easy`, `medium`, or `hard` |
| `q` | all lists | trimmed of whitespace, then UTF-8 text without NUL of at most 100 characters |
| `page` | all lists | a positive integer, default 1; a page past the last answers `items: []` |
| `size` | all lists | 1 to 100, default 20 |

Validation runs in the table's order and answers the first failure.

### Matching

`q` matches an item when it is a substring of one of its fields below, or of one element of an array field. Both sides are folded first: ASCII letters are lowercased and katakana become hiragana, so `せん` matches `セン` and `SAKI` matches `saki`. Every character in `q`, `%` and `_` included, matches only itself.

| Section | Fields |
|---|---|
| kanji | `character`, `onyomi`, `kunyomi`, `examples`, `meaning_en`, `meaning_id` |
| vocabulary | `word`, `reading`, `meaning_en`, `meaning_id` |
| grammar | `pattern`, `reading`, `meaning_en`, `meaning_id` |

Items are ordered by exact match first, when `q` is given and a folded field or element equals the folded `q`; then by level from `n5`, the lesson's week and day, and `sequence`.

### Lookup

`{level}` is checked like the `level` parameter. A `{code}` that names no item of that level and section answers `item_not_found`.

### Errors

| Status | Code | Message |
|---|---|---|
| 400 | `invalid_level` | `Level must be n5, n4, n3, n2, or n1.` |
| 400 | `invalid_week` | `Week must be a positive integer, given with level.` |
| 400 | `invalid_source` | `Source must be soumatome, shinkanzen, or minna-no-nihongo.` |
| 400 | `invalid_difficulty` | `Difficulty must be easy, medium, or hard.` |
| 400 | `invalid_query` | `Query must be valid text of at most 100 characters.` |
| 400 | `invalid_page` | `Page must be a positive integer.` |
| 400 | `invalid_size` | `Size must be 1 to 100.` |
| 404 | `item_not_found` | `Item was not found.` |

A store error answers 500 `internal_error`, as in [api](api.md).

## Acceptance criteria

- AC-1: Handler tests answer each 400 code of a list route for its case, in the order of the error table, including `week` without `level` and `difficulty=hard` ignored on `/api/kanji`.
- AC-2: Handler tests prove `page` defaults to 1 and `size` to 20, and a lookup with `level=n9` answers 400 `invalid_level`.
- AC-3: Repository integration tests prove, for each section, that `level`, `week`, `source`, and grammar `difficulty` return exactly the matching rows of `data/`, in level, week, day, and `sequence` order, with `total` counting them all and `page` and `size` slicing them.
- AC-4: Repository integration tests prove `q` matching: `せん` finds the n5 kanji `先` by its onyomi `セン`, `先生` finds `先` by its examples, an uppercase romaji `q` finds a grammar point by its `reading`, `%` matches no item that lacks it, and an item that equals `q` comes before one that only contains it.
- AC-5: Repository integration tests prove a lookup returns the row of that level and code, and answers not found for an unknown code and for a code of that section that exists only in another level.
- AC-6: A page past the last answers 200 with `items: []` and the full `total`.
- AC-7: The route test passes with the six routes in `docs/openapi.yaml`, whose error code enum lists every code above.
- AC-8: The existing lesson and quiz tests pass unchanged.
- AC-9: Against the seeded `.env` database, `GET /api/vocabulary?level=n3&q=か` answers within 50 ms.
- AC-10: Code comments and `docs/openapi.yaml` descriptions are short, direct, passive statements of the point, in the style of the existing ones; judged by `rubric`.

## Verification

```bash
set -a; . ./.env; set +a; B=127.0.0.1:$APP_PORT
go vet ./... && go vet -tags integration ./...
go test ./...                                                                     # AC-1, AC-2, AC-7, AC-8
make test-integration                                                             # AC-3 to AC-6, AC-8, on the local test_nihongo_foundation database, never the .env one
make api                                                                          # demo: serves the item routes
curl -s "$B/api/kanji?level=n5&q=せん" | jq '{total, first: .items[0].character}'          # 先 among the matches
curl -s "$B/api/grammar?level=n5&week=1&difficulty=easy&size=5" | jq '{total, n: (.items | length)}'
curl -s "$B/api/kanji/n5/k101-001" | jq .character; curl -s "$B/api/kanji/n5/nope"; curl -s "$B/api/kanji/n9/k101-001"
curl -s "$B/api/vocabulary?page=999" | jq '{total, items}'                        # AC-6
for i in 1 2 3; do curl -s -o /dev/null -w '%{time_total}\n' "$B/api/vocabulary?level=n3&q=か"; done   # AC-9, each under 0.050
rubric: comments and openapi descriptions of this branch                         # AC-10
```
