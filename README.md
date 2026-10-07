# Nihongo Foundation

Go backend for a JLPT N5 to N3 study platform.

The service converts curated kanji, vocabulary, and grammar source lists into a committed JSON dataset, seeds it from `data/` into Postgres, and serves it over a REST API organized as the Soumatome course runs: by level, section, week, and lesson. Quizzes are written by an AI provider at request time, anchored to dataset items, and never stored.

## Features

- Lesson list API with level and section filters
- Lesson detail API with its kanji, vocabulary, or grammar points, plus comparison notes, mistake cards, and expressions for grammar lessons
- AI quizzes (`POST /api/quizzes`): meaning, reading, and usage questions in English or Indonesian, each checked against its item and dropped when it fails a rule
- AI providers in a fall-through chain (OpenRouter, OpenCode Zen), or an offline `mock`
- Conversion (`make convert`) of the source lists into the committed JSON dataset, checked against each level's curriculum TOC
- Seeding (`make seed`) that makes the tables match `data/` exactly, with deterministic UUIDv5 ids
- Timestamped SQL migrations via goose
- Swagger UI at `/docs`

## Tech Stack

- Go 1.26
- PostgreSQL
- pgx/v5 with hand-written SQL, no ORM
- goose for migrations

## Getting Started

Create `.env` from the example and point `DB_*` at your local Postgres. `AI_PROVIDERS=mock` answers quizzes offline, with no API key.

```bash
cp .env.example .env
```

Install goose, create the database, apply the migrations, and seed the dataset:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
createdb -h 127.0.0.1 -U postgres nihongo_foundation
make migrate-up
make seed
```

Run the API:

```bash
make api
```

The API listens on `APP_PORT` and is reached at `APP_URL`, `http://127.0.0.1:8080` in the example. Swagger UI is at `APP_URL/docs`. A variable given to `make` overrides `.env`:

```bash
make api APP_PORT=8090 APP_URL=http://127.0.0.1:8090
```

Ask for a quiz:

```bash
curl -s -X POST http://127.0.0.1:8080/api/quizzes -d '{"level":"n5","section":"vocabulary","lang":"id","count":5}'
```

To use a real model, set `AI_PROVIDERS` to `openrouter`, `opencode_zen`, or both in fall-through order, with each provider's `*_API_KEY` and `*_MODEL` in `.env`. `make help` lists every target.

## Project Layout

```
cmd/api              REST API binary: lessons, quizzes, and the docs page
cmd/convert          converter binary: data/source/ -> data/<level>/*.json
cmd/seed             seeder binary: data/<level>/*.json -> Postgres
internal/lesson      lessons and their items: repository, handlers
internal/quiz        AI quizzes: targets, rules, providers and the fall-through chain, handler
internal/dataset     parses the source lists and builds the dataset
internal/seed        loads the dataset and writes it to the tables
internal/database    Postgres connections and goose migrations
internal/httpx       JSON responses, errors, routing, docs page
internal/config      environment configuration
docs                 OpenAPI contract, specs, ADRs
data                 committed JSON dataset per level, and the source-list snapshot in data/source
```

## Testing

```bash
make test
```

Integration tests run against `test_nihongo_foundation` on the `.env` Postgres server, rebuilt from the migrations and emptied after every run. Create the test database once, then:

```bash
createdb -h 127.0.0.1 -U postgres test_nihongo_foundation
make test-integration
```

## Documentation

- `CONTEXT.md`: the domain glossary
- `docs/adr/`: architecture decisions
- `docs/specs/`: one spec per feature, with acceptance criteria
- `docs/openapi.yaml`: the API contract the Swagger UI serves

## Notice

This is a personal study project. Not affiliated with or endorsed by the Japan Foundation, JEES, or the publishers of Soumatome, Shinkanzen Master, or Minna no Nihongo.
