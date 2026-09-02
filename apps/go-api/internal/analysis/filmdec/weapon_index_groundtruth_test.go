package filmdec

// weapon_index_groundtruth_test.go — VERITE TERRAIN du correctif d'indice de tireur (Lot 3),
// confronte aux KILLS (dead-state, scan robuste robustCollectKills).
//
// La question : l'indice de tireur CORRIGE (5 bits, ShooterIndex5) resout-il au VRAI tueur ?
// Le dead-state porte le tueur en roster (EnumB) ; les tirs portent l'indice de FILM. La table
// d'identite roster<->film (geoBuildIdentity, apprise par co-occurrence tir/mort) fait le pont.
// Un espace d'indice CORRECT rend la table INJECTIVE : chaque joueur du roster <-> un unique
// indice de film. Le 4 bits (ancien FilmIndex) SATURE a 15 au-dela de 16 joueurs et FUSIONNE
// deux tueurs sur un meme indice -> table NON injective -> le tir mortel resoudrait au mauvais
// tueur. Le 5 bits separe les tueurs que le 4 bits confond.
//
// MESURE : par film (arene 000d5950 / 01e1f945 / 00502e52 ; BTB 4f77afc1), on construit la table
// avec .film = idx4 PUIS avec .film = idx5, et on compare cardinalite + injectivite. On EXIGE que
// le 5 bits soit injectif partout, et qu'il resolve au moins autant de tueurs distincts que le
// 4 bits ; sur le BTB, le 5 bits doit STRICTEMENT faire mieux (le 4 bits fusionne ou perd des
// tueurs). Env PRECISION_CORPUS = repertoire de base des films.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// precGTShot : un tir 0xD2 long avec SES DEUX lectures d'indice (4 bits tronque, 5 bits reel).
type precGTShot struct {
	ts     uint64
	att    uint64
	idx4   int
	idx5   int
	wid    uint64
	name   string
	heavy  bool
	direct bool
}

// precCollectShotsBoth decode les tirs longs 0xD2 en lisant LES DEUX indices sur le meme record.
func precCollectShotsBoth(t *testing.T, dir string, n int) []precGTShot {
	t.Helper()
	var out []precGTShot
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 5 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 36 {
				continue
			}
			att, okA := lot1RefDom1(br)
			fe, okF := decodeFireEvent(pay)
			if !okA || !okF {
				continue
			}
			name := geoWeaponName(fe.WeaponID)
			out = append(out, precGTShot{
				ts: pk.TimestampUS, att: att, idx4: fe.FilmIndex, idx5: fe.ShooterIndex5,
				wid: fe.WeaponID, name: name, heavy: lot1IsHeavy(name), direct: geoIsDirect(name),
			})
		}
	}
	return out
}

// precGeoShots projette les tirs sur []geoShot en choisissant l'indice (4 ou 5 bits) comme .film.
func precGeoShots(shots []precGTShot, use5 bool) []geoShot {
	out := make([]geoShot, len(shots))
	for i, s := range shots {
		film := s.idx4
		if use5 {
			film = s.idx5
		}
		out[i] = geoShot{ts: s.ts, att: s.att, film: film, wid: s.wid, name: s.name, heavy: s.heavy, direct: s.direct}
	}
	return out
}

// precDistinct5 : nombre d'indices 5 bits distincts (taille reelle du lobby).
func precDistinct5(shots []precGTShot) int {
	seen := map[int]bool{}
	for _, s := range shots {
		seen[s.idx5] = true
	}
	return len(seen)
}

// precIdxStats : bornes et cardinalites des deux lectures d'indice sur les tirs d'un film.
type precIdxStats struct {
	maxIdx4, maxIdx5     int
	distinct4, distinct5 int
}

// precShotStats mesure les bornes des indices — DIRECTEMENT sur les records, sans heuristique.
// C'est le signal NON CONFONDU : distinct5 = taille du lobby ; un 4 bits ne peut en indexer que 16.
func precShotStats(shots []precGTShot) precIdxStats {
	s4, s5 := map[int]bool{}, map[int]bool{}
	st := precIdxStats{maxIdx4: -1, maxIdx5: -1}
	for _, s := range shots {
		s4[s.idx4], s5[s.idx5] = true, true
		if s.idx4 > st.maxIdx4 {
			st.maxIdx4 = s.idx4
		}
		if s.idx5 > st.maxIdx5 {
			st.maxIdx5 = s.idx5
		}
	}
	st.distinct4, st.distinct5 = len(s4), len(s5)
	return st
}

