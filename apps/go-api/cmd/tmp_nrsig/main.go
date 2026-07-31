// tmp_nrsig — SIGNATURE de non-regression du decodeur film. THROWAWAY.
// Emet un CSV deterministe couvrant, pour le film 000d5950 : les positions bipeds
// offline (quanta bruts, sans bornes de carte), les directions capturees dans le meme
// record, la vie (i4) et le bouclier (i5). Le MD5 du fichier est le temoin : toute
// derive du cadrage bit deplace au moins un quantum.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_nrsig <out.csv>
package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func main() {
	out := "nrsig.csv"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.QuantaOnly = true
	opt.MaxSpeedMPS = 0 // sans bornes monde, un seuil en m/s n'a aucun sens
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(cache, opt)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "chunk,packet,ts_us,slot,qx,qy,qz,has_body,health,has_shield,shield")
	health, shield := 0, 0
	for _, p := range pos {
		h, hok := p.HealthAt()
		s, sok := p.ShieldAt()
		if hok {
			health++
		}
		if sok {
			shield++
		}
		fmt.Fprintf(w, "%d,%d,%d,%d,%d,%d,%d,%t,%.6f,%t,%.6f\n",
			p.Chunk, p.PacketIndex, p.TimestampUS, p.Slot, p.Q[0], p.Q[1], p.Q[2], hok, h, sok, s)
	}
	w.Flush()
	f.Close()

	g, _ := os.Open(out)
	defer g.Close()
	h := md5.New()
	io.Copy(h, g)
	fmt.Printf("positions=%d  vie=%d  bouclier=%d\n", len(pos), health, shield)
	fmt.Printf("MD5(%s) = %s\n", out, hex.EncodeToString(h.Sum(nil)))
}
