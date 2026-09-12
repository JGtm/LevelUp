package filmdec

// r8_coincidence_research_test.go — MESURE 2 du lot R8 : que fait le canal i48 (rang de
// capacite portee) A L'INSTANT d'une pose `deployed` de famille `repulsor` / `thruster` ?
//
// TROIS LECTURES POSSIBLES DE CES POSES, et elles se separent sur ce canal :
//
//	USAGE       le porteur consomme une charge. Le canal doit rendre, a l'instant, soit
//	            RIEN (il lui reste des charges), soit `spent` depuis le rang de la famille.
//	ECHANGE     le porteur ramasse un AUTRE equipement : le jeu lache l'ancien a ses pieds,
//	            ce qui cree un objet en cours de vie — donc une pose `deployed`. Signature :
//	            `taken` a l'instant, avec `from` = rang de la famille de la pose.
//	SOCLE       l'objet reapparait sur son point fixe (mesure 1). Le canal ne dit rien.
//
// CRITERE ECRIT AVANT LA MESURE. Fenetre de coincidence `r8CoincWindow` = 5 frames
// (500 ms) de part et d'autre : deux fois le pas de publication du canal, assez pour
// absorber l'ecart entre la creation de l'objet et la transmission du rang, trop court
// pour attraper un ramassage sans rapport (les transitions sont rares : ~100 par film).
//
// TEMOIN OBLIGATOIRE : la meme mesure sur les poses `deployed` des familles reellement
// deployables (`wall`, `sensor`, ...). Un mur qu'on DEPOSE ne doit PAS coincider avec un
// `taken` — s'il coincidait autant que les cibles, la fenetre attraperait du hasard.

import (
	"sort"
	"testing"
)

// r8CoincWindow est la demi-fenetre de coincidence, en frames du document (100 ms).
const r8CoincWindow = 5

// r8CoincTally compte les issues d'une population de poses.
type r8CoincTally struct {
	total       int // poses examinees
	withOwner   int // poses a poseur mesure
	rankKnown   int // poses dont la palette du film nomme le rang de la famille
	changesRead int // poses dans un film qui publie `equipmentChanges`
	takenFrom   int // `taken` avec from == rang de la famille  -> ECHANGE
	takenOther  int // `taken` avec from != rang de la famille
	spentFrom   int // `spent` avec from == rang de la famille  -> USAGE
	spentOther  int // `spent` avec from != rang de la famille
	none        int // aucune transition dans la fenetre
	heldFam     int // le porteur PORTAIT la famille juste avant la pose (canal i48)
	heldOther   int
	heldUnknown int
}

// r8HeldRankAt rend le dernier rang lu par i48 pour ce slot a `frame` ou avant, et -1
// quand aucune lecture ne precede (le canal ne transmet que les changements).
func r8HeldRankAt(ab []r8Ability, slot uint32, frame int) int {
	best, bestT := -1, -1
	for _, r := range ab {
		if r.Slot != slot || r.T > frame {
			continue
		}
		if r.T > bestT {
			best, bestT = r.R, r.T
		}
	}
	return best
}

// r8ChangesNear rend les transitions du slot dans la fenetre autour de `frame`.
func r8ChangesNear(ch []r8Change, slot uint32, frame int) []r8Change {
	var out []r8Change
	for _, c := range ch {
		if c.Slot != slot {
			continue
		}
		d := c.T - frame
		if d < 0 {
			d = -d
		}
		if d <= r8CoincWindow {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].T < out[j].T })
	return out
}

// r8Coincide classe UNE pose. `rank` est le rang de sa famille (-1 : non nomme).
func r8Coincide(a *r8Artifact, p r8Placement, rank int, tl *r8CoincTally) {
	tl.total++
	if p.Owner < 0 {
		return
	}
	tl.withOwner++
	slot := uint32(p.Owner) //nolint:gosec // Owner >= 0 teste juste au-dessus
	if rank >= 0 {
		tl.rankKnown++
		switch r8HeldRankAt(a.Abilities, slot, p.T0) {
		case rank:
			tl.heldFam++
		case -1:
			tl.heldUnknown++
		default:
			tl.heldOther++
		}
	}
	if len(a.Changes) == 0 {
		return
	}
	tl.changesRead++
	near := r8ChangesNear(a.Changes, slot, p.T0)
	if len(near) == 0 {
		tl.none++
		return
	}
	for _, c := range near {
		switch {
		case c.Kind == "taken" && c.From == rank && rank >= 0:
			tl.takenFrom++
		case c.Kind == "taken":
			tl.takenOther++
		case c.Kind == "spent" && c.From == rank && rank >= 0:
			tl.spentFrom++
		default:
			tl.spentOther++
		}
	}
}

func TestR8CoincidenceI48(t *testing.T) {
	corpus := r8LoadCorpus(t)
	pops := map[string]*r8CoincTally{}
	get := func(k string) *r8CoincTally {
		if pops[k] == nil {
			pops[k] = &r8CoincTally{}
		}
		return pops[k]
	}
	for _, a := range corpus {
		ranks := map[string]int{
			"repulsor": r8RankOfFamily(a, "repulsor"),
			"thruster": r8RankOfFamily(a, "thruster"),
		}
		for i := range a.Placements {
			p := a.Placements[i]
			if r8OriginOrUnknown(p.Origin) != "deployed" {
				continue
			}
			switch {
			case r8TargetFamilies[p.Family]:
				k := p.Family
				if r8AtSocle(a.Placements, i) {
					k += " (socle)"
				}
				r8Coincide(a, p, ranks[p.Family], get(k))
			case r8DeployableFamilies[p.Family]:
				// Temoin : le rang de la famille n'est pas nomme, on ne teste que la
				// PRESENCE d'une transition dans la fenetre.
				r8Coincide(a, p, -1, get("TEMOIN "+p.Family))
			}
		}
	}
	r8LogCoincidence(t, pops)
}

func r8LogCoincidence(t *testing.T, pops map[string]*r8CoincTally) {
	t.Helper()
	keys := make([]string, 0, len(pops))
	for k := range pops {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("fenetre de coincidence : +/- %d frames (%d ms)", r8CoincWindow, r8CoincWindow*100)
	t.Logf("%-24s %6s %6s %6s %6s | %6s %6s %6s %6s %6s | %6s %6s %6s",
		"population", "poses", "poseur", "rang", "chgLu",
		"takFrm", "takAut", "spnFrm", "spnAut", "aucun",
		"portFm", "portAu", "portNC")
	for _, k := range keys {
		c := pops[k]
		t.Logf("%-24s %6d %6d %6d %6d | %6d %6d %6d %6d %6d | %6d %6d %6d",
			k, c.total, c.withOwner, c.rankKnown, c.changesRead,
			c.takenFrom, c.takenOther, c.spentFrom, c.spentOther, c.none,
			c.heldFam, c.heldOther, c.heldUnknown)
	}
}
