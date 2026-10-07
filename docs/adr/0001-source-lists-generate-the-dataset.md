# Source lists generate the dataset

The source lists are curated outside this project and are also rendered by the nihongo-path site, so the database cannot be where the content is edited. A snapshot of the lists in `data/source/` is converted into JSON in `data/`, which is committed and never edited by hand, and the seeder loads only that JSON. A fix to the data goes into the source lists or the converter's rules, never into the JSON or the database, because the next conversion overwrites both.

## Considered options

- **Database as the editable source, seeded once.** Rejected: edits would diverge from the lists the site renders, with no way to reconcile them.
- **Seeder parses the markdown directly.** Rejected: a parser bug would reach the database with no reviewable artifact in between. The committed JSON shows every change to the data as a diff before it is seeded.
