package filmdec

// deto_preuve_finder_test.go — CHERCHE dans le corpus les films les plus riches en tirs de
// LANCEUR EXPLOSIF DIRECT (M41 SPNKr, Skewer...), pour moissonner assez de KILLS explosifs sur
// la verite terrain de la preuve (TestDetoPreuveRobuste). Un Fiesta tire peu de roquettes ;
// certaines listes (Rockets, Heavies) en tirent beaucoup. Garde LOT1_CORPUS.
//
// Le compte est un PROXY cheap (decodeFireEvent est a offsets fixes, sans monde) : nombre de
// tirs dont l'arme est un lanceur explosif DIRECT sur les premiers chunks. On rend aussi le
// nombre de FilmIndex distincts (taille de lobby : ~8 arene, > 8 BTB) et la signature d'axe
// (auto-detectable = la preuve tourne sans forcer LOT1_SONDE_MAP).

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

const (
	rockFinderPeek = 6   // chunks echantillonnes par film
	rockFinderTop  = 30  // films listes en tete
	rockFinderCapD = 500 // films scannes par defaut (LOT1_FINDER_CAP l'ecrase)
)

// rockFilmStat : le comptage d'un film.
type rockFilmStat struct {
	name    string
	rockets int
	films   int     // FilmIndex distincts
	axis    [3]uint // signature d'axe (bornes monde)
	chunks  int
}

// rockCountFilm compte les tirs de lanceur explosif direct et les FilmIndex distincts.
func rockCountFilm(dir string, upTo int) (rockets, distinct int) {
	seen := map[int]bool{}
	for c := 1; c <= upTo; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if int(pay[0]>>1) != FireEventType || int(pay[0])&1 != 0 {
				continue
			}
			fe, ok := decodeFireEvent(pay)
			if !ok {
				continue
			}
			seen[fe.FilmIndex] = true
			name := geoWeaponName(fe.WeaponID)
			if lot1IsHeavy(name) && geoIsDirect(name) {
				rockets++
			}
		}
	}
	return rockets, len(seen)
}

// TestDetoPreuveFindRockets liste les films du corpus les plus riches en lanceurs explosifs.
func TestDetoPreuveFindRockets(t *testing.T) {
	root := os.Getenv("LOT1_CORPUS")
	if root == "" {
		t.Skip("LOT1_CORPUS absent : recherche de films a roquettes sautee")
	}
	release := LockProcessDecode()
	defer release()
	scanCap := rockFinderCapD
	if v := os.Getenv("LOT1_FINDER_CAP"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			scanCap = k
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("corpus illisible : %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var stats []rockFilmStat
	scanned := 0
	for _, nm := range names {
		if scanned >= scanCap {
			break
		}
		dir := filepath.Join(root, nm)
		if CountFilmChunks(dir) < 8 {
			continue
		}
		scanned++
		rockets, distinct := rockCountFilm(dir, rockFinderPeek)
		if rockets == 0 {
			continue
		}
		st := rockFilmStat{name: nm, rockets: rockets, films: distinct, chunks: CountFilmChunks(dir)}
		if lay, _, err := DetectI0Layout(dir); err == nil {
			st.axis = lay.AxisW
		}
		stats = append(stats, st)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].rockets > stats[j].rockets })
	t.Logf("== %d films scannes · %d avec au moins un tir de lanceur explosif ==", scanned, len(stats))
	top := rockFinderTop
	if len(stats) < top {
		top = len(stats)
	}
	for i := 0; i < top; i++ {
		s := stats[i]
		t.Logf("%2d. %s · %d tirs explosifs (%d premiers chunks) · %d FilmIndex · axe %v · %d chunks",
			i+1, s.name, s.rockets, rockFinderPeek, s.films, s.axis, s.chunks)
	}
}