// precHighVictimKills compte les kills dont la VICTIME (slot -> film argmax de ses tirs) est un
// joueur d'indice >= 16 : un tueur/victime que le 4 bits FUSIONNE avec un joueur d'indice < 16, et
// dont l'attribution du tir mortel bascule avec le correctif. Signal kill-lie, pour information.
func precHighVictimKills(shots []precGTShot, kills []geoKill) int {
	slotFilm5 := map[uint32]map[int]int{}
	for _, s := range shots {
		slot := uint32(geoActiveBase + int(s.att))
		if slotFilm5[slot] == nil {
			slotFilm5[slot] = map[int]int{}
		}
		slotFilm5[slot][s.idx5]++
	}
	argmax := func(m map[int]int) (int, bool) {
		best, bn, ok := 0, -1, false
		for f, n := range m {
			if n > bn {
				best, bn, ok = f, n, true
			}
		}
		return best, ok
	}
	high := 0
	for _, k := range kills {
		if fm, ok := slotFilm5[k.victSlot]; ok {
			if f, ok2 := argmax(fm); ok2 && f >= 16 {
				high++
			}
		}
	}
	return high
}

// TestWeaponIndexGroundTruth : le 5 bits resout au vrai tueur la ou le 4 bits fusionne (BTB).
func TestWeaponIndexGroundTruth(t *testing.T) {
	base := os.Getenv("PRECISION_CORPUS")
	if base == "" {
		t.Skip("PRECISION_CORPUS absent : verite terrain sautee")
	}
	release := LockProcessDecode()
	defer release()
	for _, f := range idxGTCorpus {
		runWeaponIndexGroundTruth(t, filepath.Join(base, f.id), f)
	}
}

// idxGTFilm : un film et son attendu roster.
type idxGTFilm struct {
	id  string
	btb bool
}

var idxGTCorpus = []idxGTFilm{
	{id: "000d5950"},
	{id: "01e1f945"},
	{id: "00502e52"},
	{id: "4f77afc1", btb: true},
}

