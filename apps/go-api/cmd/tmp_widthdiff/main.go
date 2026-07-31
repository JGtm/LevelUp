// tmp_widthdiff — LA MESURE QUI MANQUAIT : ce que NOTRE decodeur consomme, composant par
// composant, sur le VRAI film, confronte a ce que le desérialiseur du jeu consomme reellement.
//
// POURQUOI ELLE EST DECISIVE. Toutes les mesures precedentes comparaient soit des largeurs
// ATTEIGNABLES (chaque deser execute sur des flux ALEATOIRES), soit des presences agregees. Ni
// l'une ni l'autre ne dit OU le curseur decroche sur les donnees reelles. Ici on lit les
// StartBit successifs d'un meme record : leur difference EST la largeur que notre code a
// consommee, sur les octets du film.
//
// LES DEUX COTES :
//   - NOTRE COTE : DecodeFrameRecords sur 000d5950, EntityTrace.Comps[i].StartBit.
//   - LE VRAI COTE : .ai/re_dump/ce_capture_delta.csv, 807 855 curseurs captures sur le SEUL
//     site de dispatch du jeu (FUN_14076cb60). largeur(N) = bitCursor(N+1) - bitCursor(N).
//
// PRECAUTION : les deux traces viennent de FILMS DIFFERENTS (la capture est la partie live,
// idLow 14 ; 000d5950 a idLow 11). On ne peut donc PAS aligner record par record. Ce qui se
// compare est la DISTRIBUTION DES LARGEURS par composant — elle, ne depend pas du film pour
// tous les composants dont la largeur est un litteral de l'executable.
//
// CE QUE LA SORTIE DONNE : par composant, notre largeur dominante et la vraie, l'ecart, et
// surtout la POSITION MOYENNE du composant dans l'ordre de lecture. Un composant faux qui
// arrive tot coute bien plus cher qu'un composant faux qui arrive tard.
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

// bipedTI est l'archetype du biped joueur, le seul dont la verite terrain soit dense.
const bipedTI = 35

type dist struct {
	counts map[int]int
	n      int
}

func (d *dist) add(w int) {
	if d.counts == nil {
		d.counts = map[int]int{}
	}
	d.counts[w]++
	d.n++
}

