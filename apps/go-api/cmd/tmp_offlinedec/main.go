// tmp_offlinedec — export CSV des positions bipeds décodées 100 % OFFLINE (zéro
// CheatEngine). Mince enveloppe autour de filmdec.ScanFilmBipedPositions : la grammaire
// du record biped vit dans internal/analysis/filmdec (promue depuis cet outil), ici on ne
// fait que l'export pour les validations et les visualisations.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_offlinedec [filmDir] [out.csv]
// Env   : SATURATED=1 conserve les positions écrêtées (buckets extrêmes).
package main

import (
	"bufio"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	outPath := dir + "/offline_records.csv"
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}
	opt := filmdec.DefaultScanFilmOptions()
	// Dumper de bitstream : quanta bruts uniquement (les coordonnées monde exigent les
	// bornes de la carte, cf. map_bounds.go / cmd/mapquant-build).
	opt.QuantaOnly = true
	opt.DropSaturated = os.Getenv("SATURATED") != "1"

	recs, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Println("scan:", err)
		os.Exit(1)
	}
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Println("create:", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriterSize(f, 1<<20)
	fmt.Fprintln(w, "slot,chunk,packetIndex,ts,x,y,z")
	slots := map[uint32]bool{}
	for _, r := range recs {
		slots[r.Slot] = true
		fmt.Fprintf(w, "%d,%d,%d,%d,%.5f,%.5f,%.5f\n",
			r.Slot, r.Chunk, r.PacketIndex, r.TimestampUS, r.X, r.Y, r.Z)
	}
	_ = w.Flush()
	fmt.Printf("TOTAL : %d positions, %d slots -> %s\n", len(recs), len(slots), outPath)
}
