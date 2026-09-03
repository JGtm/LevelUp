package analysis_test

// weapon_index_equivalence_test.go — LA PREUVE que la CLE DE TIREUR du NUMERATEUR de precision
// (film, filmdec) et celle du DENOMINATEUR (shared.match_weapon_shots, analysis) sont LE MEME
// CHAMP, au bit pres, sur les memes records de tir 0xD2.
//
// CONTEXTE DU BUG (Lot 3). La precision par arme = touches / tirs par joueur.
//   - DENOMINATEUR (tirs)   : analysis.ScanFireEventsB5 -> FireEvent.PlayerIndex5 (event_start+31,
//                             5 bits). C'est l'indice qu'ecrit match_weapon_shots, et le pont
//                             resolvePlayerIndices(indice->xuid) est keye dessus.
//   - NUMERATEUR (touches)  : filmdec.decodeFireEvent -> FilmIndex (bits 36..39, soit le champ
//                             attaquant x2 >>1 = 4 bits). AVANT le correctif, le mapper resolvait
//                             piToXUID[FilmIndex] : un 4 bits contre un pont keye sur 5 bits.
//
// Sous 17 joueurs (arene) les deux lectures RENDENT LA MEME VALEUR (le bit 35 est 0). Au-dela
// (BTB, >16 joueurs), le 4 bits SATURE a 15 et fusionne deux tireurs -> num et denom pointent
// des joueurs DIFFERENTS. Le correctif expose filmdec.ShooterIndex5 (bits 35..39, R(5) sans >>1)
// et key le numerateur dessus. Ce test MESURE que ShooterIndex5 == PlayerIndex5 record par record,
// sur arene ET BTB 4f77afc1 (le film ou >16 joueurs revele la saturation).
//
// LECTURE : marqueur universel d'analysis (11 bits) place le champ a event_start+31 ; l'ancre
// paquet de filmdec place le champ a bit 35 du payload. Pour un record 0xD2 dont le marqueur
// s'aligne sur le bit 1 du payload (event_start = bit 4), event_start+31 == bit 35 : MEMES BITS.
// On correle par position en octet (analysis.FireEvent.BytePos == filmdec paquet Start) et on
// verifie l'egalite. Env PRECISION_CORPUS = repertoire de base des films (data/cache/film_chunks).

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// idxEqZeroTS : la mesure ne depend pas de l'horodatage (on correle par octet, pas par temps).
func idxEqZeroTS(int) float64 { return 0 }

// idxEqFilm : un film du corpus, avec l'attendu roster (arene <=16, BTB >16).
type idxEqFilm struct {
	id  string
	btb bool // roster > 16 attendu (au moins un indice de tireur >= 16)
}

// idxEqCorpus : les films cites par la mission. 3 arene + le BTB decisif (Forge, >=16 joueurs).
var idxEqCorpus = []idxEqFilm{
	{id: "000d5950"},
	{id: "01e1f945"},
	{id: "00502e52"},
	{id: "4f77afc1", btb: true},
}

// idxEqStats : l'accumulateur d'un film.
type idxEqStats struct {
	records     int // records 0xD2 longs decodes (numerateur)
	matched     int // records correles a un event analysis (meme octet)
	mismatch    int // records ou ShooterIndex5 != PlayerIndex5 (DOIT rester 0)
	invariantKO int // records ou ShooterIndex5 & 0x0F != FilmIndex (DOIT rester 0)
	ge16        int // records dont ShooterIndex5 >= 16 (revele la saturation 4 bits)
	maxIdx5     int
	maxIdx4     int
	// corr : table de correspondance (idx4, idx5) -> nombre de records.
	corr map[[2]int]int
}

func newIdxEqStats() *idxEqStats {
	return &idxEqStats{corr: map[[2]int]int{}, maxIdx5: -1, maxIdx4: -1}
}

