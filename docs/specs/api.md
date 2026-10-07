# api

## Goal

`cmd/api` serves the seeded dataset over HTTP: lessons filtered by level and section, and one lesson with its items, documented by `docs/openapi.yaml` and browsable in Swagger UI.

## Non-goals

- No quiz, AI provider, authentication, CORS, or pagination. Later specs own them.
- No write route, and no route outside the table below.
- No change to the dataset, the converter, the seeder, or the migrations.
- No dataset loading at request time. The API reads only Postgres.

## Routes

| Method and path | 200 body | Errors |
|---|---|---|
| `GET /health` | `{"status":"ok"}` | |
| `GET /api/lessons?level=&section=` | array of Lesson | 400 `invalid_level`, 400 `invalid_section` |
| `GET /api/lessons/{lesson_id}` | Lesson detail | 404 `lesson_not_found` |
| `GET /docs`, `GET /docs/`, `GET /docs/index.html` | Swagger UI's standalone page rendering `docs/openapi.yaml`; its top bar shows the Swagger logo on the left and the dark-mode toggle in the right corner, without the Explore box, and the page starts dark when the operating system prefers dark | |
| `GET /docs/openapi.yaml` | the spec file as written, `application/yaml` | |
| `GET /` | 302 redirect to `/docs` | |

Every route of a resource starts with `/api/<resource>`, so every lesson route starts with `/api/lessons`; filters are query parameters.

`docs/openapi.yaml` is the contract for every route and body except the docs routes and redirect above. It declares no servers, so the docs page calls OpenAPI's default `/`, the address it was opened on, and shows no servers picker. Its descriptions are short and direct: one point each, no restating the route, and none about errors. It is written by hand, and a test fails when a registered route other than those is missing from it, or a path in it is not registered.

### Bodies

Keys are the column names of the data model ([data-model](data-model.md)), without the timestamps, exactly as in `data/<level>/*.json`.

- **Lesson:** `{"id","level","section","week","day","title","title_en","title_id","week_title","week_title_en","week_title_id"}`.
- **Lesson detail:** `{"lesson": Lesson, "<section>": [item]}`, where `<section>` is the lesson's section (`kanji`, `vocabulary`, or `grammar`) and each item is a row of that table. A grammar lesson also has `"comparisons"`, `"mistakes"`, and `"expressions"`, each an array of the lesson's `grammar_comparison`, `grammar_mistake`, and `grammar_expression` rows, empty when it has none. A kanji or vocabulary lesson has none of these three keys.
- **Error:** `{"error":{"code","message"}}`.

Every JSON body is written by `encoding/json`'s `Encoder` with its defaults, so it ends in a newline.

### Lessons and items

- `level` is one of `n5`, `n4`, `n3`, `n2`, `n1`, and `section` one of `kanji`, `vocabulary`, `grammar`, matched exactly. Both are optional; an absent or empty one matches every value, and they combine with AND. A filter no lesson matches answers `[]`.
- Lessons are ordered by level from `n5` to `n1`, section as kanji, vocabulary, grammar, then week, then day. Items, comparisons, mistakes, and expressions are ordered by `sequence`.
- A `lesson_id` that is not a UUID, or names no lesson, answers `lesson_not_found`.

### Errors

| Status | Code | Message |
|---|---|---|
| 400 | `invalid_level` | `Level must be n5, n4, n3, n2, or n1.` |
| 400 | `invalid_section` | `Section must be kanji, vocabulary, or grammar.` |
| 404 | `lesson_not_found` | `Lesson was not found.` |
| 404 | `not_found` | `Route was not found.` |
| 405 | `method_not_allowed` | `Method is not allowed on this route.` |
| 500 | `internal_error` | `Internal server error.` |

A 500 logs the underlying error. Its body never contains it.

## Configuration

Read once at startup from the environment, which `make` fills from `.env`. A missing required value stops startup with `<VAR> is not set; copy .env.example to .env`, and an unparseable one with an error naming the variable.

| Variable | Default | Meaning |
|---|---|---|
| `DB_HOST`, `DB_NAME`, `DB_USERNAME` | required | Postgres host, database, and user |
| `DB_PORT` | required | Postgres port, 1 to 65535 |
| `DB_PASSWORD` | empty | Postgres password |
| `DB_SSLMODE` | required | Postgres `sslmode` |
| `APP_PORT` | required | listen port, 1 to 65535 |
| `APP_URL` | required | the service's public base URL, an absolute `http` or `https` URL; a port in it must be `APP_PORT`; printed at startup as `url` and `docs` |

