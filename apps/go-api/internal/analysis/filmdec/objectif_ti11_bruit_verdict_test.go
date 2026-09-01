package filmdec

// objectif_ti11_bruit_verdict_test.go — LES QUATRE PORTES DU LOT « BRUIT », et le journal qui les
// rend lisibles. Le CRITERE est ecrit dans l'en-tete d'`objectif_ti11_bruit_test.go` ; ce fichier
// ne fait que l'appliquer et publier les chiffres, y compris quand ils disent non.
//
// L'ORACLE N'EST PAS RECOPIE UNE TROISIEME FOIS : `ti12Explosions` (garde par
// `TestNavpointTi12OracleFige`, 9 films / 28 explosions) est reutilise tel quel.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// btJournalFilm publie ce qu'un film a rendu, voie par voie.
func btJournalFilm(t *testing.T, b *btFilmBilan) {
	t.Helper()
	sc := b.sc
	t.Logf("%-9s %-26s support [%d, %d] ms (%.0f s) · %d slot(s) de bande · balayage en %s",
		b.id, b.mode, b.debutMS, b.finMS, float64(b.finMS-b.debutMS)/1000, sc.Slots,
		b.duree.Round(time.Second))
	t.Logf("           DELTA %d ancres, %d marches, %d cassees, %d chainees (%.1f %%) | "+
		"IMAGE-CLE %d records, %d marches, %d cassees, %d chainees",
		sc.Records, sc.Walked, sc.Broken, sc.Chained, ti11Part(sc.Chained, sc.Walked),
		sc.KeyRecords, sc.KeyWalked, sc.KeyBroken, sc.KeyChained)
	for v := 0; v < btVoies; v++ {
		var sb strings.Builder
		for c := 0; c < btChamps; c++ {
			s := b.voies[v][c]
			fmt.Fprintf(&sb, " %s %d/%d", btNomChamp(c), len(s.ech), s.distincts)
		}
		t.Logf("           %-13s lectures/instants distincts :%s", btNomVoie(v), sb.String())
	}
	if b.sansHorloge > 0 {
		t.Logf("           %d lecture(s) sans horloge (paquet absent du manifeste) — ecartees",
			b.sansHorloge)
	}
}

// btMotifsTemoins donne a chaque temoin des PSEUDO-explosions : le motif du film d'Assaut de
// duree la plus proche, rogne au support du temoin.
//
// POURQUOI CE TEMOIN CROISE EXISTE. Un film sans bombe n'a pas d'explosion, donc le critere n'y
// est pas applicable tel quel — et un canal qu'on ne peut pas contredire ne vaut rien. En posant
// sur le temoin le MEME motif d'evenements que sur l'Assaut, on demande a l'instrument de
// produire un exces la ou il ne PEUT PAS y en avoir. S'il en produit, c'est lui qui fabrique les
// pics, et le verdict sur l'Assaut tombe avec.
func btMotifsTemoins(t *testing.T, temoins, assauts []*btFilmBilan) {
	t.Helper()
	for _, w := range temoins {
		src := btPlusProche(w, assauts)
		if src == nil {
			continue
		}
		w.motif = src.id
		for _, e := range src.expl {
			if e >= int(w.debutMS) && e <= int(w.finMS) {
				w.expl = append(w.expl, e)
			}
		}
		sort.Ints(w.expl)
		t.Logf("TEMOIN CROISE %-9s %-26s motif emprunte a %s : %d pseudo-explosion(s) sur %d",
			w.id, w.mode, src.id, len(w.expl), len(src.expl))
	}
}

// btPlusProche rend le film d'Assaut dont la duree est la plus voisine de celle du temoin.
func btPlusProche(w *btFilmBilan, assauts []*btFilmBilan) *btFilmBilan {
	var best *btFilmBilan
	ecart := 0
	for _, a := range assauts {
		d := int(a.finMS) - int(w.finMS)
		if d < 0 {
			d = -d
		}
		if best == nil || d < ecart {
			best, ecart = a, d
		}
	}
	return best
}

