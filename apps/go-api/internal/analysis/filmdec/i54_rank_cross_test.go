package filmdec

// i54_rank_cross_test.go — INSTRUMENT DE MESURE de la PHASE D du plan
// .ai/V7.5/replay2d/PLAN_ETAT_ACTIF_EQUIPEMENT.md : le croisement JAMAIS FAIT,
// épisodes `i54` x identité `i48` de la MÊME VIE (même slot). C'était la reprise n°1 du
// registre des reports.
//
// LA PRÉDICTION FALSIFIABLE, ÉNONCÉE AVANT LA MESURE (item D.1) : si `i54`
// biped-mobility-action est l'événement d'usage des équipements de MOBILITÉ, ses épisodes
// se concentrent sur les vies à rang de mobilité — 4 (grappin), 5 (propulseur),
// 6 (répulseur) en famille A ; 20 (grappin), 21 (propulseur) en famille B — et les vies
// aux autres rangs (1, 2, 8, 9, 12, 19, 22, 23...) n'en ont pratiquement aucun.
// Si la prédiction ne tient pas, `i54` est autre chose (glissade, escalade — le corps
// porte des positions et des vecteurs), et la mobilité reste sans instant d'usage par ce
// canal (item D.3) — l'écrire est le livrable.
//
// PROTOCOLE : décision n°2 du plan — AUCUNE coïncidence ±1 s avec `i56` n'est rejouée
// (canal clairsemé, vice de conception documenté). Ici la jointure est PAR VIE, pas par
// fenêtre : un épisode appartient à un slot, le slot a une identité i48.
//
// LECTURE SEULE, gardé par I54X_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I54X_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI54CrossI48Rank$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i54xFilmEnv = "I54X_FILM"

// i54xMobilityRanks : les rangs de mobilité des deux familles de palette (relevés RECETTE
// §13-15 : famille A 4/5/6 = grappin/propulseur/répulseur ; famille B 20/21 =
// grappin/propulseur, relevé Theater du 27/07 sur 000d5950).
var i54xMobilityRanks = []int{4, 5, 6, 20, 21}

// i54xEpisodeGapUS : des lectures flag1==1 consécutives à moins d'une seconde d'écart
// portent LA MÊME action (protocole de l'instrument du 13/08, i54_research_test.go).
const i54xEpisodeGapUS = 1_000_000