## Server

- The server binds `127.0.0.1:APP_PORT`, then logs `api listening url=<APP_URL> docs=<APP_URL>/docs`; a port it cannot bind stops startup with no `api listening` line.
- Startup pings Postgres and fails when it cannot reach it.
- Read-header timeout 5s, read timeout 10s, write timeout 10s, idle timeout 60s.
- `SIGINT` or `SIGTERM` stops accepting connections and lets in-flight requests finish for up to 25s.

## Acceptance criteria

- AC-1: Against the seeded `.env` database, `GET /api/lessons` with `level` and `section` returns the ids of the matching rows of `data/<level>/lessons.json` in week and day order, for every level and section of the dataset; `level` alone returns that level's lessons, no filter returns all 209 lessons in list order, and `level=n2` returns `[]`.
- AC-2: `GET /api/lessons/{lesson_id}` for a kanji, a vocabulary, and a grammar lesson returns the lesson and exactly its rows from `data/<level>/*.json`, in `sequence` order, with the three grammar keys on the grammar lesson only.
- AC-3: `level=n9` and `section=phrases` answer 400 `invalid_level` and `invalid_section`; a non-UUID `lesson_id` and an unknown UUID answer 404 `lesson_not_found`; all in the error body shape.
- AC-4: An unknown path answers 404 `not_found`, and a wrong method on a known path answers 405 `method_not_allowed`, both in the error body shape.
- AC-5: A handler test forces a store error and gets 500 `internal_error` with no detail from the error in the body.
- AC-6: The route test passes: every registered route except the docs routes and redirect is in `docs/openapi.yaml`, and every path in it is registered.
- AC-7: `GET /docs` renders the spec in Swagger UI, and `GET /docs/openapi.yaml` declares no servers.
- AC-8: Repository integration tests pass against a test database seeded from `data/`.
- AC-9: Startup without `DB_HOST` or `APP_URL`, with `APP_PORT=abc`, with `APP_URL=localhost`, or with an `APP_URL` port other than `APP_PORT`, exits non-zero naming the variable. Startup with an unreachable database, or on a port already in use, exits non-zero, and the busy port logs no `api listening` line.
- AC-10: After `SIGTERM`, the process exits 0.

## Verification

```bash
set -a; . ./.env; set +a; B=127.0.0.1:$APP_PORT                                 # the commands below read the .env variables
go vet ./... && go vet -tags integration ./...
go test ./...                                                                     # AC-4, AC-5, AC-6, AC-7 (server)
make test-integration                                                             # AC-8, on the local test_nihongo_foundation database, never the .env one
make api                                                                          # serves on APP_PORT, docs at APP_URL/docs
for l in n5 n4 n3; do for s in kanji vocabulary grammar; do echo "$l $s $(curl -s "$B/api/lessons?level=$l&section=$s" | jq length) $(jq --arg s $s '[.[] | select(.section == $s)] | length' data/$l/lessons.json)"; done; done   # AC-1, the two counts match
curl -s $B/api/lessons | jq length; curl -s "$B/api/lessons?level=n2"   # AC-1, 209 and []
id=$(jq -r '[.[] | select(.section == "grammar")][0].id' data/n5/lessons.json); curl -s $B/api/lessons/$id | jq 'keys, (.grammar | length)'   # AC-2
curl -s "$B/api/lessons?level=n9"; curl -s "$B/api/lessons?section=phrases"; curl -s $B/api/lessons/nope; curl -s $B/api/lessons/00000000-0000-0000-0000-000000000000   # AC-3
curl -s $B/nope; curl -s -X DELETE $B/health             # AC-4
open http://$B/docs; curl -s $B/docs/openapi.yaml | grep -c '^servers:'   # AC-7, 0
DB_HOST= go run ./cmd/api; APP_URL= go run ./cmd/api; APP_PORT=abc go run ./cmd/api; APP_URL=localhost go run ./cmd/api; APP_URL=http://127.0.0.1:1 go run ./cmd/api; DB_PORT=1 go run ./cmd/api   # AC-9, each exits non-zero
kill -TERM <api pid>                                                              # AC-10, exits 0
```