// mode rend la largeur dominante et sa part.
func (d dist) mode() (int, float64) {
	best, bestN := -1, 0
	for w, n := range d.counts {
		if n > bestN || (n == bestN && w < best) {
			bestN, best = n, w
		}
	}
	if d.n == 0 {
		return -1, 0
	}
	return best, float64(bestN) / float64(d.n)
}

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "chunks du film")
	repo := flag.String("repo", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\.claude\worktrees\filmdec-continuation`, "racine du depot")
	idlow := flag.Int("idlow", 11, "largeur du champ id bas (valeur de RUNTIME, 11 sur ce film)")
	flag.Parse()

	truth, err := loadTruth(filepath.Join(*repo, ".ai", "re_dump", "ce_capture_delta.csv"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "verite terrain:", err)
		os.Exit(1)
	}
	ours, order, records := walkOurs(*dir, *idlow)

	fmt.Printf("CONFRONTATION DES LARGEURS — %d records bipedes decodes\n\n", records)
	fmt.Printf("  %-42s %-16s %-16s %-8s %s\n",
		"composant", "NOUS (mode)", "VRAI (mode)", "ecart", "rang moyen")
	type row struct {
		idx         int
		name        string
		ourW, trueW int
		ourF, trueF float64
		ourN, trueN int
		rank        float64
		diff        int
	}
	var rows []row
	for idx, d := range ours {
		t, ok := truth[idx]
		if !ok || d.n == 0 {
			continue
		}
		ow, of := d.mode()
		tw, tf := t.mode()
		rows = append(rows, row{idx, compName[idx], ow, tw, of, tf, d.n, t.n,
			order[idx].mean(), ow - tw})
	}
	// tri par RANG DE LECTURE : la premiere faute est celle qui compte.
	sort.Slice(rows, func(i, j int) bool { return rows[i].rank < rows[j].rank })
	for _, r := range rows {
		flag := ""
		if r.diff != 0 {
			flag = fmt.Sprintf("%+d", r.diff)
			if r.rank < 3 {
				flag += "  <== TOT DANS LA LECTURE"
			}
		} else {
			flag = "OK"
		}
		nm := r.name
		if nm == "" {
			nm = fmt.Sprintf("i%02d", r.idx)
		} else {
			nm = fmt.Sprintf("i%02d %s", r.idx, nm)
		}
		fmt.Printf("  %-42s %4d (%3.0f%%, n=%-6d) %4d (%3.0f%%, n=%-6d) %-8s %.2f\n",
			trunc(nm, 42), r.ourW, 100*r.ourF, r.ourN, r.trueW, 100*r.trueF, r.trueN,
			flag, r.rank)
	}

	bad := 0
	for _, r := range rows {
		if r.diff != 0 {
			bad++
		}
	}
	fmt.Printf("\n  %d composants comparables, %d divergents\n", len(rows), bad)
	fmt.Println("\n  LECTURE : trier par rang moyen met la PREMIERE faute en haut. Un composant")
	fmt.Println("  divergent qui arrive au rang 1 ou 2 decale tout le reste du record ; le meme")
	fmt.Println("  au rang 6 ne coute que la queue.")
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

type meanAcc struct {
	sum float64
	n   int
}

func (m *meanAcc) add(v float64) { m.sum += v; m.n++ }
func (m meanAcc) mean() float64 {
	if m.n == 0 {
		return 99
	}
	return m.sum / float64(m.n)
}

var compName = map[int]string{}

// walkOurs deroule NOTRE decodeur sur le film et mesure la largeur consommee par composant.
func walkOurs(dir string, idlow int) (map[int]*dist, map[int]*meanAcc, int) {
	out := map[int]*dist{}
	rank := map[int]*meanAcc{}
	records := 0
	cfg := filmdec.DefaultFrameConfig()
	cfg.IDLowBits = idlow

	// Le registre du chunk_00 nomme les composants de chaque archetype : sans lui, le walk
	// ne sait pas quel deser appeler.
	reg, err := filmdec.ParseRegistryChunk(mustChunk(dir, 0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "registre:", err)
		os.Exit(1)
	}
	// LE MONDE. L'amorçage par keyframe ne lie que ~123 slots : la quasi-totalité des records
	// delta tombe alors sur un slot non lié et le PAQUET ENTIER est abandonné (le jeu fait de
	// même : c'est un port fidèle, pas une faiblesse). Mesuré : 5 records décodés au lieu de
	// 48 000. On prend donc le dump CE quand il existe — c'est l'oracle de liaison, il couvre
	// 1 396 slots — et le keyframe seulement en repli.
	bind := loadDump(filepath.Join(dir, "world_dump_full.txt"))
	mkWorld := func(kf []byte) *filmdec.World {
		if len(bind) > 0 {
			w := filmdec.NewWorld(reg)
			for id, ti := range bind {
				w.BindFull(id, ti)
			}
			return w
		}
		return filmdec.WorldFromKeyframe(reg, kf)
	}
	n := filmdec.CountFilmChunks(dir)
	var world *filmdec.World
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			pay := p.Payload(chunk)
			if p.Type == filmdec.PacketTypeKeyframe && world == nil {
				world = mkWorld(pay)
				continue
			}
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			if world == nil {
				world = mkWorld(nil)
			}
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(pay), world, cfg)
			for _, r := range recs {
				if r.Type != 3 || r.TypeIndex != bipedTI || len(r.Trace.Comps) < 2 {
					continue
				}
				records++
				cs := r.Trace.Comps
				for i := range cs {
					end := r.Trace.EndBit
					if i+1 < len(cs) {
						end = cs[i+1].StartBit
					}
					w := end - cs[i].StartBit
					if w <= 0 || w > 4096 {
						continue
					}
					if out[cs[i].Index] == nil {
						out[cs[i].Index] = &dist{}
						rank[cs[i].Index] = &meanAcc{}
					}
					out[cs[i].Index].add(w)
					rank[cs[i].Index].add(float64(i))
					if cs[i].Name != "" {
						compName[cs[i].Index] = cs[i].Name
					}
				}
			}
		}
	}
	return out, rank, records
}

// loadTruth reconstitue la largeur reelle par composant depuis la capture. La trace est
// SEQUENTIELLE : une BAISSE de bitCursor = nouveau paquet, un changement d'eid = nouveau record.
// La largeur d'un composant est l'ecart au curseur SUIVANT DU MEME RECORD.
func loadTruth(path string) (map[int]*dist, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.Comment = '#' // la capture commence par une ligne de commentaire
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}
	need := []string{"eid", "typeIndex", "compIndex", "bitCursor"}
	for _, k := range need {
		if _, ok := col[k]; !ok {
			return nil, fmt.Errorf("colonne %q absente (en-tete : %v)", k, rows[0])
		}
	}
	out := map[int]*dist{}
	var prevEID, prevComp, prevCur = "", -1, -1
	var prevTI = -1
	for _, row := range rows[1:] {
		eid := row[col["eid"]]
		ti, _ := strconv.Atoi(row[col["typeIndex"]])
		ci, _ := strconv.Atoi(row[col["compIndex"]])
		cur, _ := strconv.Atoi(row[col["bitCursor"]])
		if prevCur >= 0 && eid == prevEID && prevTI == bipedTI && cur > prevCur {
			w := cur - prevCur
			if w > 0 && w <= 4096 {
				if out[prevComp] == nil {
					out[prevComp] = &dist{}
				}
				out[prevComp].add(w)
			}
		}
		prevEID, prevComp, prevCur, prevTI = eid, ci, cur, ti
	}
	return out, nil
}

// mustChunk lit un chunk ou meurt : sans le chunk_00 il n'y a pas de registre, donc pas de walk.
func mustChunk(dir string, i int) []byte {
	b, err := filmdec.ReadFilmChunk(dir, i)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chunk", i, ":", err)
		os.Exit(1)
	}
	return b
}

// loadDump lit le dump de liaisons slot->archetype capture sur le jeu. Absent, on retombe sur
// le keyframe (moins couvrant, mais universel et 100 % hors ligne).
func loadDump(path string) map[uint32]uint32 {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := map[uint32]uint32{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			id, e1 := strconv.ParseUint(kv[0], 10, 64)
			ti, e2 := strconv.Atoi(kv[1])
			if e1 != nil || e2 != nil {
				continue
			}
			out[uint32(id)] = uint32(ti)
		}
	}
	return out
}
