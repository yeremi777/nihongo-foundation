package quiz

import (
	"context"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/seed"
)

func TestMockWritesAPassingQuestionForEveryTargetOfTheDataset(t *testing.T) {
	datasets, err := seed.Load("../../data")
	if err != nil {
		t.Fatal(err)
	}
	for _, ds := range datasets {
		for _, lang := range []Lang{LangEN, LangID} {
			all := targets(lang, Items{Kanji: ds.Kanji, Vocabulary: ds.Vocabulary, Grammar: ds.Grammar}, []Type{TypeMeaning, TypeReading, TypeUsage})
			drafts, err := Mock{}.Generate(context.Background(), lang, all)
			if err != nil {
				t.Fatal(err)
			}
			quiz := assemble(all, drafts)
			if quiz.Dropped != 0 || len(quiz.Questions) != len(all) || len(all) == 0 {
				t.Errorf("%s %s: %d questions and %d dropped of %d targets; want none dropped", ds.Level, lang, len(quiz.Questions), quiz.Dropped, len(all))
			}
		}
	}
}
