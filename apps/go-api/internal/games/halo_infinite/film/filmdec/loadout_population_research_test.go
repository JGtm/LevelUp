package filmdec

// loadout_population_research_test.go — POURQUOI LA PREDICTION NE BOUGE PAS.
//
// Le test de prediction rend « repare=0, casse=0 » sur deux films. J'ai avance une explication
// (les vies seraient trop courtes pour couvrir deux images-cles) ; l'utilisateur la conteste et
// dit que la plupart des joueurs vivent plus de 20 s. Ce fichier ne discute pas : il compte.
//
// Il rend, pour un film : la distribution du nombre d'images-cles par vie, le nombre de vies
// portant au moins un changement d'arme, le RECOUVREMENT des deux, et — le point decisif —
// combien de changements tombent REELLEMENT a l'interieur d'une fenetre entre deux images-cles
// de la meme vie. Si ce dernier nombre est nul alors que les deux populations existent, la
// cause n'est pas la duree de vie : c'est un defaut de jointure.
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.

import (
	"os"
	"sort"
	"testing"
)

func TestLoadoutPopulation(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	evBySlot := map[uint32][]hwEvent{}
	for _, e := range ev {
		evBySlot[e.Slot] = append(evBySlot[e.Slot], e)
	}
	ref := hwKeyframeRef(t, dir)

	// Distribution du nombre d'images-cles par vie, et duree couverte.
	var counts []int
	var spans []uint64
	multi := 0
	for _, list := range ref.bySlot {
		counts = append(counts, len(list))
		if len(list) >= 2 {
			multi++
			spans = append(spans, list[len(list)-1].TimestampUS-list[0].TimestampUS)
		}
	}
	sort.Ints(counts)
	sort.Slice(spans, func(i, j int) bool { return spans[i] < spans[j] })
	med := func(v []int) int {
		if len(v) == 0 {
			return 0
		}
		return v[len(v)/2]
	}
	medU := func(v []uint64) uint64 {
		if len(v) == 0 {
			return 0
		}
		return v[len(v)/2]
	}
	t.Logf("VIES (slots) vues aux images-cles = %d ; mediane d'images-cles par vie = %d ; "+
		"vies couvrant >= 2 images-cles = %d (%.0f %%)",
		len(counts), med(counts), multi, 100*float64(multi)/float64(len(counts)))
	t.Logf("DUREE couverte par ces vies : mediane = %d s ; max = %d s",
		medU(spans)/1_000_000, func() uint64 {
			if len(spans) == 0 {
				return 0
			}
			return spans[len(spans)-1] / 1_000_000
		}())

	// Recouvrement des deux populations, et position reelle des changements.
	slotsWithEv, overlap := 0, 0
	inside, before, after := 0, 0, 0
	for slot, list := range evBySlot {
		slotsWithEv++
		kf := ref.bySlot[slot]
		if len(kf) >= 2 {
			overlap++
		}
		if len(kf) == 0 {
			after += len(list)
			continue
		}
		lo, hi := kf[0].TimestampUS, kf[len(kf)-1].TimestampUS
		for _, e := range list {
			switch {
			case e.TimestampUS < lo:
				before++
			case e.TimestampUS > hi:
				after++
			default:
				inside++
			}
		}
	}
	t.Logf("VIES portant au moins un changement d'arme = %d ; dont couvrant >= 2 images-cles = %d",
		slotsWithEv, overlap)
	t.Logf("POSITION des %d changements : AVANT la 1re image-cle de leur vie = %d ; "+
		"DANS l'intervalle couvert = %d ; APRES la derniere (ou vie sans image-cle) = %d",
		len(ev), before, inside, after)
	t.Log("LECTURE : si « AVANT » domine, les changements se produisent juste apres la " +
		"reapparition, avant que la premiere image-cle de la vie ne les enregistre — et le test " +
		"de prediction ne peut structurellement rien voir. Ce n'est alors NI une duree de vie " +
		"trop courte, NI un defaut du canal : c'est un mauvais choix de population de mesure.")
}
