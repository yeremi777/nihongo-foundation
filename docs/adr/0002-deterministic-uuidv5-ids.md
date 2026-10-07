# Deterministic UUIDv5 ids

Every row's UUID is a version 5 UUID derived from its natural key, such as `n3:kanji:k101-001`, so the same row has the same id on every conversion, on every machine, and in every environment. Random UUIDs were rejected because each fresh seed would hand clients new ids for the same kanji; serial integers were rejected because they depend on insert order and leak row counts. The converter assigns the ids, so the JSON carries them and the seeder never generates one.

## Consequences

Changing a row's natural key (its level, section, or code) changes its id. Codes therefore must not be renumbered once published.
