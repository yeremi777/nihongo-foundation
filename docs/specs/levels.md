# levels

## Goal

`GET /api/levels` returns each **Level** that has lessons, with its **Sections** and, per section, the number of weeks, lessons, and items, so a client learns the shape of the curriculum in one call.

## Non-goals

- No week titles and no lesson list. [api](api.md) owns `/api/lessons`.
- No query parameter, filter, or paging.
- No level without lessons: n2 and n1 are absent until the dataset holds them.
- No change to the dataset, the converter, the seeder, or the migrations.

## Route

| Method and path | 200 body | Errors |
|---|---|---|
| `GET /api/levels` | array of Level | |

A Level is `{"level", "sections": [Section]}` and a Section is `{"section", "weeks", "lessons", "items"}`, where `weeks` counts distinct weeks of the section in the level, `lessons` its lessons, and `items` its rows of `kanji`, `vocabulary`, or `grammar`. Levels are ordered from `n5`, sections as `kanji`, `vocabulary`, `grammar`. A section without lessons is left out. The counts and the lists are read in one snapshot.

## Acceptance criteria

- AC-1: Repository integration tests prove the counts of each level and section from `data/`, the order of levels and sections, and that a level or section without lessons is absent.
- AC-2: A handler test proves the 200 body shape and an empty array when no lesson exists.
- AC-3: Against the seeded `.env` database, the counts of every level and section equal those computed from `data/<level>/lessons.json`, `kanji.json`, `vocabulary.json`, and `grammar.json`.
- AC-4: The route test passes with `GET /api/levels` in `docs/openapi.yaml` under a new `levels` tag.
- AC-5: Code comments and `docs/openapi.yaml` descriptions are short, direct, passive statements of the point; judged by `rubric`.

## Verification

```bash
set -a; . ./.env; set +a; B=127.0.0.1:$APP_PORT
go vet ./... && go vet -tags integration ./...
go test ./...                                                                    # AC-2, AC-4
make test-integration                                                            # AC-1, on the local test_nihongo_foundation database, never the .env one
make api AI_PROVIDERS=mock
for l in n5 n4 n3; do for s in kanji vocabulary grammar; do
  api=$(curl -s $B/api/levels | jq -c --arg l $l --arg s $s '.[] | select(.level == $l) | .sections[] | select(.section == $s) | [.weeks, .lessons, .items]')
  src=$(jq -c --arg s $s '[[.[] | select(.section == $s) | .week] | unique | length, ([.[] | select(.section == $s)] | length)]' data/$l/lessons.json)
  echo "$l $s $api $src"
done; done                                                                       # AC-3, [weeks,lessons,items] against [weeks,lessons]; items against the row counts of data/<level>/<section>.json
rubric: comments and openapi descriptions of this branch                         # AC-5
```
