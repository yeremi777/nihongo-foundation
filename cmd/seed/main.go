// Command seed loads the committed JSON dataset into the database named by the DB_* variables.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yeremi777/nihongo-foundation/internal/database"
	"github.com/yeremi777/nihongo-foundation/internal/seed"
)

const seedTimeout = 2 * time.Minute

func main() {
	dir := flag.String("data", "data", "directory holding one folder of JSON per level")
	flag.Parse()

	if err := run(*dir); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	dsn, err := database.DSNFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	datasets, err := seed.Load(dir)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()
	conn, err := database.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	if err := seed.Seed(ctx, conn, datasets); err != nil {
		return err
	}
	for _, ds := range datasets {
		fmt.Printf("%s: %d lessons, %d kanji, %d vocabulary, %d grammar, %d comparisons, %d mistake cards, %d expressions\n",
			ds.Level, len(ds.Lessons), len(ds.Kanji), len(ds.Vocabulary), len(ds.Grammar),
			len(ds.GrammarComparisons), len(ds.GrammarMistakes), len(ds.GrammarExpressions))
	}
	return nil
}
