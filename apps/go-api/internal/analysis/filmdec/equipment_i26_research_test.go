package filmdec

// equipment_i26_research_test.go — INSTRUMENT DE MESURE (pas de production).
//
// LA QUESTION (utilisateur, 2026-08-30) : les HANDLES d'i26 `unit-equipment-component` — la
// liste de jusqu'a 7 optionnels que le porteur emet — designent-ils des ENTITES ? Chaque
// entree fait `porte(1) + valeur(13) + queue(2)` : les largeurs exactes d'un slot d'entite
// (13 bits) et d'une generation (2 bits). Si (valeur, queue) = (slot, gen) d'objets ti=37,
// i26 est LE FIL objet <-> porteur que la proximite ne sait pas donner pour l'equipement
// (refutation de la mesure D : l'equipement tombe en tas avec les grenades).
//
// TROIS MESURES, criteres enonces avant lecture :
//
//  1. APPARTENANCE : les valeurs des handles presents tombent-elles dans la BANDE DE SLOTS
//     ti=37 (celle des images-cles) ? Temoin : la bande ti=42, et la bande bipede. Un canal
//     d'entites d'equipement doit ecraser ses temoins.
//  2. VIE : la paire (valeur, queue) est-elle RECENSEE comme vie ti=37 aux images-cles,
//     a un instant proche de l'emission ?
//  3. LE FIL DU RAMASSAGE : autour d'une prise i48 (schema 26) du MEME slot bipede, la liste
//     i26 de ce slot gagne-t-elle un handle NOUVEAU dans la fenetre [-1 s, +1 s] ? Si oui, ce
//     handle -> (slot, gen) -> pose ti=37 -> GlobalID eqip = l'identite REELLE de l'objet
//     ramasse.
//
// GARDE : HW_FILM, meme convention que les autres instruments du lot.

import (
	"os"
	"sort"
	"testing"
)

// eqiScan rend toutes les emissions d i26 — le scan de production, trie par instant.
func eqiScan(t *testing.T, s hwSetup) []UnitEquipmentEmission {
	t.Helper()
	out, err := ScanFilmUnitEquipment(s.dir)
	if err != nil {
		t.Fatalf("balayage i26 : %v", err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// eqiBands rend les bandes de slots par archetype, lues des images-cles.
func eqiBands(dir string) (ti37, ti42 map[uint32]bool, biped SlotBand) {
	n := CountFilmChunks(dir)
	ti37 = worldObjectSlotBandDir(dir, n, EquipmentTypeIndex)
	ti42 = worldObjectSlotBandDir(dir, n, GroundWeaponTypeIndex)
	biped = bipedSlotBandDir(dir, chunkList(n))
	return ti37, ti42, biped
}

func chunkList(n int) []int {
	out := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, i)
	}
	return out
}

func TestI26HandlesAppartenance(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : si les handles d i26 designent des entites " +
		"d equipement, la part de leurs valeurs dans la bande ti=37 ECRASE les temoins " +
		"(bande ti=42, bande bipede). Des parts comparables = un compteur, pas une reference.")

	s := hwResolve(t, dir)
	ev := eqiScan(t, s)
	if len(ev) == 0 {
		t.Log("VERDICT : aucune emission i26 lue.")
		return
	}
	ti37, ti42, biped := eqiBands(dir)
	kf := ScanFilmWorldObjectKeyframes(dir, EquipmentTypeIndex)

	var emissions, entries, present, absent int
	var in37, in42, inBiped, inLives int
	countHist := map[int]int{}
	headHist := map[uint32]int{}
	tailHist := map[uint32]int{}
	for _, e := range ev {
		emissions++
		countHist[len(e.Read.Entries)]++
		headHist[e.Read.Head]++
		for _, en := range e.Read.Entries {
			entries++
			if !en.Present {
				absent++
				continue
			}
			present++
			tailHist[en.Tail]++
			if ti37[en.Val] {
				in37++
			}
			if ti42[en.Val] {
				in42++
			}
			if biped.Has(en.Val) {
				inBiped++
			}
			if seen := kf.SeenUS[EquipmentLifeKey{Slot: en.Val, Gen: en.Tail}]; len(seen) > 0 {
				inLives++
			}
		}
	}
	t.Logf("EMISSIONS i26 = %d ; entrees=%d (presentes=%d, portes fermees=%d)",
		emissions, entries, present, absent)
	t.Logf("TAILLE DE LISTE : %v ; EN-TETE R(3) : %v ; QUEUE R(2) des presentes : %v",
		countHist, headHist, tailHist)
	if present == 0 {
		t.Log("VERDICT : aucune entree presente — i26 n emet que des portes fermees ici.")
		return
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(present) }
	t.Logf("APPARTENANCE des valeurs presentes : bande ti=37 = %d (%.1f %%) ; bande ti=42 = "+
		"%d (%.1f %%) ; bande bipede = %d (%.1f %%)",
		in37, pct(in37), in42, pct(in42), inBiped, pct(inBiped))
	t.Logf("PAIRES (valeur, queue) RECENSEES comme vies ti=37 aux images-cles = %d (%.1f %%)",
		inLives, pct(inLives))
}