// btGate0 — PRESENCE ET TAILLE D'ECHANTILLON. C'est la porte qui dit si la question est seulement
// posable : le nombre d'INSTANTS DISTINCTS borne tout ce qui suit.
func btGate0(t *testing.T, assauts []*btFilmBilan) {
	t.Helper()
	t.Logf("########## GATE 0 PRESENCE — %d film(s) d'Assaut mesures", len(assauts))
	for v := 0; v < btVoies; v++ {
		for c := 0; c < btChamps; c++ {
			n, d, films := 0, 0, 0
			for _, b := range assauts {
				n += len(b.voies[v][c].ech)
				d += b.voies[v][c].distincts
				if len(b.voies[v][c].ech) > 0 {
					films++
				}
			}
			t.Logf("   %-13s %-22s %8d lecture(s) · %6d instant(s) distinct(s) · %d film(s)",
				btNomVoie(v), btNomChamp(c), n, d, films)
		}
	}
}

// btGate1 — VALIDITE DE L'INSTRUMENT PAR LE TEMOIN CROISE, et elle passe AVANT tout verdict sur
// l'Assaut.
func btGate1(t *testing.T, temoins []*btFilmBilan) {
	t.Helper()
	t.Logf("########## GATE 1 TEMOIN CROISE — %d film(s) non-Assaut au meme instrument",
		len(temoins))
	if len(temoins) == 0 {
		t.Logf("   AUCUN TEMOIN MESURE — les Strongholds n'ont aucun slot ti=11 dans leurs "+
			"images-cles (le MIROIR du chantier). Sans temoin, la porte ne peut pas trancher.%s", "")
		return
	}
	for c := 0; c < btChamps; c++ {
		e := btNouvelleEpreuve(temoins, btVoieDelta, c, nil)
		btPublier(t, "TEMOIN CROISE DELTA "+btNomChamp(c), btEprouver(e), btPMax)
	}
}

// btGate2 — LE CRITERE DU CHANTIER sur les trois champs et les trois voies.
func btGate2(t *testing.T, assauts []*btFilmBilan) {
	t.Helper()
	t.Logf("########## GATE 2 EXCES — fenetre [%d s, %d s[ avant explosion, %d explosion(s)",
		btFenetreBasMS/1000, btFenetreHautMS/1000, btNbExplosions(assauts))
	for v := 0; v < btVoies; v++ {
		for c := 0; c < btChamps; c++ {
			e := btNouvelleEpreuve(assauts, v, c, nil)
			btPublier(t, btNomVoie(v)+" "+btNomChamp(c), btEprouver(e), btPMax)
		}
	}
	t.Logf("--- DETAIL PAR EXPLOSION (voie DELTA, i12 progress) : ce qu'un agregat cacherait")
	btDetail(t, btNouvelleEpreuve(assauts, btVoieDelta, btChampProgres, nil))
}

// btGate3 — i14 PAR VALEUR : une valeur d'etat qui se concentre avant les explosions serait
// « en cours d'armement ».
//
// HUIT TESTS SIMULTANES, DONC HUIT CHANCES DE SE TROMPER : le seuil est divise par huit
// (Bonferroni), et il l'est AVANT la mesure, pas apres avoir vu laquelle des huit ressort.
func btGate3(t *testing.T, assauts, temoins []*btFilmBilan) {
	t.Helper()
	seuil := btPMax / btEtats
	t.Logf("########## GATE 3 i14 PAR VALEUR — huit valeurs testees, seuil de Bonferroni "+
		"p <= %.5f", seuil)
	btDistribution(t, "ASSAUT", assauts)
	btDistribution(t, "TEMOIN", temoins)
	for v := 0; v < btEtats; v++ {
		val := uint64(v)
		e := btNouvelleEpreuve(assauts, btVoieDelta, btChampEtat, func(x uint64) bool {
			return x == val
		})
		if e.tirages() == 0 {
			t.Logf("--- DELTA i14 state = %d : AUCUNE lecture", v)
			continue
		}
		btPublier(t, fmt.Sprintf("DELTA i14 state = %d", v), btEprouver(e), seuil)
	}
}

