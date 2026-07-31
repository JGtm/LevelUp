// tmp_grapple — un joueur est-il TRACTE vers la position d'un equipement a courte vie ?
//
// HYPOTHESE (utilisateur, 2026-07-26) : « qu'est-ce qui peut etre un equipement qui a une
// localisation si courte, un grappin peut-etre ? Est-ce qu'on a un joueur qui suit la
// trajectoire vers ces coordonnees ? »
//
// Elle explique l'anomalie mesuree la veille : les vies de l'archetype equipement (ti=37) ont
// une duree mediane de 1,18 s, absurde pour un mur deployable qui tient une quinzaine de
// secondes -- mais exactement la duree d'un tir de grappin.
//
// LE DISCRIMINANT N'EST PAS LA CONVERGENCE, C'EST LA VITESSE. Un joueur marche aussi vers une
// arme au sol pour la ramasser : la convergence seule ne separerait rien. Mais un Spartan
// plafonne a ~2,75-3,00 m/s a pied (mesure independante sur ce corpus), alors qu'un grappin
// TRACTE. C'est donc la vitesse d'approche qu'il faut mesurer, et elle doit sortir de la plage
// pedestre.
//
// TROIS CONTROLES NEGATIFS, sans lesquels le chiffre ne vaut rien :
//  1. instants PERMUTES : les memes positions d'equipement, a des instants tires au hasard.
//  2. archetype TEMOIN ti=42 (armes au sol) : on s'attend a de la convergence LENTE, donc a
//     un taux de traction bas. C'est le controle le plus severe, car il partage la
//     convergence avec l'hypothese et n'en differe QUE par la vitesse.
//  3. taux de base : quelle fraction du temps un joueur depasse le plafond pedestre, tous
//     instants confondus ? Si c'est deja 40 %, le critere ne discrimine rien.
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

// walkCapMS est le plafond de vitesse pedestre mesure sur ce corpus (le pic s'effondre
// au-dela de 3,00 m/s). Au-dessus, le joueur n'avance pas par ses jambes.
const walkCapMS = 3.0

// approachWindowUS est la fenetre autour de la fin de vie de l'equipement dans laquelle on
// cherche l'approche : de 0,5 s avant a 1,5 s apres.
const (
	windowBeforeUS = 500_000
	windowAfterUS  = 1_500_000
	// nearMeters : distance en deca de laquelle on considere que le joueur a REJOINT le point.
	nearMeters = 2.5
)

type sample struct {
	t       uint64
	x, y, z float32
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
		if !p.HasWorld {
			continue
		}
		bySlot[p.Slot] = append(bySlot[p.Slot], sample{p.TimestampUS, p.X, p.Y, p.Z})
	}
	for s := range bySlot {
		v := bySlot[s]
		sort.Slice(v, func(i, j int) bool { return v[i].t < v[j].t })
		bySlot[s] = v
	}
	fmt.Printf("  %d positions de joueur sur %d slots\n", len(pos), len(bySlot))
	fmt.Printf("  TAUX DE BASE : %.1f %% du temps un joueur depasse %.1f m/s\n\n",
		100*baseRate(bySlot), walkCapMS)

	for _, a := range []struct {
		ti    int
		label string
	}{
		{filmdec.EquipmentTypeIndex, "EQUIPEMENT ti=37 — l'hypothese du grappin"},
		{42, "ARME AU SOL ti=42 — controle le plus severe (convergence LENTE attendue)"},
		{filmdec.ProjectileTypeIndex, "PROJECTILE ti=41 — second controle"},
	} {
		tracks, err := filmdec.ScanFilmWorldObjects(*dir, &wr, a.ti)
		fmt.Printf("=== %s ===\n", a.label)
		if err != nil {
			fmt.Println("   ", err)
			continue
		}
		measure(tracks, bySlot)
		fmt.Println()
	}
}

