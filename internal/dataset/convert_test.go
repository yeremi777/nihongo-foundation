package dataset

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const fixtureTOC = `# N3 Curriculum

## 文法 GRAMMAR — Soumatome N3 (6 Weeks · 6 Days each)

### Week 1 — がんばらなくちゃ！ *(I have to stick at it! / Aku harus terus berusaha!)*

| ID | Day | Pattern | Reading | English | Indonesian | Formula | Diff | Src |
|----|-----|---------|---------|---------|------------|---------|------|-----|
| g101 | 1 | 〜れる・〜られる | reru/rareru | Passive | Bentuk pasif | V+れる | hard | B |
| g102 | 1 | 〜させる | saseru | Causative | Kausatif | V+させる | medium | K |

## 語彙 VOCABULARY

### 📖 Soumatome N3 — 6 Weeks

#### Week 1 — 家事をしましょう *(Let's Do Some Housework / Ayo Lakukan Pekerjaan Rumah)*

| ID | Day | Topic | English | Indonesian |
|----|-----|-------|---------|------------|
| v101 | 1 | キッチンで | In the kitchen | Di dapur |

## 漢字 KANJI

### 📖 Soumatome N3 — 6 Weeks

#### Week 1 — でかける① *(Go Out (1) / Keluar ①)*

| ID | Day | Topic | English | Indonesian |
|----|-----|-------|---------|------------|
| k101 | 1 | 駐車場 | Parking lot | Tempat parkir |

## 📊 Summary
`

const fixtureKanji = `# N3 Kanji Master List — Soumatome

---

## Week 1 — でかける① (Going Out 1)

### Day 1 — 駐車場 (Parking Lot)
| Kanji | On | Kun | English | Indonesian | Examples |
|-------|----|-----|---------|------------|---------|
| 駐 | チュウ | — | park vehicles | parkir kendaraan | 駐車、駐車場 |
| 向 | コウ | むこう、むかう | direction | arah | 方向、向かう |
`

const fixtureVocabulary = `# N3 Vocabulary Master List — Soumatome

## Week 1 — 家事をしましょう (Let’s Do Some Housework)

### Day 1 — キッチンで／リビングで (In the kitchen / in the living room)
| Item ID | Word | Reading | Part of speech | English | Indonesian | Usage/Notes |
| ------- | ---- | ------- | -------------- | ------- | ---------- | ----------- |
| v101-001 | キッチン | キッチン | Noun | kitchen | dapur | — |
| v101-002 | 水が凍る | みずがこおる | phrase · 自動詞 | water freezes | air membeku | 凍る uses が |
`

