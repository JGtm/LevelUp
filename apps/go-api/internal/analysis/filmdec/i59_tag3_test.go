package filmdec

// i59_tag3_test.go — INSTRUMENT DE MESURE de la PHASE E du plan
// .ai/V7.5/replay2d/PLAN_ETAT_ACTIF_EQUIPEMENT.md : compter et DATER les occurrences
// tag==3 d'i59 `biped-spartan-ability-non-predicted-state` — SANS décoder le corps.
//
// AU MOMENT DE CETTE MESURE (16/08, phase E), LE CORPS N'ÉTAIT PAS PORTÉ : la branche
// tag==3 appelle FUN_142f25e90 — ce que cet instrument a établi, c'est que ce portage
// VALAIT un lot (115/117 lectures tag==3 à porteur identifié = rang 20, grappin). Le
// corps est PORTÉ depuis (plan PLAN_GRAPPIN_LIGNE phase 0, components_biped_anchor.go) ;
// cet instrument reste la mesure de compte/co-datation de la phase E — sa marche s'arrête
// toujours à i59 (cible), le tag externe lui suffit.
//
// E.2 — CROISEMENT avec les signaux de la phase C : si les instants tag==3 co-datent avec
// les naissances d'entités ti=37 ou les transitions d'`equipment-activated`, le portage du
// corps devient un lot justifié — à inscrire au registre comme reprise, PAS à exécuter ici.
// Canaux clairsemés : écarts au plus proche voisin d'abord, fenêtres larges et témoins
// décalés ±5 s ensuite (décision n°2 du plan).
//
// LECTURE SEULE, gardé par I59_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I59_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI59Tag3Count$' -timeout 60m -v

import (
	"os"
	"testing"
)

const i59FilmEnv = "I59_FILM"

// i59Sample est une lecture du tag R(2) d'i59, datée et attribuée à un slot.
type i59Sample struct {
	slot uint32
	tsUS uint64
	tag  uint32
}

func TestI59Tag3Count(t *testing.T) {
	dir := os.Getenv(i59FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	idx59 := s.arch.indicesOfFirst("biped-spartan-ability-non-predicted-state-component")
	if idx59 < 0 {
		idx59 = s.arch.indicesOfFirst("biped-spartan-ability-non-predicted-state")
	}
	if idx59 < 0 {
		t.Fatalf("biped-spartan-ability-non-predicted-state absent de l'archétype — composants : %v",
			s.arch.Components)
	}
	t.Logf("composant %d du registre biped : %q", idx59, s.arch.component(idx59))
	slotRanks := eaSlotRanks(t, dir)

	samples, records, with59, read, unread := i59Scan(s, idx59)
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("E.1 — RECORDS delta biped %d · masque∋i59 %d · LUS %d · illisibles %d",
		records, with59, read, unread)
	tags := map[uint32]int{}
	var tag3 []i59Sample
	for _, sm := range samples {
		tags[sm.tag]++
		if sm.tag == 3 {
			tag3 = append(tag3, sm)
		}
	}
	t.Logf("  tags R(2) : %s", i48RenderU32(tags))
	t.Logf("  tag==3 : %d occurrences, datées ci-dessous", len(tag3))
	var tag3Ts []uint64
	for _, sm := range tag3 {
		t.Logf("    slot %-5d rangs i48 %v · t=%.2f s", sm.slot, eaRankSet(slotRanks[sm.slot]),
			float64(sm.tsUS)/1e6)
		tag3Ts = append(tag3Ts, sm.tsUS)
	}
	if len(tag3) == 0 {
		t.Log("VERDICT E : aucun tag==3 sur ce film — rien à co-dater, portage du corps sans objet ici")
		return
	}
	// E.2 — les signaux de la phase C, sur le MÊME film.
	lives := eaScanTi37Lives(t, dir)
	var births []uint64
	for _, l := range lives {
		births = append(births, l.firstUS)
	}
	births = eaSortedU64(births)
	acts := i57hActivatedTransitions(t, dir)
	var actTs []uint64
	for _, a := range acts {
		actTs = append(actTs, a.tsUS)
	}
	actTs = eaSortedU64(actTs)
	tag3Ts = eaSortedU64(tag3Ts)
	t.Logf("== E.2 — CO-DATATION : tag==3 %d · naissances ti=37 %d · transitions activated %d ==",
		len(tag3Ts), len(births), len(actTs))
	i57hPairReport(t, "i59 tag==3 -> naissance ti=37 la plus proche", tag3Ts, births)
	i57hPairReport(t, "i59 tag==3 -> transition activated la plus proche", tag3Ts, actTs)
	t.Log("RAPPEL du gate E : le compte, la co-datation, et la RECOMMANDATION écrite — le " +
		"portage du corps n'est PAS exécuté ici")
}

// i59Scan balaye les records biped et fait publier le tag d'i59 par le déser de
// production. La marche s'arrête à i59 (cible) : le corps tag==3 n'est jamais parcouru.
func i59Scan(s eaFilmSetup, idx59 int) (samples []i59Sample, records, with59, read, unread int) {
	var capt struct {
		tag uint64
		got bool
	}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) {
		capt.tag, capt.got = uint64(st.Tag), true
	})
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
				if eaMaskHas(idx, idx59) {
					with59++
					capt.got = false
					if eaWalkThrough(pay, i0, total, idx, s, idx59) && capt.got {
						read++
						samples = append(samples, i59Sample{slot: slot, tsUS: pk.TimestampUS, tag: uint32(capt.tag)})
					} else {
						unread++
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	return samples, records, with59, read, unread
}
