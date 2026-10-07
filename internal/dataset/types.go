package dataset

// Section is one kind of study material.
type Section string

const (
	SectionKanji      Section = "kanji"
	SectionVocabulary Section = "vocabulary"
	SectionGrammar    Section = "grammar"
)

// Source is a textbook an item comes from.
type Source string

const (
	SourceSoumatome      Source = "soumatome"
	SourceShinkanzen     Source = "shinkanzen"
	SourceMinnaNoNihongo Source = "minna-no-nihongo"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Each row type below is one row of the table with the same name in the
// data-model spec; its JSON keys are that table's column names.

type Lesson struct {
	ID          string  `json:"id"`
	Level       string  `json:"level"`
	Section     Section `json:"section"`
	Week        int     `json:"week"`
	Day         int     `json:"day"`
	Title       string  `json:"title"`
	TitleEN     string  `json:"title_en"`
	TitleID     string  `json:"title_id"`
	WeekTitle   string  `json:"week_title"`
	WeekTitleEN string  `json:"week_title_en"`
	WeekTitleID *string `json:"week_title_id"`
}

type Kanji struct {
	ID        string   `json:"id"`
	Level     string   `json:"level"`
	Code      string   `json:"code"`
	LessonID  string   `json:"lesson_id"`
	Sequence  int      `json:"sequence"`
	Character string   `json:"character"`
	Onyomi    []string `json:"onyomi"`
	Kunyomi   []string `json:"kunyomi"`
	Examples  []string `json:"examples"`
	MeaningEN string   `json:"meaning_en"`
	MeaningID string   `json:"meaning_id"`
	Sources   []Source `json:"sources"`
}

type Vocabulary struct {
	ID           string   `json:"id"`
	Level        string   `json:"level"`
	Code         string   `json:"code"`
	LessonID     string   `json:"lesson_id"`
	Sequence     int      `json:"sequence"`
	Word         string   `json:"word"`
	Reading      string   `json:"reading"`
	PartOfSpeech string   `json:"part_of_speech"`
	Note         *string  `json:"note"`
	MeaningEN    string   `json:"meaning_en"`
	MeaningID    string   `json:"meaning_id"`
	Sources      []Source `json:"sources"`
}

type Grammar struct {
	ID             string     `json:"id"`
	Level          string     `json:"level"`
	Code           string     `json:"code"`
	LessonID       string     `json:"lesson_id"`
	Sequence       int        `json:"sequence"`
	CurriculumCode string     `json:"curriculum_code"`
	Pattern        string     `json:"pattern"`
	Reading        *string    `json:"reading"`
	Formula        *string    `json:"formula"`
	Note           *string    `json:"note"`
	Example        *string    `json:"example"`
	Difficulty     Difficulty `json:"difficulty"`
	MeaningEN      string     `json:"meaning_en"`
	MeaningID      string     `json:"meaning_id"`
	Sources        []Source   `json:"sources"`
}

type GrammarComparison struct {
	ID         string `json:"id"`
	LessonID   string `json:"lesson_id"`
	Sequence   int    `json:"sequence"`
	PatternA   string `json:"pattern_a"`
	PatternB   string `json:"pattern_b"`
	Difference string `json:"difference"`
	UseWhen    string `json:"use_when"`
	Example    string `json:"example"`
}

type GrammarMistake struct {
	ID        string        `json:"id"`
	LessonID  string        `json:"lesson_id"`
	Sequence  int           `json:"sequence"`
	Point     string        `json:"point"`
	Lines     []MistakeLine `json:"lines"`
	MeaningEN *string       `json:"meaning_en"`
	MeaningID string        `json:"meaning_id"`
}

// Verdict says whether a mistake card sentence is one to avoid or one to use.
type Verdict string

const (
	VerdictIncorrect Verdict = "incorrect"
	VerdictCorrect   Verdict = "correct"
)

// MistakeLine is one judged sentence on a mistake card; Label keeps the
// card's own wording, such as "Less natural" or "Better".
type MistakeLine struct {
	Verdict  Verdict `json:"verdict"`
	Label    string  `json:"label"`
	Sentence string  `json:"sentence"`
}

type GrammarExpression struct {
	ID         string  `json:"id"`
	LessonID   string  `json:"lesson_id"`
	Sequence   int     `json:"sequence"`
	Expression string  `json:"expression"`
	Reading    *string `json:"reading"`
	MeaningEN  string  `json:"meaning_en"`
	MeaningID  string  `json:"meaning_id"`
	Note       *string `json:"note"`
	Example    *string `json:"example"`
}

// Dataset is every row of one level.
type Dataset struct {
	Level              string
	Lessons            []Lesson
	Kanji              []Kanji
	Vocabulary         []Vocabulary
	Grammar            []Grammar
	GrammarComparisons []GrammarComparison
	GrammarMistakes    []GrammarMistake
	GrammarExpressions []GrammarExpression
}
