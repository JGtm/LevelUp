package filmdec

// vehicules_v6_ancrage_test.go — INSTRUMENT (lot V6) : L'ANCRAGE AVAL.
//
// CE QUE MESURE CE FICHIER. `TestV6Chaine` etablit que la liste d'un paquet a tete board/exit
// se termine JUSTE APRES la charge (bit de continuation a 0 dans 99 % des cas) et qu'il reste
// 1 300 a 2 800 bits derriere : c'est la TRAME ECS. On connait donc, pour ces paquets, le bit
// exact ou la trame commence — `S = fin de l'evenement + 1`.
//
// C'EST UNE VERITE TERRAIN POUR CALIBRER UN SCORE. Si un score de decodage de trame designe ce
// S-la parmi tous les candidats, alors le meme score, applique a un paquet dont la tete est
// d'un type INCONNU, donne la longueur de cet evenement — donc la marche de la liste entiere.
//
// LE TEMOIN EST LE BALAYAGE LUI-MEME : le score est calcule a TOUS les decalages candidats, et
// on releve le rang du vrai S. Un score qui ne separe pas se voit immediatement.
//
// Garde d'environnement V6_ROOT / V6_FILMS : sans elle, tout SKIP.

import (
	"path/filepath"
	"sort"
	"testing"
)

// v6FrameScore evalue un candidat de debut de trame ECS : il rejoue la boucle de records a
// partir du bit S et rend (records aboutis, fin propre). « Fin propre » = la boucle s'est
// terminee sur un record de type 0 et il ne reste qu'un bourrage de moins de 8 bits, nul.
//
// Le monde est passe par l'appelant (amorce par les images-cles du chunk) ; il est CLONE pour
// que le balayage d'un candidat faux ne pollue pas les suivants.
func v6FrameScore(pay []byte, S int, w *World, cfg FrameConfig) (walked int, clean bool) {
	if S < 0 || S >= len(pay)*8 {
		return 0, false
	}
	snap := w.Snapshot()
	defer w.Restore(snap)
	br := NewBitReader(pay)
	br.SetBitPos(S)
	recs, err := DecodeFrameRecords(br, w, cfg)
	for i := range recs {
		if recs[i].DesyncAt == -1 {
			walked++
		}
	}
	if err != nil {
		return walked, false
	}
	rest := br.Remaining()
	if rest < 0 || rest >= 8 {
		return walked, false
	}
	for b := br.BitPos(); b < len(pay)*8; b++ {
		if readBitsAt(pay, b, 1) != 0 {
			return walked, false
		}
	}
	return walked, true
}

// v6Depth rend la PROFONDEUR DE MARCHE a partir du bit S : le nombre de records consecutifs
// dont la traversee ABOUTIT avant la premiere desynchronisation ou erreur. C'est le score
// candidat n°2, apres l'echec du critere « fin propre » (mesure : 0,0 % au vrai S).
func v6Depth(pay []byte, S int, w *World, cfg FrameConfig) int {
	if S < 0 || S >= len(pay)*8 {
		return -1
	}
	snap := w.Snapshot()
	defer w.Restore(snap)
	br := NewBitReader(pay)
	br.SetBitPos(S)
	recs, _ := DecodeFrameRecords(br, w, cfg)
	d := 0
	for i := range recs {
		if recs[i].DesyncAt != -1 {
			break
		}
		d++
	}
	return d
}

// v6AncrageStats : ce que la passe d'etalonnage releve.
type v6AncrageStats struct {
	// paquets a LISTE VIDE (verite terrain S = 2)
	emptyTotal, emptyCleanTrue, emptyCleanAny int
	// paquets a tete board/exit (verite terrain S = fin + 1)
	vehTotal, vehCleanTrue int
	// rang du vrai S dans le balayage (0 = unique gagnant)
	vehRank map[int]int
	// nombre de candidats « fin propre » dans la fenetre de balayage
	vehCleanCount map[int]int
	// etalonnage du score PROFONDEUR sur les paquets a liste vide (verite terrain S = 2)
	depthN                     int
	depthTrue                  map[int]int // profondeur au vrai S
	depthStrictBetter          map[int]int // nb de candidats STRICTEMENT meilleurs
	depthTies                  map[int]int // nb de candidats a egalite
	depthWinnerUnique, depthEq int
}

func newV6Ancrage() *v6AncrageStats {
	return &v6AncrageStats{vehRank: map[int]int{}, vehCleanCount: map[int]int{},
		depthTrue: map[int]int{}, depthStrictBetter: map[int]int{}, depthTies: map[int]int{}}
}

// v6DepthEvery : un paquet a liste vide sur N est etalonne (le balayage coute 129 decodages).
const v6DepthEvery = 97

// calibrateDepth etalonne le score PROFONDEUR sur un paquet dont le vrai S est connu.
func (a *v6AncrageStats) calibrateDepth(pay []byte, trueS int, w *World, cfg FrameConfig) {
	base := v6Depth(pay, trueS, w, cfg)
	a.depthN++
	a.depthTrue[base]++
	better, ties := 0, 0
	for S := trueS - v6SweepWindow; S <= trueS+v6SweepWindow; S++ {
		if S == trueS || S < 2 {
			continue
		}
		switch d := v6Depth(pay, S, w, cfg); {
		case d > base:
			better++
		case d == base:
			ties++
		}
	}
	a.depthStrictBetter[better]++
	a.depthTies[ties]++
	if better == 0 && ties == 0 {
		a.depthWinnerUnique++
	} else if better == 0 {
		a.depthEq++
	}
}

