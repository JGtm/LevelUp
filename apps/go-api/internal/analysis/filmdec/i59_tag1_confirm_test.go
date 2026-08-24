package filmdec

// i59_tag1_confirm_test.go — INSTRUMENT DE MESURE de la PHASE 0-BIS du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md : confirmer (ou tuer) `i59 tag==1` comme
// canal d'usage du PROPULSEUR.
//
// D'OÙ VIENT CETTE PHASE. L'étape 0.7 a trouvé que les transitions vers `tag==1` se
// concentrent sur les vies de propulseur : 78,8 % de la masse, ZÉRO sur les 76 vies de
// grappin (le confondeur qui avait avalé `i56`), quasi-zéro ailleurs — mais sur 66
// transitions et 3 films seulement. Le superviseur a validé QUATRE seuils, écrits AVANT
// cette mesure et NON renégociables. Cet instrument les instruit tous les quatre en UNE
// passe par film, parce qu'un second décodage du même film coûte le même prix que le
// premier.
//
// ────────────────────────────────────────────────────────────────────────────────────
// LES QUATRE SEUILS, tels que validés (l'agrégation multi-films se fait hors instrument ;
// chaque exécution publie les NOMBRES BRUTS qui s'additionnent) :
//
//	(1) VOLUME — >= 150 transitions tag==1 CUMULÉES sur 8-10 films couvrant les DEUX
//	    familles de palette (rang 5 en A, rang 21 en B).
//	(2) REPRODUCTIBILITÉ — sur >= 6 films sur 8 : >= 75 % de la masse des transitions sur
//	    les vies de propulseur, <= 0,10 transition par vie-lue sur les vies de GRAPPIN,
//	    <= 0,15 sur les vies SANS identité.
//	(3) DATATION — ÉLIMINATOIRE. Les transitions doivent tomber EN COURS DE VIE, pas à
//	    l'apparition. Une dotation datée au spawn est un NÉGATIF : on ne publiera jamais un
//	    effet qui pulse à chaque réapparition. Le contrôle est celui d'`i54` (2026-08-13) :
//	    offset au début de la vie, seuil « < 2 s = au spawn », avec un TÉMOIN — la même
//	    distribution pour les transitions vers les tags 0 et 2, qui sont, eux, de purs états
//	    génériques et donnent la forme d'un signal SANS rapport avec le spawn.
//	(4) CHARGE UTILE — établir ce que `tag==1` transporte.
//
// ────────────────────────────────────────────────────────────────────────────────────
//
// COMMENT LE SEUIL (4) EST INSTRUIT, ET POURQUOI PAS EN « PORTANT LE CORPS ». Le déser de
// production (`consumeBipedSpartanAbilityNonPredictedState`, components_biped_ability.go)
// ne lit un corps QUE pour `tag==3` : le décompile de FUN_142f2679c donne un `R(2)` plat,
// et la branche FUN_142f25e90 n'est appelée que sur le tag 3. Il n'y a donc pas de corps à
// « porter » pour le tag 1 — l'affirmer demande une PREUVE PAR LE FLUX, pas une lecture de
// décompile, et c'est ce que fait cet instrument : sur les records dont le masque contient
// un composant APRÈS `i59`, la marche complète est tentée. Si `R(2)` était une largeur
// FAUSSE pour le tag 1, le curseur serait décalé et la marche aval casserait bien plus
// souvent sur ces records-là que sur les records à tag 0 ou 2. Un taux d'alignement aval
// comparable entre les tags EST la preuve que le tag 1 ne transporte rien de plus.
//
// LECTURE SEULE, gardé par I59T1_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process. AUCUNE ligne de production n'est touchée par cette phase.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I59T1_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI59Tag1Confirm$' -timeout 60m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i59t1FilmEnv = "I59T1_FILM"

// i59t1SpawnWindowS est la fenêtre « au spawn » du contrôle (3), reprise à l'identique de
// l'instrument d'`i54` du 2026-08-13 — le seuil ne se réinvente pas d'un lot à l'autre.
const i59t1SpawnWindowS = 2.0

// i59t1Read est une lecture d'i59 datée, avec son tag et l'état de la marche AVAL du record
// qui la portait (pour le seuil 4).
type i59t1Read struct {
	slot      uint32
	tsUS      uint64
	tag       uint32
	hasAfter  bool // le masque contient un composant APRÈS i59
	alignedOK bool // la marche complète du masque est allée au bout
}

