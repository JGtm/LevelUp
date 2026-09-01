package objectiveevents

// assaut_footer_research_test.go — L'ARMEMENT DE LA BOMBE DANS LE PIED DE FILM.
//
// # POURQUOI ICI, ET PAS DANS LE STATBORG
//
// L'utilisateur, le 2026-08-31 : « statborg a les prises de colline en KOTH, les jauges et
// minuteurs pour CTF, les zones en capture et tout pour Strongholds. Ce serait etrange d'avoir
// ca a un autre endroit. » L'intuition est juste, et elle designe le bon voisinage — mais pas
// le bon fichier : **les prises de zone, les prises de colline et la possession du crane ne
// viennent PAS des composants du statborg. Elles viennent des evenements `th=10` du PIED DE
// FILM** (`extractFromTh10`, cf. le dispatch d'`Extract`). Seul le CTF passe par une autre voie
// (les bursts de l'echelle de score).
//
// Or l'Assaut n'a JAMAIS ete lu la : `classifyObjectiveMode` rendait "" jusqu'a ce jour, et
// `ObjectiveTypeBomb` tombe encore dans le `default` du dispatch. Le pied de ces neuf films n'a
// donc jamais ete ouvert.
//
// # DEUX FILTRES A LEVER, ET LE SECOND EST LE PLUS INTERESSANT
//
// `decodeTh10Block` REFUSE tout bloc dont l'octet 47 ne vaut pas 10 (`th != 10`). C'est le bon
// filtre pour les zones ; c'est aussi un mur si l'Assaut porte son armement sous un AUTRE
// indice de type. Cette sonde ne filtre rien : elle releve la valeur de l'octet 47 telle
// qu'elle est, et compte.
//
// # LE CRITERE, ecrit avant la mesure — le meme que la chasse au statborg (phase A6)
//
//	COUVERTURE   au moins un evenement de cet indice avant CHAQUE explosion du releve A0.3 ;
//	CONSTANCE    les delais resserres — dispersion <= 20 % de la mediane ;
//	SENS         delai positif et sous 120 s.
//
// Un indice qui tient les trois donne l'armement, et sa mediane donne la MECHE.
//
// REGIME : garde `ASSAUT_CACHE` (racine du cache film). Aucune base, aucun reseau.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedDeFilm -v -timeout 30m

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// afExplosions : les instants d'explosion DATES du releve A0.3 (`A_PROTOCOLE.md` §2), recopies
// sans modification — le meme oracle que la phase A5.
var afExplosions = map[string][]int{
	"35b75a31": {304013, 541270, 787051},
	"69b16f5d": {154305, 278617, 310215},
	"3d58eb37": {203065, 342196, 386280},
	"34bb3bc8": {427120},
	"1c01e34f": {150546, 273787, 335637, 400853},
	"ce083875": {512505, 686401, 947537},
	"df8fcbef": {255767, 309284, 485860, 778033},
	"c75f33b8": {109549, 395724, 450833},
	"9f57c612": {83322, 298489, 353160, 469057},
}

const (
	afMecheMaxMS = 120_000
	afCVMax      = 0.20
)

// afSource lit le cache film SANS passer par `filmcache` : ce paquet lui est AMONT, et
// l'importer ferait un cycle. Trente lignes de relecture valent mieux qu'une inversion de
// dependance posee pour une sonde.
type afSource struct {
	chunks []ChunkMeta
	dir    string
}

func (s afSource) Chunks() []ChunkMeta { return s.chunks }

func (s afSource) ChunkData(index int) ([]byte, bool) {
	raw, err := os.ReadFile(filepath.Join(s.dir, fmt.Sprintf("chunk_%02d.bin", index)))
	if err != nil {
		return nil, false
	}
	return raw, true
}

// afOuvrir lit le manifeste et rend la source, ou faux si le film manque.
func afOuvrir(t *testing.T, cache, id string) (afSource, bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(cache, "film_manifests", id+".json"))
	if err != nil {
		return afSource{}, false
	}
	var mf struct {
		Chunks []struct {
			Index     int `json:"index"`
			ChunkType int `json:"chunk_type"`
			StartMS   int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		t.Fatalf("%s : manifeste illisible : %v", id, err)
	}
	src := afSource{dir: filepath.Join(cache, "film_chunks", id)}
	for _, c := range mf.Chunks {
		src.chunks = append(src.chunks, ChunkMeta{Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS})
	}
	return src, len(src.chunks) > 0
}

// afBloc est un bloc du pied, avec son indice de type LU et non filtre.
type afBloc struct {
	th, t, slot int
}