func TestI26FilDuRamassage(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : a une prise i48 d un slot, la liste i26 du MEME " +
		"slot doit gagner un handle NOUVEAU dans [-1 s, +1 s]. Seuil >= 70 % des prises, " +
		"temoin (instants decales de +30 s) <= 15 %.")

	s := hwResolve(t, dir)
	ev := eqiScan(t, s)
	i26BySlot := map[uint32][]UnitEquipmentEmission{}
	for _, e := range ev {
		i26BySlot[e.Slot] = append(i26BySlot[e.Slot], e)
	}
	changes, _, err := ScanFilmEquipmentChanges(dir, nil)
	if err != nil {
		t.Fatalf("changements d equipement : %v", err)
	}

	// newHandleNear : un handle present, ABSENT de la derniere liste anterieure a la fenetre,
	// apparait dans une emission de la fenetre.
	newHandleNear := func(slot uint32, at uint64) bool {
		list := i26BySlot[slot]
		before := map[uint32]bool{}
		for _, e := range list {
			if e.TimestampUS >= at-1_000_000 {
				break
			}
			before = map[uint32]bool{}
			for _, en := range e.Read.Entries {
				if en.Present {
					before[en.Val<<2|en.Tail] = true
				}
			}
		}
		for _, e := range list {
			if e.TimestampUS < at-1_000_000 || e.TimestampUS > at+1_000_000 {
				continue
			}
			for _, en := range e.Read.Entries {
				if en.Present && !before[en.Val<<2|en.Tail] {
					return true
				}
			}
		}
		return false
	}

	var takes, hits, witnessHits int
	for _, ch := range changes {
		if ch.Kind != EquipmentTaken && ch.Kind != EquipmentSpawned {
			continue
		}
		takes++
		if newHandleNear(ch.Slot, ch.TimestampUS) {
			hits++
		}
		if newHandleNear(ch.Slot, ch.TimestampUS+30_000_000) {
			witnessHits++
		}
	}
	if takes == 0 {
		t.Log("VERDICT : aucune prise i48 sur ce film.")
		return
	}
	t.Logf("PRISES (spawned comprises) = %d ; handle NOUVEAU dans la fenetre = %d (%.1f %%) ; "+
		"temoin +30 s = %d (%.1f %%)", takes, hits, 100*float64(hits)/float64(takes),
		witnessHits, 100*float64(witnessHits)/float64(takes))
	t.Log("Si le canal tient, l etape suivante est la RESOLUTION : handle -> vie ti=37 -> " +
		"GlobalID eqip (poses calibrees) = l identite reelle de l equipement ramasse.")

}