const fixtureGrammar = `# N3 Grammar Master List — Soumatome

---
## Week 1 — がんばらなくちゃ！ (I have to stick at it! / Aku harus terus berusaha!)

### Day 1 — ぼくにもやらせて (Let me do it / Izinkan aku juga melakukannya)

| Item ID | Pattern | Reading | English | Indonesian | Formula | Usage/Notes | Example |
| ------- | ------- | ------- | ------- | ---------- | ------- | ----------- | ------- |
| g101-a | 〜れている | rete iru | Passive state | Keadaan pasif | V(passive)+ている | Doer unknown. | 説明は書かれていません。 |
| g102 | 〜させる | saseru | Causative | Kausatif | Vない+させる | — | 野菜を食べさせました。 |

#### Grammar Comparison Notes

| Pattern A | Pattern B | Main Difference | Use This When | Example |
| --------- | --------- | --------------- | ------------- | ------- |
| 〜れている（g101-a） | 〜させる（g102） | Passive vs causative | Use g101-a for facts. | 書かれています。／食べさせました。 |

#### Common Mistake Cards

<div class="grammar-mistake-card">
<p><strong>Point:</strong> Asking permission</p>
<p><strong>❌ Incorrect:</strong> 手を洗ってください。</p>
<p><strong>✅ Correct:</strong> 手を洗わせてください。</p>
<p><strong>EN:</strong> Please let me wash my hands.</p>
<p><strong>ID:</strong> Tolong izinkan saya mencuci tangan.</p>
</div>

<div class="grammar-mistake-card">
<p><strong>Point:</strong> 〜ないかなあ often means hope</p>
<p><strong>❌ Misread:</strong> バス、早く来ないかなあ。= The bus will not come soon.</p>
<p><strong>✅ Correct meaning:</strong> I hope the bus comes soon.</p>
<p><strong>ID:</strong> Semoga busnya cepat datang.</p>
</div>

<div class="grammar-mistake-card">
<p><strong>Point:</strong> For する, both するべき and すべき are possible</p>
<p><strong>✅ Correct:</strong> 勉強するべきです。</p>
<p><strong>✅ Correct:</strong> 勉強すべきです。</p>
<p><strong>EN:</strong> You should study.</p>
<p><strong>ID:</strong> Kamu seharusnya belajar.</p>
</div>

#### Question Word Notes

| Expression | Reading | English | Indonesian | Usage/Notes | Example |
| ---------- | ------- | ------- | ---------- | ----------- | ------- |
| だれ | dare | who | siapa | Basic “who.” | あの人はだれですか。 |
| 何 | なに／なん | what | apa | — | 何の本ですか。 |

#### Potential Form Notes

| Verb Type | Rule | Example | English | Indonesian |
| --------- | ---- | ------- | ------- | ---------- |
| Group 1 | う-row → え-row + る | 書く → 書ける | can write | bisa menulis |

#### Indonesian Learner Note

In Indonesian, "seperti" can cover many meanings.

---

## Week 1 Review Focus

| Focus Area | Related IDs | What to Review |
| ---------- | ----------- | -------------- |
| Passive | g101 | Review the passive. |
`

func fixtureFiles() LevelFiles {
	return LevelFiles{
		TOC:        File{Name: "n3_curriculum_toc.md", Text: fixtureTOC},
		Kanji:      File{Name: "n3_kanji_list.md", Text: fixtureKanji},
		Vocabulary: File{Name: "n3_vocabulary_list.md", Text: fixtureVocabulary},
		Grammar:    File{Name: "n3_grammar_list.md", Text: fixtureGrammar},
	}
}