func TestI59Tag1Confirm(t *testing.T) {
	dir := os.Getenv(i59t1FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59t1FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	idx59 := s.arch.indicesOfFirst("biped-spartan-ability-non-predicted-state")
	if idx59 < 0 {
		t.Fatalf("i59 absent de l'archétype — composants : %v", s.arch.Components)
	}
	slotRanks := eaSlotRanks(t, dir)

	reads, slotFirst, records, with59 := i59t1Scan(s, idx59)
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("RECORDS delta biped %d · masque∋i59 %d · LUS %d · %d vies datées",
		records, with59, len(reads), len(slotFirst))
	if len(reads) == 0 {
		t.Log("VERDICT : aucune lecture d'i59 sur ce film")
		return
	}
	i59t1Volume(t, reads)
	trans := i59t1Transitions(reads)
	t.Log("== SEUILS (1) et (2) — MASSE ET CONTRASTE ==")
	xrTable(t, i59xSpec(1), slotRanks, i59xReadsBySlot(i59t1AsSamples(reads)), trans[1])
	i59t1Datation(t, reads, slotFirst, slotRanks)
	i59t1Payload(t, reads)
}

// i59t1Scan balaye les deltas UNE fois et collecte tout ce dont les quatre seuils ont
// besoin. `slotFirst` est daté sur TOUS les records reconnus, pas seulement sur ceux qui
// portent i59 : sinon le « début de vie » serait la première transmission d'i59 et tous les
// offsets vaudraient ~0 par construction — le contrôle (3) se donnerait raison tout seul.
func i59t1Scan(s eaFilmSetup, idx59 int) (reads []i59t1Read, slotFirst map[uint32]uint64, records, with59 int) {
	slotFirst = map[uint32]uint64{}
	var capt struct {
		tag uint32
		got bool
	}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { capt.tag, capt.got = st.Tag, true })
	defer SetAbilityNonPredictedHook(prev)

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
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
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				records++
				if f, seen := slotFirst[slot]; !seen || pk.TimestampUS < f {
					slotFirst[slot] = pk.TimestampUS
				}
				if eaMaskHas(idx, idx59) {
					with59++
					capt.got = false
					// UNE seule marche, complète : elle déclenche le hook au passage d'i59
					// (le tag) et dit si le masque a été parcouru jusqu'au bout (seuil 4).
					_, _, _, aligned := i57Walk(pay, i0, total, idx, s.lay, s.arch, -1)
					if capt.got {
						reads = append(reads, i59t1Read{
							slot: slot, tsUS: pk.TimestampUS, tag: capt.tag,
							hasAfter: idx[len(idx)-1] > idx59, alignedOK: aligned,
						})
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	return reads, slotFirst, records, with59
}

// i59t1AsSamples convertit les lectures au type attendu par le dénominateur partagé.
func i59t1AsSamples(reads []i59t1Read) []i59Sample {
	out := make([]i59Sample, 0, len(reads))
	for _, r := range reads {
		out = append(out, i59Sample{slot: r.slot, tsUS: r.tsUS, tag: r.tag})
	}
	return out
}

// i59t1Volume publie l'histogramme des tags — le premier chiffre du seuil (1).
func i59t1Volume(t *testing.T, reads []i59t1Read) {
	t.Helper()
	hist := map[uint32]int{}
	for _, r := range reads {
		hist[r.tag]++
	}
	t.Logf("tags R(2) : %s", i48RenderU32(hist))
}

// i59t1Transitions rend, par tag, le compte de transitions VERS ce tag par vie.
func i59t1Transitions(reads []i59t1Read) map[uint32]map[uint32]int {
	series := map[uint32][]i59t1Read{}
	for _, r := range reads {
		series[r.slot] = append(series[r.slot], r)
	}
	out := map[uint32]map[uint32]int{}
	for tag := uint32(0); tag < 4; tag++ {
		out[tag] = map[uint32]int{}
	}
	for slot, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		for i := 1; i < len(ss); i++ {
			if ss[i].tag != ss[i-1].tag {
				out[ss[i].tag][slot]++
			}
		}
	}
	return out
}

// i59t1Datation instruit le SEUIL (3), éliminatoire : les transitions vers le tag 1
// tombent-elles au spawn ou en cours de vie ? Le témoin est la même distribution pour les
// tags 0 et 2 — des états génériques, donc la forme d'un signal sans lien avec le spawn.
func i59t1Datation(t *testing.T, reads []i59t1Read, slotFirst map[uint32]uint64, slotRanks map[uint32][]int) {
	t.Helper()
	t.Log("== SEUIL (3) — DATATION : au spawn ou en cours de vie ? (ÉLIMINATOIRE) ==")
	series := map[uint32][]i59t1Read{}
	for _, r := range reads {
		series[r.slot] = append(series[r.slot], r)
	}
	offs := map[uint32][]float64{}
	var propOffs []float64
	for slot, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		isProp := xrHasAnyRank(slotRanks[slot], xrThrusterRanks)
		for i := 1; i < len(ss); i++ {
			if ss[i].tag == ss[i-1].tag {
				continue
			}
			o := float64(ss[i].tsUS-slotFirst[slot]) / 1e6
			offs[ss[i].tag] = append(offs[ss[i].tag], o)
			if ss[i].tag == 1 && isProp {
				propOffs = append(propOffs, o)
			}
		}
	}
	for tag := uint32(0); tag < 4; tag++ {
		i59t1OffsetLine(t, fmt.Sprintf("transitions -> tag %d", tag), offs[tag])
	}
	i59t1OffsetLine(t, "transitions -> tag 1 SUR VIES DE PROPULSEUR (la population jugée)", propOffs)
	t.Logf("RAPPEL du critère (3) : si la quasi-totalité des transitions tag 1 des vies de "+
		"propulseur tombe à moins de %.0f s du début de la vie ALORS que les tags 0 et 2 "+
		"s'étalent, le tag 1 date une DOTATION et non un usage — verdict NÉGATIF, "+
		"éliminatoire, on classe le propulseur.", i59t1SpawnWindowS)
}