// TestWeaponIndexNumDenomEquivalence PROUVE num-cle == denom-cle sur les records 0xD2 (arene + BTB).
func TestWeaponIndexNumDenomEquivalence(t *testing.T) {
	base := os.Getenv("PRECISION_CORPUS")
	if base == "" {
		t.Skip("PRECISION_CORPUS absent : preuve d'equivalence sautee")
	}
	release := filmdec.LockProcessDecode()
	defer release()

	for _, f := range idxEqCorpus {
		dir := filepath.Join(base, f.id)
		if _, err := os.Stat(dir); err != nil {
			t.Logf("film %s absent du corpus (%v) : saute", f.id, err)
			continue
		}
		st := measureFilmIndexEquivalence(t, dir)
		logIdxEqFilm(t, f, st)

		if st.mismatch != 0 {
			t.Errorf("film %s : %d records ou ShooterIndex5 != PlayerIndex5 (num-cle != denom-cle)",
				f.id, st.mismatch)
		}
		if st.invariantKO != 0 {
			t.Errorf("film %s : %d records ou ShooterIndex5 & 0x0F != FilmIndex (invariant 4/5 bits rompu)",
				f.id, st.invariantKO)
		}
		if st.matched == 0 {
			t.Errorf("film %s : aucun record correle analysis<->filmdec (correlation cassee)", f.id)
		}
		if f.btb && st.ge16 == 0 {
			t.Errorf("film BTB %s : aucun indice de tireur >= 16 — la saturation 4 bits ne serait "+
				"pas revelee ; le film attendu doit avoir > 16 joueurs", f.id)
		}
		if !f.btb && st.ge16 != 0 {
			t.Errorf("film arene %s : %d records d'indice >= 16 inattendus (arene <= 16 joueurs)",
				f.id, st.ge16)
		}
	}
}

// measureFilmIndexEquivalence parcourt les chunks : lecture DENOMINATEUR (analysis.ScanFireEventsB5)
// indexee par octet, puis lecture NUMERATEUR (filmdec, paquets 0xD2) correlee au meme octet.
func measureFilmIndexEquivalence(t *testing.T, dir string) *idxEqStats {
	t.Helper()
	st := newIdxEqStats()
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue // film partiel : chunk illisible ignore, comme les scanners de production
		}
		byByte := map[int]int{} // BytePos -> PlayerIndex5 (denominateur)
		for _, ev := range analysis.ScanFireEventsB5(data, idxEqZeroTS) {
			byByte[ev.BytePos] = ev.PlayerIndex5
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta || pk.Size < 5 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 { // type 36 (105) variante LONGUE (porte l'arme)
				continue
			}
			idx5 := filmdec.ReadShooterIndex5(pay) // NUMERATEUR (5 bits, corrige)
			idx4 := filmdec.ReadAttackerIndex(pay) // ancien numerateur (4 bits, tronque)
			if idx5 < 0 || idx4 < 0 {
				continue
			}
			accumulateIdxEq(st, pk.Start, idx4, idx5, byByte)
		}
	}
	return st
}

// accumulateIdxEq enregistre un record : invariant 4/5 bits, correlation au denominateur, table.
func accumulateIdxEq(st *idxEqStats, start, idx4, idx5 int, byByte map[int]int) {
	st.records++
	st.corr[[2]int{idx4, idx5}]++
	if idx5 > st.maxIdx5 {
		st.maxIdx5 = idx5
	}
	if idx4 > st.maxIdx4 {
		st.maxIdx4 = idx4
	}
	if idx5 >= 16 {
		st.ge16++
	}
	if (idx5 & 0x0F) != idx4 {
		st.invariantKO++
	}
	if pi5, ok := byByte[start]; ok {
		st.matched++
		if pi5 != idx5 {
			st.mismatch++
		}
	}
}

// logIdxEqFilm publie la table de correspondance d'un film.
func logIdxEqFilm(t *testing.T, f idxEqFilm, st *idxEqStats) {
	t.Helper()
	kind := "ARENE"
	if f.btb {
		kind = "BTB  "
	}
	t.Logf("== film %s [%s] : %d records 0xD2 · %d correles au denominateur · mismatch %d · "+
		"invariant KO %d · idx5>=16 : %d · max idx4=%d max idx5=%d ==",
		f.id, kind, st.records, st.matched, st.mismatch, st.invariantKO, st.ge16, st.maxIdx4, st.maxIdx5)
	keys := make([][2]int, 0, len(st.corr))
	for k := range st.corr {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][1] != keys[j][1] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	for _, k := range keys {
		note := ""
		if k[1] >= 16 {
			note = "  <- >16 joueurs : le 4 bits (idx4) l'aurait fusionne avec " + itoa(k[1]&0x0F)
		}
		t.Logf("   idx4=%2d  idx5=%2d  n=%d%s", k[0], k[1], st.corr[k], note)
	}
}

// itoa : petit formateur local (evite un import strconv pour une seule note).
func itoa(v int) string {
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
