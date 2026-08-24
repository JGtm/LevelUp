package filmdec

// i56_rank_cross_test.go — INSTRUMENT DE MESURE de l'item 0.4 du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md : le canal DÉDIÉ d'un usage de répulseur ou
// de propulseur, cherché PAR RANG comme il l'a été pour le camouflage (i28) et le
// surbouclier (i5), et NON par croisement temporel avec `i54` (mort, décision D1).
//
// OÙ IL VIT, ET POURQUOI PAS OÙ LE PLAN L'ANNONÇAIT. Le plan nomme
// `internal/analysis/replay/i56_capacity_episodes_research_test.go`. Il ne peut pas y
// vivre : la mesure a besoin du hook non exporté d'i56 (`abilityEnergyHook`), du détecteur
// de records (`matchBipedHeader`) et des désers de production (`consumeByName`), tous
// internes à `filmdec`. Le nom retenu suit le patron du croisement voisin
// (`i54_rank_cross_test.go`), qui pose exactement la même jointure par vie.
//
// LA PRÉDICTION FALSIFIABLE, ÉNONCÉE AVANT LA MESURE. Si `i56`
// biped-spartan-ability-energy date l'usage d'une capacité à charges, alors ses CHUTES de
// quartet fort (une charge consommée — sémantique confirmée le 2026-08-14, et 282 chutes
// sur 282 franchissant un multiple de 16 le 2026-08-15) se concentrent sur les vies dont
// l'identité `i48` est celle d'une capacité à charges : rang 6 (répulseur) et rang 5 / 21
// (propulseur). Les vies aux autres rangs n'en ont pratiquement aucune.
//
// LE CONTRÔLE QUI PEUT ÉCHOUER, ET LE DÉNOMINATEUR QUI LE REND LISIBLE : cf.
// `rank_cross_shared_test.go`, qui porte le tableau et son critère (partagé avec l'item 0.5,
// pour que les deux canaux se jugent au même bar).
//
// LECTURE SEULE, gardé par I56X_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I56X_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
//	  go test ./internal/analysis/filmdec/ -run '^TestI56CrossI48Rank$' -timeout 30m -v

import (
	"os"
	"sort"
	"testing"
)

const i56xFilmEnv = "I56X_FILM"

// i56xSpec : les deux capacités du périmètre (décision D2). Les rangs viennent de la source
// unique `rank_cross_shared_test.go` — ils sont partagés avec les instruments d'`i59`.
var i56xSpec = xrSpec{
	event: "CHUTES i56",
	groups: []xrGroup{
		{name: "RÉPULSEUR (rang 6)", ranks: xrRepulsorRanks},
		{name: "PROPULSEUR (rangs 5/21)", ranks: xrThrusterRanks},
	},
}

// i56xSample est UNE lecture d'i56 localisée, telle que le déserialiseur de production l'a
// publiée. Même forme que celle de l'instrument des chutes, pour que les deux se comparent.
type i56xSample struct {
	slot uint32
	tsUS uint64
	ch   [AbilityEnergyCharges]int
}

func TestI56CrossI48Rank(t *testing.T) {
	dir := os.Getenv(i56xFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i56xFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	t.Logf("composant %d du registre biped : %q", i56Index, s.arch.component(i56Index))
	slotRanks := eaSlotRanks(t, dir)

	energy, records, with56, read56, unread56 := i56xScan(t, s)
	t.Logf("RECORDS delta biped %d · masque∋i56 %d (%.3f %%) · i56 LU %d · illisible %d",
		records, with56, 100*float64(with56)/float64(max(records, 1)), read56, unread56)
	if len(energy) == 0 {
		t.Log("VERDICT : aucune lecture d'i56 sur ce film — rien à croiser")
		return
	}
	xrTable(t, i56xSpec, slotRanks, i56xReadsBySlot(energy), i56xDropsBySlot(energy))
}

// i56xScan parcourt les paquets delta et lit i56 par le DÉSERIALISEUR DE PRODUCTION partout
// où le masque l'annonce — c'est la marche qui déclenche le hook, on ne relit pas les bits
// à côté de lui.
func i56xScan(t *testing.T, s eaFilmSetup) (out []i56xSample, records, with56, read56, unread56 int) {
	t.Helper()
	var (
		hook i56xSample
		got  bool
	)
	prev := abilityEnergyHook
	SetAbilityEnergyHook(func(_ uint32, ch [AbilityEnergyCharges]int) {
		hook.ch, got = ch, true
	})
	defer SetAbilityEnergyHook(prev)

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
				if eaMaskHas(idx, i56Index) {
					with56++
					got = false
					if eaWalkThrough(pay, i0, total, idx, s, i56Index) && got {
						read56++
						out = append(out, i56xSample{slot: slot, tsUS: pk.TimestampUS, ch: hook.ch})
					} else {
						unread56++
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	return out, records, with56, read56, unread56
}

// i56xReadsBySlot compte les lectures d'i56 par vie. C'est LE dénominateur du tableau : une
// vie sans lecture ne peut pas porter de chute, et ne dit donc rien sur l'exclusivité.
func i56xReadsBySlot(energy []i56xSample) map[uint32]int {
	out := map[uint32]int{}
	for _, e := range energy {
		out[e.slot]++
	}
	return out
}

// i56xDropsBySlot compte les CHUTES de charge par vie : une valeur 7 bits dont le QUARTET
// FORT décroît d'une lecture à la suivante, sur le même emplacement de la même vie. Même
// définition qu'`i56_drops_test.go` (2026-08-15), pour que les deux mesures se comparent.
func i56xDropsBySlot(energy []i56xSample) map[uint32]int {
	type key struct {
		slot uint32
		ch   int
	}
	series := map[key][]i56xSample{}
	for _, e := range energy {
		for c := 0; c < AbilityEnergyCharges; c++ {
			if e.ch[c] != AbilityEnergyUnarmed {
				series[key{e.slot, c}] = append(series[key{e.slot, c}], e)
			}
		}
	}
	out := map[uint32]int{}
	for k, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		for i := 1; i < len(ss); i++ {
			prev, cur := ss[i-1].ch[k.ch], ss[i].ch[k.ch]
			if (cur>>4)&0xF < (prev>>4)&0xF {
				out[k.slot]++
			}
		}
	}
	return out
}
