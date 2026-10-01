package grammar

// etat_actif_shared_test.go — AIDES PARTAGÉES des cinq instruments du plan
// .ai/V7.5/replay2d/PLAN_ETAT_ACTIF_EQUIPEMENT.md (phases A camo, B surbouclier,
// C déployables, D mobilité, E i59). Un seul exemplaire de la marche, de la jointure
// slot -> rang i48 et du groupage en épisodes : cinq copies auraient re-divergé (règle
// des ≤ 2 copies, CLAUDE.md n°6).
//
// LA MARCHE est celle des instruments précédents (i48_rank_test.go, i57_reach_test.go) :
// composants du masque consommés par les désers de PRODUCTION (consumeByName), qui
// publient via leurs hooks — on ne relit jamais les bits à côté du déserialiseur
// (décision n°4 du plan). Elle rend false dès qu'un composant intermédiaire n'est pas
// porté ou que la marche déborde : au-delà, la position du curseur n'est plus digne de
// confiance.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"slices"
	"sort"
	"testing"
)

// eaFilmSetup porte ce que tout instrument biped doit charger d'un film.
type eaFilmSetup struct {
	dir    string
	chunks []int
	slots  SlotBand
	lay    profile.I0Layout
	arch   Archetype
}

// eaSetupBiped charge chunks, bande de slots biped, découpage i0 et archétype biped.
func eaSetupBiped(t *testing.T, dir string) eaFilmSetup {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	if slots.Count() == 0 {
		t.Fatalf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	lay, _, err := detectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	return eaFilmSetup{dir: dir, chunks: chunks, slots: slots, lay: lay, arch: i48Archetype(t, dir)}
}

// eaWalkThrough marche les composants du masque avec les désers de PRODUCTION et consomme
// AUSSI le composant cible — c'est cette consommation qui déclenche le hook installé par
// l'instrument. La marche s'arrête après la cible.
func eaWalkThrough(pay []byte, i0, total int, idx []int, s eaFilmSetup, target int) bool {
	at := i0 + s.lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return false
		}
		name := s.arch.component(id)
		if name == "" {
			return false
		}
		br := LecteurSur(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), s.arch.Level(id))
		if id == target {
			// La cible est consommée : le hook a publié. `ported` peut être faux sur une
			// branche non déterminable (i57 tag==3) — le tag, lu AVANT la branche, est
			// valide ; c'est la SUITE du record qui ne l'est plus, et on ne la lit pas.
			return br.BitPos() <= total
		}
		if !ported || br.BitPos() > total {
			return false
		}
		at = br.BitPos()
	}
	return false
}

// eaMaskHas dit si la liste d'index du masque contient l'index visé.
func eaMaskHas(idx []int, target int) bool {
	return slices.Contains(idx, target)
}

// eaSlotRanks lit les identités i48 du film par le BALAYAGE DE PRODUCTION
// (ScanFilmAbilityRanks) et les rend par slot. Un slot est UNE VIE ; une vie peut
// légitimement changer de rang (ramassage d'un autre équipement) — c'est pourquoi la
// jointure rend l'ENSEMBLE des rangs transmis, pas seulement le majoritaire.
func eaSlotRanks(t *testing.T, dir string) map[uint32][]int {
	t.Helper()
	ranks, st, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Fatalf("identités i48 illisibles dans %s : %v", dir, err)
	}
	t.Logf("i48 (production) : %d identités · records %d · masque∋i48 %d · lues %d · "+
		"illisibles %d · sans identité %d", len(ranks), st.Records, st.WithI48, st.Read,
		st.Unread, st.Gated)
	out := map[uint32][]int{}
	for _, r := range ranks {
		out[r.Slot] = append(out[r.Slot], r.Rank)
	}
	return out
}

// eaRankSet rend les rangs distincts, triés, d'une vie.
func eaRankSet(ranks []int) []int {
	seen := map[int]bool{}
	for _, r := range ranks {
		seen[r] = true
	}
	out := make([]int, 0, len(seen))
	for r := range seen {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// eaHasRank dit si la vie a transmis au moins une fois ce rang.
func eaHasRank(ranks []int, want int) bool {
	return slices.Contains(ranks, want)
}

// eaEpisode est un groupe de lectures consécutives d'un même slot à moins de gapUS d'écart :
// un état répliqué tant qu'il est actif est UN épisode, pas autant d'événements.
type eaEpisode struct {
	slot    uint32
	startUS uint64
	endUS   uint64
	n       int
}

// eaGroupEpisodes groupe des horodatages TRIÉS d'un slot en épisodes.
func eaGroupEpisodes(slot uint32, ts []uint64, gapUS uint64) []eaEpisode {
	var out []eaEpisode
	for i, x := range ts {
		if i == 0 || x-ts[i-1] > gapUS {
			out = append(out, eaEpisode{slot: slot, startUS: x, endUS: x, n: 1})
			continue
		}
		out[len(out)-1].endUS = x
		out[len(out)-1].n++
	}
	return out
}

// eaSortedU64 rend une copie triée.
func eaSortedU64(ts []uint64) []uint64 {
	out := append([]uint64(nil), ts...)
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}