// i59t1OffsetLine publie une distribution d'offsets et la part « au spawn ».
func i59t1OffsetLine(t *testing.T, label string, o []float64) {
	t.Helper()
	if len(o) == 0 {
		t.Logf("  %-58s : aucune transition", label)
		return
	}
	sort.Float64s(o)
	near := 0
	for _, x := range o {
		if x < i59t1SpawnWindowS {
			near++
		}
	}
	t.Logf("  %-58s : n=%3d · AU SPAWN (< %.0f s) %3d (%5.1f %%) · médiane %7.1f s · p90 %7.1f s",
		label, len(o), i59t1SpawnWindowS, near, 100*float64(near)/float64(len(o)),
		o[len(o)/2], o[len(o)*9/10])
}

// i59t1Payload instruit le SEUIL (4) : `tag==1` transporte-t-il quelque chose ? Si `R(2)`
// était une largeur fausse pour ce tag, le curseur serait décalé après lui et la marche
// AVAL casserait plus souvent sur ces records que sur ceux des autres tags.
func i59t1Payload(t *testing.T, reads []i59t1Read) {
	t.Helper()
	t.Log("== SEUIL (4) — CHARGE UTILE : alignement de la marche APRÈS i59, par tag ==")
	type acc struct{ withAfter, aligned int }
	byTag := map[uint32]*acc{}
	for _, r := range reads {
		if !r.hasAfter {
			continue
		}
		if byTag[r.tag] == nil {
			byTag[r.tag] = &acc{}
		}
		byTag[r.tag].withAfter++
		if r.alignedOK {
			byTag[r.tag].aligned++
		}
	}
	for tag := uint32(0); tag < 4; tag++ {
		a := byTag[tag]
		if a == nil || a.withAfter == 0 {
			t.Logf("  tag %d : aucun record avec un composant APRÈS i59 — non testable ici", tag)
			continue
		}
		t.Logf("  tag %d : %4d records avec composant aval · marche complète OK %4d (%5.1f %%)",
			tag, a.withAfter, a.aligned, 100*float64(a.aligned)/float64(a.withAfter))
	}
	t.Log("RAPPEL du critère (4) : un taux d'alignement aval COMPARABLE entre les tags prouve " +
		"que R(2) est la largeur complète du tag 1 — donc qu'il ne transporte AUCUNE charge " +
		"utile, et qu'il est un pur changement d'état (comme l'interrupteur i28 du camo). Un " +
		"taux effondré sur le seul tag 1 dirait l'inverse : une charge non portée, et toute " +
		"la mesure de l'étape 0.7 serait à refaire sur une grammaire corrigée.")
}
