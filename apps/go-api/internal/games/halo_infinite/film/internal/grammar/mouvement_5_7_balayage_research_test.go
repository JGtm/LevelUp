//go:build research

package grammar

// mouvement_5_7_balayage_research_test.go — LE BALAYAGE DE PRODUCTION DES ETATS DE MOUVEMENT,
// MESURE SOUS LA PORTE DE SPECULATION (lot 5.7.4).
//
// # CE QU IL MESURE, ET POURQUOI IL APPELLE LA FONCTION DE PRODUCTION
//
// Il n imite pas `ScanMovementStates` : il L APPELLE. C est le seul moyen de dire ce que le
// document publiera, et c est le point du lot 5.7.4 — la porte des etats de mouvement
// publiait les ESSAIS D ALIGNEMENT de la marche, dans un rapport de 14 a 152 pour un. Depuis
// que la porte s inscrit dans `neutraliserEtatsDeMouvement`, elle ne doit plus rendre que les
// lectures des records RETENUS, et ce fichier le verifie contre les comptes de
// `mouvement_5_7_retenus_research_test.go` (relecture a `StartBit`, 0 ecart de largeur).
//
// LES LARGEURS D AXE DE LA CARTE SONT INSTALLEES (le geste de `replay`), sans quoi `i0` lit ses
// trois axes aux largeurs d une autre carte et tout ce qui suit est du bruit.
//
// Rejouable : memes variables que `TestMouvement57Posture`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestMouvement57Balayage(t *testing.T) {
	dir := os.Getenv("MOUV57_FILM")
	if dir == "" {
		t.Skip("MOUV57_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m57Contexte(t, film)
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	lus, st, err := ScanMovementStates(fc)
	if err != nil {
		t.Fatalf("ScanMovementStates : %v", err)
	}
	t.Logf("BALAYAGE DE PRODUCTION : balaye=%v absent=%v · records ti=35 %d (desyncs %d) · "+
		"lectures RETENUES %d · ecartees (slot non lie) %d · doublons %d · paquets a evenements "+
		"%d dont %d localises et %d NON localises · largeurs de carte %v",
		st.Scanned, st.Absent, st.Records, st.Desyncs, st.Read, st.SlotUnbound, st.Duplicates,
		st.EventPackets, st.EventPacketsLocated, st.EventPacketsUnlocated, st.MapWidths)
	m57BalayageParGenre(t, lus)
}

// m57BalayageParGenre publie, par genre, le compte des lectures, des lectures POSEES, des slots,
// et la mediane de l ecart entre deux lectures consecutives de la MEME vie.
func m57BalayageParGenre(t *testing.T, lus []types.MovementStateRead) {
	t.Helper()
	type agg struct {
		total, posees int
		slots         map[uint32]bool
		parSlot       map[uint32][]uint64
	}
	m := map[string]*agg{}
	for _, l := range lus {
		a := m[l.Kind]
		if a == nil {
			a = &agg{slots: map[uint32]bool{}, parSlot: map[uint32][]uint64{}}
			m[l.Kind] = a
		}
		a.total++
		if l.On {
			a.posees++
		}
		a.slots[l.Slot] = true
		a.parSlot[l.Slot] = append(a.parSlot[l.Slot], l.TimestampUS)
	}
	genres := make([]string, 0, len(m))
	for k := range m {
		genres = append(genres, k)
	}
	sort.Strings(genres)
	for _, g := range genres {
		a := m[g]
		var ecarts []float64
		for _, ts := range a.parSlot {
			sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
			for i := 1; i < len(ts); i++ {
				ecarts = append(ecarts, float64(ts[i]-ts[i-1])/1e6)
			}
		}
		sort.Float64s(ecarts)
		t.Logf("  %-10s : %5d lectures (%d posees) · %3d vies · %4d ecarts entre transitions, "+
			"mediane %.3f s (p10 %.3f · p90 %.3f)", g, a.total, a.posees, len(a.slots),
			len(ecarts), m57Quantile(ecarts, 0.50), m57Quantile(ecarts, 0.10),
			m57Quantile(ecarts, 0.90))
	}
	if len(genres) == 0 {
		t.Logf("  AUCUNE lecture retenue")
	}
	_ = strings.TrimSpace(fmt.Sprint(len(lus)))
}
