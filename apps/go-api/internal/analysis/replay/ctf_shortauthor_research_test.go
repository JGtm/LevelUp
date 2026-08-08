package replay

// ctf_shortauthor_research_test.go — INSTRUMENT DE RECHERCHE #6 (v7.5, colonne ①).
//
// # LA QUESTION, HÉRITÉE DE L'INSTRUMENT #5
//
// La variante COURTE du record type 105 est très probablement un TIR : hors Fiesta, longs+courts
// rendent 84 à 99 % des tirs de l'API, contre 72-87 % pour les longs seuls. Mais son AUTEUR n'est
// pas lisible à l'offset du long : le champ s'y concentre sur deux index, en laisse quatre à zéro
// et produit des valeurs hors roster (9, 14, 15).
//
// **Où le record court porte-t-il son tireur ?** Ce fichier balaie les offsets pour le trouver.
//
// # LE CRITÈRE EST POSÉ AVANT LA MESURE, ET IL EST STRICT — SINON LE HASARD GAGNE
//
// Balayer ~250 offsets sur 3 largeurs, c'est ~750 candidats par film. À ce volume, des champs
// passeront « par chance » : un compteur de munitions ou un identifiant d'entité tronqué peuvent
// très bien tomber dans [0,7] et couvrir huit valeurs. Trois garde-fous, écrits ici AVANT d'avoir
// vu un seul résultat :
//
//	1. COUVERTURE      les huit index du roster présents, AUCUN à zéro.
//	2. PURETÉ          moins de 1 % de valeurs hors [0,7].
//	3. CORRÉLATION     le profil par joueur des COURTS doit suivre celui des LONGS. Un joueur qui
//	                   tire beaucoup en long doit produire beaucoup de courts. C'est le contrôle
//	                   INDÉPENDANT : un champ qui satisfait 1 et 2 par hasard n'a aucune raison de
//	                   corréler avec l'activité de tir mesurée par une AUTRE source.
//	4. REPRODUCTIBILITÉ le même offset doit passer sur les QUATRE films. Un offset qui ne tient
//	                   que sur un film est une coïncidence, pas une découverte.
//
// Le critère 4 est le plus important : c'est lui qui rend la recherche par balayage licite.
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_SHORTAUTHOR_FILMS="0edb8512,64e8adfa,9aeca4b3,000d5950" \
//	  go test ./internal/analysis/replay/ -run CTFShortAuthor -timeout 60m

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const ctfShortAuthorFilmsEnv = "CTF_SHORTAUTHOR_FILMS"

const (
	// authorScanMaxBit : borne du balayage. Le champ d'attaquant du long est au bit 36 ; on
	// balaie largement au-delà pour couvrir un décalage de plusieurs champs.
	authorScanMaxBit = 256
	// authorPurityMax : part maximale de valeurs hors [0,7] tolérée (critère 2).
	authorPurityMax = 0.01
	// authorCorrMin : corrélation minimale entre le profil des courts et celui des longs
	// (critère 3). 0,5 est déjà exigeant sur huit points.
	authorCorrMin = 0.5
)

// authorCand est un candidat (offset, largeur, décalage) et son score sur un film.
type authorCand struct {
	bit, width int
	shift      bool // true = valeur décalée d'un bit, comme le champ du long
	covered    int  // index distincts du roster observés
	outside    float64
	corr       float64
}

func (c authorCand) key() string {
	return fmt.Sprintf("%d/%d/%v", c.bit, c.width, c.shift)
}

func (c authorCand) passes() bool {
	return c.covered == 8 && c.outside <= authorPurityMax && c.corr >= authorCorrMin
}

