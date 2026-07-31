// tmp_deployed — les EQUIPEMENTS DEPLOYES ont-ils une position lisible ?
//
// CE QUI MOTIVE LA SONDE (remarque utilisateur, 2026-07-26) : « y a d'autres equipements
// aussi, les traqueurs, repair equipement, shroud machin, qui se deploient au sol, donc lies
// a des coordonnees aussi ». Un objet pose au sol est une entite du monde : il doit avoir une
// position, comme un projectile.
//
// CE QUE LA SONDE TRANCHE, et il faut le dire avant de regarder les chiffres : un equipement
// DEPLOYE est IMMOBILE. Sa signature n'est donc pas la parabole d'un projectile, c'est
// exactement l'inverse -- une position CONSTANTE pendant plusieurs secondes. C'est un temoin
// aussi discriminant, et il ne peut pas etre confondu avec le precedent.
//
// TROIS CONTROLES, parce qu'un compte seul ne prouve rien :
//  1. IMMOBILITE : l'ecart-type de la position sur la vie doit etre quasi nul pour un
//     equipement, et grand pour un projectile. On mesure les deux, cote a cote.
//  2. AU SOL : la derniere position doit etre proche de la surface reconstruite. On se
//     contente ici de la distribution des altitudes, qui doit etre serree.
//  3. CONTROLE NEGATIF : le meme balayage sur un archetype qu'on sait mobile (ti=41) doit
//     rendre l'inverse sur le critere d'immobilite.
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

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "chunks du film")
	mapName := flag.String("map", "Cliffhanger", "carte (bornes de dequantification)")
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

	type arch struct {
		ti    int
		label string
	}
	for _, a := range []arch{
		{filmdec.EquipmentTypeIndex, "EQUIPEMENT (ti=37) — attendu IMMOBILE"},
		{filmdec.ProjectileTypeIndex, "PROJECTILE (ti=41) — controle negatif, attendu MOBILE"},
		{42, "ARME AU SOL (ti=42) — second controle, attendu immobile aussi"},
	} {
		tracks, err := filmdec.ScanFilmWorldObjects(*dir, &wr, a.ti)
		fmt.Printf("\n=== %s ===\n", a.label)
		if err != nil {
			fmt.Println("   ", err)
			continue
		}
		report(tracks)
	}
}

// report mesure l'immobilite et l'altitude d'un ensemble de vies.
func report(tracks []filmdec.ProjectileTrack) {
	if len(tracks) == 0 {
		fmt.Println("    aucune vie")
		return
	}
	var spreads, durs, zs []float64
	still := 0
	for _, t := range tracks {
		var sx, sy, sxx, syy float64
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, p := range t.Pts {
			x, y := float64(p.X), float64(p.Y)
			sx += x
			sy += y
			sxx += x * x
			syy += y * y
			if float64(p.Z) < lo {
				lo = float64(p.Z)
			}
			if float64(p.Z) > hi {
				hi = float64(p.Z)
			}
		}
		n := float64(len(t.Pts))
		vx, vy := sxx/n-(sx/n)*(sx/n), syy/n-(sy/n)*(sy/n)
		sd := math.Sqrt(math.Max(0, vx) + math.Max(0, vy))
		spreads = append(spreads, sd)
		durs = append(durs, float64(t.Pts[len(t.Pts)-1].TimestampUS-t.Pts[0].TimestampUS)/1e6)
		zs = append(zs, float64(t.Pts[len(t.Pts)-1].Z))
		_ = lo
		_ = hi
		if sd < 0.25 { // un objet pose ne derive pas de plus de 25 cm
			still++
		}
	}
	fmt.Printf("    %d vies · duree mediane %.2f s\n", len(tracks), median(durs))
	fmt.Printf("    dispersion XY : mediane %.2f m, 90e centile %.2f m\n",
		median(spreads), pct(spreads, 0.90))
	fmt.Printf("    IMMOBILES (dispersion < 25 cm) : %d/%d = %.1f %%\n",
		still, len(tracks), 100*float64(still)/float64(len(tracks)))
	fmt.Printf("    altitude de la derniere position : mediane %.2f m, 10e-90e [%.2f ; %.2f]\n",
		median(zs), pct(zs, 0.10), pct(zs, 0.90))
}

func median(v []float64) float64 { return pct(v, 0.5) }

func pct(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(q * float64(len(s)-1))
	return s[i]
}
