package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var levelDir = regexp.MustCompile(`^n[1-5]$`)

// Run converts every level under sourceDir and writes its JSON under outDir.
// Every level converts before any file is written, so a failure leaves outDir as it was.
func Run(sourceDir, outDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	var datasets []Dataset
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !levelDir.MatchString(e.Name()) {
			return fmt.Errorf("%s: %q is not a level directory", sourceDir, e.Name())
		}
		files, err := LoadLevel(filepath.Join(sourceDir, e.Name()), e.Name())
		if err != nil {
			return err
		}
		ds, err := ConvertLevel(e.Name(), files)
		if err != nil {
			return err
		}
		datasets = append(datasets, ds)
	}
	if len(datasets) == 0 {
		return fmt.Errorf("%s: no level directories", sourceDir)
	}
	for _, ds := range datasets {
		if err := writeLevel(filepath.Join(outDir, ds.Level), ds); err != nil {
			return err
		}
	}
	return nil
}

// LoadLevel reads the four source files of one level from dir.
func LoadLevel(dir, level string) (LevelFiles, error) {
	read := func(suffix string) (File, error) {
		path := filepath.Join(dir, level+suffix)
		b, err := os.ReadFile(path)
		return File{Name: path, Text: string(b)}, err
	}
	var files LevelFiles
	var err error
	for _, f := range []struct {
		dst    *File
		suffix string
	}{
		{&files.TOC, "_curriculum_toc.md"},
		{&files.Kanji, "_kanji_list.md"},
		{&files.Vocabulary, "_vocabulary_list.md"},
		{&files.Grammar, "_grammar_list.md"},
	} {
		if *f.dst, err = read(f.suffix); err != nil {
			return LevelFiles{}, err
		}
	}
	return files, nil
}

func writeLevel(dir string, ds Dataset) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range []struct {
		name string
		rows any
	}{
		{"lessons", ds.Lessons},
		{"kanji", ds.Kanji},
		{"vocabulary", ds.Vocabulary},
		{"grammar", ds.Grammar},
		{"grammar_comparisons", ds.GrammarComparisons},
		{"grammar_mistakes", ds.GrammarMistakes},
		{"grammar_expressions", ds.GrammarExpressions},
	} {
		if err := writeJSON(filepath.Join(dir, f.name+".json"), f.rows); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON writes v as indented JSON with Japanese text and markup unescaped,
// replacing the file only once the new content is fully on disk.
func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
