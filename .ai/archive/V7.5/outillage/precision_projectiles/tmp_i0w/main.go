// Commande tmp_i0w — TEST T2 : les LARGEURS D AXE de `object-position-component` (`ti=41`).
//
// La grammaire est etablie (note §2.2) mais ses largeurs viennent d une globale de runtime :
// elles ne sont pas derivables au desassembleur. Elles se MESURENT, par la methode deja
// eprouvee sur l autre i0 (`DetectI0Layout`) : pour un champ quantifie de largeur W dont la
// valeur bouge peu d une frame a la suivante, le TAUX DE BASCULE par position de bit vaut ~50 %
// sur le LSB et DOUBLE du MSB vers le LSB. Le profil de trois champs contigus est une DENT DE
// SCIE — montee geometrique puis effondrement au MSB du champ suivant.
//
// Aucun a priori de largeur n entre dans la mesure. On lit les frontieres sur le profil.
//
// Population : les records `ti=41` du flux DELTA, apparies par SLOT dans l ordre du flux (deux
// records consecutifs du MEME projectile). Le composant i0 est localise par son `StartBit`, que
// `CompResult` porte deja — rien n est re-scanne.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	projectileTI = 41
	i0Name       = "object-position-component"
	window       = 64 // bits observes apres la porte
)

type sample struct {
	slot uint32
	gate int
	bits [window]byte
}

func main() {
	films := flag.String("films", "", "racine du cache de films (lecture seule)")
	limit := flag.Int("limit", 40, "nombre maximum de films")
	flag.Parse()
	if *films == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_i0w -films <dir> [-limit N]")
		os.Exit(2)
	}
	dirs := listFilms(*films, *limit)

	var byGate [2][]sample
	for _, d := range dirs {
		for _, s := range collect(filepath.Join(*films, d)) {
			byGate[s.gate] = append(byGate[s.gate], s)
		}
	}
	fmt.Printf("echantillons i0 de ti=41 : porte=0 -> %d   porte=1 -> %d\n\n",
		len(byGate[0]), len(byGate[1]))

	if len(byGate[1]) > 0 {
		runT3(byGate[1])
	}
	for g := 0; g < 2; g++ {
		fmt.Printf("=== PORTE = %d %s ===\n", g, map[int]string{0: "(plage de la CARTE)", 1: "(plage PAR DEFAUT, les 59 bits opaques)"}[g])
		profile(byGate[g])
		fmt.Println()
	}
}

func listFilms(root string, limit int) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cache illisible: %v\n", err)
		os.Exit(1)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// collect rend les echantillons i0 des records ti=41 d un film, DANS L ORDRE DU FLUX.
