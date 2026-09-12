package filmdec

// lot1_visee_calib_research_test.go — LOT 1 : CALIBRER (et donc PROUVER ou REFUTER) la visee du
// type 36 par l'ORACLE DE TRAME, le seul discriminant (le vecteur R(30) ne l'est pas a 30 bits).
//
// PRINCIPE : pour les paquets 0xD2 modaux (type 36, 0 cible, 0 composante), on decode l'en-tete
// + les deux composites + la visee R(30) — position CONNUE. Restent des champs post-visee de
// longueur variable, puis le bit de continuation, puis la TRAME. On BALAYE la longueur K a
// sauter apres la visee : pour chaque K, on lit continuation puis la trame et on mesure sa
// PROFONDEUR (records/paquet). Le K qui MAXIMISE la profondeur est la longueur post-visee ; si
// le pic est NET (profondeur ~2 au bon K, ~0.2 ailleurs), le cadrage jusqu'a la visee est PROUVE
// bout en bout (composites + visee compris) — c'est le juge qui manquait.
//
// Critere ecrit avant la mesure : PROUVE si un K rend une profondeur >= 1.0 record/paquet ET
// >= 3x la mediane des autres K (pic net). Sinon NON CONCLUANT (post-visee a longueur variable
// -> demande son RE, comme damage_aftermath).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris.

import (
	"os"
	"sort"
	"testing"
)

func TestLot1ViseeCalibration(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	const kMax = 96
	depthByK := make([]int, kMax+1) // records totaux de la trame par longueur post-visee K
	nModal := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		cfg2 := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg2)
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			aimEnd, ok := lot1Type36AimEnd(pay)
			if !ok {
				continue
			}
			nModal++
			for k := 0; k <= kMax; k++ {
				p := aimEnd + k
				if p+1 >= len(pay)*8 {
					break
				}
				w := NewWorld(reg)
				w.Restore(snap)
				br := NewBitReader(pay)
				br.Skip(p + 1) // K bits post-visee + 1 bit de continuation
				recs, _ := DecodeFrameRecords(br, w, DefaultFrameConfig())
				depthByK[k] += len(recs)
			}
		}
	}
	if nModal < 30 {
		t.Skipf("seulement %d paquets 0xD2 modaux : trop peu pour calibrer", nModal)
	}
	// Meilleur K et pic.
	bestK, bestD := 0, -1
	for k, d := range depthByK {
		if d > bestD {
			bestK, bestD = k, d
		}
	}
	sorted := append([]int(nil), depthByK...)
	sort.Ints(sorted)
	median := sorted[len(sorted)/2]
	profBest := float64(bestD) / float64(nModal)
	t.Logf("== calibration visee type 36 : %d paquets modaux, K balaye 0..%d ==", nModal, kMax)
	t.Logf("MEILLEUR K post-visee = %d : %d records (%.2f/paquet) · mediane des K : %d records",
		bestK, bestD, profBest, median)
	// Top 5 pour voir la forme du pic.
	type kd struct{ k, d int }
	var top []kd
	for k, d := range depthByK {
		top = append(top, kd{k, d})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].d > top[j].d })
	line := ""
	for i := 0; i < 6 && i < len(top); i++ {
		line += " K=" + itoa(top[i].k) + ":" + itoa(top[i].d)
	}
	t.Logf("  profil (K:records, decroissant) :%s", line)
	net := profBest >= 1.0 && bestD >= 3*median && median > 0
	t.Logf("VERDICT (pic net : profondeur >= 1/paquet ET >= 3x la mediane) : %s", lot1Verdict(net))
}

// lot1Type36AimEnd decode l'en-tete + composites + visee d'un paquet 0xD2 modal et rend la
// position de bit APRES la visee R(30), ou ok=false si le paquet n'est pas un type 36 modal.
func lot1Type36AimEnd(pay []byte) (int, bool) {
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 36 {
		return 0, false
	}
	// ref0 dom1 sonde, ref1/ref2 dom8/dom7 (R(13))
	if br.ReadBit() {
		w := 13
		if br.ReadBit() {
			w = 9
		}
		br.Skip(w + 2)
	}
	for range 2 {
		if br.ReadBit() {
			br.Skip(15)
		}
	}
	estCourt := br.ReadBit()
	estBloc := br.ReadBit()
	br.Skip(8)
	if br.ReadBit() {
		br.Skip(5)
	}
	if !br.ReadBit() {
		br.Skip(2)
	}
	if br.ReadBit() {
		br.Skip(32)
	}
	br.Skip(32) // variant_name
	br.Skip(2)
	if estBloc {
		br.Skip(1)
		if br.ReadBit() {
			return 0, false // horodatage non resolu
		}
	}
	if estCourt {
		return 0, false
	}
	// comptes
	var nCibles, nComp uint64
	if !br.ReadBit() {
		if br.ReadBit() {
			nCibles = 1
		} else {
			nCibles = br.ReadBits(4)
		}
		if !br.ReadBit() {
			if br.ReadBit() {
				nComp = 1
			} else {
				nComp = br.ReadBits(4)
			}
		}
	}
	if nCibles != 0 || nComp != 0 {
		return 0, false // seulement le cas modal
	}
	lot1SkipCd5b8(br)
	lot1SkipEff64(br)
	br.Skip(30) // visee R(30)
	return br.BitPos(), true
}

// itoa : petit entier -> chaine (evite d'importer strconv pour deux usages).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
