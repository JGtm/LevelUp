//go:build research

package replay

// p4_equipes_faits_research_test.go — SONDE P4 (campagne « retours rejeu », 2026-09-23), volet
// FAITS PERSISTES. Relit le fichier de faits d'UN film (jamais le film) et rend :
//   - l'ORIGINE de l'horloge du rejeu (premier horodatage de position) : frame = (ts - origine) / 100 ms,
//     la meme horloge que les documents publies ; les deux autres volets de P4 la prennent en entree ;
//   - la table de chunk_00 et le roster killsource (bots declares) ;
//   - chaque VIE : (slot, creation) -> index de participant LU dans le record de creation, fenetre
//     de positions [premiere..derniere] ;
//   - la REGLE DES PLACES SUR LES TIRS : pour chaque index de tireur s, et chaque index de
//     participant p, combien de tirs de s tombent DANS une vie de p (tolerance 2 frames) et combien
//     HORS de toute vie de p. Un tireur n'est pas mort quand il tire : un seul tir hors des vies de p
//     exclut p ; le tireur est le p a 0 tir hors vie. L'hypothese : s = PLACE, pas index de
//     participant (Claudors, index 10, tire sous l'index 5).
//
// ETALONNAGE : pour tout index s tenu par un seul occupant du debut a la fin (sieges de la table de
// depart), le p a 0 tir hors vie doit etre s lui-meme — c'est le temoin positif de la jointure.
//
// Lecture seule, sous la voie film :
//
//	P4_FAITS=<data>/cache/film_facts/halo_infinite/b1ad85eb.filmfacts.bin \
//	P4_CATALOGUE=<data>/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test -tags research -count=1 -run '^TestP4EquipesFaits$' -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

const p4ToleranceFrames = 2

func p4Entree(t *testing.T, blob []byte) profile.MapQuantEntry {
	t.Helper()
	ent, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	if ent.LayoutDetected {
		return profile.MapQuantEntry{Module: ent.MapModule}
	}
	if p := os.Getenv("P4_CATALOGUE"); p != "" {
		cat, err := profile.LoadMapQuantCatalog(p)
		if err != nil {
			t.Fatalf("catalogue : %v", err)
		}
		for _, e := range cat.Maps {
			if e.Module == ent.MapModule && e.AxisWidths == ent.AxisW {
				return e
			}
		}
	}
	return profile.MapQuantEntry{Module: ent.MapModule, AxisWidths: ent.AxisW}
}

// p4Vie : une vie = un record de creation de bipede et les positions de son slot jusqu'a la
// creation suivante du meme slot.
type p4Vie struct {
	slot       uint32
	idx        int
	idxLu      bool
	crea, a, b int64
	n          int
}

func TestP4EquipesFaits(t *testing.T) {
	chemin := os.Getenv("P4_FAITS")
	if chemin == "" {
		t.Skip("P4_FAITS absent : sonde sautee")
	}
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	f, err := DecodeFilmFactsFile(blob, p4Entree(t, blob))
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	in := f.Facts.FilmInputs
	var origin uint64
	for i, p := range in.Positions {
		if i == 0 || p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	fr := func(ts uint64) int64 { return (int64(ts) - int64(origin)) / 100_000 }
	t.Logf("ORIGINE %d us (frame = (ts - origine) / 100 ms)", origin)
	for _, s := range in.FilmTable.Seats {
		t.Logf("  table chunk_00 : siege %2d xuid=%d %q", s.FilmIndex, s.XUID, s.Gamertag)
	}
	if f.Kills != nil {
		t.Logf("  killsource : bots=%+v botsSuccedes=%d", f.Kills.Roster.Bots, f.Kills.Roster.BotsSuccedes)
	}
	vies := p4Vies(in.BipedCreations, in.Positions, fr)
	for _, v := range vies {
		t.Logf("  VIE slot %4d idx=%2d(lu=%v) creation f%5d positions [%5d..%5d] n=%d",
			v.slot, v.idx, v.idxLu, v.crea, v.a, v.b, v.n)
	}
	p4TirsContreVies(t, in.Fire, vies, fr)
}

// p4Vies decoupe les positions de chaque slot par ses creations.
func p4Vies(creas []grammar.BipedCreation, pos []grammar.BipedPosition, fr func(uint64) int64) []*p4Vie {
	parSlot := map[uint32][]*p4Vie{}
	for _, c := range creas {
		parSlot[c.Slot] = append(parSlot[c.Slot], &p4Vie{
			slot: c.Slot, idx: int(c.ParticipantIndex), idxLu: c.HasIndex, crea: fr(c.TimestampUS), a: -1, b: -1,
		})
	}
	for _, l := range parSlot {
		sort.Slice(l, func(i, j int) bool { return l[i].crea < l[j].crea })
	}
	for _, p := range pos {
		l := parSlot[p.Slot]
		f0 := fr(p.TimestampUS)
		var cible *p4Vie
		for _, v := range l {
			if v.crea <= f0 {
				cible = v
			}
		}
		if cible == nil {
			continue // position anterieure a toute creation lue : hors jointure (comptee par slot ailleurs)
		}
		if cible.a < 0 || f0 < cible.a {
			cible.a = f0
		}
		if f0 > cible.b {
			cible.b = f0
		}
		cible.n++
	}
	var out []*p4Vie
	for _, l := range parSlot {
		for _, v := range l {
			if v.n > 0 {
				out = append(out, v)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].crea != out[j].crea {
			return out[i].crea < out[j].crea
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// p4TirsContreVies : pour chaque index de tireur, dans / hors les vies de chaque index de participant.
func p4TirsContreVies(t *testing.T, tirs []grammar.FireEvent, vies []*p4Vie, fr func(uint64) int64) {
	t.Helper()
	parTireur := map[int][]int64{}
	for _, e := range tirs {
		parTireur[e.FilmIndex] = append(parTireur[e.FilmIndex], fr(e.TimestampUS))
	}
	parIdx := map[int][]*p4Vie{}
	for _, v := range vies {
		if v.idxLu {
			parIdx[v.idx] = append(parIdx[v.idx], v)
		}
	}
	idxs := make([]int, 0, len(parIdx))
	for k := range parIdx {
		idxs = append(idxs, k)
	}
	sort.Ints(idxs)
	tireurs := make([]int, 0, len(parTireur))
	for k := range parTireur {
		tireurs = append(tireurs, k)
	}
	sort.Ints(tireurs)
	for _, s := range tireurs {
		fs := parTireur[s]
		sort.Slice(fs, func(i, j int) bool { return fs[i] < fs[j] })
		var sb strings.Builder
		zeroHors := []int{}
		for _, p := range idxs {
			dans := 0
			for _, f0 := range fs {
				if p4DansUneVie(f0, parIdx[p]) {
					dans++
				}
			}
			fmt.Fprintf(&sb, " p%d:%d/%d", p, dans, len(fs)-dans)
			if dans == len(fs) {
				zeroHors = append(zeroHors, p)
			}
		}
		t.Logf("TIREUR %2d : %3d tirs [f%d..f%d] | dans/hors par index de participant :%s | 0 hors = %v",
			s, len(fs), fs[0], fs[len(fs)-1], sb.String(), zeroHors)
	}
}

func p4DansUneVie(f0 int64, vies []*p4Vie) bool {
	for _, v := range vies {
		if f0 >= v.a-p4ToleranceFrames && f0 <= v.b+p4ToleranceFrames {
			return true
		}
	}
	return false
}