func TestConvertLevel(t *testing.T) {
	ds, err := ConvertLevel("n3", fixtureFiles())
	if err != nil {
		t.Fatal(err)
	}

	if got := len(ds.Lessons); got != 3 {
		t.Fatalf("lessons = %d, want 3", got)
	}
	kanjiLesson := ds.Lessons[0]
	want := Lesson{
		ID: rowID("n3:lesson:kanji:1:1"), Level: "n3", Section: SectionKanji, Week: 1, Day: 1,
		Title: "駐車場", TitleEN: "Parking Lot", TitleID: "Tempat parkir",
		WeekTitle: "でかける①", WeekTitleEN: "Going Out 1", WeekTitleID: ptr("Keluar ①"),
	}
	assertLesson(t, kanjiLesson, want)
	grammarLesson := ds.Lessons[2]
	assertLesson(t, grammarLesson, Lesson{
		ID: rowID("n3:lesson:grammar:1:1"), Level: "n3", Section: SectionGrammar, Week: 1, Day: 1,
		Title: "ぼくにもやらせて", TitleEN: "Let me do it", TitleID: "Izinkan aku juga melakukannya",
		WeekTitle: "がんばらなくちゃ！", WeekTitleEN: "I have to stick at it!", WeekTitleID: ptr("Aku harus terus berusaha!"),
	})
	if got := ds.Lessons[1].TitleEN; got != "In the kitchen / in the living room" {
		t.Errorf("vocabulary title_en = %q, want the whole parenthetical", got)
	}

	if len(ds.Kanji) != 2 {
		t.Fatalf("kanji = %d, want 2", len(ds.Kanji))
	}
	k := ds.Kanji[0]
	if k.Code != "k101-001" || k.ID != rowID("n3:kanji:k101-001") || k.LessonID != kanjiLesson.ID || k.Sequence != 1 {
		t.Errorf("kanji identity = %s %s %s %d", k.Code, k.ID, k.LessonID, k.Sequence)
	}
	if !slices.Equal(k.Onyomi, []string{"チュウ"}) || len(k.Kunyomi) != 0 || k.Kunyomi == nil || !slices.Equal(k.Examples, []string{"駐車", "駐車場"}) {
		t.Errorf("kanji readings = %#v %#v %#v", k.Onyomi, k.Kunyomi, k.Examples)
	}
	if !slices.Equal(k.Sources, []Source{SourceSoumatome}) {
		t.Errorf("kanji sources = %v", k.Sources)
	}
	if ds.Kanji[1].Code != "k101-002" {
		t.Errorf("second kanji code = %s, want k101-002", ds.Kanji[1].Code)
	}

	v := ds.Vocabulary[0]
	if v.PartOfSpeech != "noun" || v.Note != nil || v.Code != "v101-001" {
		t.Errorf("vocabulary = %+v", v)
	}
	if got := ds.Vocabulary[1].Note; got == nil || *got != "凍る uses が" {
		t.Errorf("vocabulary note = %v", got)
	}

	g := ds.Grammar[0]
	if g.Code != "g101-a" || g.CurriculumCode != "g101" || g.Difficulty != DifficultyHard {
		t.Errorf("grammar = %+v", g)
	}
	if !slices.Equal(g.Sources, []Source{SourceSoumatome, SourceShinkanzen}) {
		t.Errorf("grammar B sources = %v, want soumatome and shinkanzen", g.Sources)
	}
	if g2 := ds.Grammar[1]; !slices.Equal(g2.Sources, []Source{SourceShinkanzen}) || g2.Note != nil {
		t.Errorf("grammar K = %+v", g2)
	}

	c := ds.GrammarComparisons
	if len(c) != 1 || c[0].PatternA != "〜れている（g101-a）" || c[0].LessonID != grammarLesson.ID ||
		c[0].ID != rowID("n3:grammar_comparison:1:1:1") {
		t.Errorf("comparisons = %+v", c)
	}
	m := ds.GrammarMistakes
	if len(m) != 3 {
		t.Fatalf("mistakes = %d, want 3", len(m))
	}
	if m[0].ID != rowID("n3:grammar_mistake:1:1:1") || m[0].MeaningEN == nil || *m[0].MeaningEN != "Please let me wash my hands." ||
		m[0].MeaningID != "Tolong izinkan saya mencuci tangan." {
		t.Errorf("first mistake = %+v", m[0])
	}
	wantLines := [][]MistakeLine{
		{{VerdictIncorrect, "Incorrect", "手を洗ってください。"}, {VerdictCorrect, "Correct", "手を洗わせてください。"}},
		{{VerdictIncorrect, "Misread", "バス、早く来ないかなあ。= The bus will not come soon."}, {VerdictCorrect, "Correct meaning", "I hope the bus comes soon."}},
		{{VerdictCorrect, "Correct", "勉強するべきです。"}, {VerdictCorrect, "Correct", "勉強すべきです。"}},
	}
	for i, want := range wantLines {
		if !slices.Equal(m[i].Lines, want) {
			t.Errorf("mistake %d lines = %+v, want %+v", i+1, m[i].Lines, want)
		}
	}
	if m[1].MeaningEN != nil {
		t.Errorf("mistake without an EN line has meaning_en %q, want nil", *m[1].MeaningEN)
	}

	e := ds.GrammarExpressions
	if len(e) != 2 || e[0].Expression != "だれ" || e[0].MeaningEN != "who" || e[0].MeaningID != "siapa" ||
		e[0].LessonID != grammarLesson.ID || e[0].ID != rowID("n3:grammar_expression:1:1:1") || e[1].Note != nil {
		t.Errorf("expressions = %+v", e)
	}
}

