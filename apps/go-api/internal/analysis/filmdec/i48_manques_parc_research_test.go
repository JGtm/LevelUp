package filmdec

// i48_manques_parc_research_test.go — INSTRUMENT DE MESURE (pas de production). Lot R2.2
// du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03 : GENERALISER le verdict du cas index.
//
// LA METHODE. Pour chaque film de la liste : balayage STRICT complet (replique production,
// cf. i48_manques_research_test.go) pour dresser les emissions par vie et localiser chaque
// SAUT du compteur R(3) (pas != 1 modulo 8 entre deux emissions d'une meme vie). Puis, pour
// chaque saut (plafonne), balayage RELACHE de la SEULE fenetre [avant, apres] et verdict :
//
//   - SCANNER  : au moins un candidat marche jusqu'a i48 par les desers de production, au
//     bon slot, portant un compteur PREDIT (avant+1 .. avant+pas-1 modulo 8) — les octets
//     existent, le balayage strict les rejette (gardes du candidat en toutes lettres).
//   - FILM     : aucun candidat plausible sur toute la fenetre — la perte est de
//     replication, incompressible.
//
// GARDE : I48M_PARC (patron TRANSLOC_FILM) = repertoires de films separes par des
// virgules ; I48M_MAXJUMPS plafonne les sauts examines (defaut 12 — balayages BORNES).
//
//	CGO_ENABLED=0 I48M_PARC='<depot>/data/cache/film_chunks/01e1f945,<...>/0a44c6cc' \
//	  I48M_MAXJUMPS=12 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI48ManquesParc$' -v -timeout 60m

import (
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	i48mParcEnv = "I48M_PARC"
	i48mMaxJEnv = "I48M_MAXJUMPS"
)

// i48mJump est UN saut de compteur entre deux emissions strictes d'une meme vie.
type i48mJump struct {
	slot          uint32
	before, after i48mCand
	step          int
	want          map[uint32]bool // compteurs predits des emissions manquantes
}

