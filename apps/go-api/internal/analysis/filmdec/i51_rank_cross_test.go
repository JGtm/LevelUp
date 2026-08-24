package filmdec

// i51_rank_cross_test.go — INSTRUMENT DE MESURE de l'item 0.5 du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md : le CANDIDAT SECONDAIRE, interrogé
// uniquement parce que l'item 0.4 a échoué pour les deux capacités du périmètre.
//
// CE QUE `i51` EST, D'APRÈS LE BINAIRE, ET CE QUE ÇA IMPLIQUE POUR LA PRÉDICTION.
// `biped-emp-timer-component` (FUN_142f02830, R(8), minuteur quantifié 0..10 s) mesure
// « combien de temps le joueur reste neutralisé » — un effet SUBI par la vie qui le porte,
// pas une action qu'elle déclenche. Le plan le désigne quand même parce qu'il n'avait
// JAMAIS été interrogé, et un candidat jamais mesuré ne se réfute pas par raisonnement.
// La prédiction est donc faible et énoncée telle quelle : SI le répulseur ou le propulseur
// écrivait dans ce champ à l'usage, ses déclenchements se concentreraient sur les vies de
// rang 6 / 5 / 21. La lecture attendue par la sémantique est l'inverse — des déclenchements
// répartis sur les vies TOUCHÉES par une grenade à impulsion ou un Disruptor, sans rapport
// avec la capacité portée.
//
// CE QUI EST COMPTÉ COMME ÉVÉNEMENT. Un minuteur DÉCROÎT par construction : compter ses
// décréments compterait le temps qui passe. L'événement retenu est le DÉCLENCHEMENT — le
// passage d'un quantum nul à un quantum non nul sur la même vie. C'est le seul instant qui
// puisse correspondre à une action.
//
// LE TABLEAU ET SON CRITÈRE sont ceux de `rank_cross_shared_test.go`, identiques à ceux de
// l'item 0.4 : les deux canaux se jugent au même bar, sans quoi la comparaison ne vaut rien.
//
// LECTURE SEULE, gardé par I51X_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I51X_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
//	  go test ./internal/analysis/filmdec/ -run '^TestI51CrossI48Rank$' -timeout 30m -v

import (
	"os"
	"sort"
	"testing"
)

const i51xFilmEnv = "I51X_FILM"

// i51xSpec reprend les groupes de l'item 0.4 : mêmes cibles, même bar.
var i51xSpec = xrSpec{
	event: "DÉCLENCHEMENTS i51",
	groups: []xrGroup{
		{name: "RÉPULSEUR (rang 6)", ranks: []int{6}},
		{name: "PROPULSEUR (rangs 5/21)", ranks: []int{5, 21}},
	},
}

// i51xSample est UNE lecture d'i51 localisée, telle que le déserialiseur de production l'a
// publiée.
type i51xSample struct {
	slot  uint32
	tsUS  uint64
	quant uint32
}

func TestI51CrossI48Rank(t *testing.T) {
	dir := os.Getenv(i51xFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i51xFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	idx51 := s.arch.indicesOfFirst("biped-emp-timer-component")
	if idx51 < 0 {
		idx51 = s.arch.indicesOfFirst("biped-emp-timer")
	}
	if idx51 < 0 {
		t.Fatalf("biped-emp-timer absent de l'archétype — composants : %v", s.arch.Components)
	}
	t.Logf("composant %d du registre biped : %q", idx51, s.arch.component(idx51))
	slotRanks := eaSlotRanks(t, dir)

	timers, records, with51, read51, unread51 := i51xScan(t, s, idx51)
	t.Logf("RECORDS delta biped %d · masque∋i51 %d (%.3f %%) · i51 LU %d · illisible %d",
		records, with51, 100*float64(with51)/float64(max(records, 1)), read51, unread51)
	if len(timers) == 0 {
		t.Log("VERDICT : aucune lecture d'i51 sur ce film — le canal ne date rien ici")
		return
	}
	i51xLogShape(t, timers)
	xrTable(t, i51xSpec, slotRanks, i51xReadsBySlot(timers), i51xTriggersBySlot(timers))
}

// i51xScan parcourt les paquets delta et lit i51 par le DÉSERIALISEUR DE PRODUCTION partout
// où le masque l'annonce — c'est la marche qui déclenche le hook.
func i51xScan(t *testing.T, s eaFilmSetup, idx51 int) (out []i51xSample, records, with51, read51, unread51 int) {
	t.Helper()
	var (
		quant uint32
		got   bool
	)
	prev := empTimerHook
	SetEmpTimerHook(func(q uint32) { quant, got = q, true })
	defer SetEmpTimerHook(prev)

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
				if eaMaskHas(idx, idx51) {
					with51++
					got = false
					if eaWalkThrough(pay, i0, total, idx, s, idx51) && got {
						read51++
						out = append(out, i51xSample{slot: slot, tsUS: pk.TimestampUS, quant: quant})
					} else {
						unread51++
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	return out, records, with51, read51, unread51
}

// i51xLogShape publie la forme du signal. Un champ qui ne vaut jamais que 0 n'a aucun
// déclenchement à porter, et c'est le premier résultat possible — il doit se lire d'un coup
// d'œil, avant tout tableau d'exclusivité.
func i51xLogShape(t *testing.T, timers []i51xSample) {
	t.Helper()
	hist := map[uint64]int{}
	slots := map[uint32]bool{}
	nonZero := 0
	for _, s := range timers {
		hist[uint64(s.quant)]++
		slots[s.slot] = true
		if s.quant != 0 {
			nonZero++
		}
	}
	t.Logf("i51 : %d lectures sur %d vies · quanta NON NULS %d (%.1f %%)",
		len(timers), len(slots), nonZero, 100*float64(nonZero)/float64(max(len(timers), 1)))
	t.Logf("i51 quanta transmis : %s", equipRenderU64(hist))
}

// i51xReadsBySlot compte les lectures d'i51 par vie — le dénominateur du tableau.
func i51xReadsBySlot(timers []i51xSample) map[uint32]int {
	out := map[uint32]int{}
	for _, s := range timers {
		out[s.slot]++
	}
	return out
}

// i51xTriggersBySlot compte les DÉCLENCHEMENTS par vie : passage d'un quantum nul à un
// quantum non nul, sur la même vie, d'une lecture à la suivante. Les décréments qui suivent
// ne sont PAS comptés — ils datent l'écoulement du minuteur, pas son armement.
func i51xTriggersBySlot(timers []i51xSample) map[uint32]int {
	series := map[uint32][]i51xSample{}
	for _, s := range timers {
		series[s.slot] = append(series[s.slot], s)
	}
	out := map[uint32]int{}
	for slot, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		for i := 1; i < len(ss); i++ {
			if ss[i-1].quant == 0 && ss[i].quant != 0 {
				out[slot]++
			}
		}
	}
	return out
}