func TestConvertLevelDecodesSourceLettersPerLevel(t *testing.T) {
	files := fixtureFiles()
	files.TOC.Text = strings.Replace(files.TOC.Text, "| hard | B |", "| hard | B | L8 |", 1)
	files.TOC.Text = strings.Replace(files.TOC.Text, "| Diff | Src |", "| Diff | Src | MNN |", 1)
	files.TOC.Text = strings.Replace(files.TOC.Text, "| medium | K |", "| medium | S | — |", 1)
	files.TOC.Text = strings.Replace(files.TOC.Text, "|------|-----|", "|------|-----|-----|", 1)

	ds, err := ConvertLevel("n5", files)
	if err != nil {
		t.Fatal(err)
	}
	if got := ds.Grammar[0].Sources; !slices.Equal(got, []Source{SourceSoumatome, SourceMinnaNoNihongo}) {
		t.Errorf("n5 B = %v, want soumatome and minna-no-nihongo", got)
	}
}

func TestConvertLevelFailures(t *testing.T) {
	tests := []struct {
		name     string
		file     func(*LevelFiles) *File
		old, new string
		wantFile string
		wantMsg  string
	}{
		{
			name: "list day without TOC day",
			file: func(f *LevelFiles) *File { return &f.Kanji },
			old:  "方向、向かう |", new: "方向、向かう |\n\n### Day 2 — 横断歩道 (Pedestrian Crossing)\n| Kanji | On | Kun | English | Indonesian | Examples |\n|---|---|---|---|---|---|\n| 横 | オウ | よこ | side | samping | 横断 |",
			wantFile: "n3_kanji_list.md", wantMsg: "no curriculum TOC day",
		},
		{
			name: "TOC day without list day",
			file: func(f *LevelFiles) *File { return &f.TOC },
			old:  "| k101 | 1 | 駐車場 | Parking lot | Tempat parkir |", new: "| k101 | 1 | 駐車場 | Parking lot | Tempat parkir |\n| k102 | 2 | 横断歩道 | Crossing | Penyeberangan |",
			wantFile: "n3_curriculum_toc.md", wantMsg: "no lesson in the kanji list",
		},
		{
			name: "vocabulary code from another lesson",
			file: func(f *LevelFiles) *File { return &f.Vocabulary },
			old:  "| v101-002 |", new: "| v102-002 |",
			wantFile: "n3_vocabulary_list.md", wantMsg: "does not belong to lesson v101",
		},
		{
			name: "grammar curriculum code missing from TOC",
			file: func(f *LevelFiles) *File { return &f.Grammar },
			old:  "| g102 |", new: "| g103 |",
			wantFile: "n3_grammar_list.md", wantMsg: "curriculum code g103 is not in the curriculum TOC",
		},
		{
			name: "source letter not defined for the level",
			file: func(f *LevelFiles) *File { return &f.TOC },
			old:  "| medium | K |", new: "| medium | M |",
			wantFile: "n3_curriculum_toc.md", wantMsg: `source letter "M" is not defined for n3`,
		},
		{
			name: "duplicate code",
			file: func(f *LevelFiles) *File { return &f.Vocabulary },
			old:  "| v101-002 |", new: "| v101-001 |",
			wantFile: "n3_vocabulary_list.md", wantMsg: "code v101-001 repeats",
		},
		{
			name: "duplicate kanji character",
			file: func(f *LevelFiles) *File { return &f.Kanji },
			old:  "| 向 |", new: "| 駐 |",
			wantFile: "n3_kanji_list.md", wantMsg: "kanji 駐 repeats",
		},
		{
			name: "required cell empty",
			file: func(f *LevelFiles) *File { return &f.Kanji },
			old:  "| direction |", new: "| — |",
			wantFile: "n3_kanji_list.md", wantMsg: "English is empty",
		},
		{
			name: "unknown table in a grammar lesson",
			file: func(f *LevelFiles) *File { return &f.Grammar },
			old:  "| Verb Type | Rule | Example | English | Indonesian |", new: "| Verb Group | Rule | Example | English | Indonesian |",
			wantFile: "n3_grammar_list.md", wantMsg: "unexpected table in a grammar lesson",
		},
		{
			name: "mistake card without a correct sentence",
			file: func(f *LevelFiles) *File { return &f.Grammar },
			old:  "<p><strong>✅ Correct:</strong> 手を洗わせてください。</p>\n", new: "",
			wantFile: "n3_grammar_list.md", wantMsg: "mistake card has no ✅ sentence",
		},
		{
			name: "mistake card with an unknown label",
			file: func(f *LevelFiles) *File { return &f.Grammar },
			old:  "<strong>Point:</strong> Asking permission", new: "<strong>Topic:</strong> Asking permission",
			wantFile: "n3_grammar_list.md", wantMsg: `unknown mistake card label "Topic"`,
		},
		{
			name: "grammar heading without one separator",
			file: func(f *LevelFiles) *File { return &f.Grammar },
			old:  "(Let me do it / Izinkan aku juga melakukannya)", new: "(Let me do it)",
			wantFile: "n3_grammar_list.md", wantMsg: `exactly one " / "`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := fixtureFiles()
			f := tt.file(&files)
			if !strings.Contains(f.Text, tt.old) {
				t.Fatalf("fixture %s lacks %q", f.Name, tt.old)
			}
			f.Text = strings.Replace(f.Text, tt.old, tt.new, 1)

			_, err := ConvertLevel("n3", files)
			if err == nil {
				t.Fatal("conversion succeeded, want an error")
			}
			located := regexp.MustCompile(`^` + regexp.QuoteMeta(tt.wantFile) + `:\d+: `)
			if !located.MatchString(err.Error()) || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %q, want %s:<line>: ...%s", err, tt.wantFile, tt.wantMsg)
			}
		})
	}
}

