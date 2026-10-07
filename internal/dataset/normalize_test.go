package dataset

import (
	"slices"
	"testing"
)

func TestNormalizePartOfSpeech(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Noun; suru-verb", "noun / する-verb"},
		{"Na-adj/Noun", "な-adjective / noun"},
		{"Noun", "noun"},
		{"Verb phrase", "verb phrase"},
		{"Suru-verb phrase", "する-verb phrase"},
		{"する-verb", "する-verb"},
		{"I-adjective/Verb phrase", "い-adjective / verb phrase"},
		{"na-adjective", "な-adjective"},
		{"adverb／na-adjective", "adverb / な-adjective"},
		{"adverb / noun", "adverb / noun"},
		{"collocation／verb phrase", "collocation / verb phrase"},
		{"phrase · 他動詞", "phrase · 他動詞"},
		{"verb · 自動詞 / 他動詞", "verb · 自動詞 / 他動詞"},
		{"  honorific verb ", "honorific verb"},
	}
	for _, tt := range tests {
		if got := normalizePartOfSpeech(tt.in); got != tt.want {
			t.Errorf("normalizePartOfSpeech(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSplitList(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"セイ、ショウ", []string{"セイ", "ショウ"}},
		{"先生、 先月,先に", []string{"先生", "先月", "先に"}},
		{"—", []string{}},
		{"", []string{}},
	}
	for _, tt := range tests {
		if got := splitList(tt.in); !slices.Equal(got, tt.want) || got == nil {
			t.Errorf("splitList(%q) = %#v, want %#v", tt.in, got, tt.want)
		}
	}
}

func TestOptional(t *testing.T) {
	if got := optional("—"); got != nil {
		t.Errorf("optional(—) = %q, want nil", *got)
	}
	if got := optional("  "); got != nil {
		t.Errorf("optional(blank) = %q, want nil", *got)
	}
	if got := optional(" 冷蔵庫に入れる "); got == nil || *got != "冷蔵庫に入れる" {
		t.Errorf("optional(text) = %v, want 冷蔵庫に入れる", got)
	}
}
