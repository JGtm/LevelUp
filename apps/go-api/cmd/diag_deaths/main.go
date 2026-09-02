// diag_deaths — dump du fil des morts d'un film (diagnostic ponctuel, lot sièges 2026-09-02).
// Usage : go run ./cmd/diag_deaths <filmDir>
package main

import (
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

func main() {
	deaths, err := replay.ScanFilmDeaths(os.Args[1])
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	zero := 0
	for _, d := range deaths {
		if d.XUID == 0 {
			zero++
		}
	}
	fmt.Printf("morts=%d dont xuid=0 : %d\n", len(deaths), zero)
	for i, d := range deaths {
		if d.XUID == 0 || i < 5 || i > len(deaths)-6 {
			fmt.Printf("  t=%7dms xuid=%d gt=%q\n", d.TimeMS, d.XUID, d.Gamertag)
		}
	}
}