func TestI54CrossI48Rank(t *testing.T) {
	dir := os.Getenv(i54xFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i54xFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	idx54 := s.arch.indicesOfFirst("biped-mobility-action-component")
	if idx54 < 0 {
		idx54 = s.arch.indicesOfFirst("biped-mobility-action")
	}
	if idx54 < 0 {
		t.Fatalf("biped-mobility-action absent de l'archétype — composants : %v", s.arch.Components)
	}
	t.Logf("composant %d du registre biped : %q", idx54, s.arch.component(idx54))
	slotRanks := eaSlotRanks(t, dir)

	var (
		capt struct {
			flag1, got bool
		}
		records, with54, read, unread, flag1On int
		onTs                                   = map[uint32][]uint64{}
	)
	prev := mobilityActionHook
	SetMobilityActionHook(func(flag1, _ bool) { capt.flag1, capt.got = flag1, true })
	defer SetMobilityActionHook(prev)

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
				if eaMaskHas(idx, idx54) {
					with54++
					capt.got = false
					if eaWalkThrough(pay, i0, total, idx, s, idx54) && capt.got {
						read++
						if capt.flag1 {
							flag1On++
							onTs[slot] = append(onTs[slot], pk.TimestampUS)
						}
					} else {
						unread++
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("RECORDS delta biped %d · masque∋i54 %d · LUS %d · illisibles %d · flag1==1 %d",
		records, with54, read, unread, flag1On)
	i54xTable(t, onTs, slotRanks)
}

// i54xLifeRow agrège les épisodes d'un groupe de vies.
type i54xLifeRow struct {
	lives, withEp, episodes int
	durs                    []float64
}

func (r *i54xLifeRow) add(eps []eaEpisode) {
	r.lives++
	if len(eps) > 0 {
		r.withEp++
	}
	r.episodes += len(eps)
	for _, e := range eps {
		r.durs = append(r.durs, float64(e.endUS-e.startUS)/1e6)
	}
}

func (r i54xLifeRow) String() string {
	perLife := 0.0
	if r.lives > 0 {
		perLife = float64(r.episodes) / float64(r.lives)
	}
	return fmt.Sprintf("%3d vies · %3d avec épisode · %4d épisodes · %.2f épisode/vie",
		r.lives, r.withEp, r.episodes, perLife)
}

// i54xTable publie le tableau épisodes x rang (gate D) : par rang d'abord, puis les trois
// groupes (mobilité / autres rangs / sans identité), puis les durées.
func i54xTable(t *testing.T, onTs map[uint32][]uint64, slotRanks map[uint32][]int) {
	episodes := map[uint32][]eaEpisode{}
	for sl, ts := range onTs {
		episodes[sl] = eaGroupEpisodes(sl, eaSortedU64(ts), i54xEpisodeGapUS)
	}
	// Tableau PAR RANG : une vie à plusieurs rangs (ramassage en cours de vie) compte dans
	// CHAQUE rang qu'elle a transmis — dit tel quel, le tableau par groupe est disjoint, lui.
	rankRows := map[int]*i54xLifeRow{}
	for sl, ranks := range slotRanks {
		for _, r := range eaRankSet(ranks) {
			if rankRows[r] == nil {
				rankRows[r] = &i54xLifeRow{}
			}
			rankRows[r].add(episodes[sl])
		}
	}
	rankKeys := make([]int, 0, len(rankRows))
	for r := range rankRows {
		rankKeys = append(rankKeys, r)
	}
	sort.Ints(rankKeys)
	t.Log("== GATE D — TABLEAU ÉPISODES i54 x RANG i48 (une vie multi-rangs compte dans chaque rang) ==")
	for _, r := range rankKeys {
		mob := ""
		for _, m := range i54xMobilityRanks {
			if r == m {
				mob = "  <- MOBILITÉ"
			}
		}
		t.Logf("  rang %-2d : %s%s", r, rankRows[r], mob)
	}
	// Groupes DISJOINTS, prédiction D.1.
	var gMob, gOther, gNoID i54xLifeRow
	allSlots := map[uint32]bool{}
	for sl := range slotRanks {
		allSlots[sl] = true
	}
	for sl := range episodes {
		allSlots[sl] = true
	}
	for sl := range allSlots {
		ranks, hasID := slotRanks[sl]
		isMob := false
		for _, m := range i54xMobilityRanks {
			if eaHasRank(ranks, m) {
				isMob = true
			}
		}
		switch {
		case isMob:
			gMob.add(episodes[sl])
		case hasID:
			gOther.add(episodes[sl])
		default:
			gNoID.add(episodes[sl])
		}
	}
	t.Logf("  GROUPE MOBILITÉ (4/5/6/20/21) : %s", gMob)
	t.Logf("  GROUPE autres rangs           : %s", gOther)
	t.Logf("  GROUPE sans identité i48      : %s", gNoID)
	i54xDurations(t, gMob.durs, "mobilité")
	i54xDurations(t, gOther.durs, "autres rangs")
	t.Log("RAPPEL de la prédiction D.1 : les épisodes se concentrent sur le groupe MOBILITÉ ; " +
		"les vies aux autres rangs n'en ont pratiquement aucun. Sinon, i54 est autre chose " +
		"(glissade, escalade) et la mobilité reste sans instant d'usage par ce canal.")
}

// i54xDurations publie la distribution des durées d'épisode d'un groupe.
func i54xDurations(t *testing.T, durs []float64, label string) {
	if len(durs) == 0 {
		t.Logf("  durées (%s) : aucun épisode", label)
		return
	}
	sort.Float64s(durs)
	t.Logf("  durées (%s) : n=%d · médiane %.2f s · p90 %.2f s · max %.2f s",
		label, len(durs), durs[len(durs)/2], durs[len(durs)*9/10], durs[len(durs)-1])
}
