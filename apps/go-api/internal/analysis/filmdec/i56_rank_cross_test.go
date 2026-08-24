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
// LE CONTRÔLE QUI PEUT ÉCHOUER, ET LE DÉNOMINATEUR QUI LE REND LISIBLE. Une exclusivité ne
// se juge PAS sur un compte de chutes : un rang sans chute peut n'avoir aucune LECTURE.
// Le tableau publie donc, par rang, les vies IDENTIFIÉES, les vies ayant au moins une
// lecture d'i56, les lectures, puis seulement les chutes — et le taux se prend sur les
// vies LUES, jamais sur les vies identifiées. Sans cette colonne, « 0 chute sur le rang 2 »
// se lirait comme une exclusivité alors que ce serait un trou de transmission.
//
// LECTURE SEULE, gardé par I56X_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I56X_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
//	  go test ./internal/analysis/filmdec/ -run '^TestI56CrossI48Rank$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i56xFilmEnv = "I56X_FILM"

// i56xRepulsorRank est le rang de palette du répulseur (famille A). Aucun rang famille B
// n'a été établi pour lui — un film de famille B ne le contredit donc pas, il l'ignore.
const i56xRepulsorRank = 6

// i56xThrusterRanks sont les rangs du propulseur : 5 en famille A (RECETTE_LOADOUT §13),
// 21 en famille B (`ability_evade`, nommé par murmur3 le 2026-08-18, REGISTRE_REPORTS).
var i56xThrusterRanks = []int{5, 21}

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
	drops := i56xDropsBySlot(energy)
	reads := i56xReadsBySlot(energy)
	i56xTable(t, slotRanks, reads, drops)
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

// i56xRow agrège un groupe de vies. `lives` compte les vies IDENTIFIÉES, `withRead` celles
// qui ont au moins une lecture d'i56 : c'est sur ces dernières que le taux se prend.
type i56xRow struct {
	lives, withRead, reads, withDrop, drops int
}

func (r *i56xRow) add(reads, drops int) {
	r.lives++
	r.reads += reads
	r.drops += drops
	if reads > 0 {
		r.withRead++
	}
	if drops > 0 {
		r.withDrop++
	}
}

func (r i56xRow) String() string {
	per := 0.0
	if r.withRead > 0 {
		per = float64(r.drops) / float64(r.withRead)
	}
	return fmt.Sprintf("%3d vies · %3d avec lecture i56 · %4d lectures · %3d avec chute · "+
		"%4d chutes · %.2f chute/vie-lue", r.lives, r.withRead, r.reads, r.withDrop, r.drops, per)
}

// i56xTable publie le tableau chutes x rang, puis les groupes disjoints par capacité.
func i56xTable(t *testing.T, slotRanks map[uint32][]int, reads, drops map[uint32]int) {
	rows := map[int]*i56xRow{}
	for sl, ranks := range slotRanks {
		for _, r := range eaRankSet(ranks) {
			if rows[r] == nil {
				rows[r] = &i56xRow{}
			}
			rows[r].add(reads[sl], drops[sl])
		}
	}
	keys := make([]int, 0, len(rows))
	for r := range rows {
		keys = append(keys, r)
	}
	sort.Ints(keys)
	t.Log("== TABLEAU CHUTES i56 x RANG i48 (une vie multi-rangs compte dans chaque rang) ==")
	for _, r := range keys {
		t.Logf("  rang %-2d : %s%s", r, rows[r], i56xTag(r))
	}
	i56xGroups(t, slotRanks, reads, drops)
	t.Log("RAPPEL du critère (item 0.4) : la quasi-totalité de la masse de chutes doit tomber " +
		"sur les vies du rang CIBLE, et 0 ou quasi-0 sur les autres rangs LUS. Un rang sans " +
		"lecture ne prouve rien. Verdict PAR CAPACITÉ, jamais groupé.")
}

// i56xTag marque les rangs cibles dans le tableau.
func i56xTag(r int) string {
	if r == i56xRepulsorRank {
		return "  <- RÉPULSEUR"
	}
	for _, x := range i56xThrusterRanks {
		if r == x {
			return "  <- PROPULSEUR"
		}
	}
	return ""
}

// i56xGroups publie les groupes DISJOINTS : répulseur, propulseur, autres rangs, sans
// identité. Une vie qui a transmis les DEUX rangs cibles est comptée au répulseur et dite
// telle quelle — le cas est rare et ne doit pas se dissoudre en silence.
func i56xGroups(t *testing.T, slotRanks map[uint32][]int, reads, drops map[uint32]int) {
	var gRep, gThr, gOther, gNoID i56xRow
	both := 0
	all := map[uint32]bool{}
	for sl := range slotRanks {
		all[sl] = true
	}
	for sl := range reads {
		all[sl] = true
	}
	for sl := range all {
		ranks, hasID := slotRanks[sl]
		isRep := eaHasRank(ranks, i56xRepulsorRank)
		isThr := false
		for _, x := range i56xThrusterRanks {
			if eaHasRank(ranks, x) {
				isThr = true
			}
		}
		if isRep && isThr {
			both++
		}
		switch {
		case isRep:
			gRep.add(reads[sl], drops[sl])
		case isThr:
			gThr.add(reads[sl], drops[sl])
		case hasID:
			gOther.add(reads[sl], drops[sl])
		default:
			gNoID.add(reads[sl], drops[sl])
		}
	}
	t.Logf("  GROUPE RÉPULSEUR  (rang 6)      : %s", gRep)
	t.Logf("  GROUPE PROPULSEUR (rangs 5/21)  : %s", gThr)
	t.Logf("  GROUPE autres rangs identifiés  : %s", gOther)
	t.Logf("  GROUPE sans identité i48        : %s", gNoID)
	if both > 0 {
		t.Logf("  (dont %d vies ayant transmis les DEUX rangs cibles, comptées au répulseur)", both)
	}
}
