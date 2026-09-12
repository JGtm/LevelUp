package filmdec

// fire_inventory_research_test.go — LA MESURE FONCTIONNELLE : nos changements d'arme
// expliquent-ils ce qu'on voit TIRER ?
//
// POURQUOI CET ORACLE. Un tir est une preuve directe et datee a la milliseconde qu'une arme
// etait EN MAIN. Il y en a des centaines par match, contre une poignee de releves d'images-cles
// espaces de 20 s. C'est le seul oracle dense dont on dispose hors ligne.
//
// POURQUOI SANS PONT TIREUR -> VIE. Le film numerote les tireurs par un index interne
// (FireEvent.FilmIndex) qui n'est PAS un slot de bipede, et le pont entre les deux n'existe pas
// dans ce paquet. On mesure donc l'UNION : a l'instant d'un tir, la famille tiree fait-elle
// partie des armes qu'on croit portees par QUELQU'UN ? C'est plus faible qu'un test par joueur,
// mais ce n'est pas trivial — huit joueurs portent une quinzaine de familles sur la trentaine
// du catalogue — et surtout c'est DISCRIMINANT : une arme ramassee entre deux images-cles
// n'appartient a l'union que si le flux delta l'y a mise.
//
// LE TEMOIN EST LA MESURE, PAS UN ORNEMENT : la meme union construite avec les SEULES
// images-cles. L'ecart entre les deux EST l'apport du canal.
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.

import (
	"os"
	"sort"
	"testing"
)

// fiUnionAt construit l'ensemble des familles portees par l'ensemble des vies a l'instant at.
// withDelta = false rend l'union des seules images-cles (le temoin).
func fiUnionAt(
	ref hwKFRef, evBySlot map[uint32][]hwEvent, at uint64, withDelta bool,
) map[uint32]bool {
	union := map[uint32]bool{}
	for slot, list := range ref.bySlot {
		var base map[uint32]bool
		for _, k := range list {
			if k.TimestampUS <= at {
				base = hwFamilies(k)
			}
		}
		if base == nil {
			continue
		}
		if withDelta {
			perComp := map[int]uint32{}
			for _, e := range evBySlot[slot] {
				if e.TimestampUS <= at {
					lpApply(base, perComp, e.CompIndex, e.IDHigh)
				}
			}
		}
		for f := range base {
			union[f] = true
		}
	}
	return union
}

func TestFireEventsExplainedByInventory(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : a l'instant de chaque tir, la famille de l'arme " +
		"tiree doit appartenir a l'union des inventaires reconstitues. On compare DEUX " +
		"reconstitutions : images-cles seules (temoin) et images-cles + flux delta. Le canal " +
		"n'apporte quelque chose que si la seconde explique NETTEMENT plus de tirs que la premiere.")

	fires, err := ScanFilmFireEvents(dir)
	if err != nil {
		t.Fatalf("evenements de tir illisibles : %v", err)
	}
	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	evBySlot := map[uint32][]hwEvent{}
	for _, e := range ev {
		evBySlot[e.Slot] = append(evBySlot[e.Slot], e)
	}
	ref := hwKeyframeRef(t, dir)

	// Les tirs sont tries : on reconstruit l'union une fois par instant distinct.
	sort.SliceStable(fires, func(i, j int) bool { return fires[i].TimestampUS < fires[j].TimestampUS })

	var total, okBase, okDelta, gained, lost int
	var missing []string
	for _, f := range fires {
		fam := uint32(f.WeaponID >> 32)
		if fam == 0 {
			continue
		}
		total++
		base := fiUnionAt(ref, evBySlot, f.TimestampUS, false)
		full := fiUnionAt(ref, evBySlot, f.TimestampUS, true)
		b, d := base[fam], full[fam]
		if b {
			okBase++
		}
		if d {
			okDelta++
		}
		switch {
		case d && !b:
			gained++
		case b && !d:
			lost++
		}
		if !d && len(missing) < 12 {
			missing = append(missing, hwName(fam))
		}
	}
	if total == 0 {
		t.Log("VERDICT : aucun evenement de tir portant une arme sur ce film. Rien a conclure.")
		return
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(total) }
	t.Logf("TIRS portant une arme = %d", total)
	t.Logf("EXPLIQUES par les images-cles SEULES (temoin) = %d (%.1f %%)", okBase, pct(okBase))
	t.Logf("EXPLIQUES par images-cles + flux delta        = %d (%.1f %%)", okDelta, pct(okDelta))
	t.Logf("le delta EXPLIQUE EN PLUS %d tirs (%.1f %%) et en fait perdre %d (%.1f %%)",
		gained, pct(gained), lost, pct(lost))
	if len(missing) > 0 {
		t.Logf("echantillon d'armes tirees et INEXPLIQUEES : %v", missing)
	}
	t.Log("LECTURE : « explique en plus » est l'apport NET du canal — des tirs qu'on ne pouvait " +
		"pas justifier avec les seules images-cles et que les changements d'arme justifient. " +
		"« fait perdre » signale l'inverse : un changement qui retire une arme que le joueur " +
		"utilise encore, donc une lecture fausse.")
}
