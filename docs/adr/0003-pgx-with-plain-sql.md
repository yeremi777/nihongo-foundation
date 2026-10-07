# pgx with plain SQL

The Go service standard this project follows uses GORM. Here the database work is six tables, one bulk upsert in the seeder, and a handful of filtered reads, so repositories use `pgx/v5` with SQL written by hand and scan rows with `pgx.RowToStructByName`. GORM was rejected because every column would be declared twice, in the Goose migration and on a model struct, with nothing keeping the two in step, and because `text[]` columns would need an extra array type; sqlc was rejected because a code generator costs more than it saves at about ten queries. Goose remains the migration tool.

## Consequences

SQL errors surface at run time, not at compile time, so every repository query needs a test against a real Postgres.
