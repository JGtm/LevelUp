package filmdec

// equipment_change_research_test.go — INSTRUMENT DE MESURE (pas de production).
//
// LA QUESTION. Pour les ARMES, le canal de changement est `weapon-state-type-info` : il
// n'entre au masque du flux delta que lorsque l'identite d'un emplacement CHANGE, donc chaque
// emission est une prise ou un lacher. Pour l'EQUIPEMENT (la capacite d'armure : grappin,
// repulseur, mur, capteur, propulseur), l'equivalent doit etre i48
// `biped-desired-ability-set-component` — deja porte, deja decode, deja publie en LECTURES
// BRUTES par le document (`abilities[]`), mais JAMAIS en evenements.
//
// CE QU'IL FAUT MESURER AVANT D'ECRIRE QUOI QUE CE SOIT :
//
//  1. i48 emet-il PLUS D'UNE FOIS PAR VIE ? Si non, ce n'est qu'une annonce de reapparition
//     et il n'y a aucun ramassage a en tirer. `ability_rank.go` dit « a peu pres une
//     transmission par vie », ce qui ne tranche pas.
//  2. Les emissions successives d'une meme vie changent-elles de RANG ? Un changement de rang
//     en cours de vie EST un ramassage d'equipement — il n'y a pas d'autre cause.
//  3. La PORTE OUVERTE (AbilitySetNoRank) est-elle un etat « plus d'equipement » ? Si oui
//     c'est le lacher / l'epuisement, et `ScanFilmAbilityRanks` le JETTE aujourd'hui.
//  4. Le compteur R(3) suit-il les changements ? S'il s'incremente a chaque changement, il
//     donne un temoin independant du nombre de ramassages.
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.
//
//	CGO_ENABLED=0 HW_FILM=<depot>/data/cache/film_chunks/64e8adfa \
//	  go test ./internal/analysis/filmdec/ -run EquipmentChange -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// compBipedDesiredAbilitySet : les deux orthographes acceptees par le dispatch pour i48.
var compBipedDesiredAbilitySet = []string{
	"biped-desired-ability-set-component", "biped-desired-ability-set",
}

// eqEmission est UNE emission d'i48, gatee ou non, rattachee a son record.
type eqEmission struct {
	Slot        uint32
	Chunk       int
	TimestampUS uint64
	Counter     uint32
	// Rank vaut AbilitySetNoRank quand la porte est OUVERTE — l'etat que le balayage de
	// production jette et que cet instrument conserve.
	Rank int
}

// eqAbilityIndex resout l'index d'i48 par NOM dans l'archetype du film.
func eqAbilityIndex(t *testing.T, arch Archetype) int {
	t.Helper()
	for id := 0; id < archetypeBlockSlots; id++ {
		name := arch.component(id)
		for _, want := range compBipedDesiredAbilitySet {
			if name == want {
				return id
			}
		}
	}
	t.Fatalf("aucun %v dans l archetype biped du registre", compBipedDesiredAbilitySet)
	return -1
}

