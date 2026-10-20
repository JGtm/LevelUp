//go:build research

package grammar

// va_revue_canal_research_test.go — LOT VA, REVUE (2026-10-06) : D OU VIENNENT LES LECTURES DU CANAL
// DES MORTS, TRAME PAR TRAME. Un instrument de recherche : aucune sortie de production ne change.
//
// Par film (carte par VA_CATALOGUE / VA_CARTES, contexte de cuisson [NewFilmContextForMap], profil de
// la cuisson par VA_PROFILS), la marche des trames de production est parcourue ; chaque trame delta
// a evenements est ecrite avec son instant, la facon dont la cuisson a trouve le debut de sa vue B
// ([lecture.DebutDeVueB]), ce debut, celui que le localisateur de la cuisson rendrait sans la fin de
// la vue A ([localiserLaListe]) et, pour une liste que la cuisson n a pas localisee, le debut que le
// canal des morts recupere ([debutRecupere]). Les lectures de morts et d occupation se rattachent a
// leur trame par leur instant.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	VA_CATALOGUE=<map_quant_bounds.json> VA_CARTES="id=Carte;..." [VA_PROFILS=<profils>] \
//	  go test -tags=research -count=1 -run '^TestVARevueOriginesDuCanal$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestVARevueOriginesDuCanal ecrit `va_revue_canal.tsv` : une ligne par trame a evenements.
func TestVARevueOriginesDuCanal(t *testing.T) {
	racine, sortie, films := b2Env(t)
	lignes := []string{"film\tinstant_us\tchunk\tpaquet\tdebut_par\tdebut\tdebut_sans_E\tdebut_recupere"}
	for _, id := range films {
		e, ok := vaCarte(t, id)
		if !ok {
			t.Fatalf("%s : carte inconnue", id)
		}
		film, err := source.LoadDir(filepath.Join(racine, id), nil)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		fc := NewFilmContextForMap(film, &e, DefaultScanFilmOptions().Layout)
		vaPoserLeProfilDeLaCuisson(t, id, fc)
		m, err := fc.nouveauMarcheurDesTrames(nil)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		n := 0
		m.parcourir(func(tr *trameLue) bool {
			p := tr.paquet
			if !listeAnnoncee(&p.VueA) {
				return true
			}
			sansE, recupere := -2, -2
			if p.Debut == lecture.DebutParVueA {
				sansE, _ = localiserLaListe(p.Payload, m.monde, m.cfg)
			}
			if tr.debut < 0 {
				recupere, _ = debutRecupere(p.Payload, m.monde, m.cfg)
			}
			lignes = append(lignes, rnTab(id, p.TS, p.Chunk, p.Index, int(p.Debut), tr.debut, sansE, recupere))
			n++
			return true
		})
		t.Logf("%s : %d trame(s) a evenements", id, n)
	}
	b2Ecrire(t, sortie, "va_revue_canal.tsv", lignes)
}
