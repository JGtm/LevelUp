package filmdec

// loadout_prediction_research_test.go — LA MESURE QUI COMPTE : sait-on, a tout instant, ce
// que chaque joueur tient dans les mains ?
//
// POURQUOI CELLE-CI ET PAS LA PRECEDENTE. Le taux de « couverture » mesurait combien
// d'ARRIVEES d'arme reperees par les images-cles trouvaient un evenement delta. Il melangeait
// deux causes : un evenement manquant de notre cote, et une arrivee FANTOME cote images-cles
// (le lecteur d'image-cle est un balayage ancre, il peut rater une famille a un releve et la
// trouver au suivant, ce qui fabrique une fausse arrivee). Un chiffre qui melange deux causes
// ne dit rien d'actionnable.
//
// LA MESURE DIRECTE, elle, ne melange rien : on PREDIT. On part de l'inventaire lu a une
// image-cle, on applique tous les changements du flux delta jusqu'a l'image-cle suivante, et
// on compare a ce que cette image-cle montre reellement. Si la prediction tombe juste, le
// canal est complet. Si elle tombe faux, il manque quelque chose — et le TEMOIN dit combien
// le flux delta apporte : la meme comparaison SANS appliquer aucun changement.
//
// C'est la question du produit, mot pour mot : « quelle arme ce joueur tient-il maintenant ? »
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.

import (
	"os"
	"sort"
	"testing"
)

// lpApply applique une emission d'identite a un inventaire vu comme un ENSEMBLE de familles.
// Les images-cles ne donnent qu'un ensemble : on ne peut donc pas suivre les emplacements un
// a un, on suit ce que le joueur PORTE.
func lpApply(set map[uint32]bool, perComp map[int]uint32, comp int, fam uint32) {
	if old, ok := perComp[comp]; ok && old != noVariant {
		delete(set, old)
	}
	if fam != noVariant {
		set[fam] = true
	}
	perComp[comp] = fam
}

// lpKeys rend la signature triee d'un ensemble de familles.
func lpKeys(set map[uint32]bool) string {
	out := make([]uint32, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	s := ""
	for _, f := range out {
		s += hwName(f) + "|"
	}
	return s
}

// TestLoadoutPrediction mesure la prediction d'inventaire d'une image-cle a la suivante.
func TestLoadoutPrediction(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : partant de l'inventaire d'une image-cle et en " +
		"appliquant les changements du flux delta, l'inventaire predit doit egaler celui de " +
		"l'image-cle suivante. Le TEMOIN est la meme comparaison SANS appliquer les changements " +
		"(donc « rien n'a bouge »). Le canal n'apporte quelque chose que si la prediction bat " +
		"nettement le temoin.")

	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	evBySlot := map[uint32][]hwEvent{}
	for _, e := range ev {
		evBySlot[e.Slot] = append(evBySlot[e.Slot], e)
	}
	ref := hwKeyframeRef(t, dir)

	var pairs, exact, witness, improved, broken int
	for slot, list := range ref.bySlot {
		for i := 1; i < len(list); i++ {
			lo, hi := list[i-1].TimestampUS, list[i].TimestampUS
			want := lpKeys(hwFamilies(list[i]))
			base := lpKeys(hwFamilies(list[i-1]))

			pred := hwFamilies(list[i-1])
			perComp := map[int]uint32{}
			for _, e := range evBySlot[slot] {
				if e.TimestampUS > lo && e.TimestampUS <= hi {
					lpApply(pred, perComp, e.CompIndex, e.IDHigh)
				}
			}
			got := lpKeys(pred)

			pairs++
			okPred, okBase := got == want, base == want
			if okPred {
				exact++
			}
			if okBase {
				witness++
			}
			switch {
			case okPred && !okBase:
				improved++
			case !okPred && okBase:
				broken++
				t.Logf("  CASSE slot=%d [%d->%d]ms  %s  =(delta)=>  %s  mais l'image-cle dit %s",
					slot, lo/1000, hi/1000, base, got, want)
			}
		}
	}
	if pairs == 0 {
		t.Log("VERDICT : aucune paire d'images-cles consecutives. Rien a conclure.")
		return
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(pairs) }
	t.Logf("PAIRES D'IMAGES-CLES CONSECUTIVES = %d", pairs)
	t.Logf("PREDICTION EXACTE avec le flux delta = %d (%.1f %%)", exact, pct(exact))
	t.Logf("TEMOIN « rien n'a bouge »            = %d (%.1f %%)", witness, pct(witness))
	t.Logf("le delta REPARE %d paires (%.1f %%) et en CASSE %d (%.1f %%)",
		improved, pct(improved), broken, pct(broken))
	t.Log("LECTURE : « repare » = le joueur a change d'arme et le delta l'a dit correctement. " +
		"« casse » = le delta a annonce un changement que l'image-cle suivante contredit — " +
		"c'est la seule categorie qui signale une ERREUR de lecture, et non un simple manque.")
}
