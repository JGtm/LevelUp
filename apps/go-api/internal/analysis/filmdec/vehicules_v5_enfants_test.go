package filmdec

// vehicules_v5_enfants_test.go — LOT V5, PISTE 3 : LES ENTITÉS-ENFANTS (sièges, tourelles).
//
// L'HYPOTHÈSE. L'attachement runtime est prouvé côté exécutable
// (`Object_AttachObjectToMarker`, cf. GHIDRA_ATTACHEMENT_VEHICULE_2026-09-01.md). Si un SIÈGE
// occupé — ou la tourelle d'un artilleur — existe comme entité répliquée, alors une entité
// APPARAÎT dans la table d'image-clé au début d'un épisode et DISPARAÎT à sa fin. Cette
// coïncidence de VIE est mesurable sans décoder le moindre corps de record.
//
// LA MESURE. Pour chaque entité de la table d'image-clé, on connaît sa première et sa
// dernière apparition. On demande : combien d'entités ont une vie CONTENUE dans un épisode
// attesté (aux tolérances d'une image-clé près) ? Et combien en attend-on par hasard, vu la
// part du temps que les épisodes couvrent ?
//
// TÉMOIN INTÉGRÉ : la même mesure avec les épisodes décalés de 37 s.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5Enfants -v -timeout 180m

import (
	"sort"
	"testing"
)

// v5Vie est l'intervalle de vie d'une entité dans la table d'image-clé.
type v5Vie struct {
	Slot, TI       int
	Debut, Fin     uint64
	NombreParution int
}

// TestV5Enfants confronte la vie des entités aux épisodes attestés.
func TestV5Enfants(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5EnfantsUnFilm(t, dir)
	}
}

func v5EnfantsUnFilm(t *testing.T, dir string) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5 ENFANTS %s : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	vies := map[int]*v5Vie{}
	var tsKf []uint64
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		tsKf = append(tsKf, kf[0].TS)
		for _, r := range kf {
			v := vies[r.Slot]
			if v == nil {
				v = &v5Vie{Slot: r.Slot, TI: r.TI, Debut: r.TS}
				vies[r.Slot] = v
			}
			v.Fin = r.TS
			v.NombreParution++
			v.TI = r.TI
		}
	}
	if len(tsKf) < 2 {
		t.Logf("V5 ENFANTS %s : pas assez d'images-clés", dir)
		return
	}
	sort.Slice(tsKf, func(i, j int) bool { return tsKf[i] < tsKf[j] })
	duree := tsKf[len(tsKf)-1] - tsKf[0]

	compte := func(decal uint64) (contenues int, parTI map[int]int) {
		parTI = map[int]int{}
		for _, v := range vies {
			for _, e := range eps {
				if v.Debut >= e.DebutUS+decal && v.Fin <= e.FinUS+decal {
					contenues++
					parTI[v.TI]++
					break
				}
			}
		}
		return contenues, parTI
	}
	// Part du temps couverte par les épisodes : c'est l'attente par hasard pour une entité
	// dont la vie est courte. Sans elle, « des entités vivent pendant un épisode » ne dit rien.
	var couvert uint64
	for _, e := range eps {
		couvert += e.FinUS - e.DebutUS
	}
	part := 0.0
	if duree > 0 {
		part = float64(couvert) / float64(duree)
	}
	reel, parTI := compte(0)
	temoin, _ := compte(v5DecalTemoinUS)
	t.Logf("V5 ENFANTS %s — %d entités distinctes dans les images-clés, %d épisodes couvrant "+
		"%.1f %% du temps", dir, len(vies), len(eps), part*100)
	t.Logf("    entités dont la VIE ENTIÈRE tombe dans un épisode : réel=%d, témoin décalé=%d",
		reel, temoin)
	for ti, n := range parTI {
		t.Logf("        dont ti=%d : %d", ti, n)
	}
	// Combien d'entités apparaissent ET disparaissent entre deux images-clés consécutives ?
	// Une entité vue une seule fois ne peut pas être appariée à un épisode long : c'est la
	// borne de résolution de cette piste, et elle doit être dite.
	uneSeule := 0
	for _, v := range vies {
		if v.NombreParution == 1 {
			uneSeule++
		}
	}
	t.Logf("    entités vues à UNE SEULE image-clé : %d/%d (résolution de la piste)",
		uneSeule, len(vies))
}
