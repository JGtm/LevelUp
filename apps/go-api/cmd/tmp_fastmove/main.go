// tmp_fastmove — OU un joueur va-t-il plus vite que ses jambes ne le permettent ?
//
// METHODE (redirection utilisateur, 2026-07-26) : « une piste pour le grappin c'est de voir ou
// un joueur se deplace plus vite que prevu (hors canon humain) ».
//
// C'est l'inverse de la sonde precedente, et c'est mieux pose : au lieu de partir d'entites
// d'equipement dont la BANDE DE SLOTS est incertaine, on part du MOUVEMENT, que nos 171 826
// positions decrivent bien. Le signal est cherche la ou la donnee est solide.
//
// TROIS CAUSES CONNUES font depasser le plafond pedestre. Il faut les separer, sinon on compte
// des chutes pour des grappins :
//
//	CANON HUMAIN  depart depuis un point FIXE de la carte, connu par recurrence spatiale
//	              (14.584, 20.141, -0.468) -- 9 lancements, 13 cm de dispersion.
//	CHUTE LIBRE   acceleration verticale constante NEGATIVE, deplacement horizontal ~constant.
//	GRAPPIN       traction vers un point : trajectoire quasi RECTILIGNE, vitesse qui MONTE puis
//	              s'arrete net a l'arrivee, et une composante verticale qui peut etre POSITIVE
//	              (on se hisse) sans etre balistique.
//
// LE DISCRIMINANT RETENU est la RECTITUDE : rapport entre la distance parcourue et la distance
// a vol d'oiseau sur l'episode. Une chute ou un lancement decrivent une parabole (rapport > 1) ;
// une traction tire droit (rapport ~ 1). C'est mesurable sans rien supposer du jeu.
//
// CONTROLE NEGATIF : la meme mesure sur des episodes LENTS tires au hasard, qui doivent donner
// une distribution de rectitude differente -- sinon le critere ne separe rien.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	walkCapMS   = 3.0  // plafond pedestre mesure sur ce corpus
	minEpisodeN = 3    // au moins 3 pas consecutifs : un pic isole est du bruit de quantification
	cannonR     = 4.0  // rayon d'exclusion autour du canon humain, en metres
	maxStepS    = 0.35 // au-dela, deux echantillons ne sont pas consecutifs
)

var cannon = [3]float64{14.584, 20.141, -0.468}

type sample struct {
	t       uint64
	x, y, z float32
}

