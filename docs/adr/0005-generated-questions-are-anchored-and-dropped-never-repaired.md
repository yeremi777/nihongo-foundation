# Generated questions are anchored to their item and dropped, never repaired

A model's question can be wrong, and the learner cannot tell. Every **Question** is built for one item the server picked; wherever that item's row holds the fact a question type asks for, the server writes it, and the model writes only what the row lacks: distractors, an explanation, and, for some types, a sentence or reading. Model output that breaks a rule for its type is dropped, and the quiz reports how many were dropped. Nothing is rewritten to pass.

## Considered options

- **Repair, as jlpt-foundation did.** Rejected: about 1,100 lines for kanji alone rebuilt choices, swapped in dataset distractors, and fell back to hardcoded kana, so a broken model answer reached the learner looking like a good one.
- **The model writes every field and the server compares the answer to the row.** Rejected: a reworded meaning such as "ahead, before" for "previous, ahead, before" fails the comparison, so correct questions are dropped for wording the server already holds.
- **The model writes every field and only its shape is checked.** Rejected: a wrong answer passes unnoticed.

## Consequences

A quiz can hold fewer **Questions** than requested. Kanji reading and grammar usage answers are not in any column, so they stay unchecked beyond their form.