func TestRunLeavesExistingJSONOnFailure(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	writeFixtureLevel(t, src, "n3", strings.Replace(fixtureKanji, "| direction |", "| — |", 1))
	existing := filepath.Join(out, "n3", "kanji.json")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("previous\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(src, out); err == nil {
		t.Fatal("Run succeeded, want an error")
	}
	if got, _ := os.ReadFile(existing); string(got) != "previous\n" {
		t.Errorf("kanji.json = %q, want it untouched", got)
	}
}

func TestRunIsDeterministic(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	writeFixtureLevel(t, src, "n3", fixtureKanji)

	if err := Run(src, out); err != nil {
		t.Fatal(err)
	}
	first := readTree(t, out)
	if err := Run(src, out); err != nil {
		t.Fatal(err)
	}
	if second := readTree(t, out); !mapsEqual(first, second) {
		t.Error("second run changed the output")
	}
	for _, name := range []string{"lessons", "kanji", "vocabulary", "grammar", "grammar_comparisons", "grammar_mistakes", "grammar_expressions"} {
		if _, ok := first[filepath.Join("n3", name+".json")]; !ok {
			t.Errorf("missing n3/%s.json", name)
		}
	}
	if !strings.Contains(first[filepath.Join("n3", "kanji.json")], `"kunyomi": []`) {
		t.Error("empty readings must be written as [], not null")
	}
}

func writeFixtureLevel(t *testing.T, dir, level, kanji string) {
	t.Helper()
	files := map[string]string{
		"_curriculum_toc.md":  fixtureTOC,
		"_kanji_list.md":      kanji,
		"_vocabulary_list.md": fixtureVocabulary,
		"_grammar_list.md":    fixtureGrammar,
	}
	if err := os.MkdirAll(filepath.Join(dir, level), 0o755); err != nil {
		t.Fatal(err)
	}
	for suffix, text := range files {
		if err := os.WriteFile(filepath.Join(dir, level, level+suffix), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		rel, _ := filepath.Rel(root, path)
		tree[rel] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func assertLesson(t *testing.T, got, want Lesson) {
	t.Helper()
	if got.WeekTitleID == nil || want.WeekTitleID == nil || *got.WeekTitleID != *want.WeekTitleID {
		t.Errorf("week_title_id = %v, want %v", deref(got.WeekTitleID), deref(want.WeekTitleID))
	}
	got.WeekTitleID, want.WeekTitleID = nil, nil
	if got != want {
		t.Errorf("lesson =\n %+v\nwant\n %+v", got, want)
	}
}

func ptr(s string) *string { return &s }

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
