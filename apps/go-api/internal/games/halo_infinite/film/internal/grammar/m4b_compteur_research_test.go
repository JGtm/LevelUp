//go:build research

package grammar

// m4b_compteur_research_test.go — LOT M4b : LE CONTROLE DU TIR CONTINU PAR LE NUMERO DE TIR.
// Mesure seule.
//
// Le numero de tir d un joueur (record 36, huit bits) avance a CHAQUE tir, emis ou non
// (`FUN_14202f3a0`, sonde P1-S3). Entre deux records 36 du meme joueur, son SAUT moins un compte
// les tirs qui n ont pas ete ecrits — ceux du tir continu. Pour chaque rafale du joueur (marche de
// production, [ScanMarcheDesTrames]) encadree par deux de ses records sans autre rafale entre eux,
// l instrument publie le saut, la duree LUE de la rafale (hors trous) et la cadence qu ils
// impliquent, a comparer a la cadence lue dans le tag de l arme.
//
// Rejouable : memes variables que `m4b_tir_continu_research_test.go` ; S3_INDEX = -1 pour tous.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestM4bCompteurDeTir publie le controle par le numero de tir (en-tete du fichier).
func TestM4bCompteurDeTir(t *testing.T) {
	cad := s3LireCadre(t)
	tc := t516Cadre(t)
	m, err := ScanMarcheDesTrames(tc.fc)
	if err != nil {
		t.Fatalf("ScanMarcheDesTrames : %v", err)
	}
	fire, err := ScanFireEvents(m511Film(t))
	if err != nil {
		t.Fatalf("ScanFireEvents : %v", err)
	}
	if chemin := os.Getenv("M4B_DUMP_RECORDS"); chemin != "" {
		var lignes []string
		for _, e := range fire {
			lignes = append(lignes, fmt.Sprintf("%d %d %d %v", e.FilmIndex, cad.trame(e.TimestampUS), e.FireNumber,
				e.Unit.Slot))
		}
		if err := os.WriteFile(chemin, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
			t.Fatalf("ecriture %s : %v", chemin, err)
		}
	}
	parJoueur := map[int][]FireEvent{}
	for _, e := range fire {
		if e.HasShooter {
			parJoueur[e.FilmIndex] = append(parJoueur[e.FilmIndex], e)
		}
	}
	for _, evs := range parJoueur {
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].TimestampUS < evs[j].TimestampUS })
	}
	rafales := map[int][]types.ContinuousFireBurst{}
	for _, r := range m.ContinuousFire {
		if r.Hand == 0 && !r.Barrel && r.Input == 0 {
			rafales[r.FilmIndex] = append(rafales[r.FilmIndex], r)
		}
	}
	for j, rs := range rafales {
		if cad.index >= 0 && j != cad.index {
			continue
		}
		m4bControlerLeJoueur(t, cad, j, rs, parJoueur[j])
	}
}

// m4bControlerLeJoueur publie, pour les rafales d un joueur, le saut du numero de tir qui les
// encadre.
func m4bControlerLeJoueur(t *testing.T, cad s3Cadre, j int, rs []types.ContinuousFireBurst, evs []FireEvent) {
	t.Helper()
	for k, r := range rs {
		avant, apres := -1, -1
		for i, e := range evs {
			if e.TimestampUS <= r.StartUS {
				avant = i
			}
			if e.TimestampUS >= r.EndUS && apres < 0 {
				apres = i
			}
		}
		if avant < 0 || apres < 0 {
			t.Logf("joueur %d rafale t%d..%d : pas de record 36 des deux cotes", j, cad.trame(r.StartUS),
				cad.trame(r.EndUS))
			continue
		}
		a, b := evs[avant], evs[apres]
		seule := (k == 0 || rs[k-1].EndUS <= a.TimestampUS) && (k+1 == len(rs) || rs[k+1].StartUS >= b.TimestampUS)
		saut := int(b.FireNumber-a.FireNumber) - 1 // modulo 256 par l arithmetique des octets
		lue := r.EndUS - r.StartUS
		for _, h := range r.Holes {
			lue -= h.EndUS - h.StartUS
		}
		t.Logf("joueur %d rafale t%d..%d (%.2f s, lue %.2f s, trous %d, %s..%s) : records t%d n%d -> t%d n%d, "+
			"saut %d (seule entre eux %v) -> %.1f coups/s sur la duree, %.1f sur la part lue", j,
			cad.trame(r.StartUS), cad.trame(r.EndUS), float64(r.EndUS-r.StartUS)/1e6, float64(lue)/1e6,
			len(r.Holes), r.StartBound, r.EndBound, cad.trame(a.TimestampUS), a.FireNumber,
			cad.trame(b.TimestampUS), b.FireNumber, saut, seule,
			float64(saut)/(float64(r.EndUS-r.StartUS)/1e6), float64(saut)/(float64(max(lue, 1))/1e6))
	}
}
