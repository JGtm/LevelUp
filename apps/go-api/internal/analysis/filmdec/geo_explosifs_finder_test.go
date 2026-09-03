package filmdec

// geo_explosifs_finder_test.go — CHERCHE dans le corpus un film BTB jouable par la geometrie :
// beaucoup de tireurs (index FilmIndex distincts > 8) ET une carte a signature de largeurs
// d'axe UNIQUE (donc bornes monde auto-detectables sans forcage). Garde LOT1_CORPUS (racine du
// corpus). Borne : scanne au plus geoFinderCap films, s'arrete a geoFinderWant candidats.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	geoFinderCap  = 200 // films scannes au maximum (le corpus en compte des milliers)
	geoFinderWant = 8   // candidats BTB retenus au-dela desquels on s'arrete
	geoFinderMinF = 9   // FilmIndex distincts pour qualifier « BTB » (> 8 tireurs)
	geoFinderPeek = 5   // chunks echantillonnes pour compter les tireurs
)

// geoBTBUsableSigs : signatures [X,Y,Z] de cartes a bornes monde exploitables pour un film BTB.
// Le canevas FORGE [15,15,17] regroupe 59 cartes fo## qui PARTAGENT les memes bornes (canevas
// standard) : n'importe laquelle donne des positions correctes (LOT1_SONDE_MAP="flood gulch").
// highpower/oasis/breaker ont une signature UNIQUE (auto-detectable sans forcage).
var geoBTBUsableSigs = map[[3]uint]string{
	{15, 15, 17}: "forge (canevas partage)",
	{18, 19, 17}: "highpower",
	{15, 15, 14}: "oasis",
	{13, 13, 12}: "breaker",
}

// geoDistinctFilmIndex compte les FilmIndex distincts des tirs des premiers chunks (cheap :
// decodeFireEvent est a offsets fixes, sans monde).
func geoDistinctFilmIndex(dir string, upTo int) int {
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
			if fe, ok := decodeFireEvent(pay); ok {
				seen[fe.FilmIndex] = true
			}
		}
	}
	return len(seen)
}

// TestGeoFindBTB liste les films BTB a carte auto-detectable du corpus.
func TestGeoFindBTB(t *testing.T) {
	root := os.Getenv("LOT1_CORPUS")
	if root == "" {
		t.Skip("LOT1_CORPUS absent : recherche de film BTB sautee")
	}
	release := LockProcessDecode()
	defer release()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("corpus illisible : %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	scanned, found := 0, 0
	for _, nm := range names {
		if scanned >= geoFinderCap || found >= geoFinderWant {
			break
		}
		dir := filepath.Join(root, nm)
		if CountFilmChunks(dir) < 8 {
			continue
		}
		scanned++
		lay, _, err := DetectI0Layout(dir)
		if err != nil {
			continue
		}
		mapName, ok := geoBTBUsableSigs[lay.AxisW]
		if !ok {
			continue
		}
		nf := geoDistinctFilmIndex(dir, geoFinderPeek)
		if nf < geoFinderMinF {
			continue
		}
		found++
		t.Logf("BTB candidat : %s · carte %s (signature %v) · %d FilmIndex distincts · %d chunks",
			nm, mapName, lay.AxisW, nf, CountFilmChunks(dir))
	}
	t.Logf("== %d films scannes · %d candidats BTB (carte auto-detectable, >= %d tireurs) ==",
		scanned, found, geoFinderMinF)
}