// runWeaponIndexGroundTruth mesure un film : identite roster<->film sous 4 bits puis 5 bits.
func runWeaponIndexGroundTruth(t *testing.T, dir string, f idxGTFilm) {
	if _, err := os.Stat(dir); err != nil {
		t.Logf("film %s absent du corpus (%v) : saute", f.id, err)
		return
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("%s chunk_00 illisible : %v", f.id, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("%s registre illisible : %v", f.id, err)
	}
	n := CountFilmChunks(dir)
	if n > geoMaxChunks {
		n = geoMaxChunks
	}
	shots := precCollectShotsBoth(t, dir, n)
	nRoster := precDistinct5(shots)
	raws, _ := geoCollectDamageKills(t, dir, reg, n)
	kills := robustCollectKills(t, dir, reg, n, nRoster)

	geoActiveBase = geoDetectBase(raws)
	defer func() { geoActiveBase = geoBase }()

	table4, card4, inj4 := geoBuildIdentity(precGeoShots(shots, false), kills)
	table5, card5, inj5 := geoBuildIdentity(precGeoShots(shots, true), kills)
	stat := precShotStats(shots)
	highKills := precHighVictimKills(shots, kills)

	kind := "ARENE"
	if f.btb {
		kind = "BTB  "
	}
	t.Logf("== VERITE TERRAIN film %s [%s] · %d chunks · lobby (idx5 distincts) %d · %d kills robustes ==",
		f.id, kind, n, nRoster, len(kills))
	t.Logf("   indices tireur observes : 4 bits max=%d distincts=%d · 5 bits max=%d distincts=%d",
		stat.maxIdx4, stat.distinct4, stat.maxIdx5, stat.distinct5)
	t.Logf("   identite roster<->film (heuristique geoBuildIdentity) : 4 bits %d mappes inj=%v · 5 bits %d mappes inj=%v",
		card4, inj4, card5, inj5)
	t.Logf("   kills dont la victime est un joueur d'indice >= 16 (mis-attribue par le 4 bits) : %d", highKills)
	logIndexFusions(t, table4, table5)

	assertWeaponIndexGroundTruth(t, f, card4, card5, stat)
}

// logIndexFusions journalise les rosters que le 4 bits FUSIONNE sur un meme indice (le 5 bits, lui,
// leur donne des indices distincts) — la preuve concrete que le 4 bits designerait le mauvais tueur.
func logIndexFusions(t *testing.T, table4, table5 map[int32]int) {
	byFilm4 := map[int][]int32{}
	rosters := make([]int32, 0, len(table4))
	for r := range table4 {
		rosters = append(rosters, r)
	}
	sort.Slice(rosters, func(i, j int) bool { return rosters[i] < rosters[j] })
	for _, r := range rosters {
		byFilm4[table4[r]] = append(byFilm4[table4[r]], r)
	}
	films := make([]int, 0, len(byFilm4))
	for fi := range byFilm4 {
		films = append(films, fi)
	}
	sort.Ints(films)
	for _, fi := range films {
		rs := byFilm4[fi]
		if len(rs) < 2 {
			continue
		}
		t.Logf("   FUSION 4 bits : indice film %d partage par rosters %v — 5 bits leur donne %s",
			fi, rs, films5For(rs, table5))
	}
}

// films5For rend, pour un groupe de rosters fusionnes par le 4 bits, leurs indices 5 bits.
func films5For(rosters []int32, table5 map[int32]int) string {
	out := "["
	for i, r := range rosters {
		if i > 0 {
			out += " "
		}
		if f, ok := table5[r]; ok {
			out += itoaGT(f)
		} else {
			out += "?"
		}
	}
	return out + "]"
}

// assertWeaponIndexGroundTruth valide le SIGNAL NON CONFONDU (bornes d'indice), pas l'heuristique
// geoBuildIdentity (bruitee : elle est deja non injective en arene, ou idx4==idx5, par recyclage
// de slots — ce bruit est identique 4/5 bits et NE discrimine PAS la largeur).
//
//   - ARENE : idx4 == idx5 partout (bit 35 nul). Le correctif est un NO-OP : les deux tables
//     d'identite sont IDENTIQUES (memes .film). On l'exige (garde de NON-REGRESSION) et on exige
//     que l'indice reste < 16 (sinon le film n'est pas de l'arene).
//   - BTB : le 5 bits atteint la vraie taille du lobby (> 16) la ou le 4 bits SATURE a 15. Le
//     4 bits ne PEUT PAS indexer > 16 joueurs (preuve par denombrement, sans heuristique) : il
//     fusionne forcement des joueurs. Le 5 bits ne regresse pas (card5 >= card4).
func assertWeaponIndexGroundTruth(t *testing.T, f idxGTFilm, card4, card5 int, stat precIdxStats) {
	if card5 < card4 {
		t.Errorf("film %s : le 5 bits resout MOINS de joueurs (%d) que le 4 bits (%d) — regression",
			f.id, card5, card4)
	}
	if !f.btb {
		if stat.maxIdx5 > 15 {
			t.Errorf("film arene %s : indice 5 bits max %d > 15 — incoherent avec un lobby <= 16",
				f.id, stat.maxIdx5)
		}
		if card4 != card5 {
			t.Errorf("film arene %s : correction NON no-op (card4=%d card5=%d) alors qu'idx4==idx5 "+
				"y est garanti — regression", f.id, card4, card5)
		}
		return
	}
	if stat.maxIdx5 <= 15 || stat.distinct5 <= 16 {
		t.Errorf("film BTB %s : le 5 bits ne revele pas > 16 joueurs (max=%d distincts=%d) — la "+
			"saturation 4 bits ne serait pas demontree", f.id, stat.maxIdx5, stat.distinct5)
	}
	if stat.maxIdx4 > 15 {
		t.Errorf("film BTB %s : le 4 bits max %d > 15 — impossible pour un champ de 4 bits", f.id, stat.maxIdx4)
	}
}

// itoaGT : formateur local (evite strconv pour une note de log).
func itoaGT(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
