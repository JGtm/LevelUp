//go:build research

package grammar

// movement_states_probe_research_test.go — LA SONDE DU BALAYAGE DE PRODUCTION (lot 5.3.6).
//
// Elle joue `ScanMovementStates` — la marche ANCREE, celle que la cuisson emploie — sur un film
// reel, et publie ses denominateurs. C est le controle sur pieces du port : un balayage qui rend
// zero lecture sur un film ou la mesure de 5.3.4 en compte des milliers serait un port mort.
//
//	MOUV536_FILM=<dir> MOUV536_CARTE=snowbound MOUV536_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -run '^TestMouvement536Sonde$' ./...grammar/

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestMouvement536Sonde(t *testing.T) {
	dir := os.Getenv("MOUV536_FILM")
	if dir == "" {
		t.Skip("MOUV536_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	os.Setenv("MOUV534_CARTE", os.Getenv("MOUV536_CARTE"))
	os.Setenv("MOUV534_BORNES", os.Getenv("MOUV536_BORNES"))
	fc := m534Contexte(t, film)
	reads, st, err := ScanMovementStates(fc)
	if err != nil {
		t.Fatalf("ScanMovementStates : %v", err)
	}
	t.Logf("BALAYAGE DE PRODUCTION : scanne=%v absent=%v · %d records dont %d desynchronises · "+
		"%d lectures retenues · %d paquets (%d a evenements, %d localises, %d non localises) · "+
		"%d slots non lies · %d doublons · largeurs %v",
		st.Scanned, st.Absent, st.Records, st.Desyncs, st.Read, st.Packets, st.EventPackets,
		st.EventPacketsLocated, st.EventPacketsUnlocated, st.SlotUnbound, st.Duplicates,
		st.MapWidths)
	parGenre := map[string]int{}
	parGenreOn := map[string]int{}
	slots := map[uint32]bool{}
	for _, r := range reads {
		parGenre[r.Kind]++
		if r.On {
			parGenreOn[r.Kind]++
		}
		slots[r.Slot] = true
	}
	genres := make([]string, 0, len(parGenre))
	for k := range parGenre {
		genres = append(genres, k)
	}
	sort.Strings(genres)
	for _, k := range genres {
		t.Logf("  %-9s : %d lectures, dont %d POSEES (%.1f %%)", k, parGenre[k], parGenreOn[k],
			100*float64(parGenreOn[k])/float64(parGenre[k]))
	}
	t.Logf("  slots distincts : %d · premiere lecture ts=%d · derniere ts=%d", len(slots),
		m536Premier(reads), m536Dernier(reads))
}

func m536Premier(r []types.MovementStateRead) uint64 {
	if len(r) == 0 {
		return 0
	}
	return r[0].TimestampUS
}

func m536Dernier(r []types.MovementStateRead) uint64 {
	if len(r) == 0 {
		return 0
	}
	return r[len(r)-1].TimestampUS
}
