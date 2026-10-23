package dataset

import (
	"slices"
	"testing"
)

func TestLevelsAndDifficultiesAreCopiesInListOrder(t *testing.T) {
	Levels()[0], Difficulties()[0] = "n9", "trivial"
	if got := Levels(); !slices.Equal(got, []string{"n5", "n4", "n3", "n2", "n1"}) {
		t.Errorf("Levels() = %v", got)
	}
	if got := Difficulties(); !slices.Equal(got, []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard}) {
		t.Errorf("Difficulties() = %v", got)
	}
}