type episode struct {
	slot           uint32
	t0, t1         uint64
	n              int
	peak, mean     float64
	pathLen, chord float64
	dz             float64
	fromCannon     bool
	startX, startY float64
	endX, endY     float64
}

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "chunks du film")
	mapName := flag.String("map", "Cliffhanger", "carte")
	repo := flag.String("repo", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\.claude\worktrees\filmdec-continuation`, "racine du depot")
	flag.Parse()

	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(*repo,
		"data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalogue:", err)
		os.Exit(1)
	}
	e, err := cat.Lookup(*mapName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "carte:", err)
		os.Exit(1)
	}
	wr := e.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &wr
	pos, err := filmdec.ScanFilmBipedPositions(*dir, scan)
	if err != nil {
		fmt.Fprintln(os.Stderr, "positions:", err)
		os.Exit(1)
	}
	bySlot := map[uint32][]sample{}
	for _, p := range pos {
		if p.HasWorld {
			bySlot[p.Slot] = append(bySlot[p.Slot], sample{p.TimestampUS, p.X, p.Y, p.Z})
		}
	}
	for s := range bySlot {
		v := bySlot[s]
		sort.Slice(v, func(i, j int) bool { return v[i].t < v[j].t })
		bySlot[s] = v
	}

	// DISTRIBUTION DES VITESSES — avant de choisir un seuil, le mesurer. Un "plafond" que
	// 24 % des pas depassent n'est pas un plafond.
	var all []float64
	for _, sm := range bySlot {
		for i := 1; i < len(sm); i++ {
			dt := float64(sm[i].t-sm[i-1].t) / 1e6
			if dt > 0 && dt <= maxStepS {
				all = append(all, dist(sm[i-1], sm[i])/dt)
			}
		}
	}
	sort.Float64s(all)
	fmt.Println("  DISTRIBUTION DES VITESSES DE PAS (m/s) :")
	for _, q := range []float64{0.50, 0.75, 0.90, 0.95, 0.99, 0.999, 0.9999} {
		fmt.Printf("    %6.2f%% : %6.2f\n", 100*q, all[int(q*float64(len(all)-1))])
	}
	fmt.Printf("    maximum : %.2f   (n = %d pas)\n\n", all[len(all)-1], len(all))

	eps, slow := collect(bySlot)
	fmt.Printf("EPISODES RAPIDES — %d positions, %d slots\n\n", len(pos), len(bySlot))
	fmt.Printf("  %d episodes au-dessus de %.1f m/s sur >= %d pas consecutifs\n", len(eps), walkCapMS, minEpisodeN)

	var fromCannon int
	for _, x := range eps {
		if x.fromCannon {
			fromCannon++
		}
	}
	fmt.Printf("  dont %d partent du canon humain (rayon %.0f m) -> ECARTES\n", fromCannon, cannonR)

	rest := eps[:0]
	for _, x := range eps {
		if !x.fromCannon {
			rest = append(rest, x)
		}
	}
	fmt.Printf("  reste %d episodes rapides inexpliques\n\n", len(rest))

	// RECTITUDE : chemin / corde. 1.00 = ligne droite ; une parabole depasse nettement.
	str := make([]float64, 0, len(rest))
	for _, x := range rest {
		if x.chord > 0.5 {
			str = append(str, x.pathLen/x.chord)
		}
	}
	slowStr := make([]float64, 0, len(slow))
	for _, x := range slow {
		if x.chord > 0.5 {
			slowStr = append(slowStr, x.pathLen/x.chord)
		}
	}
	fmt.Println("  RECTITUDE (chemin / corde ; 1.00 = ligne droite)")
	fmt.Printf("    episodes RAPIDES : mediane %.3f  (n=%d)\n", median(str), len(str))
	fmt.Printf("    CONTROLE, episodes LENTS : mediane %.3f  (n=%d)\n", median(slowStr), len(slowStr))
	fmt.Printf("    part des rapides quasi RECTILIGNES (< 1,05) : %.1f %%   [controle lent : %.1f %%]\n",
		fracBelow(str, 1.05), fracBelow(slowStr, 1.05))

	fmt.Println("\n  MONTEE (dz > +1 m sur l'episode) — une chute ne monte pas")
	up := 0
	for _, x := range rest {
		if x.dz > 1.0 {
			up++
		}
	}
	upSlow := 0
	for _, x := range slow {
		if x.dz > 1.0 {
			upSlow++
		}
	}
	fmt.Printf("    rapides qui MONTENT : %d/%d = %.1f %%   [controle lent : %.1f %%]\n",
		up, len(rest), pctOf(up, len(rest)), pctOf(upSlow, len(slow)))

	fmt.Println("\n  CANDIDATS GRAPPIN (rapides, rectilignes < 1,05, qui montent de plus d'1 m) :")
	n := 0
	for _, x := range rest {
		if x.chord <= 0.5 || x.pathLen/x.chord >= 1.05 || x.dz <= 1.0 {
			continue
		}
		n++
		if n <= 15 {
			fmt.Printf("    slot %3d  t=%6.1fs  %4.2fs  pic %5.2f m/s  dz %+5.2f m  (%.1f,%.1f) -> (%.1f,%.1f)\n",
				x.slot, float64(x.t0)/1e6, float64(x.t1-x.t0)/1e6, x.peak, x.dz,
				x.startX, x.startY, x.endX, x.endY)
		}
	}
	fmt.Printf("    TOTAL : %d candidats\n", n)
}

// collect extrait les episodes rapides, et un echantillon d'episodes LENTS de meme longueur
// pour servir de controle a la mesure de rectitude.
func collect(bySlot map[uint32][]sample) (fast, slow []episode) {
	for slot, sm := range bySlot {
		i := 0
		for i < len(sm)-1 {
			dt := float64(sm[i+1].t-sm[i].t) / 1e6
			if dt <= 0 || dt > maxStepS {
				i++
				continue
			}
			v := dist(sm[i], sm[i+1]) / dt
			j := i
			isFast := v > walkCapMS
			var path, peak, sumv float64
			cnt := 0
			for j < len(sm)-1 {
				d2 := float64(sm[j+1].t-sm[j].t) / 1e6
				if d2 <= 0 || d2 > maxStepS {
					break
				}
				vv := dist(sm[j], sm[j+1]) / d2
				if (vv > walkCapMS) != isFast {
					break
				}
				path += dist(sm[j], sm[j+1])
				sumv += vv
				if vv > peak {
					peak = vv
				}
				cnt++
				j++
			}
			if cnt >= minEpisodeN {
				a, b := sm[i], sm[j]
				ep := episode{
					slot: slot, t0: a.t, t1: b.t, n: cnt, peak: peak, mean: sumv / float64(cnt),
					pathLen: path, chord: dist(a, b), dz: float64(b.z - a.z),
					startX: float64(a.x), startY: float64(a.y),
					endX: float64(b.x), endY: float64(b.y),
				}
				ep.fromCannon = math.Hypot(math.Hypot(float64(a.x)-cannon[0], float64(a.y)-cannon[1]),
					float64(a.z)-cannon[2]) < cannonR
				if isFast {
					fast = append(fast, ep)
				} else if len(slow) < 4000 {
					slow = append(slow, ep)
				}
			}
			if j == i {
				j++
			}
			i = j
		}
	}
	return fast, slow
}

func dist(a, b sample) float64 {
	dx, dy, dz := float64(a.x-b.x), float64(a.y-b.y), float64(a.z-b.z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func fracBelow(v []float64, x float64) float64 {
	if len(v) == 0 {
		return 0
	}
	n := 0
	for _, a := range v {
		if a < x {
			n++
		}
	}
	return 100 * float64(n) / float64(len(v))
}

func pctOf(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
