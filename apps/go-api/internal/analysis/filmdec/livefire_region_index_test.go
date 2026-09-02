package filmdec

// livefire_region_index_test.go — SONDE C1 (lot C catalogues, 2026-08-27) : LA VALEUR de
// l'index de region d'i0 sur une carte a PLUS de deux regions de compression.
//
// CONTEXTE. Live Fire (module sgh_interlock) declare QUATRE regions de compression, portees
// par ds/globals/common (sonde himap TestSondeInterlockSbsp) dans l'ordre
// [7047b96f, d88e1d88, a59f5052, 91c336c1]. L'index de region vaut alors
// ceilLog2(4) = 2 bits — le cas que le commentaire de map_quant_control_test.go predisait
// (« l'axe X sera le premier a mentir ») : DetectI0Layout, gate fixe a 5, lit [13 12 11]
// la ou les largeurs de la 2e region (d88e1d88, la seule dont l'emprise contient les ancres
// d'objectifs du catalogue) valent [12 12 11].
//
// CE QUE CETTE SONDE MESURE : la DISTRIBUTION DES VALEURS des bits 4 et 5 d'i0 (les 2 bits
// d'index sous l'hypothese gate=6). Si les records portent la valeur 01 (= region 1 de
// l'ordre, d88e1d88), l'hypothese « 4 regions, arene = region 1 » est confirmee par le film
// lui-meme, et les bornes a cataloguer sont celles de d88e1d88. Toute autre distribution
// se publie et s'instruit avant d'ecrire quoi que ce soit au catalogue.
//
// LECTURE SEULE, garde par LIVEFIRE_IDX_FILM, saute partout ailleurs (CI comprise).
//
//	CGO_ENABLED=0 LIVEFIRE_IDX_FILM=<repo>/data/cache/film_chunks/60ae07c4 \
//	  go test ./internal/analysis/filmdec/ -run '^TestLiveFireRegionIndex$' -v

import (
	"os"
	"testing"
)

const liveFireIdxEnv = "LIVEFIRE_IDX_FILM"

func TestLiveFireRegionIndex(t *testing.T) {
	dir := os.Getenv(liveFireIdxEnv)
	if dir == "" {
		t.Skipf("%s absent : sonde sautee", liveFireIdxEnv)
	}
	release := LockProcessDecode()
	defer release()

	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	scanned := make([]int, 0, detectMaxChunks)
	for c := 1; c <= n && len(scanned) < detectMaxChunks; c++ {
		scanned = append(scanned, c)
	}
	slots := bipedSlotBandDir(dir, scanned)
	if len(slots) == 0 {
		t.Fatalf("aucun slot biped dans %s", dir)
	}
	var samples []i0Sample
	for _, c := range scanned {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			samples = collectI0Samples(pk.Payload(data), slots, c, pk.Index, samples)
		}
	}
	if len(samples) == 0 {
		t.Fatalf("aucun record i0 candidat dans %s", dir)
	}
	// Distribution des valeurs du couple (bit 4, bit 5) — les 2 bits d'index sous gate=6.
	var dist [4]int
	for _, s := range samples {
		dist[s.bit(4)<<1|s.bit(5)]++
	}
	t.Logf("%s : %d records — index 2 bits (b4b5) : 00=%d 01=%d 10=%d 11=%d",
		dir, len(samples), dist[0], dist[1], dist[2], dist[3])
}