// v6ChunkWorld amorce un monde a partir des images-cles d'un chunk.
func v6ChunkWorld(reg *Registry, data []byte, pks []FilmPacket) *World {
	w := NewWorld(reg)
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
	}
	return w
}

// v6SweepWindow est la demi-largeur (bits) du balayage de candidats autour du vrai S.
const v6SweepWindow = 64

// scanFilm etalonne le score sur un film.
func (a *v6AncrageStats) scanFilm(t *testing.T, dir string, cfg FrameConfig) {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return
	}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		pks := WalkPackets(data)
		w := v6ChunkWorld(reg, data, pks)
		for _, p := range pks {
			if p.Type != PacketTypeDelta || p.Size < 2 {
				continue
			}
			a.samplePacket(p.Payload(data), w, cfg)
		}
	}
}

// samplePacket etalonne le score sur un paquet.
func (a *v6AncrageStats) samplePacket(pay []byte, w *World, cfg FrameConfig) {
	typ, present := PacketHeadEventType(pay)
	if !present {
		a.emptyTotal++
		if _, clean := v6FrameScore(pay, DefaultPacketPreambleBits, w, cfg); clean {
			a.emptyCleanTrue++
		}
		if a.emptyTotal%v6DepthEvery == 0 {
			a.calibrateDepth(pay, DefaultPacketPreambleBits, w, cfg)
		}
		return
	}
	if typ != EventBipedBoardVehicle && typ != EventUnitExitVehicle {
		return
	}
	end, ok := v6EventEnd(pay, 1, typ)
	if !ok {
		return
	}
	trueS := end + 1
	a.vehTotal++
	trueWalked, trueClean := v6FrameScore(pay, trueS, w, cfg)
	if trueClean {
		a.vehCleanTrue++
	}
	better, cleanN := 0, 0
	for S := trueS - v6SweepWindow; S <= trueS+v6SweepWindow; S++ {
		if S == trueS || S < 2 {
			continue
		}
		ww, cl := v6FrameScore(pay, S, w, cfg)
		if cl {
			cleanN++
		}
		if cl && !trueClean {
			better++
			continue
		}
		if cl == trueClean && ww > trueWalked {
			better++
		}
	}
	if trueClean {
		cleanN++
	}
	a.vehRank[better]++
	a.vehCleanCount[cleanN]++
}

// v6TopIntHist : histogramme trie par CLE (lisible pour des rangs).
func v6TopIntHist(h map[int]int, n int) string {
	var keys []int
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := ""
	for i, k := range keys {
		if i >= n {
			s += " …"
			break
		}
		s += " " + itoa(k) + "×" + itoa(h[k])
	}
	return s
}

// TestV6Ancrage : le score de trame designe-t-il le VRAI debut de trame ?
func TestV6Ancrage(t *testing.T) {
	dirs := v6FilmDirs(t)
	release := LockProcessDecode()
	defer release()
	cfg := DefaultFrameConfig()
	a := newV6Ancrage()
	for _, d := range dirs {
		t.Logf("film %s…", filepath.Base(filepath.Clean(d)))
		a.scanFilm(t, d, cfg)
	}
	t.Logf("== V6 ANCRAGE ==")
	if a.emptyTotal > 0 {
		t.Logf("paquets a LISTE VIDE : %d — fin propre au vrai S(=2) : %d (%.1f %%)",
			a.emptyTotal, a.emptyCleanTrue, 100*float64(a.emptyCleanTrue)/float64(a.emptyTotal))
	}
	if a.vehTotal == 0 {
		t.Skip("aucun paquet vehicule")
	}
	t.Logf("paquets a tete VEHICULE : %d — fin propre au vrai S : %d (%.1f %%)",
		a.vehTotal, a.vehCleanTrue, 100*float64(a.vehCleanTrue)/float64(a.vehTotal))
	t.Logf("rang du vrai S (0 = gagnant unique), fenetre ±%d bits :%s",
		v6SweepWindow, v6TopIntHist(a.vehRank, 12))
	t.Logf("nombre de candidats a fin propre dans la fenetre :%s", v6TopIntHist(a.vehCleanCount, 12))
	if a.depthN > 0 {
		t.Logf("-- score PROFONDEUR, etalonne sur %d paquets a LISTE VIDE (vrai S = 2) --", a.depthN)
		t.Logf("  profondeur au vrai S            :%s", v6TopIntHist(a.depthTrue, 12))
		t.Logf("  candidats STRICTEMENT meilleurs :%s", v6TopIntHist(a.depthStrictBetter, 12))
		t.Logf("  candidats a EGALITE             :%s", v6TopIntHist(a.depthTies, 12))
		t.Logf("  vrai S gagnant UNIQUE : %d (%.1f %%) · gagnant EX AEQUO : %d (%.1f %%)",
			a.depthWinnerUnique, 100*float64(a.depthWinnerUnique)/float64(a.depthN),
			a.depthEq, 100*float64(a.depthEq)/float64(a.depthN))
	}
}
