package replay

// ctf_shortvariant_research_test.go — INSTRUMENT DE RECHERCHE #5 (v7.5, colonne ①).
//
// # CE QU'IL INSTRUIT
//
// Le rejeu perd 13 à 31 % des tirs d'un match. J'ai d'abord écrit que c'était parce qu'il
// « balaie les bits là où la piste E lit les paquets » — **c'était FAUX** : `ScanFilmFireEvents`
// utilise déjà `WalkPackets`. La vraie différence tient à UNE LIGNE :
//
//	if int(pay[0]>>1) != FireEventType || int(pay[0])&1 != 0 { continue }
//	                                      ^^^^^^^^^^^^^^^^^^ la VARIANTE COURTE est écartée
//
// La piste E sélectionne `pay[0]>>1 == 105` sans regarder le bit de variante ; le rejeu exige
// `&1 == 0`. **L'écart mesuré entre les deux, ce sont les records COURTS.**
//
// # LA QUESTION, ET POURQUOI ELLE N'EST PAS TRANCHÉE D'AVANCE
//
// Le dossier dit du record court : « NE PORTE PAS d'arme (mesuré : 3/313), sa sémantique n'est
// pas établie — il n'est pas émis par ce décodeur ». Le décodeur a donc RAISON de s'abstenir
// tant qu'on ignore ce que c'est. Émettre comme « tir » un record qui serait un rechargement,
// une touche ou une fin de rafale gonflerait le calque avec des faux — exactement l'erreur que
// ce chantier refuse partout ailleurs.
//
// Ce fichier ne suppose donc RIEN. Il mesure quatre choses, et c'est leur faisceau qui tranche :
//
//	1. combien de courts, contre combien de longs
//	2. long seul, et long+court, RAPPORTÉS AUX TIRS DE L'API — si long+court colle et que long
//	   seul manque, le court est un tir ; s'il dépasse, il est autre chose
//	3. l'index d'attaquant du court est-il lisible au MÊME offset, et couvre-t-il les 8 joueurs ?
//	   Un champ qui ne se distribue pas sur le roster n'est pas un index de joueur.
//	4. l'écart temporel entre un court et le long le plus proche du MÊME joueur — un court qui
//	   suit systématiquement un long à quelques ms est une SUITE de tir, pas un tir de plus.
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_SHORT_FILMS="64e8adfa,0edb8512,000d5950" \
//	  go test ./internal/analysis/replay/ -run CTFShortVariant -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const ctfShortFilmsEnv = "CTF_SHORT_FILMS"

// shortRec : un record type 105, sa variante, son instant et son index d'attaquant lu au MÊME
// offset que pour le long. Lire au même offset est l'HYPOTHÈSE TESTÉE, pas un acquis.
type shortRec struct {
	tUS   uint64
	long  bool
	pi    int
	bits  int
	first byte
}

func TestCTFShortVariant(t *testing.T) {
	spec := os.Getenv(ctfShortFilmsEnv)
	if spec == "" {
		t.Skipf("variante courte non demandée : %s vide", ctfShortFilmsEnv)
	}
	cache, outDir := os.Getenv(ctfCacheEnv), os.Getenv(ctfOutEnv)
	if cache == "" || outDir == "" {
		t.Fatalf("%s et %s sont requis", ctfCacheEnv, ctfOutEnv)
	}
	for _, short := range strings.Split(spec, ",") {
		short = strings.TrimSpace(short)
		t.Run(short, func(t *testing.T) {
			b := ctfShortReport(t, filepath.Join(cache, "film_chunks", short), short)
			if err := os.WriteFile(filepath.Join(outDir, short+"_variante.txt"), []byte(b), 0o644); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			t.Logf("\n%s", b)
		})
	}
}

func ctfShortReport(t *testing.T, dir, short string) string {
	t.Helper()
	recs := scanAllFireRecords(t, dir)
	var b strings.Builder
	fmt.Fprintf(&b, "film\t%s\nrecords_105\t%d\n", short, len(recs))
	nLong, nShort := 0, 0
	for _, r := range recs {
		if r.long {
			nLong++
		} else {
			nShort++
		}
	}
	fmt.Fprintf(&b, "longs\t%d\tcourts\t%d\tpart_courts\t%.4f\n", nLong, nShort, ratio(nShort, len(recs)))
	ctfWriteShortIndexes(&b, recs)
	ctfWriteShortSizes(&b, recs)
	ctfWriteShortDelays(&b, recs)
	return b.String()
}

