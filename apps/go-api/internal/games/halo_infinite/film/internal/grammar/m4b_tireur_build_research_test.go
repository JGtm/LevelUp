//go:build research

package grammar

// m4b_tireur_build_research_test.go — LOT M4b : LE TIREUR LU PAR LA GRAMMAIRE CONTRE L ANCIEN
// OFFSET FIXE (bits 35..39), sur le film M4B_FILM. Mesure seule.

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestM4bTireurContreOffsetFixe(t *testing.T) {
	dir := os.Getenv("M4B_FILM")
	if dir == "" {
		t.Skip("M4B_FILM absent")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	accord, desaccord, sansTireur, n := 0, 0, 0, 0
	type cle struct {
		probe, court, bloc bool
		ancien, nouveau    int
	}
	hist := map[cle]int{}
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, p := range pks {
			if p.Type != PacketTypeDelta || p.Size < 6 {
				continue
			}
			pay := p.Payload(chunk)
			if typ, ok := PacketHeadEventType(pay); !ok || typ != TypeTirArme {
				continue
			}
			n++
			ancien := int(readBitsAt(pay, 35, 5))
			e, ok := decodeFireEvent(pay)
			if !ok {
				hist[cle{ancien: ancien, nouveau: -9}]++
				continue
			}
			switch {
			case !e.HasShooter:
				sansTireur++
			case e.FilmIndex == ancien:
				accord++
			default:
				desaccord++
			}
			hist[cle{e.Unit.Probe, e.Short, e.Bloc, ancien, e.FilmIndex}]++
		}
	}
	t.Logf("records 36 en tete : %d · accord %d · desaccord %d · sans tireur %d", n, accord, desaccord, sansTireur)
	var lignes []string
	for k, v := range hist {
		if k.ancien == k.nouveau {
			continue
		}
		lignes = append(lignes, fmt.Sprintf("%6d  sonde %v court %v bloc %v ancien %2d nouveau %2d", v, k.probe, k.court, k.bloc, k.ancien, k.nouveau))
	}
	sort.Strings(lignes)
	for i := len(lignes) - 1; i >= 0 && i >= len(lignes)-25; i-- {
		t.Log(lignes[i])
	}
}

// TestM4bBitsDeTete publie les 56 premiers bits de M4B_N records 36 de tete et leur lecture.
func TestM4bBitsDeTete(t *testing.T) {
	dir := os.Getenv("M4B_FILM")
	if dir == "" {
		t.Skip("M4B_FILM absent")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, p := range pks {
			if p.Type != PacketTypeDelta || p.Size < 8 || n >= 12 {
				continue
			}
			pay := p.Payload(chunk)
			if typ, ok := PacketHeadEventType(pay); !ok || typ != TypeTirArme {
				continue
			}
			n++
			bits := make([]byte, 0, 56)
			for i := 0; i < 56; i++ {
				bits = append(bits, '0'+byte(readBitsAt(pay, i, 1)))
			}
			e, _ := decodeFireEvent(pay)
			t.Logf("%s  ancien(35,5)=%2d  grammaire : unite %+v tireur %d numero %d arme %016x", string(bits),
				readBitsAt(pay, 35, 5), e.Unit, e.FilmIndex, e.FireNumber, e.WeaponID)
		}
	}
}