// i48mFindJumps localise les sauts de compteur dans les emissions strictes d'un film.
func i48mFindJumps(ems []i48mCand) []i48mJump {
	bySlot := map[uint32][]i48mCand{}
	for _, e := range ems {
		bySlot[e.Slot] = append(bySlot[e.Slot], e)
	}
	slots := make([]uint32, 0, len(bySlot))
	for sl := range bySlot {
		slots = append(slots, sl)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []i48mJump
	for _, sl := range slots {
		list := bySlot[sl]
		sort.Slice(list, func(i, j int) bool { return list[i].TS < list[j].TS })
		for i := 1; i < len(list); i++ {
			step := (int(list[i].Counter) - int(list[i-1].Counter) + 8) % 8
			if step == 1 || step == 0 {
				continue
			}
			want := map[uint32]bool{}
			for k := 1; k < step; k++ {
				want[uint32((int(list[i-1].Counter)+k)%8)] = true
			}
			out = append(out, i48mJump{
				slot: sl, before: list[i-1], after: list[i], step: step, want: want,
			})
		}
	}
	return out
}

// i48mJumpVerdict balaye la fenetre d'UN saut et classe : candidats marches au compteur
// predit (hits), marches hors prediction (off), rejetes sans marche (reste).
func i48mJumpVerdict(s i48mSetup, j i48mJump) (hits, off []i48mCand, total, pkts int) {
	cands, p, _ := i48mRelaxed(s, j.before.TS, j.after.TS)
	pkts = p
	for _, c := range cands {
		if c.Slot != j.slot {
			continue
		}
		if c.i48mKey() == j.before.i48mKey() || c.i48mKey() == j.after.i48mKey() {
			continue // les deux emissions connues elles-memes
		}
		total++
		if !c.Walked {
			continue
		}
		if j.want[c.Counter%8] {
			hits = append(hits, c)
		} else {
			off = append(off, c)
		}
	}
	return hits, off, total, pkts
}

// TestI48ManquesParc — echantillon de sauts de compteur sur plusieurs films : pour chacun,
// verdict SCANNER (octets retrouves) ou FILM (aucun candidat plausible), et taux global.
func TestI48ManquesParc(t *testing.T) {
	parc := os.Getenv(i48mParcEnv)
	if parc == "" {
		t.Skipf("%s absent : instrument de mesure saute", i48mParcEnv)
	}
	maxJumps := 12
	if v := os.Getenv(i48mMaxJEnv); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxJumps = n
		}
	}
	release := LockProcessDecode()
	defer release()

	examined, scanner, film, missTotal, hitTotal := 0, 0, 0, 0, 0
	for _, dir := range strings.Split(parc, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" || examined >= maxJumps {
			continue
		}
		s := i48mResolve(t, dir)
		ems, unread := i48mStrict(s, 0, math.MaxUint64)
		jumps := i48mFindJumps(ems)
		t.Logf("FILM %s : %d emissions strictes, %d records i48 illisibles, %d sauts de compteur",
			dir, len(ems), len(unread), len(jumps))
		for _, j := range jumps {
			if examined >= maxJumps {
				break
			}
			examined++
			missTotal += j.step - 1
			hits, offPred, cands, pkts := i48mJumpVerdict(s, j)
			t.Logf("  SAUT slot %d : c%d@%dus -> c%d@%dus (pas %d, %d manquante(s)) — fenetre %d paquets, %d candidats slot, %d marches hors prediction",
				j.slot, j.before.Counter, j.before.TS, j.after.Counter, j.after.TS,
				j.step, j.step-1, pkts, cands, len(offPred))
			for _, o := range offPred {
				t.Logf("    (hors prediction : @%dus %s gardes[%s] masque %v c%d rang %s — plancher de bruit)",
					o.TS, o.Variant, o.Guards, o.Idx, o.Counter, eqRankLabel(o.Rank))
			}
			if len(hits) == 0 {
				film++
				t.Logf("    VERDICT FILM : aucun candidat marche au compteur predit sur toute la fenetre")
				continue
			}
			scanner++
			hitTotal += len(hits)
			for _, h := range hits {
				t.Logf("    VERDICT SCANNER : @%dus (chunk %d pkt %d off %d) %s gardes[%s] masque %v c%d rang %s",
					h.TS, h.Chunk, h.Pkt, h.Off, h.Variant, h.Guards, h.Idx,
					h.Counter, eqRankLabel(h.Rank))
			}
		}
	}
	if examined == 0 {
		t.Log("Aucun saut de compteur dans les films fournis.")
		return
	}
	t.Logf("BILAN : %d sauts examines — %d SCANNER (octets retrouves), %d FILM ; %d emissions manquantes annoncees, %d retrouvees (taux de recuperable %.0f %% des sauts, %.0f %% des emissions)",
		examined, scanner, film, missTotal, hitTotal,
		100*float64(scanner)/float64(examined), 100*float64(hitTotal)/float64(missTotal))
}

// i48mConservative dit si un candidat appartient aux DEUX formes du correctif propose
// (rapport R2), les seules aux signatures structurelles recurrentes sur le parc :
//
//   - « sans i0 » : en-tete de production intact (tag=1, bits 16-17 nuls), comptage 2..7,
//     indices strictement croissants mais premier != 0 — Guards se reduit a `noI0`.
//   - « masque dense » : tag=1, bit16=0, porte de masque a 1, R(64) lu bit k = composant
//     63-k, i0 present ET absolu de la bonne region — Guards se reduit a `dense`.
//
// Dans les deux cas la marche de production doit avoir atteint i48. Les candidats a tag
// etranger, comptage 1, i0 non absolu ou ordre de bits inverse restent EXCLUS : sur le
// parc mesure, ces profils sont ceux du plancher de bruit.
func i48mConservative(c i48mCand) bool {
	if !c.Walked {
		return false
	}
	return (c.Variant == "count" && c.Guards == "noI0 ") ||
		(c.Variant == "dense/msb=true" && c.Guards == "dense ")
}

