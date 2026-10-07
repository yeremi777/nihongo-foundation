// Command convert turns the source-list snapshot into the committed JSON dataset.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

func main() {
	source := flag.String("source", "data/source", "directory holding one folder of source markdown per level")
	out := flag.String("out", "data", "directory the JSON dataset is written to")
	flag.Parse()

	if err := dataset.Run(*source, *out); err != nil {
		fmt.Fprintln(os.Stderr, "convert:", err)
		os.Exit(1)
	}
}
