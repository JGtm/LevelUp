package filmdec

// held_weapon_control_research_test.go — LE CONTROLE DE COMPLETUDE, et la classification.
//
// Deux questions, deux tests :
//
//   - TestHeldWeaponDeltaClasse : que SONT les emissions d'identite ? Un lacher se lit sans
//     rien supposer (l'emplacement passe a "absent") ; une prise et un echange se distinguent
//     par l'etat precedent, quand il est connu.
//   - TestHeldWeaponDeltaCorrobore : les emissions couvrent-elles les changements que
//     l'oracle des images-cles revele ? Et le canal i42 (selection) comble-t-il les trous ?
//
// GARDE : HW_FILM, meme convention que held_weapon_delta_research_test.go.

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/weaponv3"
)

// hwCatalogue rend le predicat d'appartenance au catalogue de production.
func hwCatalogue() map[uint32]bool {
	m := make(map[uint32]bool, len(weaponv3.KnownWeaponHigh32))
	for f := range weaponv3.KnownWeaponHigh32 {
		m[f] = true
	}
	return m
}

// hwName nomme une famille, ou la rend en hexa si elle est hors catalogue.
func hwName(v uint32) string {
	if v == noVariant {
		return "(vide)"
	}
	if n, ok := weaponv3.KnownWeaponHigh32[v]; ok {
		return n
	}
	return fmt.Sprintf("0x%08x", v)
}