// i48mChain porte les temoins de continuite d'un jeu d'emissions (per-vie).
type i48mChain struct {
	ems, repeats, jumps, missed, firstOff int
}

// i48mChainStats rejoue le temoin de production (countEquipmentCounterStep) sur un jeu
// d'emissions — c'est la MEME arithmetique modulo 8, pour des avant/apres comparables.
func i48mChainStats(ems []i48mCand) i48mChain {
	st := i48mChain{ems: len(ems)}
	bySlot := map[uint32][]i48mCand{}
	for _, e := range ems {
		bySlot[e.Slot] = append(bySlot[e.Slot], e)
	}
	for _, list := range bySlot {
		sort.Slice(list, func(i, j int) bool {
			if list[i].TS != list[j].TS {
				return list[i].TS < list[j].TS
			}
			return list[i].Off < list[j].Off
		})
		for i, e := range list {
			if i == 0 {
				if e.Counter != equipmentFirstCounter {
					st.firstOff++
				}
				continue
			}
			step := (int(e.Counter) - int(list[i-1].Counter) + 8) % 8
			switch step {
			case 0:
				st.repeats++
			case 1:
			default:
				st.jumps++
				st.missed += step - 1
			}
		}
	}
	return st
}

// TestI48ManquesAvantApres mesure le correctif propose, INCONDITIONNELLEMENT (pas de
// prediction) : balayage strict + les deux formes conservatrices sur le film ENTIER, puis
// temoins de continuite avant/apres. Une fausse emission se verrait : elle creerait une
// repetition ou un NOUVEAU saut (7 chances sur 8 pour un compteur aleatoire).
func TestI48ManquesAvantApres(t *testing.T) {
	parc := os.Getenv(i48mParcEnv)
	if parc == "" {
		t.Skipf("%s absent : instrument de mesure saute", i48mParcEnv)
	}
	release := LockProcessDecode()
	defer release()
	var tb, ta i48mChain
	var totalAdd int
	for _, dir := range strings.Split(parc, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		s := i48mResolve(t, dir)
		strictEms, _ := i48mStrict(s, 0, math.MaxUint64)
		cands, _, _ := i48mRelaxed(s, 0, math.MaxUint64)
		seen := map[string]bool{}
		for _, e := range strictEms {
			seen[e.i48mKey()] = true
		}
		merged := append([]i48mCand{}, strictEms...)
		added, addNoI0, addDense := 0, 0, 0
		for _, c := range cands {
			if seen[c.i48mKey()] || !i48mConservative(c) {
				continue
			}
			seen[c.i48mKey()] = true
			merged = append(merged, c)
			added++
			if c.Variant == "count" {
				addNoI0++
			} else {
				addDense++
			}
		}
		before, after := i48mChainStats(strictEms), i48mChainStats(merged)
		totalAdd += added
		t.Logf("FILM %s : AVANT ems=%d sauts=%d manquees=%d rep=%d 1eOff=%d | +%d acceptees (%d sans-i0, %d dense) | APRES ems=%d sauts=%d manquees=%d rep=%d 1eOff=%d",
			dir, before.ems, before.jumps, before.missed, before.repeats, before.firstOff,
			added, addNoI0, addDense, after.ems, after.jumps, after.missed, after.repeats, after.firstOff)
		tb.ems += before.ems
		tb.jumps += before.jumps
		tb.missed += before.missed
		tb.repeats += before.repeats
		tb.firstOff += before.firstOff
		ta.ems += after.ems
		ta.jumps += after.jumps
		ta.missed += after.missed
		ta.repeats += after.repeats
		ta.firstOff += after.firstOff
	}
	t.Logf("TOTAL : AVANT ems=%d sauts=%d manquees=%d rep=%d 1eOff=%d | +%d | APRES ems=%d sauts=%d manquees=%d rep=%d 1eOff=%d",
		tb.ems, tb.jumps, tb.missed, tb.repeats, tb.firstOff, totalAdd,
		ta.ems, ta.jumps, ta.missed, ta.repeats, ta.firstOff)
}
