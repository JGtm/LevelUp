//go:build research

package grammar

// instruments_partages_etat_actif_research_test.go — aides de mesure de etat_actif_shared_test.go dont
// les seuls utilisateurs sont des instruments `research` (J12.7 lint : inutilisees dans le build par
// defaut). Deplacement pur.

import (
	"sort"
	"testing"
)

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
	_, lay := contexteDuFilm(t, dir)
	lg := profilDeCarte(lay).LargeursObjetDuMonde()
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
			limit := total - (worldObjectHeaderBits + worldObjectIndexBits + projPosBits(lg))
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
