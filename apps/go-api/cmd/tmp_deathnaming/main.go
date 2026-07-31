// cmd/tmp_deathnaming — ETAPE 2 du plan PLAN_REJEU_2D_FIABILISATION : mesurer le REPLI
// avant de le brancher.
//
// LE REPLI, ET POURQUOI CE N'EST PAS UN VOTE. Chaque vie de biped se termine par une mort.
// Le chunk highlight du film porte le fil des morts : une victime, datee, par son XUID.
// On nomme donc chaque vie par LA MORT QUI LA TERMINE — une jointure sur un fait, pas une
// election. C'est la branche que l'etape 1 a rendue necessaire : son echec est mesure et
// consigne au plan.
//
// CE QUI EST VERIFIE ICI PLUTOT QUE REPRIS. Les chiffres du plan (91 vies sur 99, ecart
// median 0,0 image) ont ete produits avec un CSV de killer_victim_pairs, donc AVEC LA BASE.
// Le plan affirme par ailleurs que le fil des morts est film-seul. Les deux peuvent etre
// vrais, mais ce n'est pas la meme mesure : on la refait ici sur le chunk highlight, qui
// est dans le film, et on publie l'ecart s'il y en a un.
//
// L'IDENTITE EST LE XUID. Le fil des morts nomme par xuid, pas par index — c'est la
// decision n°2 du plan (« un index est un ordre, jamais une identite »). Le pont vers
// l'index de tir se resout separement, par affectation globale sur huit valeurs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

