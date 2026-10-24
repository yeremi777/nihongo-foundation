# Quizzes come from an AI provider or from the dataset

A quiz is written by an AI provider at `POST /api/quizzes`, or built from the dataset alone at `POST /api/quizzes/dataset`. The service now has clients that bundle no content, so they cannot build a quiz themselves, and every AI quiz costs money, needs a key, and can fail. The dataset quiz needs none of that and cannot ask a wrong question, because its prompt, answer, and explanation come from the item's row. This supersedes the AI-only part of [ADR-0004](0004-quizzes-are-ai-generated-and-never-stored.md); a quiz is still never stored.

## Considered options

- **A `source` field on `POST /api/quizzes`.** Rejected: the 503 check runs before the body is read, and one error table would cover two different failure sets.
- **Fall back to a dataset quiz when the AI providers fail.** Rejected: a client could not tell which quiz it got, and the failure would be hidden.
- **Meaning questions only.** Rejected: reading is held in the row for vocabulary and kanji, so it can be asked and checked exactly.

## Consequences

The dataset quiz is one more writer of drafts behind the rules of [ADR-0005](0005-generated-questions-are-anchored-and-dropped-never-repaired.md), so there is one place that builds and checks a question. It asks no `usage` question, because the dataset holds no blank sentences; only the AI quiz asks it.
