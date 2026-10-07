//go:build research

package grammar

// zone_proprietaire_noms_research_test.go — CE QUE LE FILM DIT DE CHAQUE SLOT ti=13 : le NOM de sa
// propriete (`i0`, porte par les lectures d image-cle, [ManagedPropertyRead.Name]), son tag et sa
// valeur a la premiere image-cle, et les changements de sa serie delta.
//
// C est l instrument qui a releve les blocs de proprietes de zone (index, cle de nommage,
// proprietaire, pousseur, jauge) et leur vocabulaire (`replay/zone_states_owner_nom.go`). Il lit le
// balayage de PRODUCTION, sans second lecteur.
//
//	ZPROP_FILM=<abs>/data/cache/film_chunks/114b0040 [ZPROP_DETAIL=1] \
//	  go test -tags=research -count=1 -v -run '^TestZoneProprietaireNoms$' ./internal/games/halo_infinite/film/internal/grammar/
//
// UN SEUL FILM PAR INVOCATION, aucune base, aucun artefact ecrit.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestZoneProprietaireNoms rend une ligne par slot ti=13 (et, avec ZPROP_DETAIL, les series).
func TestZoneProprietaireNoms(t *testing.T) {
	dir := os.Getenv("ZPROP_FILM")
	if dir == "" {
		t.Skip("instrument de mesure : ZPROP_FILM requis")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	sc, err := ScanManagedProperties(contexteDeBobine(film))
	if err != nil {
		t.Fatalf("balayage : %v", err)
	}
	cle, delta := zpParSlot(sc.KeyReads), zpParSlot(sc.Reads)
	t0 := zpOrigine(sc)
	t.Logf("FILM %s : %d records ti=13 aux images-cles (%d fermes), %d lectures delta",
		filepath.Base(dir), sc.KeyRecords, sc.KeyClosed, len(sc.Reads))
	for _, s := range zpSlots(cle, delta) {
		t.Logf("R %d %s", s, zpResume(cle[s], delta[s]))
		if os.Getenv("ZPROP_DETAIL") != "" {
			t.Logf("    image-cle %s", zpSerie(cle[s], t0))
			t.Logf("    delta     %s", zpSerie(delta[s], t0))
		}
	}
}

// zpParSlot range les lectures SCALAIRES par slot, dans l ordre du temps.
func zpParSlot(rs []ManagedPropertyRead) map[uint32][]ManagedPropertyRead {
	out := map[uint32][]ManagedPropertyRead{}
	for _, r := range rs {
		if r.Field == ManagedPropertyScalar {
			out[r.Slot] = append(out[r.Slot], r)
		}
	}
	for s := range out {
		sort.SliceStable(out[s], func(i, j int) bool { return out[s][i].TimestampUS < out[s][j].TimestampUS })
	}
	return out
}

// zpOrigine rend le plus petit horodatage des deux voies.
func zpOrigine(sc ManagedPropertyScan) uint64 {
	t0 := uint64(0)
	for _, rs := range [][]ManagedPropertyRead{sc.Reads, sc.KeyReads} {
		for _, r := range rs {
			if t0 == 0 || r.TimestampUS < t0 {
				t0 = r.TimestampUS
			}
		}
	}
	return t0
}

// zpSlots rend les slots des deux voies, tries.
func zpSlots(a, b map[uint32][]ManagedPropertyRead) []uint32 {
	vus := map[uint32]bool{}
	for s := range a {
		vus[s] = true
	}
	for s := range b {
		vus[s] = true
	}
	out := make([]uint32, 0, len(vus))
	for s := range vus {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// zpResume : le nom, le tag et la valeur de la premiere image-cle, et le nombre de CHANGEMENTS
// chaines de la serie delta sous ce tag.
func zpResume(cle, delta []ManagedPropertyRead) string {
	nom, tag, val := "-", -1, "-"
	if len(cle) > 0 {
		if cle[0].Named {
			nom = fmt.Sprintf("%d", cle[0].Name)
		}
		tag = cle[0].Tag
		if cle[0].HasValue {
			val = zpVal(cle[0].Tag, cle[0].Value)
		}
	}
	chg, prev := 0, uint64(1<<63)
	for _, r := range delta {
		if !r.Chained || r.Tag != tag || !r.HasValue {
			continue
		}
		if r.Value != prev {
			chg++
		}
		prev = r.Value
	}
	return fmt.Sprintf("%10s t%-2d %-12s chg=%d", nom, tag, val, chg)
}

// zpSerie colle une serie en ne gardant que les CHANGEMENTS (tag, valeur), l instant en secondes
// depuis t0 ; une lecture non chainee porte un `~`.
func zpSerie(rs []ManagedPropertyRead, t0 uint64) string {
	if len(rs) == 0 {
		return "-"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[%d] ", len(rs))
	prevTag, prevVal, n := -1, uint64(0), 0
	for _, r := range rs {
		if r.Tag == prevTag && r.Value == prevVal {
			continue
		}
		prevTag, prevVal = r.Tag, r.Value
		if n++; n > 40 {
			b.WriteString("...")
			break
		}
		mark, val := "", "muet"
		if !r.Chained {
			mark = "~"
		}
		if r.HasValue {
			val = zpVal(r.Tag, r.Value)
		}
		fmt.Fprintf(&b, "%s%.1fs:t%d=%s ", mark, float64(r.TimestampUS-t0)/1e6, r.Tag, val)
	}
	return b.String()
}

// zpVal rend une valeur lisible : la jauge dequantifiee, le neutre en `N`.
func zpVal(tag int, v uint64) string {
	switch {
	case tag == ManagedPropertyTagQuant:
		return fmt.Sprintf("%.3f", float64(ManagedPropertyQuantValue(v)))
	case v == 0xFFFFFFFF:
		return "N"
	}
	return fmt.Sprintf("%d", v)
}