func main() {
	// Deux racines DISTINCTES, et ce n'est pas un accident : le catalogue de bornes est
	// versionne, donc il vit dans le worktree ; le cache de films ne l'est pas et reste
	// dans l'arbre principal. Les confondre donne « catalogue illisible ».
	repo := flag.String("repo", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`,
		"racine portant data/cache/film_chunks")
	bounds := flag.String("bounds", "",
		"racine portant data/titles/.../map_quant_bounds.json (defaut : racine du depot courant)")
	match := flag.String("match", "000d5950", "identifiant du match")
	mapName := flag.String("map", "Cliffhanger", "carte du match (bornes de dequantification)")
	flag.Parse()

	filmDir := filepath.Join(*repo, "data", "cache", "film_chunks", *match)
	boundsRoot := *bounds
	if boundsRoot == "" {
		r, err := title.FindRepoRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, "racine du depot:", err)
			os.Exit(1)
		}
		boundsRoot = r
	}
	rng, err := worldRange(boundsRoot, *mapName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bornes de carte:", err)
		os.Exit(1)
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = rng
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(filmDir, opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "positions:", err)
		os.Exit(1)
	}
	fire, err := filmdec.ScanFilmFireEvents(filmDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "events de tir:", err)
		os.Exit(1)
	}
	deaths, err := loadFilmDeaths(filmDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fil des morts:", err)
		os.Exit(1)
	}
	fmt.Printf("=== ETAPE 2 — REPLI « nommer la vie par la mort qui la termine »\n")
	fmt.Printf("film %s · %d positions · %d events de tir · %d morts au fil (film-seul)\n\n",
		*match, len(pos), len(fire), len(deaths))

	lives := buildLives(pos)
	fmt.Printf("--- 1. LES VIES ---\n")
	fmt.Printf("vies construites (slot + continuite) : %d\n", len(lives))

	fmt.Printf("\n--- 2. CALAGE DE L'HORLOGE ---\n")
	off, n := bestOffset(lives, deaths)
	fmt.Printf("decalage retenu (centre du plateau) : %d ms -> %d morts appariables\n", off, n)

	fmt.Printf("\n--- 3. APPARIEMENT ET TEMOIN ---\n")
	named := matchDeaths(lives, deaths, off)
	ctrl := controlMatch(lives, deaths, off)
	fmt.Printf("vies nommees par la mort qui les termine : %d / %d (%.1f %%)\n",
		named, len(lives), pct(named, len(lives)))
	fmt.Printf("TEMOIN (memes morts, instants tires au hasard sur la meme plage) : %d\n", ctrl)
	reportLifeSpans(lives)

	fmt.Printf("\n--- 4. PONT INDEX DE TIR -> XUID ---\n")
	bridge := solveBridge(lives, fire)
	reportBridge(bridge)

	fmt.Printf("\n--- 5. COUVERTURE DU RATTACHEMENT ---\n")
	reportCoverage(lives, fire, bridge, filmDir)

	fmt.Printf("\n--- 6. CONTROLE PAR SOURCE DISJOINTE (loadout des images-cles) ---\n")
	reportLoadoutControl(lives, fire, bridge, filmDir)

	prodDeaths, err := replay.ScanFilmDeaths(filmDir)
	if err != nil {
		fmt.Println("fil des morts (chemin de production):", err)
		return
	}
	runPipeline(*match, filmDir, rng, prodDeaths)
}

// worldRange charge les bornes de dequantification de la carte.
func worldRange(repo, mapName string) (*filmdec.Vec3Range, error) {
	p := title.NewPathResolver(repo).MapQuantBoundsPath(title.DefaultSlug)
	cat, err := filmdec.LoadMapQuantCatalog(p)
	if err != nil {
		return nil, err
	}
	e, err := cat.Lookup(mapName)
	if err != nil {
		return nil, err
	}
	r := e.Range()
	return &r, nil
}

// loadFilmDeaths lit le fil des morts DANS LE FILM (chunk highlight), sans aucune base.
// Le chunk highlight est le dernier du manifest ; CountFilmChunks le compte comme les
// autres, et son parseur est deja du code de production (analysis.ParseHighlightEvents).
func loadFilmDeaths(filmDir string) ([]Death, error) {
	n := filmdec.CountFilmChunks(filmDir)
	if n == 0 {
		return nil, fmt.Errorf("aucun chunk dans %s", filmDir)
	}
	raw, err := os.ReadFile(filepath.Join(filmDir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		return nil, err
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		return nil, err
	}
	var out []Death
	for _, e := range evs {
		if e.EventType != analysis.EventTypeDeath {
			continue
		}
		out = append(out, Death{XUID: e.XUID, Gamertag: e.Gamertag, TimeMS: int64(e.TimeMS), LifeIndex: -1})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out, nil
}

// reportLifeSpans publie la distribution des durees de vie : un repli qui nomme des vies
// doit d'abord montrer que ses vies sont des vies, et pas des morceaux de trajectoire.
func reportLifeSpans(lives []Life) {
	if len(lives) == 0 {
		return
	}
	var d []float64
	bySlot := map[uint32]int{}
	for _, l := range lives {
		d = append(d, float64(l.EndUS-l.StartUS)/1e6)
		bySlot[l.Slot]++
	}
	sort.Float64s(d)
	multi := 0
	for _, n := range bySlot {
		if n > 1 {
			multi++
		}
	}
	fmt.Printf("slots distincts : %d ; slots portant PLUSIEURS vies : %d\n", len(bySlot), multi)
	// UN SLOT PEUT-IL CHANGER DE PORTEUR ? Le pont publie est une table slot -> joueur ;
	// si un slot portait successivement deux identites, cette table serait fausse par
	// construction, quelle que soit la qualite du nommage. C'est la seule hypothese du
	// repli qui n'a pas encore ete mesuree.
	xuidsOf := map[uint32]map[uint64]bool{}
	for _, l := range lives {
		if l.XUID == 0 {
			continue
		}
		if xuidsOf[l.Slot] == nil {
			xuidsOf[l.Slot] = map[uint64]bool{}
		}
		xuidsOf[l.Slot][l.XUID] = true
	}
	mixed := 0
	for _, m := range xuidsOf {
		if len(m) > 1 {
			mixed++
		}
	}
	fmt.Printf("slots dont les vies nommees portent PLUSIEURS identites : %d / %d\n",
		mixed, len(xuidsOf))
	fmt.Printf("duree de vie (s) : mediane %.1f · p10 %.1f · p90 %.1f · max %.1f\n",
		median(d), quantile(d, 0.10), quantile(d, 0.90), d[len(d)-1])
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func median(v []float64) float64 { return quantile(v, 0.5) }

func quantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	i := int(q * float64(len(v)-1))
	return v[i]
}
