package decfilm_test

import (
	"fmt"
	"log"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// Read the kill feed of a Halo Infinite Theater film. The directory holds the film's chunks as
// downloaded (the module does not download them: bytes arrive from your own storage). The film
// below is a minimal fixture of this module, in which every player identifier is fictitious.
func ExampleScanDeaths() {
	film, err := decfilm.LoadDir("../replay/testdata/minifilm_000d5950", nil)
	if err != nil {
		log.Fatal(err)
	}
	deaths, err := decfilm.ScanDeaths(film)
	if err != nil {
		log.Fatal(err)
	}
	victims := map[uint64]bool{}
	for _, d := range deaths {
		victims[d.XUID] = true
	}
	fmt.Printf("%d deaths, %d distinct victims\n", len(deaths), len(victims))
	// Output: 93 deaths, 8 distinct victims
}