// eqScan rend TOUTES les emissions d'i48 du flux delta, portes ouvertes comprises.
func eqScan(t *testing.T, s hwSetup) []eqEmission {
	t.Helper()
	idx48 := eqAbilityIndex(t, s.arch)

	var last struct {
		counter uint32
		rank    int
		got     bool
	}
	prev := abilitySetHook
	SetAbilitySetHook(func(counter uint64, rank, _ int) {
		last.counter, last.rank, last.got = uint32(counter), rank, true
	})
	defer SetAbilitySetHook(prev)

	var out []eqEmission
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+s.minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				if maskHas(idx, idx48) {
					last.got = false
					if walkRecordTo(pay, i0, total, idx, s.lay, s.arch, idx48) && last.got {
						out = append(out, eqEmission{
							Slot: slot, Chunk: c, TimestampUS: pk.TimestampUS,
							Counter: last.counter, Rank: last.rank,
						})
					}
					last.got = false
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// eqRankLabel rend un rang lisible, la porte ouverte comprise.
func eqRankLabel(r int) string {
	if r == AbilitySetNoRank {
		return "(aucun)"
	}
	return fmt.Sprintf("rang %d", r)
}

// eqTally porte les comptages d'un film, pour tenir la fonction de mesure sous le seuil.
type eqTally struct {
	gated, changes, counterMoves, counterSame int
	multi, monoRank                           int
	// counterStep1 : transitions dont le compteur avance EXACTEMENT de 1 (modulo 8).
	// counterJump : transitions ou il saute — chaque saut denonce des emissions MANQUEES,
	// et missed les compte. C.est le temoin de completude que les armes n.avaient pas.
	counterStep1, counterJump, missed int
	// firstCounter : la valeur du compteur a la premiere emission de chaque vie.
	firstCounter map[uint32]int
	perSlot      []int
	rankHist     map[int]int
}

// eqCount agrege les emissions par vie.
func eqCount(bySlot map[uint32][]eqEmission) eqTally {
	tl := eqTally{rankHist: map[int]int{}, firstCounter: map[uint32]int{}}
	for slot, list := range bySlot {
		tl.perSlot = append(tl.perSlot, len(list))
		distinct := map[int]bool{}
		for i, e := range list {
			tl.rankHist[e.Rank]++
			if e.Rank == AbilitySetNoRank {
				tl.gated++
			}
			distinct[e.Rank] = true
			if i == 0 {
				tl.firstCounter[slot] = int(e.Counter)
				continue
			}
			if list[i-1].Rank != e.Rank {
				tl.changes++
			}
			if list[i-1].Counter != e.Counter {
				tl.counterMoves++
			} else {
				tl.counterSame++
			}
			// Le compteur est sur 3 bits : l'avance se lit MODULO 8, et un pas de 1 dit
			// « aucune emission entre les deux ». Tout autre pas denonce des manquantes.
			step := (int(e.Counter) - int(list[i-1].Counter) + 8) % 8
			if step == 1 {
				tl.counterStep1++
			} else {
				tl.counterJump++
				tl.missed += (step + 7) % 8
			}
		}
		if len(list) > 1 {
			tl.multi++
		}
		if len(distinct) == 1 {
			tl.monoRank++
		}
	}
	sort.Ints(tl.perSlot)
	return tl
}

// eqLogChronology sort la suite des rangs des vies qui en montrent plusieurs.
func eqLogChronology(t *testing.T, bySlot map[uint32][]eqEmission, origin uint64) {
	t.Helper()
	slots := make([]uint32, 0, len(bySlot))
	for sl := range bySlot {
		slots = append(slots, sl)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	shown := 0
	for _, sl := range slots {
		list := bySlot[sl]
		distinct := map[int]bool{}
		for _, e := range list {
			distinct[e.Rank] = true
		}
		if len(distinct) < 2 || shown >= 12 {
			continue
		}
		shown++
		seq := ""
		for _, e := range list {
			sec := int64(e.TimestampUS-origin) / 1_000_000
			seq += fmt.Sprintf("%s@%02d:%02ds(c%d)  ", eqRankLabel(e.Rank), sec/60, sec%60, e.Counter)
		}
		t.Logf("   vie %-5d : %s", sl, seq)
	}
	if shown == 0 {
		t.Log("   aucune : chaque vie ne montre qu un seul rang.")
	}
}

func TestEquipmentChangeCanal(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : pour qu i48 serve de canal de RAMASSAGE " +
		"d equipement, il faut qu une part non negligeable des vies porte au moins DEUX " +
		"emissions de rangs DIFFERENTS. Si chaque vie n emet qu une fois, ou toujours le " +
		"meme rang, i48 n est qu une annonce de reapparition et il n y a pas d evenement.")

	s := hwResolve(t, dir)
	ev := eqScan(t, s)
	if len(ev) == 0 {
		t.Log("VERDICT : aucune emission i48. Rien a conclure sur ce film.")
		return
	}

	bySlot := map[uint32][]eqEmission{}
	for _, e := range ev {
		bySlot[e.Slot] = append(bySlot[e.Slot], e)
	}
	tl := eqCount(bySlot)

	t.Logf("EMISSIONS = %d sur %d vies", len(ev), len(bySlot))
	t.Logf("PAR VIE : mediane=%d  max=%d  ; vies a plusieurs emissions = %d (%.1f %%)",
		tl.perSlot[len(tl.perSlot)/2], tl.perSlot[len(tl.perSlot)-1], tl.multi,
		100*float64(tl.multi)/float64(len(bySlot)))
	t.Logf("VIES A UN SEUL RANG = %d (%.1f %%) — les autres ont CHANGE d equipement",
		tl.monoRank, 100*float64(tl.monoRank)/float64(len(bySlot)))
	t.Logf("CHANGEMENTS DE RANG en cours de vie = %d", tl.changes)
	t.Logf("PORTE OUVERTE (aucun equipement) = %d emissions (%.1f %%)",
		tl.gated, 100*float64(tl.gated)/float64(len(ev)))
	t.Logf("COMPTEUR R(3) : bouge sur %d transitions, identique sur %d",
		tl.counterMoves, tl.counterSame)
	t.Logf("CONTINUITE DU COMPTEUR : pas de +1 sur %d transitions, SAUT sur %d "+
		"(soit %d emissions manquees estimees)", tl.counterStep1, tl.counterJump, tl.missed)
	fc := map[int]int{}
	for _, c := range tl.firstCounter {
		fc[c]++
	}
	t.Logf("COMPTEUR A LA PREMIERE EMISSION DE CHAQUE VIE : %v "+
		"(une valeur unique dirait que le compteur repart a chaque vie)", fc)

	type kv struct{ r, n int }
	hist := make([]kv, 0, len(tl.rankHist))
	for r, n := range tl.rankHist {
		hist = append(hist, kv{r, n})
	}
	sort.Slice(hist, func(i, j int) bool { return hist[i].n > hist[j].n })
	line := ""
	for i, h := range hist {
		if i >= 12 {
			break
		}
		line += fmt.Sprintf("%s=%d  ", eqRankLabel(h.r), h.n)
	}
	t.Logf("RANGS OBSERVES (top 12) : %s", line)

	t.Log("CHRONOLOGIE DES VIES A PLUSIEURS RANGS (les 12 premieres) :")
	eqLogChronology(t, bySlot, ev[0].TimestampUS)
}

// TestEquipmentPremiereEmissionContreDebutDeVie tranche la question qui decide de tout :
// la PREMIERE emission i48 d'une vie est-elle l'annonce de la reapparition (le joueur nait
// avec cet equipement) ou un RAMASSAGE en cours de vie ?
//
// La reponse change la lecture de 34 vies sur 44 du premier film mesure : si c'est une
// annonce de naissance, ces vies n'ont ramasse RIEN ; si c'est un ramassage, elles ont
// toutes ramasse une fois.
//
// LE TEMOIN est le premier echantillon de position de la meme vie — la naissance du bipede
// dans le flux. Un ecart proche de zero dit « annonce de naissance » ; un ecart etale sur
// des dizaines de secondes dit « ramassage ».
func TestEquipmentPremiereEmissionContreDebutDeVie(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := hwResolve(t, dir)
	ev := eqScan(t, s)
	if len(ev) == 0 {
		t.Log("VERDICT : aucune emission i48.")
		return
	}
	first := map[uint32]uint64{}
	for _, e := range ev {
		if at, ok := first[e.Slot]; !ok || e.TimestampUS < at {
			first[e.Slot] = e.TimestampUS
		}
	}

	pos, err := ScanFilmBipedPositions(dir, ScanFilmOptions{QuantaOnly: true})
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	birth := map[uint32]uint64{}
	for _, p := range pos {
		if at, ok := birth[p.Slot]; !ok || p.TimestampUS < at {
			birth[p.Slot] = p.TimestampUS
		}
	}

	var gaps []int64
	var noBirth int
	for slot, at := range first {
		b, ok := birth[slot]
		if !ok {
			noBirth++
			continue
		}
		gaps = append(gaps, (int64(at)-int64(b))/1000)
	}
	if len(gaps) == 0 {
		t.Log("VERDICT : aucune vie appariee a une naissance. Rien a conclure.")
		return
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i] < gaps[j] })
	var under1s, under3s int
	for _, g := range gaps {
		if g >= -1000 && g <= 1000 {
			under1s++
		}
		if g >= -3000 && g <= 3000 {
			under3s++
		}
	}
	t.Logf("VIES APPARIEES = %d (%d sans naissance connue)", len(gaps), noBirth)
	t.Logf("ECART naissance -> premiere emission i48 (ms) : min=%d  p25=%d  mediane=%d  p75=%d  max=%d",
		gaps[0], gaps[len(gaps)/4], gaps[len(gaps)/2], gaps[3*len(gaps)/4], gaps[len(gaps)-1])
	t.Logf("A MOINS D UNE SECONDE de la naissance = %d (%.1f %%) ; a moins de trois = %d (%.1f %%)",
		under1s, 100*float64(under1s)/float64(len(gaps)),
		under3s, 100*float64(under3s)/float64(len(gaps)))
	t.Log("LECTURE : une majorite ecrasante sous la seconde = la premiere emission est " +
		"l ANNONCE DE NAISSANCE (le joueur reapparait avec cet equipement), et seules les " +
		"emissions SUIVANTES sont des ramassages. Un etalement sur des dizaines de secondes = " +
		"la premiere emission est deja un ramassage.")
}

