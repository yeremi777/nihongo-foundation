# Nihongo Foundation

The curated Japanese study dataset for JLPT N5 to N3, organized as the Soumatome course runs: by level, section, week, and lesson. It serves the dataset to clients and builds quizzes from it.

## Language

### Curriculum

**Level**:
A JLPT stage: N5, N4, or N3.
_Avoid_: grade

**Section**:
One kind of study material: kanji, vocabulary, or grammar.
_Avoid_: category, type

**Week**:
A numbered, titled group of lessons in one section of one level.
_Avoid_: unit

**Lesson**:
One day of study in a week.
_Avoid_: day, session, topic

### Items

**Kanji**:
One character studied in a lesson, with its readings, meaning, and example words.

**Vocabulary item**:
One word or phrase studied in a lesson.
_Avoid_: word, vocab

**Grammar point**:
One pattern studied in a lesson.
_Avoid_: grammar item, sub-item

**Code**:
An item's stable identifier within its level, such as `k101-001`, `v101-001`, or `g101-a`.
_Avoid_: item ID

**Curriculum code**:
The table-of-contents identifier a grammar point belongs to; `g101-a` and `g101-b` share `g101`.
_Avoid_: parent ID

**Comparison note**:
Two patterns from one lesson set side by side, with when to use each.

**Mistake card**:
An incorrect sentence and its correction, with the meaning in English and Indonesian.

### Sources

**Source**:
A textbook an item comes from: Soumatome, Shinkanzen, or Minna no Nihongo.
_Avoid_: src, reference

**Source list**:
The per-level, per-section markdown list that defines the items.
_Avoid_: master list, reference list

**Curriculum TOC**:
The per-level table of contents that the source lists are checked against.

### Practice

**Quiz**:
A set of AI-generated questions built from items, never stored.

## Relationships

- A **Level** has one or more **Weeks** in each **Section**; a **Week** has one or more **Lessons**
- A **Lesson** belongs to exactly one **Section** and holds the items of that section only
- Every **Kanji**, **Vocabulary item**, and **Grammar point** belongs to exactly one **Lesson**
- A **Code** is unique within its **Level**; the same **Code** can appear in another **Level**
- A **Curriculum code** groups one or more **Grammar points**, which may sit in different **Lessons**
- A **Comparison note** and a **Mistake card** belong to exactly one grammar **Lesson**
- Every item has one or more **Sources**
- A **Source list** defines the items; the **Curriculum TOC** only checks them and fills what the list lacks
- A **Quiz** draws on items of one **Level** and **Section**

## Example dialogue

> **Dev:** "g112-a is in Day 4 and g112-b is in Day 5. Is that one **Grammar point** split over two **Lessons**?"
> **Curator:** "No. They are two **Grammar points** with one **Curriculum code**, g112. Each belongs to its own **Lesson**."
>
> **Dev:** "The N5 TOC lists gm01 to gm14 under Minna no Nihongo. Are those **Grammar points**?"
> **Curator:** "Not yet. Only rows in a **Source list** are items. The **Curriculum TOC** cannot add one."

## Flagged ambiguities

- "day" meant both a **Lesson** and its position number in the week. Resolved: the thing is a **Lesson**; "day" is only its number.
- "Item ID" in the source lists is the **Code**. Resolved: **Code** is the term; the database also gives each item a UUID, which is not a domain term.
- The TOC letter `B` means Soumatome with Minna no Nihongo in N5 and N4, but Soumatome with Shinkanzen in N3. Resolved: a **Source** is always a named textbook, never a letter.
- "sub-item" was used for `g101-a`. Resolved: `g101-a` is a **Grammar point** in its own right; `g101` is its **Curriculum code**.
