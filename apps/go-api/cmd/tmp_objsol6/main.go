// tmp_objsol6 — INVENTAIRE DES OBJETS AU SOL d un film : ce que les keyframes rendent
// vraiment une fois la position declaree hors d atteinte. THROWAWAY (producteur du JSON).
//
// CE QUI EST MESURE ICI, ET RIEN D AUTRE :
//   - la VIE de chaque entite ti=42 (arme au sol) et ti=37 (equipement) : slot, generation,
//     premier et dernier keyframe ou elle est declaree, nombre de keyframes ;
//   - le TYPE de l objet : les identifiants de FAMILLE d arme (high-32) contenus dans son
//     record, resolus par le catalogue de production weaponv3. Le mecanisme est celui de
//     keyframe_loadout.go, dont l ancrage anti-hasard est deja mesure (911 occurrences pour
//     0,52 attendue par pur hasard, ~1750x) et dont la ventilation par archetype attribue
//     deja 397 occurrences a ti=42 ;
//   - la RECURRENCE TEMPORELLE d une meme famille d arme, seul substitut disponible au
//     discriminant spatial prescrit — la POSITION de ces entites n est pas decodable
//     aujourd hui (deux voies essayees, toutes deux refutees sur piece, cf. l en-tete des
//     outils tmp_objsol2 et tmp_objsol5).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_objsol6 [filmDir] [sortie.json]
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

type obs struct {
	tSec     float64
	kfIndex  int
	families []uint32
}