func collect(dir string) []sample {
	raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
	if err != nil {
		return nil
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		return nil
	}
	w := filmdec.NewWorld(reg)
	cfg := filmdec.DefaultFrameConfig()
	var out []sample
	for c := 1; c <= filmdec.CountFilmChunks(dir); c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			pay := p.Payload(chunk)
			if p.Type == filmdec.PacketTypeKeyframe {
				for _, r := range filmdec.WalkKeyframeWorld(pay) {
					w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
				}
				continue
			}
			if p.Type != filmdec.PacketTypeDelta || len(pay) < 2 {
				continue
			}
			recs, _ := filmdec.DecodeFrameInfer(pay, w, cfg)
			for _, r := range recs {
				if r.TypeIndex != projectileTI {
					continue
				}
				for _, comp := range r.Trace.Comps {
					if comp.Name != i0Name || comp.StartBit <= 0 {
						continue
					}
					s := sample{slot: r.Slot, gate: int(bitAt(pay, comp.StartBit))}
					for i := 0; i < window; i++ {
						s.bits[i] = bitAt(pay, comp.StartBit+1+i)
					}
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func bitAt(buf []byte, pos int) byte {
	if pos < 0 || pos>>3 >= len(buf) {
		return 0
	}
	return (buf[pos>>3] >> uint(7-(pos&7))) & 1
}

// profile calcule le taux de bascule par position de bit entre records CONSECUTIFS du MEME slot.
func profile(ss []sample) {
	if len(ss) < 2 {
		fmt.Println("  (pas assez d echantillons)")
		return
	}
	prev := map[uint32]*sample{}
	var pairs int
	var toggles [window]int
	for i := range ss {
		s := ss[i]
		if p, ok := prev[s.slot]; ok {
			pairs++
			for b := 0; b < window; b++ {
				if p.bits[b] != s.bits[b] {
					toggles[b]++
				}
			}
		}
		cp := s
		prev[s.slot] = &cp
	}
	if pairs == 0 {
		fmt.Println("  (aucune paire consecutive d un meme slot)")
		return
	}
	fmt.Printf("  paires (meme slot, consecutives) : %d\n", pairs)
	fmt.Print("  taux de bascule par position :\n")
	for b := 0; b < window; b++ {
		if b%16 == 0 {
			fmt.Printf("   b%02d ", b)
		}
		fmt.Printf("%5.2f ", float64(toggles[b])/float64(pairs))
		if b%16 == 15 {
			fmt.Println()
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
//  T3 — LE DISCRIMINANT PHYSIQUE, parce que le profil de bascule ne transfere pas
// ─────────────────────────────────────────────────────────────────────────────
//
// La methode `DetectI0Layout` suppose une valeur qui BOUGE PEU d une frame a la suivante :
// c est vrai d un bipede, c est FAUX d un projectile, qui traverse la carte. Mesure a l appui,
// le profil est plat entre 0.10 et 0.45 sans dent de scie lisible. Forcer une lecture dessus
// serait exactement le defaut que ce chantier s interdit (un balayage FABRIQUE des
// distributions credibles).
//
// LE DISCRIMINANT DE REMPLACEMENT EST PHYSIQUE, et il est bien plus fort : un projectile vole
// DROIT et a vitesse ~constante. Si le decoupage en largeurs est le bon, les positions
// successives d un meme projectile sont COLINEAIRES et REGULIEREMENT ESPACEES ; si le decoupage
// est faux, les bits d un axe polluent le suivant et le nuage n a aucune structure.
//
// Le test rend, par decoupage candidat, la part de projectiles dont la trajectoire est
// colineaire a une tolerance donnee, contre une NULLE (les memes echantillons, slots melanges).

// dequant rend la valeur monde d un entier quantifie sur w bits dans [lo,hi]
// (FUN_1406d7f18 : raw * (hi-lo)/2^w + lo + demi-pas).
func dequant(raw uint64, w int, lo, hi float64) float64 {
	scale := (hi - lo) / float64(uint64(1)<<uint(w))
	return float64(raw)*scale + lo + scale*0.5
}

func readW(bits *[window]byte, at, w int) uint64 {
	var v uint64
	for i := 0; i < w; i++ {
		if at+i < window {
			v = v<<1 | uint64(bits[at+i])
		} else {
			v <<= 1
		}
	}
	return v
}

// colinearite rend la part de trajectoires (>= 3 points) dont les points sont colineaires :
// on mesure la distance maximale des points intermediaires au segment [premier, dernier],
// rapportee a la longueur du segment. Une trajectoire droite rend un ratio proche de 0.
func colinearite(ss []sample, split [3]int, tol float64) (float64, int) {
	type pt struct{ x, y, z float64 }
	traj := map[uint32][]pt{}
	for i := range ss {
		b := ss[i].bits
		off := 0
		var p pt
		vals := [3]*float64{&p.x, &p.y, &p.z}
		for a := 0; a < 3; a++ {
			*vals[a] = dequant(readW(&b, off, split[a]), split[a], -100, 100)
			off += split[a]
		}
		traj[ss[i].slot] = append(traj[ss[i].slot], p)
	}
	var ok, tot int
	for _, ps := range traj {
		if len(ps) < 3 {
			continue
		}
		tot++
		ax, ay, az := ps[0].x, ps[0].y, ps[0].z
		bx, by, bz := ps[len(ps)-1].x, ps[len(ps)-1].y, ps[len(ps)-1].z
		dx, dy, dz := bx-ax, by-ay, bz-az
		seg := dx*dx + dy*dy + dz*dz
		if seg <= 1e-9 {
			continue
		}
		worst := 0.0
		for _, q := range ps[1 : len(ps)-1] {
			t := ((q.x-ax)*dx + (q.y-ay)*dy + (q.z-az)*dz) / seg
			px, py, pz := ax+t*dx-q.x, ay+t*dy-q.y, az+t*dz-q.z
			d2 := px*px + py*py + pz*pz
			if d2 > worst {
				worst = d2
			}
		}
		if worst/seg <= tol*tol {
			ok++
		}
	}
	if tot == 0 {
		return 0, 0
	}
	return float64(ok) / float64(tot), tot
}

// runT3 confronte les decoupages candidats de 57 bits (3 axes) au critere de colinearite,
// avec une NULLE qui melange les slots : si la nulle fait aussi bien, le decoupage ne lit rien.
func runT3(ss []sample) {
	fmt.Println("=== T3 — COLINEARITE DES TRAJECTOIRES (porte = 1, plage +-100) ===")
	fmt.Printf("%-14s %10s %10s %12s\n", "decoupage", "traj>=3", "colineaire", "nulle melangee")
	shuffled := make([]sample, len(ss))
	copy(shuffled, ss)
	for i := range shuffled { // permutation deterministe des slots
		shuffled[i].slot = ss[(i*7919+13)%len(ss)].slot
	}
	for _, split := range [][3]int{{19, 19, 19}, {18, 19, 20}, {20, 19, 18}, {17, 20, 20}, {13, 13, 14}} {
		if split[0]+split[1]+split[2] > window {
			continue
		}
		r, tot := colinearite(ss, split, 0.05)
		n, _ := colinearite(shuffled, split, 0.05)
		fmt.Printf("%2d/%2d/%2d      %10d %10.4f %12.4f\n", split[0], split[1], split[2], tot, r, n)
	}
	fmt.Println()
}