// measure : pour chaque vie, cherche le joueur qui s'approche le plus du point final, et
// mesure sa vitesse d'approche. Puis rejoue la mesure avec les instants permutes.
func measure(tracks []filmdec.ProjectileTrack, bySlot map[uint32][]sample) {
	var reach, fast int
	var speeds []float64
	for _, t := range tracks {
		last := t.Pts[len(t.Pts)-1]
		d, v := bestApproach(bySlot, last.TimestampUS, last.X, last.Y, last.Z)
		if d <= nearMeters {
			reach++
			speeds = append(speeds, v)
			if v > walkCapMS {
				fast++
			}
		}
	}
	// Controle : memes points, instants DECALES en bloc (7 s), ce qui casse le lien temporel
	// sans toucher a la geometrie -- un joueur passe toujours au meme endroit tot ou tard.
	var reachC, fastC int
	for _, t := range tracks {
		last := t.Pts[len(t.Pts)-1]
		d, v := bestApproach(bySlot, last.TimestampUS+7_000_000, last.X, last.Y, last.Z)
		if d <= nearMeters {
			reachC++
			if v > walkCapMS {
				fastC++
			}
		}
	}
	fmt.Printf("    %d vies\n", len(tracks))
	fmt.Printf("    un joueur A MOINS DE %.1f m du point final : %d (%.1f %%)   [temoin decale : %d, %.1f %%]\n",
		nearMeters, reach, pctOf(reach, len(tracks)), reachC, pctOf(reachC, len(tracks)))
	if reach > 0 {
		fmt.Printf("    parmi eux, approche PLUS VITE que la marche (%.1f m/s) : %d (%.1f %%)   [temoin : %.1f %%]\n",
			walkCapMS, fast, pctOf(fast, reach), pctOf(fastC, maxInt(reachC, 1)))
		sort.Float64s(speeds)
		fmt.Printf("    vitesse d'approche : mediane %.2f m/s, 90e centile %.2f m/s, max %.2f m/s\n",
			speeds[len(speeds)/2], speeds[int(0.9*float64(len(speeds)-1))], speeds[len(speeds)-1])
	}
}

// bestApproach rend, sur tous les slots, la distance minimale atteinte dans la fenetre et la
// vitesse a laquelle le joueur s'en est approche.
func bestApproach(bySlot map[uint32][]sample, at uint64, x, y, z float32) (float64, float64) {
	bestD, bestV := math.Inf(1), 0.0
	lo := int64(at) - windowBeforeUS
	hi := int64(at) + windowAfterUS
	for _, sm := range bySlot {
		var prev *sample
		for i := range sm {
			ts := int64(sm[i].t)
			if ts < lo {
				prev = &sm[i]
				continue
			}
			if ts > hi {
				break
			}
			d := dist(sm[i], x, y, z)
			if d < bestD {
				bestD = d
				bestV = 0
				if prev != nil && sm[i].t > prev.t {
					dt := float64(sm[i].t-prev.t) / 1e6
					// vitesse du joueur sur le dernier pas avant le point le plus proche
					bestV = dist3(*prev, sm[i]) / dt
				}
			}
			prev = &sm[i]
		}
	}
	return bestD, bestV
}

func dist(s sample, x, y, z float32) float64 {
	dx, dy, dz := float64(s.x-x), float64(s.y-y), float64(s.z-z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func dist3(a, b sample) float64 {
	dx, dy, dz := float64(a.x-b.x), float64(a.y-b.y), float64(a.z-b.z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// baseRate : fraction des pas de position ou le joueur depasse le plafond pedestre. C'est le
// niveau du hasard du critere de vitesse -- sans lui, un taux de traction eleve ne prouve rien.
func baseRate(bySlot map[uint32][]sample) float64 {
	n, fast := 0, 0
	for _, sm := range bySlot {
		for i := 1; i < len(sm); i++ {
			dt := float64(sm[i].t-sm[i-1].t) / 1e6
			if dt <= 0 || dt > 0.5 {
				continue
			}
			n++
			if dist3(sm[i-1], sm[i])/dt > walkCapMS {
				fast++
			}
		}
	}
	if n == 0 {
		return 0
	}
	return float64(fast) / float64(n)
}

func pctOf(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