func TestCTFShortAuthor(t *testing.T) {
	spec := os.Getenv(ctfShortAuthorFilmsEnv)
	if spec == "" {
		t.Skipf("recherche d'auteur non demandée : %s vide", ctfShortAuthorFilmsEnv)
	}
	cache, outDir := os.Getenv(ctfCacheEnv), os.Getenv(ctfOutEnv)
	if cache == "" || outDir == "" {
		t.Fatalf("%s et %s sont requis", ctfCacheEnv, ctfOutEnv)
	}
	films := strings.Split(spec, ",")
	passed := map[string]int{}
	byFilm := map[string][]authorCand{}
	var b strings.Builder
	for _, short := range films {
		short = strings.TrimSpace(short)
		cands := ctfScanShortAuthor(t, filepath.Join(cache, "film_chunks", short))
		byFilm[short] = cands
		for _, c := range cands {
			if c.passes() {
				passed[c.key()]++
			}
		}
		fmt.Fprintf(&b, "film\t%s\tcandidats_retenus\t%d\n", short, countPassing(cands))
		writeTopCands(&b, short, cands)
	}
	writeIntersection(&b, passed, len(films))
	path := filepath.Join(outDir, "short_author_scan.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}
	t.Logf("\n%s", b.String())
}

// ctfScanShortAuthor balaie les offsets et rend un candidat par (bit, largeur, décalage).
func ctfScanShortAuthor(t *testing.T, dir string) []authorCand {
	t.Helper()
	longs, shorts := ctfSplitFireRecords(t, dir)
	if len(shorts) == 0 || len(longs) == 0 {
		return nil
	}
	// Profil de référence : combien de tirs LONGS par index de joueur.
	ref := make([]float64, 8)
	for _, pay := range longs {
		if pi := filmdec.ReadAttackerIndex(pay); pi >= 0 && pi < 8 {
			ref[pi]++
		}
	}
	var out []authorCand
	for width := 3; width <= 6; width++ {
		for bit := 0; bit+width <= authorScanMaxBit; bit++ {
			for _, shift := range []bool{false, true} {
				if c, ok := scoreAuthorCand(shorts, ref, bit, width, shift); ok {
					out = append(out, c)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].corr > out[j].corr })
	return out
}

// scoreAuthorCand mesure un candidat. ok=false si trop peu de records portent ce champ.
func scoreAuthorCand(shorts [][]byte, ref []float64, bit, width int, shift bool) (authorCand, bool) {
	hist := make([]float64, 8)
	total, outside := 0, 0
	for _, pay := range shorts {
		if len(pay)*8 < bit+width {
			continue
		}
		v := int(readBitsAtLocal(pay, bit, width))
		if shift {
			v >>= 1
		}
		total++
		if v < 0 || v > 7 {
			outside++
			continue
		}
		hist[v]++
	}
	if total < len(shorts)/2 { // le champ doit exister dans la plupart des records
		return authorCand{}, false
	}
	covered := 0
	for _, n := range hist {
		if n > 0 {
			covered++
		}
	}
	return authorCand{bit: bit, width: width, shift: shift, covered: covered,
		outside: float64(outside) / float64(total), corr: pearson(hist, ref)}, true
}

// ctfSplitFireRecords rend les payloads bruts des records 105, longs et courts séparés.
func ctfSplitFireRecords(t *testing.T, dir string) (longs, shorts [][]byte) {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != filmdec.FireEventType {
				continue
			}
			cp := append([]byte(nil), pay...)
			if pay[0]&1 == 0 {
				longs = append(longs, cp)
				continue
			}
			shorts = append(shorts, cp)
		}
	}
	return longs, shorts
}

// readBitsAtLocal lit n bits MSB-first à partir du bit `at`. Copie LOCALE et assumée : le
// `readBitsAt` de filmdec n'est pas exporté, et exporter un lecteur de bits générique pour un
// balayage de recherche exposerait plus que nécessaire. Elle est bornée, contrairement à lui.
func readBitsAtLocal(pay []byte, at, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		idx := (at + i) >> 3
		if idx >= len(pay) {
			return v << uint(n-i)
		}
		v = v<<1 | uint32((pay[idx]>>(7-uint((at+i)&7)))&1)
	}
	return v
}

// pearson rend la corrélation linéaire entre deux profils de même longueur.
func pearson(a, b []float64) float64 {
	n := float64(len(a))
	var sa, sb float64
	for i := range a {
		sa, sb = sa+a[i], sb+b[i]
	}
	ma, mb := sa/n, sb/n
	var num, da, db float64
	for i := range a {
		x, y := a[i]-ma, b[i]-mb
		num += x * y
		da += x * x
		db += y * y
	}
	if da == 0 || db == 0 {
		return 0
	}
	return num / math.Sqrt(da*db)
}

func countPassing(cands []authorCand) int {
	n := 0
	for _, c := range cands {
		if c.passes() {
			n++
		}
	}
	return n
}

func writeTopCands(b *strings.Builder, short string, cands []authorCand) {
	fmt.Fprintf(b, "# %s — meilleurs candidats par correlation\n", short)
	for i, c := range cands {
		if i >= 8 {
			break
		}
		fmt.Fprintf(b, "cand\tbit\t%d\tlargeur\t%d\tdecale\t%v\tcouverts\t%d\thors_roster\t%.4f\tcorr\t%.4f\tretenu\t%v\n",
			c.bit, c.width, c.shift, c.covered, c.outside, c.corr, c.passes())
	}
	fmt.Fprintf(b, "\n")
}

// writeIntersection — LE CRITÈRE 4, celui qui rend le balayage licite.
func writeIntersection(b *strings.Builder, passed map[string]int, nFilms int) {
	fmt.Fprintf(b, "# candidats retenus sur les %d films (critere 4 : reproductibilite)\n", nFilms)
	keys := make([]string, 0, len(passed))
	for k := range passed {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return passed[keys[i]] > passed[keys[j]] })
	if len(keys) == 0 {
		fmt.Fprintf(b, "AUCUN candidat ne passe les criteres 1-3 sur un seul film.\n")
		return
	}
	for i, k := range keys {
		if i >= 12 {
			break
		}
		fmt.Fprintf(b, "offset\t%s\tfilms\t%d/%d\t%s\n", k, passed[k], nFilms,
			map[bool]string{true: "RETENU SUR TOUS", false: ""}[passed[k] == nFilms])
	}
}