// scanAllFireRecords lit TOUS les records type 105, les deux variantes — c'est le seul endroit
// du dépôt qui le fasse, et c'est délibérément un instrument de recherche : la production
// continue de n'émettre que les longs tant que la sémantique du court n'est pas établie.
func scanAllFireRecords(t *testing.T, dir string) []shortRec {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	var out []shortRec
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
			r := shortRec{tUS: p.TimestampUS, long: pay[0]&1 == 0, pi: -1,
				bits: len(pay) * 8, first: pay[0]}
			if len(pay)*8 >= filmdec.FireHeadBits {
				r.pi = filmdec.ReadAttackerIndex(pay)
			}
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tUS < out[j].tUS })
	return out
}

// ctfWriteShortIndexes — LE TEST QUI PEUT DISQUALIFIER LE COURT. Si l'index lu au même offset ne
// se distribue pas sur les huit joueurs, ce n'est pas un index d'attaquant, et le court ne peut
// pas être publié comme un tir de joueur.
func ctfWriteShortIndexes(b *strings.Builder, recs []shortRec) {
	byIdx := map[int]int{}
	byIdxLong := map[int]int{}
	for _, r := range recs {
		if r.pi < 0 {
			continue
		}
		if r.long {
			byIdxLong[r.pi]++
			continue
		}
		byIdx[r.pi]++
	}
	fmt.Fprintf(b, "\n# index d'attaquant lu au MEME offset — courts contre longs\n")
	keys := map[int]bool{}
	for k := range byIdx {
		keys[k] = true
	}
	for k := range byIdxLong {
		keys[k] = true
	}
	ks := make([]int, 0, len(keys))
	for k := range keys {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Fprintf(b, "index\t%d\tcourts\t%d\tlongs\t%d\n", k, byIdx[k], byIdxLong[k])
	}
	fmt.Fprintf(b, "index_distincts_courts\t%d\tindex_distincts_longs\t%d\n", len(byIdx), len(byIdxLong))
}

// ctfWriteShortSizes ventile la TAILLE des payloads : un record court qui aurait une taille
// propre et constante est un autre TYPE d'enregistrement, pas un tir amputé.
func ctfWriteShortSizes(b *strings.Builder, recs []shortRec) {
	sizes := map[int]int{}
	for _, r := range recs {
		if !r.long {
			sizes[r.bits]++
		}
	}
	type kv struct{ bits, n int }
	rows := make([]kv, 0, len(sizes))
	for s, n := range sizes {
		rows = append(rows, kv{s, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	fmt.Fprintf(b, "\n# taille du payload des COURTS (bits) — top 8\n")
	for i, r := range rows {
		if i >= 8 {
			break
		}
		fmt.Fprintf(b, "bits\t%d\tn\t%d\n", r.bits, r.n)
	}
}

// ctfWriteShortDelays mesure l'écart entre un court et le long le plus proche EN AMONT du même
// index. Un court qui suit systématiquement un long à quelques millisecondes est une SUITE du
// même tir ; un court indépendant est un candidat tir à part entière.
func ctfWriteShortDelays(b *strings.Builder, recs []shortRec) {
	var d []int64
	orphans := 0
	for i, r := range recs {
		if r.long || r.pi < 0 {
			continue
		}
		best := int64(-1)
		for j := i - 1; j >= 0; j-- {
			if recs[j].long && recs[j].pi == r.pi {
				best = int64(r.tUS-recs[j].tUS) / 1000
				break
			}
		}
		if best < 0 {
			orphans++
			continue
		}
		d = append(d, best)
	}
	fmt.Fprintf(b, "\n# ecart court -> long precedent du MEME index (ms)\n")
	if len(d) == 0 {
		fmt.Fprintf(b, "aucun\torphelins\t%d\n", orphans)
		return
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	fmt.Fprintf(b, "n\t%d\torphelins\t%d\tp10\t%d\tp25\t%d\tmediane\t%d\tp75\t%d\tp90\t%d\n",
		len(d), orphans, d[len(d)/10], d[len(d)/4], d[len(d)/2], d[len(d)*3/4], d[len(d)*9/10])
	under50 := 0
	for _, v := range d {
		if v <= 50 {
			under50++
		}
	}
	fmt.Fprintf(b, "part_sous_50ms\t%.4f\n", ratio(under50, len(d)))
}