type vie struct {
	TI          int      `json:"archetype_ti"`
	Slot        int      `json:"slot"`
	Gen         int      `json:"generation"`
	Apparition  float64  `json:"apparition_s"`
	Disparition float64  `json:"derniere_vue_s"`
	NbKeyframes int      `json:"keyframes"`
	Familles    []string `json:"familles_hex"`
	Libelles    []string `json:"libelles"`
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	out := ""
	if len(os.Args) > 2 {
		out = os.Args[2]
	}

	known := map[uint32]string{}
	for f, n := range weaponv3.KnownWeaponHigh32 {
		known[f] = n
	}
	fmt.Printf("catalogue de familles d arme (weaponv3) : %d entrees\n", len(known))

	n := filmdec.CountFilmChunks(dir)
	var t0 uint64
	first := true
	kfIdx := 0
	// cle : (ti, slot, gen)
	type key struct{ ti, slot, gen int }
	vies := map[key]*vie{}
	var ordre []key
	nRec := map[int]int{}
	nRecAvecFamille := map[int]int{}
	nOcc := map[int]int{}

	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if first {
				t0, first = pk.TimestampUS, false
			}
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kfIdx++
			t := float64(pk.TimestampUS-t0) / 1e6
			pay := pk.Payload(data)
			recs := filmdec.WalkKeyframeWorld(pay)
			sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
			starts := make([]int, len(recs))
			for i, r := range recs {
				starts[i] = r.Bit
			}
			// balayage des familles, attribuees au record qui les CONTIENT
			famParRec := map[int][]uint32{}
			total := len(pay) * 8
			var w uint32
			for b := 0; b < total; b++ {
				w = w<<1 | uint32(bitAt(pay, b))
				if b < 31 {
					continue
				}
				if _, ok := known[w]; !ok {
					continue
				}
				ri := recordContaining(starts, b-31)
				if ri < 0 {
					continue
				}
				famParRec[ri] = append(famParRec[ri], w)
			}
			for ri, r := range recs {
				if r.TI != 42 && r.TI != 37 {
					continue
				}
				nRec[r.TI]++
				k := key{r.TI, r.Slot, r.Gen}
				v := vies[k]
				if v == nil {
					v = &vie{TI: r.TI, Slot: r.Slot, Gen: r.Gen, Apparition: t, Disparition: t}
					vies[k] = v
					ordre = append(ordre, k)
				}
				v.NbKeyframes++
				if t < v.Apparition {
					v.Apparition = t
				}
				if t > v.Disparition {
					v.Disparition = t
				}
				if fams := famParRec[ri]; len(fams) > 0 {
					nRecAvecFamille[r.TI]++
					nOcc[r.TI] += len(fams)
					for _, f := range fams {
						v.Familles = ajout(v.Familles, fmt.Sprintf("%08x", f))
						v.Libelles = ajout(v.Libelles, known[f])
					}
				}
			}
		}
	}
	_ = obs{}

	fmt.Printf("%d keyframes · vies distinctes (ti,slot,gen) : %d\n", kfIdx, len(ordre))
	for _, ti := range []int{42, 37} {
		fmt.Printf("  ti=%d : %d records cumules · %d portant au moins une famille (%.1f %%) · %d occurrences\n",
			ti, nRec[ti], nRecAvecFamille[ti], 100*float64(nRecAvecFamille[ti])/float64(max1(nRec[ti])), nOcc[ti])
	}

	sort.Slice(ordre, func(i, j int) bool {
		a, b := vies[ordre[i]], vies[ordre[j]]
		if a.Apparition != b.Apparition {
			return a.Apparition < b.Apparition
		}
		return a.Slot < b.Slot
	})

	var liste []vie
	for _, k := range ordre {
		liste = append(liste, *vies[k])
	}

	// --- bande de slots : les entites presentes des le PREMIER keyframe ---
	fmt.Println("\n=== ti=42 : vies portant un libelle ===")
	nommes := 0
	for _, v := range liste {
		if v.TI != 42 || len(v.Libelles) == 0 {
			continue
		}
		nommes++
		fmt.Printf("  slot %5d gen %d  t %6.1f..%6.1f s (%2d kf)  %v\n",
			v.Slot, v.Gen, v.Apparition, v.Disparition, v.NbKeyframes, v.Libelles)
	}
	fmt.Printf("  -> %d vies ti=42 nommees\n", nommes)

	fmt.Println("\n=== ti=37 : vies portant un libelle ===")
	n37 := 0
	for _, v := range liste {
		if v.TI != 37 || len(v.Libelles) == 0 {
			continue
		}
		n37++
		fmt.Printf("  slot %5d gen %d  t %6.1f..%6.1f s (%2d kf)  %v\n",
			v.Slot, v.Gen, v.Apparition, v.Disparition, v.NbKeyframes, v.Libelles)
	}
	fmt.Printf("  -> %d vies ti=37 nommees\n", n37)

	// --- recurrence temporelle par libelle (substitut au discriminant spatial) ---
	fmt.Println("\n=== recurrence temporelle par libelle (ti=42) ===")
	parLib := map[string][]float64{}
	for _, v := range liste {
		if v.TI != 42 {
			continue
		}
		for _, l := range v.Libelles {
			parLib[l] = append(parLib[l], v.Apparition)
		}
	}
	libs := make([]string, 0, len(parLib))
	for l := range parLib {
		libs = append(libs, l)
	}
	sort.Slice(libs, func(i, j int) bool { return len(parLib[libs[i]]) > len(parLib[libs[j]]) })
	for _, l := range libs {
		ts := parLib[l]
		sort.Float64s(ts)
		var ecarts []float64
		for i := 1; i < len(ts); i++ {
			ecarts = append(ecarts, ts[i]-ts[i-1])
		}
		med := 0.0
		if len(ecarts) > 0 {
			e := append([]float64{}, ecarts...)
			sort.Float64s(e)
			med = e[len(e)/2]
		}
		fmt.Printf("  %-22s %3d apparitions · ecart median %6.1f s · instants %v\n",
			l, len(ts), med, arrondi(ts))
	}

	if out != "" {
		doc := map[string]any{
			"bloc": "objets_au_sol",
			"film": filmID(dir),
			"vies": liste,
		}
		blob, _ := json.MarshalIndent(doc, "", " ")
		_ = os.WriteFile(out, blob, 0o644)
		fmt.Printf("\necrit : %s (%d octets)\n", out, len(blob))
	}
}

func max1(n int) int {
	if n == 0 {
		return 1
	}
	return n
}

func arrondi(v []float64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(int(x*10)) / 10
	}
	return out
}

func ajout(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func bitAt(b []byte, p int) byte {
	if idx := p >> 3; idx < len(b) {
		return (b[idx] >> (7 - uint(p&7))) & 1
	}
	return 0
}

func recordContaining(starts []int, at int) int {
	lo, hi := 0, len(starts)
	for lo < hi {
		mid := (lo + hi) / 2
		if starts[mid] > at {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo - 1
}

func filmID(dir string) string {
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' || dir[i] == '\\' {
			return dir[i+1:]
		}
	}
	return dir
}
