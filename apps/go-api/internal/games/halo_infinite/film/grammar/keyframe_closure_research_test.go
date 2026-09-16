package grammar

// keyframe_closure_research_test.go — L INVENTAIRE DE LA FERMETURE D IMAGE-CLE, PAR ARCHETYPE.
//
// # CE QU IL SERT
//
// Le ratchet (`keyframe_closure_ratchet_test.go`) garde la couverture sur les bobines par build ;
// CET instrument la mesure sur des FILMS ENTIERS du cache, et imprime, par archetype, la liste
// ORDONNEE des composants avec leur statut — porte, BLOQUANT (le non-porte LE PLUS FREQUENT),
// et ce qui se trouve derriere lui, donc inaccessible tant qu il n est pas porte.
// C est le DIMENSIONNEMENT du lot 3.6 : il dit quels composants porter, et dans quel ordre.
//
// Dans une image-cle il n y a pas de masque de presence : TOUS les composants de l archetype sont
// la, dans l ordre du registre. Un seul composant sans lecteur bloque donc tout ce qui le suit,
// et porter LE bloquant debloque d un coup tout ce qui etait derriere lui.
//
// Aucun code de production n est touche : l instrument appelle `KeyframeClosure`, la meme
// fonction que le ratchet, et lit le registre pour nommer les composants.
//
// USAGE (garde par environnement, saute sans elle) :
//
//	CHUNK00_FILMS='C:/.../film_chunks/a521164d;C:/.../film_chunks/60ae07c4' \
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run KeyframeClosureInventaire -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// TestKeyframeClosureInventaire imprime la fermeture et l inventaire des composants par
// archetype, film par film.
func TestKeyframeClosureInventaire(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		inventaireUnFilm(t, dir)
	}
}

// inventaireUnFilm mesure un film et imprime son tableau.
func inventaireUnFilm(t *testing.T, dir string) {
	t.Helper()
	nom := filepath.Base(dir)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Logf("%s : ECARTE (LoadDir : %v)", nom, err)
		return
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Logf("%s : ECARTE (registre illisible : %v)", nom, err)
		return
	}
	stats, err := KeyframeClosure(fc)
	if err != nil {
		t.Logf("%s : ECARTE (fermeture : %v)", nom, err)
		return
	}
	tis := make([]int, 0, len(stats))
	for ti := range stats {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	t.Logf("=== %s : %d archetype(s) mesures ===", nom, len(tis))
	var ferme, total int
	for _, ti := range tis {
		s := stats[uint32(ti)] //nolint:gosec // ti vient d'une cle uint32
		ferme, total = ferme+s.Closed, total+s.Total
		t.Logf("  ti=%-3d %6d/%-6d (%5.1f %%)  bloquant: %s",
			ti, s.Closed, s.Total, pourcent(s.Closed, s.Total), ouAucun(s.Blocking))
		imprimerComposants(t, reg, ti, s.Blocking)
	}
	t.Logf("  TOTAL %s : %d/%d (%.1f %%)", nom, ferme, total, pourcent(ferme, total))
}

// imprimerComposants liste les composants de l archetype DANS L ORDRE DU REGISTRE, en marquant
// celui qui bloque et tout ce qui le suit (inaccessible tant qu il n est pas porte).
func imprimerComposants(t *testing.T, reg *Registry, ti int, bloquant string) {
	t.Helper()
	arch, ok := reg.Archetype(ti)
	if !ok {
		return
	}
	apres := false
	for i, nom := range arch.Components {
		etiquette := "porte"
		switch {
		case fmt.Sprintf("i%d %s", i, nom) == bloquant:
			etiquette, apres = "BLOQUANT", true
		case apres:
			etiquette = "derriere le bloquant"
		}
		t.Logf("      i%-3d %-62s %s", i, nom, etiquette)
	}
}

// pourcent rend un taux en pour-cent, 0 quand le denominateur est nul.
func pourcent(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

// ouAucun rend « (aucun) » pour un bloquant vide — un champ vide dans un tableau se lit comme une
// donnee manquante, pas comme une absence de bloquant.
func ouAucun(s string) string {
	if s == "" {
		return "(aucun)"
	}
	return s
}