// btDistribution publie l'histogramme des huit valeurs de i14, voie par voie.
func btDistribution(t *testing.T, titre string, bs []*btFilmBilan) {
	t.Helper()
	for v := 0; v < btVoies; v++ {
		var compte [btEtats]int
		hors := 0
		for _, b := range bs {
			for _, e := range b.voies[v][btChampEtat].ech {
				if e.v < btEtats {
					compte[e.v]++
					continue
				}
				hors++
			}
		}
		t.Logf("   %s %-13s i14 : %v (hors domaine R(3) : %d)", titre, btNomVoie(v), compte, hors)
	}
}

// btNbExplosions rend le nombre d'explosions du corpus.
func btNbExplosions(bs []*btFilmBilan) int {
	n := 0
	for _, b := range bs {
		n += len(b.expl)
	}
	return n
}

// btPublier ecrit les chiffres d'une epreuve et son verdict.
func btPublier(t *testing.T, titre string, r btResultat, seuil float64) {
	t.Helper()
	t.Logf("--- %s : %d lecture(s), %d repetition(s) temoin, %d explosion(s)",
		titre, r.tirages, r.reps, r.explosions)
	if r.tirages == 0 {
		t.Logf("    AUCUNE LECTURE — rien a mesurer, et c'est le resultat")
		return
	}
	btHistogramme(t, r)
	t.Logf("    FENETRE : observe %d · TEMOIN A %.1f +- %.1f (x%.2f, p=%.4f) · "+
		"TEMOIN B %.1f +- %.1f (x%.2f, p=%.4f)", r.obs.fenetre,
		r.fenA.moyenne(), r.fenA.ecart(), r.fenA.enrich(), r.fenA.p(),
		r.fenB.moyenne(), r.fenB.ecart(), r.fenB.enrich(), r.fenB.p())
	t.Logf("    CONSISTANCE : %d/%d explosion(s) en exces · TEMOIN A %.1f +- %.1f (p=%.4f) · "+
		"TEMOIN B %.1f +- %.1f (p=%.4f)", r.obs.exces, r.explosions,
		r.excA.moyenne(), r.excA.ecart(), r.excA.p(),
		r.excB.moyenne(), r.excB.ecart(), r.excB.p())
	verdict := "NEGATIF sur ce canal"
	if r.passe(seuil) {
		verdict = "CANDIDAT — exces, significativite et consistance passent"
	}
	t.Logf("    VERDICT %s : %s (exige x%.2f et p <= %.5f contre le TEMOIN B)",
		titre, verdict, btEnrichMin, seuil)
}

// btHistogramme publie l'histogramme des delais, observe contre temoin B.
func btHistogramme(t *testing.T, r btResultat) {
	t.Helper()
	for i := 0; i < btSeaux; i++ {
		haut := "inf"
		if i+1 < btSeaux {
			haut = fmt.Sprintf("%d", btBornesMS[i+1]/1000)
		}
		t.Logf("    delai [%3d s, %4s s[ : observe %8d · temoin B %10.1f · %s",
			btBornesMS[i]/1000, haut, r.obs.histo[i], r.histoB[i],
			btRapport(float64(r.obs.histo[i]), r.histoB[i]))
	}
	t.Logf("    %d lecture(s) qu'aucune explosion ne suit — ecartees de l'histogramme", r.obs.sans)
}

// btDetail imprime une ligne PAR EXPLOSION : un agregat cache toujours le cas qui explique tout.
func btDetail(t *testing.T, e btEpreuve) {
	t.Helper()
	for _, b := range e.bs {
		inst := e.suite(b)
		_, par := btFenetre(inst, b.expl)
		att := btAttendus(b, len(inst))
		for i, ex := range b.expl {
			t.Logf("    %s %7d ms : fenetre %6d lecture(s) · attendu %8.1f · %s",
				b.id, ex, par[i], att[i], btRapport(float64(par[i]), att[i]))
		}
	}
}

// btRapport rend le rapport observe / attendu, ou « n/a » quand l'attendu est nul.
func btRapport(obs, att float64) string {
	if att <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("x%.2f", obs/att)
}
