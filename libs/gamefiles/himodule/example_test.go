package himodule_test

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/JGtm/halo-infinite-files-reader/himodule"
)

// Open a Halo Infinite .module archive (it is mapped into memory, not copied), list the tags
// of one group and decompress them. Kraken blocks are decoded through libs/gamefiles/ooz.
func ExampleOpen() {
	// Path inside the local Halo Infinite install: <deploy>/pc/levels/multi/<map>/<map>-rtx-new.module
	path := filepath.Join("deploy", "pc", "levels", "multi", "ridgeline", "ridgeline-rtx-new.module")

	m, err := himodule.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	for _, f := range m.Files("sbsp") { // "" lists every entry
		tag, err := m.Extract(f)
		if err != nil {
			log.Printf("tag %08x: %v", f.GlobalID, err)
			continue
		}
		fmt.Printf("%s %08x: %d bytes (%d compressed)\n", f.Group, f.GlobalID, len(tag), f.CompSize)
	}
}
