package filmdec

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
	"sort"
	"testing"
)

// eaFilmSetup porte ce que tout instrument biped doit charger d'un film.
type eaFilmSetup struct {
	dir    string
	chunks []int
	slots  map[uint32]bool
	lay    I0Layout
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
	if len(slots) == 0 {
		t.Fatalf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	lay, _, err := DetectI0Layout(dir)
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
		br := NewBitReader(pay)
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
	for _, id := range idx {
		if id == target {
			return true
		}
	}
	return false
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
	for _, r := range ranks {
		if r == want {
			return true
		}
	}
	return false
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

// eaNearestDelta rend le plus petit écart SIGNÉ (autre − ts) entre ts et une liste TRIÉE
// d'horodatages, en microsecondes. ok=false si la liste est vide.
func eaNearestDelta(ts uint64, sorted []uint64) (int64, bool) {
	if len(sorted) == 0 {
		return 0, false
	}
	i := sort.Search(len(sorted), func(k int) bool { return sorted[k] >= ts })
	best := int64(0)
	got := false
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(sorted) {
			continue
		}
		d := int64(sorted[k]) - int64(ts)
		if !got || abs64(d) < abs64(best) {
			best, got = d, true
		}
	}
	return best, got
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// eaCountWithin compte les éléments de `events` ayant un voisin dans `sorted` à moins de
// win µs, après décalage des événements de shift µs (témoin décalé, protocole des mesures
// des 13-15/08).
func eaCountWithin(events []uint64, sorted []uint64, win uint64, shift int64) int {
	n := 0
	for _, e := range events {
		ts := int64(e) + shift
		if ts < 0 {
			continue
		}
		if d, ok := eaNearestDelta(uint64(ts), sorted); ok && abs64(d) <= int64(win) {
			n++
		}
	}
	return n
}

// eaLifeKey identifie une vie d'objet du monde — LA PAIRE (slot, génération), comme pour
// les projectiles : le pool de slots reboucle.
type eaLifeKey struct{ slot, gen uint32 }

// eaLife est l'intervalle observé d'une vie ti=37 dans les paquets delta, et le masque de
// son DERNIER record (c'est lui qui porte une éventuelle fin lisible, cf. projectiles :
// at-rest sur le dernier record dans 78 cas sur 79).
type eaLife struct {
	firstUS, lastUS uint64
	n               int
	lastIdx         []int
}

// eaScanTi37Lives balaye les records delta de ti=37 (même détecteur que la production :
// matchWorldObjectRecord + i0 en tête de masque) et rend les vies observées. AUCUN
// composant n'est décodé : seul l'en-tête (slot, gen, masque) est lu — l'intervalle et le
// masque du dernier record suffisent aux phases C et E.
func eaScanTi37Lives(t *testing.T, dir string) map[eaLifeKey]*eaLife {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBandDir(dir, n, EquipmentTypeIndex)
	if len(band) == 0 {
		t.Fatalf("aucun slot d'archétype ti=%d dans les keyframes de %s", EquipmentTypeIndex, dir)
	}
	lives := map[eaLifeKey]*eaLife{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			limit := total - (worldObjectHeaderBits + worldObjectIndexBits + projPosBits())
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok || rec.Idx[0] != 0 {
					continue
				}
				k := eaLifeKey{rec.Slot, rec.Gen}
				l := lives[k]
				if l == nil {
					l = &eaLife{firstUS: pk.TimestampUS}
					lives[k] = l
				}
				if pk.TimestampUS < l.firstUS {
					l.firstUS = pk.TimestampUS
				}
				if pk.TimestampUS >= l.lastUS {
					l.lastUS = pk.TimestampUS
					l.lastIdx = append([]int(nil), rec.Idx...)
				}
				l.n++
				p = rec.After - 1
			}
		}
	}
	return lives
}

// eaSortedU64 rend une copie triée.
func eaSortedU64(ts []uint64) []uint64 {
	out := append([]uint64(nil), ts...)
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}
