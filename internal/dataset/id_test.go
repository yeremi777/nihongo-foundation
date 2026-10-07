package dataset

import "testing"

// Expected values come from Python's uuid.uuid5, an independent implementation.
func TestRowID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"n3:kanji:k101-001", "fc8dd6cd-2c9a-54ed-b26c-46bea5ba7513"},
		{"n3:lesson:kanji:1:1", "14e32c15-30cc-55e1-9c6d-ffdee839bdda"},
		{"n5:grammar:g101", "0fc3e1b0-09fd-5cbf-9712-67c053f7e08e"},
	}
	for _, tt := range tests {
		if got := rowID(tt.name); got != tt.want {
			t.Errorf("rowID(%q) = %s, want %s", tt.name, got, tt.want)
		}
	}
}
