//go:build research

package killsource

// p4_botmeta_horodatage_research_test.go — SONDE P4 (campagne « retours rejeu », 2026-09-23) :
// L INSTANT DE CHAQUE PAQUET BOT_METADATA.
//
// `loadBotMeta` agrege les paquets de type 12 de tout le film et deduplique par (slot, bid) : il
// perd QUAND chaque bot est declare. Cet instrument relit les memes paquets avec le meme lecteur
// d entrees (`scanBotEntries`) et rend, paquet par paquet : chunk, rang, horodatage, frame du rejeu,
// `nbBots`, et les entrees (slot, bid, nom). C est la moitie « bot » du lien bot -> entite ti=9 par
// le temps (M2.1) ; l autre moitie est `grammar/p4_entites_ti9_research_test.go`.
//
// Lecture seule, un film, sous la voie film :
//
//	P4_FILM=<data>/cache/film_chunks/b1ad85eb P4_ORIGIN_US=<origine> \
//	  go test -tags research -count=1 -run '^TestP4BotMetaHorodatage$' -v \
//	  ./internal/games/halo_infinite/film/internal/facts/killsource/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestP4BotMetaHorodatage(t *testing.T) {
	dir := os.Getenv("P4_FILM")
	if dir == "" {
		t.Skip("P4_FILM absent : sonde sautee")
	}
	origine, err := strconv.ParseUint(os.Getenv("P4_ORIGIN_US"), 10, 64)
	if err != nil {
		t.Fatalf("P4_ORIGIN_US illisible : %v", err)
	}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	n := 0
	for i := range f.packets {
		p := &f.packets[i]
		if p.typ != packetTypeBotMeta {
			continue
		}
		n++
		nb := -1
		if len(p.payload) >= 4 {
			nb = int(uint32(p.payload[0])<<24 | uint32(p.payload[1])<<16 | uint32(p.payload[2])<<8 | uint32(p.payload[3]))
		}
		var sb strings.Builder
		for _, b := range scanBotEntries(p.payload) {
			fmt.Fprintf(&sb, " {slot %d bid %d %q}", b.Slot, b.BotID, b.Name)
		}
		t.Logf("BOT_METADATA chunk %2d pk %3d ts %d f%5d | taille %d o | nbBots=%d |%s", p.chunk, p.idx, p.ts,
			(int64(p.ts)-int64(origine))/100_000, len(p.payload), nb, sb.String())
	}
	m := loadBotMeta(f)
	t.Logf("BOT_METADATA : %d paquet(s) ; agregat loadBotMeta nbBots=%d bots=%+v", n, m.NBots, m.Bots)
}
