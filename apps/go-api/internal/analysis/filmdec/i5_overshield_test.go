package filmdec

// i5_overshield_test.go — INSTRUMENT DE MESURE de la PHASE B du plan
// .ai/V7.5/replay2d/PLAN_ETAT_ACTIF_EQUIPEMENT.md : le SURBOUCLIER se lit-il dans le
// bouclier déjà décodé (i5 object-shield-vitality) ?
//
// B.1 — LA SÉMANTIQUE DE `Point.sh`, VÉRIFIÉE SUR PIÈCES AVANT LA MESURE (et c'est ce qui
// impose de mesurer CÔTÉ FILM, item B.3) : la chaîne de production est
// decodeObjectShieldVitality (déquant [0, 4], vitality.go) -> BipedPosition.ShieldAt()
// (offline_biped.go:99) -> ShieldFraction — qui CLAMPE à 1.0 — -> Point.Sh
// (replay/build.go, decimateTracks). L'artefact ne peut donc PAS discriminer un
// surbouclier d'un bouclier plein : tout ce qui dépasse 1.0 est écrasé AVANT la
// sérialisation. La mesure se fait ici sur la valeur i5 NON clampée (champ Shield [0..4]
// et quantum brut Q), capturée par le MÊME balayage de production
// (ScanFilmBipedPositions, CaptureDirs) — aucune relecture à côté du déser.
//
// B.2 — LE CONTRÔLE QUI PEUT ÉCHOUER, ÉNONCÉ AVANT LA MESURE : sur les porteurs i48 de
// rang 9 (surbouclier, famille A — `084a804d` : 8 lectures ; `06dfe6d9` : 2), le bouclier
// doit DÉPASSER 1.0 (quantum > 64 : le témoin de forme du 2026-08-05 donne [0, 64] pour
// 27 404 quanta d'un film sans mesure de surbouclier) d'une manière SOUTENUE. Témoin
// interne : les vies aux autres rangs ne doivent jamais dépasser 1.0. Si les deux groupes
// dépassent, ou si aucun ne dépasse, le surbouclier ne se lit pas par ce canal — verdict
// NON, chiffré.
//
// LECTURE SEULE, gardé par I5_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I5_FILM=<repo>/data/cache/film_chunks/084a804d \
//	  go test ./internal/analysis/filmdec/ -run '^TestI5OvershieldDiscriminability$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i5FilmEnv = "I5_FILM"

// i5FullShieldQ est le quantum du bouclier plein NORMAL : déquant [0,4] sur 8 bits,
// 1.0 <-> q = 255/4 = 63,75, et le témoin de forme du rejeu donne des quanta dans [0, 64].
// Le discriminant de surbouclier est donc q > i5FullShieldQ.
const i5FullShieldQ = 64

// i5Sample est une mesure de bouclier localisée : le quantum brut et la valeur [0..4].
type i5Sample struct {
	slot  uint32
	tsUS  uint64
	q     uint8
	value float32
}

func TestI5OvershieldDiscriminability(t *testing.T) {
	dir := os.Getenv(i5FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i5FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	slotRanks := eaSlotRanks(t, dir)

	// Balayage de PRODUCTION : mêmes records, mêmes désers que le rejeu — seule différence,
	// on garde la valeur i5 AVANT le clamp de ShieldFraction.
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	opt.CaptureDirs = true
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage biped impossible : %v", err)
	}
	var samples []i5Sample
	for _, p := range pos {
		if !p.HasShield {
			continue
		}
		samples = append(samples, i5Sample{
			slot: p.Slot, tsUS: p.TimestampUS, q: p.Shield.Q, value: p.Shield.Shield,
		})
	}
	t.Logf("positions biped %d · mesures de bouclier i5 %d", len(pos), len(samples))
	if len(samples) == 0 {
		t.Log("VERDICT B : aucune mesure de bouclier — rien à discriminer")
		return
	}
	i5LogGroups(t, samples, slotRanks)
	i5LogCarriers(t, samples, slotRanks)
}

// i5Group agrège un groupe de vies (rang 9 / autres rangs / sans identité).
type i5Group struct {
	slots, n, over, over2 int
	maxQ                  uint8
}

func (g *i5Group) add(list []i5Sample) {
	g.slots++
	for _, sm := range list {
		g.n++
		if sm.q > i5FullShieldQ {
			g.over++
		}
		if int(sm.q) >= 2*i5FullShieldQ {
			g.over2++
		}
		if sm.q > g.maxQ {
			g.maxQ = sm.q
		}
	}
}

func (g i5Group) String() string {
	if g.n == 0 {
		return fmt.Sprintf("%d vies · aucune mesure", g.slots)
	}
	return fmt.Sprintf("%d vies · %d mesures · q>64 : %d (%.2f %%) · q>=128 : %d · q max %d (%.3f)",
		g.slots, g.n, g.over, 100*float64(g.over)/float64(g.n), g.over2,
		g.maxQ, float64(g.maxQ)/255*4)
}