// afScanTousLesIndices reprend `scanTh10Events` en RELEVANT l'octet 47 au lieu de l'exiger
// egal a 10. C'est la seule difference, et c'est tout l'objet de la sonde.
func afScanTousLesIndices(data []byte) []afBloc {
	total := len(data) * 8
	var out []afBloc
	seen := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := readByteAtBit(data, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xstart := xe - 64
		if seen[xstart] {
			continue
		}
		x := readU64LEAtBit(data, xstart)
		if x <= minXUID || x >= maxXUID {
			continue
		}
		seen[xstart] = true
		if b, ok := afDecodeBloc(data, xstart, total); ok {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// afDecodeBloc : `decodeTh10Block` sans son filtre sur l'indice de type.
func afDecodeBloc(data []byte, xstart, total int) (afBloc, bool) {
	win := xstart + 20000
	if win > total {
		win = total
	}
	for b := xstart; b <= win-32; b++ {
		if readByteAtBit(data, b) == 0 && readByteAtBit(data, b+8) == 0 &&
			readByteAtBit(data, b+16) == 0x2e && readByteAtBit(data, b+24) == 0xe0 {
			ebs := b - 60*8
			if ebs < xstart {
				return afBloc{}, false
			}
			t := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
				int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
			return afBloc{
				th:   int(readByteAtBit(data, ebs+47*8)),
				t:    t,
				slot: int(readByteAtBit(data, ebs+36*8)),
			}, true
		}
	}
	return afBloc{}, false
}

// TestAssautPiedDeFilm ouvre le pied des neuf films et cherche l'armement parmi TOUS les
// indices de type.
func TestAssautPiedDeFilm(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	films := make([]string, 0, len(afExplosions))
	for id := range afExplosions {
		films = append(films, id)
	}
	sort.Strings(films)

	tally := map[int]int{}
	delais := map[int][]float64{}
	couverts := map[int]int{}
	total := 0
	for _, id := range films {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			t.Fatalf("film %s absent du cache (%s)", id, cache)
		}
		footer, ok := footerData(src)
		if !ok {
			t.Logf("%s : AUCUN PIED DE FILM lisible", id)
			continue
		}
		blocs := afScanTousLesIndices(footer)
		parTh := map[int][]int{}
		for _, b := range blocs {
			tally[b.th]++
			parTh[b.th] = append(parTh[b.th], b.t)
		}
		t.Logf("%s : pied de %d octets, %d bloc(s), indices %v",
			id, len(footer), len(blocs), afIndicesTries(parTh))
		exps := afExplosions[id]
		total += len(exps)
		for th, ts := range parTh {
			for _, ms := range exps {
				meilleur := -1
				for _, p := range ts {
					d := ms - p
					if d > 0 && d <= afMecheMaxMS && (meilleur < 0 || d < meilleur) {
						meilleur = d
					}
				}
				if meilleur >= 0 {
					couverts[th]++
					delais[th] = append(delais[th], float64(meilleur))
				}
			}
		}
	}

	t.Logf("TALLY des indices de type sur les 9 films : %v", afTallyTrie(tally))
	var tenus []int
	for th := range delais {
		if couverts[th] < total {
			continue
		}
		if _, cv := afMedianeEtCV(delais[th]); cv <= afCVMax {
			tenus = append(tenus, th)
		}
	}
	sort.Ints(tenus)
	if len(tenus) == 0 {
		t.Logf("AUCUN INDICE NE TIENT LES TROIS CRITERES (%d explosions). Les couvertures :", total)
		type l struct {
			th, n   int
			med, cv float64
		}
		var ls []l
		for th, n := range couverts {
			med, cv := afMedianeEtCV(delais[th])
			ls = append(ls, l{th, n, med, cv})
		}
		sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })
		for i, x := range ls {
			if i >= 10 {
				break
			}
			t.Logf("  th=%d : %d/%d couvertes, mediane %.1f s, dispersion %.0f %%",
				x.th, x.n, total, x.med/1000, x.cv*100)
		}
		return
	}
	for _, th := range tenus {
		med, cv := afMedianeEtCV(delais[th])
		t.Logf("CANDIDAT th=%d : %d/%d explosions couvertes, meche mediane %.1f s, dispersion %.0f %%",
			th, couverts[th], total, med/1000, cv*100)
	}
}

func afIndicesTries(m map[int][]int) []string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		out = append(out, fmt.Sprintf("th=%d(%d)", k, len(m[k])))
	}
	return out
}

func afTallyTrie(m map[int]int) []string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		out = append(out, fmt.Sprintf("th=%d:%d", k, m[k]))
	}
	return out
}

func afMedianeEtCV(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, math.Inf(1)
	}
	tri := append([]float64(nil), xs...)
	sort.Float64s(tri)
	med := tri[len(tri)/2]
	if med == 0 {
		return med, math.Inf(1)
	}
	var s float64
	for _, x := range xs {
		s += (x - med) * (x - med)
	}
	return med, math.Sqrt(s/float64(len(xs))) / med
}