// TestEquipmentPorteOuverteContreFinDeVie tranche le SENS de la porte ouverte.
//
// Une emission « aucun equipement » a deux causes possibles, et elles ne se publient pas
// pareil : le joueur a CONSOMME son equipement (evenement de jeu, a montrer), ou il est
// MORT (l'inventaire se vide, et le fil des morts le dit deja). Le temoin est le dernier
// echantillon de position de la vie.
func TestEquipmentPorteOuverteContreFinDeVie(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := hwResolve(t, dir)
	ev := eqScan(t, s)
	pos, err := ScanFilmBipedPositions(dir, ScanFilmOptions{QuantaOnly: true})
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	end := map[uint32]uint64{}
	for _, p := range pos {
		if at, ok := end[p.Slot]; !ok || p.TimestampUS > at {
			end[p.Slot] = p.TimestampUS
		}
	}

	var gaps []int64
	for _, e := range ev {
		if e.Rank != AbilitySetNoRank {
			continue
		}
		if last, ok := end[e.Slot]; ok {
			gaps = append(gaps, (int64(last)-int64(e.TimestampUS))/1000)
		}
	}
	if len(gaps) == 0 {
		t.Log("VERDICT : aucune emission a porte ouverte sur ce film.")
		return
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i] < gaps[j] })
	var atDeath int
	for _, g := range gaps {
		if g <= 1000 {
			atDeath++
		}
	}
	t.Logf("EMISSIONS A PORTE OUVERTE = %d", len(gaps))
	t.Logf("TEMPS RESTANT A VIVRE apres l emission (ms) : min=%d  mediane=%d  max=%d",
		gaps[0], gaps[len(gaps)/2], gaps[len(gaps)-1])
	t.Logf("DANS LA DERNIERE SECONDE DE LA VIE = %d (%.1f %%)",
		atDeath, 100*float64(atDeath)/float64(len(gaps)))
	t.Log("LECTURE : une majorite dans la derniere seconde = la porte ouverte est la MORT, " +
		"et le fil des morts le dit deja. Un etalement = c est la CONSOMMATION de " +
		"l equipement, un evenement de jeu a part entiere.")
}