// i5LogGroups publie le contrôle B.2 : dépassement de 1.0 par groupe de rang i48.
func i5LogGroups(t *testing.T, samples []i5Sample, slotRanks map[uint32][]int) {
	bySlot := map[uint32][]i5Sample{}
	for _, sm := range samples {
		bySlot[sm.slot] = append(bySlot[sm.slot], sm)
	}
	var g9, gOther, gNoID i5Group
	qHist := map[int]int{}
	for sl, list := range bySlot {
		ranks, hasID := slotRanks[sl]
		switch {
		case eaHasRank(ranks, 9):
			g9.add(list)
		case hasID:
			gOther.add(list)
		default:
			gNoID.add(list)
		}
		for _, sm := range list {
			qHist[int(sm.q)]++
		}
	}
	t.Log("== CONTRÔLE B.2 — dépassement du bouclier plein (q > 64) par groupe de rang i48 ==")
	t.Logf("  GROUPE rang 9 (surbouclier) : %s", g9)
	t.Logf("  GROUPE autres rangs         : %s", gOther)
	t.Logf("  GROUPE sans identité i48    : %s", gNoID)
	over := 0
	for q, n := range qHist {
		if q > i5FullShieldQ {
			over += n
		}
	}
	t.Logf("  histogramme global : %d quanta distincts · %d mesures au-delà de 64 · %s",
		len(qHist), over, i28RenderCapped(qHist, 40))
	t.Log("RAPPEL du critère B.2 : le groupe rang 9 doit dépasser 1.0 de manière soutenue, " +
		"les autres jamais — sinon verdict NON")
}

// i5LogCarriers publie, vie par vie de rang 9, la trajectoire du dépassement — c'est elle
// qui écrit la règle de détection (« sh > X pendant Y ») si elle existe.
func i5LogCarriers(t *testing.T, samples []i5Sample, slotRanks map[uint32][]int) {
	bySlot := map[uint32][]i5Sample{}
	for _, sm := range samples {
		bySlot[sm.slot] = append(bySlot[sm.slot], sm)
	}
	slots := make([]uint32, 0, len(bySlot))
	for sl := range bySlot {
		slots = append(slots, sl)
	}
	sort.Slice(slots, func(a, b int) bool { return slots[a] < slots[b] })
	t.Log("== VIES RANG 9 — dépassements dans le temps ==")
	found := false
	for _, sl := range slots {
		if !eaHasRank(slotRanks[sl], 9) {
			continue
		}
		found = true
		list := bySlot[sl]
		sort.Slice(list, func(a, b int) bool { return list[a].tsUS < list[b].tsUS })
		var overTs []uint64
		maxQ := uint8(0)
		for _, sm := range list {
			if sm.q > i5FullShieldQ {
				overTs = append(overTs, sm.tsUS)
			}
			if sm.q > maxQ {
				maxQ = sm.q
			}
		}
		eps := eaGroupEpisodes(sl, overTs, 2_000_000)
		t.Logf("  slot %-5d rangs %v : %d mesures · %d au-delà de 64 · q max %d (%.3f) · %d épisodes q>64",
			sl, eaRankSet(slotRanks[sl]), len(list), len(overTs), maxQ, float64(maxQ)/255*4, len(eps))
		for _, e := range eps {
			t.Logf("    épisode : début t=%.2f s · durée %.2f s · %d mesures",
				float64(e.startUS-list[0].tsUS)/1e6, float64(e.endUS-e.startUS)/1e6, e.n)
		}
	}
	if !found {
		t.Log("  aucune vie de rang 9 avec mesure de bouclier dans ce film")
	}
	// Témoin inversé : les épisodes q>64 des vies NON rang 9, s'il y en a, sont publiés —
	// un dépassement hors porteur réfuterait le discriminant.
	t.Log("== TÉMOIN — dépassements q>64 sur les vies NON rang 9 ==")
	leaks := 0
	for _, sl := range slots {
		if eaHasRank(slotRanks[sl], 9) {
			continue
		}
		list := bySlot[sl]
		n := 0
		maxQ := uint8(0)
		for _, sm := range list {
			if sm.q > i5FullShieldQ {
				n++
			}
			if sm.q > maxQ {
				maxQ = sm.q
			}
		}
		if n > 0 {
			leaks++
			t.Logf("  slot %-5d rangs %v : %d/%d mesures au-delà de 64 · q max %d",
				sl, eaRankSet(slotRanks[sl]), n, len(list), maxQ)
		}
	}
	if leaks == 0 {
		t.Log("  aucun dépassement hors rang 9")
	}
}