// hwIdentities filtre les emissions d'identite, triees dans le temps.
func hwIdentities(ev []hwEvent) []hwEvent {
	var out []hwEvent
	for _, e := range ev {
		if e.Kind == hwKindIdentity {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// TestHeldWeaponDeltaClasse classe chaque emission d'identite. Le LACHER se lit sans
// hypothese : l'emplacement passe a "vide". La PRISE et l'ECHANGE demandent l'etat
// precedent, qui n'existe qu'a partir de la deuxieme emission d'un meme couple
// (slot, emplacement) — la premiere reste INDETERMINEE, et c'est ecrit comme tel.
func TestHeldWeaponDeltaClasse(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))

	kfRef := hwKeyframeRef(t, dir)

	type key struct {
		slot uint32
		comp int
	}
	prev := map[key]uint32{}
	seen := map[key]bool{}
	var lachers, prises, echanges, dejaPortees, indetermines int
	for _, e := range ev {
		k := key{e.Slot, e.CompIndex}
		var kind, from string
		switch {
		case e.IDHigh == noVariant:
			kind, from, lachers = "LACHER     ", hwName(prev[k]), lachers+1
		case seen[k] && prev[k] == noVariant:
			kind, from, prises = "PRISE      ", "(vide)", prises+1
		case seen[k]:
			kind, from, echanges = "ECHANGE    ", hwName(prev[k]), echanges+1
		default:
			// Premiere emission du couple : l'etat de depart vient du SPAWN, donc de
			// l'image-cle la plus recente qui precede. La reference est un ENSEMBLE de
			// familles, pas un emplacement : on ne demande donc que l'appartenance.
			ref, ok := kfRef.setAt(e.Slot, e.TimestampUS)
			switch {
			case !ok:
				kind, from, indetermines = "INDETERMINE", "(aucune image-cle)", indetermines+1
			case ref[e.IDHigh]:
				kind, from, dejaPortees = "DEJA PORTEE", "(deja au spawn)", dejaPortees+1
			default:
				kind, from, prises = "PRISE      ", "(absente au spawn)", prises+1
			}
		}
		t.Logf("  %s t=%dms slot=%d i%d  %s -> %s",
			kind, e.TimestampUS/1000, e.Slot, e.CompIndex, from, hwName(e.IDHigh))
		seen[k], prev[k] = true, e.IDHigh
	}
	t.Logf("CLASSEMENT : lachers=%d prises=%d echanges=%d deja_portees=%d indetermines=%d (total=%d)",
		lachers, prises, echanges, dejaPortees, indetermines, len(ev))
	t.Log("LECTURE : le LACHER est le cas non ambigu. Une PRISE de premiere emission est une " +
		"famille ABSENTE du loadout de spawn le plus recent : c'est une acquisition, pas un " +
		"changement d'emplacement. DEJA PORTEE = l'arme etait deja la, le delta ne fait que " +
		"la re-annoncer (changement d'emplacement, pas ramassage).")
}

// hwScanEvents est hwScan sans ses compteurs, pour les appelants qui ne veulent que le flux.
func hwScanEvents(s hwSetup) []hwEvent {
	ev, _, _ := hwScan(s)
	return ev
}

// hwFamilies rend l'ensemble des familles d'un releve d'image-cle.
func hwFamilies(k KeyframeLoadout) map[uint32]bool {
	m := map[uint32]bool{}
	for _, f := range k.Families {
		m[f] = true
	}
	return m
}

// TestHeldWeaponDeltaCorrobore mesure LE taux qui decide : parmi les arrivees d'arme que les
// images-cles revelent, combien sont EXPLIQUEES par une emission d'identite portant la MEME
// famille sur le MEME slot dans la fenetre ? Et, pour celles qui ne le sont pas, le canal
// i42 fournit-il un instant unique ou les dater ?
func TestHeldWeaponDeltaCorrobore(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := hwResolve(t, dir)
	all := hwScanEvents(s)
	idBySlot, selBySlot := map[uint32][]hwEvent{}, map[uint32][]hwEvent{}
	for _, e := range all {
		if e.Kind == hwKindIdentity {
			idBySlot[e.Slot] = append(idBySlot[e.Slot], e)
		} else {
			selBySlot[e.Slot] = append(selBySlot[e.Slot], e)
		}
	}

	kf, err := ScanFilmKeyframeLoadouts(dir, hwCatalogue())
	if err != nil {
		t.Fatalf("images-cles illisibles : %v", err)
	}
	bySlotKF := map[uint32][]KeyframeLoadout{}
	for _, k := range kf {
		bySlotKF[k.Slot] = append(bySlotKF[k.Slot], k)
	}

	t.Log("CRITERE (avant lecture) : une ARRIVEE d'arme entre deux images-cles est EXPLIQUEE " +
		"s'il existe une emission d'identite sur le meme slot, dans la fenetre, dont la famille " +
		"est CETTE arme. Pour les autres, on compte les emissions i42 de la fenetre : une seule " +
		"= la prise est DATABLE sans etre nommee par le delta ; plusieurs = ambigu.")

	arrivals, explained, datable, ambiguous, blind := 0, 0, 0, 0, 0
	for slot, list := range bySlotKF {
		sort.SliceStable(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i := 1; i < len(list); i++ {
			prev, cur := hwFamilies(list[i-1]), hwFamilies(list[i])
			lo, hi := list[i-1].TimestampUS, list[i].TimestampUS
			for fam := range cur {
				if prev[fam] {
					continue
				}
				arrivals++
				if hwHasFamily(idBySlot[slot], lo, hi, fam) {
					explained++
					continue
				}
				switch n := hwCountIn(selBySlot[slot], lo, hi); {
				case n == 1:
					datable++
					t.Logf("  DATABLE PAR i42 slot=%d fenetre=[%d,%d]ms arme=%s",
						slot, lo/1000, hi/1000, hwName(fam))
				case n > 1:
					ambiguous++
					t.Logf("  AMBIGU (%d i42) slot=%d fenetre=[%d,%d]ms arme=%s",
						n, slot, lo/1000, hi/1000, hwName(fam))
				default:
					blind++
					t.Logf("  AVEUGLE slot=%d fenetre=[%d,%d]ms arme=%s",
						slot, lo/1000, hi/1000, hwName(fam))
				}
			}
		}
	}
	if arrivals == 0 {
		t.Log("VERDICT : aucune arrivee d'arme entre images-cles. Rien a conclure.")
		return
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(arrivals) }
	t.Logf("RESULTAT : arrivees=%d ; NOMMEES ET DATEES par i43..i46=%d (%.1f %%) ; "+
		"DATABLES par i42 seul=%d (%.1f %%) ; ambigues=%d (%.1f %%) ; aveugles=%d (%.1f %%)",
		arrivals, explained, pct(explained), datable, pct(datable),
		ambiguous, pct(ambiguous), blind, pct(blind))
	t.Logf("COUVERTURE COMBINEE (nommee + datable) = %.1f %%", pct(explained+datable))
}

// hwHasFamily dit si une emission d'identite de la famille fam tombe dans [lo, hi].
func hwHasFamily(ev []hwEvent, lo, hi uint64, fam uint32) bool {
	for _, e := range ev {
		if e.TimestampUS >= lo && e.TimestampUS <= hi && e.IDHigh == fam {
			return true
		}
	}
	return false
}

// hwCountIn compte les emissions tombant dans [lo, hi].
func hwCountIn(ev []hwEvent, lo, hi uint64) int {
	n := 0
	for _, e := range ev {
		if e.TimestampUS >= lo && e.TimestampUS <= hi {
			n++
		}
	}
	return n
}

// hwKFRef donne, pour un slot et un instant, l'ensemble des familles portees au dernier
// releve d'image-cle qui precede — c'est-a-dire l'etat de SPAWN vu de plus pres.
type hwKFRef struct {
	bySlot map[uint32][]KeyframeLoadout
}

// hwKeyframeRef charge les images-cles du film et les indexe par slot.
func hwKeyframeRef(t *testing.T, dir string) hwKFRef {
	t.Helper()
	kf, err := ScanFilmKeyframeLoadouts(dir, hwCatalogue())
	if err != nil {
		t.Fatalf("images-cles illisibles : %v", err)
	}
	r := hwKFRef{bySlot: map[uint32][]KeyframeLoadout{}}
	for _, k := range kf {
		r.bySlot[k.Slot] = append(r.bySlot[k.Slot], k)
	}
	for slot := range r.bySlot {
		l := r.bySlot[slot]
		sort.SliceStable(l, func(i, j int) bool { return l[i].TimestampUS < l[j].TimestampUS })
		r.bySlot[slot] = l
	}
	return r
}

// setAt rend l'ensemble des familles du dernier releve a ts <= at ; a defaut le PREMIER
// releve du slot (une emission peut preceder la premiere image-cle de la vie). Le second
// retour dit si un releve existe pour ce slot.
func (r hwKFRef) setAt(slot uint32, at uint64) (map[uint32]bool, bool) {
	l := r.bySlot[slot]
	if len(l) == 0 {
		return nil, false
	}
	pick := l[0]
	for _, k := range l {
		if k.TimestampUS <= at {
			pick = k
		}
	}
	return hwFamilies(pick), true
}
