// cmd/mappos-build — construit le catalogue des POSITIONS REELLEMENT JOUEES par carte, a
// partir des artefacts de rejeu deja decodes (data/cache/replays/{title}/{matchId}.json), et
// l'ecrit dans data/titles/{slug}/reference/map_positions_jouees.json (PathResolver).
//
// A QUOI CA SERT : `himap.RogneAuxPositionsJouees` efface la matiere loin de tout endroit ou un
// joueur s'est reellement tenu. C'est le temoin le plus solide du chantier des fonds de carte —
// une position courue ne se deduit pas, elle s'observe, et elle prouve qu'il y avait du sol.
//
// DECIMATION : les positions brutes sont massivement redondantes (366 768 points sur 13 matchs
// de Dredge pour 1 008 cellules d'un metre distinctes). On ne garde qu'un point par cellule de
// `--pas` metres : le fichier reste petit et le masque est identique, la cuisson dilatant de
// toute facon d'un rayon bien superieur au pas.
//
// FILTRE DE RARETE — mesure du 2026-08-30 sur Dredge, et c est ce qui rend le masque
// utilisable. Sans filtre, le nuage des positions s etend jusqu a 268 m du centre de l arene ;
// en n exigeant que DEUX matchs distincts par cellule, il retombe a 27 m, et a TROIS a 19,4 m —
// soit le 99e centile du nuage lui-meme. Ce que la mesure dit : quelques positions isolees,
// vues une seule fois dans un seul match, tirent des BRAS hors de l arene (verdict utilisateur :
// « en haut en bas et a gauche et a droite on a comme des bras, 8 en tout »). Une cellule
// traversee dans plusieurs matchs differents, elle, est du terrain.
//
// On compte les MATCHS DISTINCTS plutot que les occurrences brutes : un joueur immobile gonfle
// le compte d une cellule sans rien prouver de plus, alors que deux matchs differents sont deux
// observations independantes.
//
// Usage :
//
//	mappos-build --cle <mapId> [--carte <nom>] [--title slug] [--pas M]
//	             [--min-matchs N] [--min-occurrences N] <rejeu.json>...
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/domain/title"
)

// pisteRejeu : la seule part de l'artefact de rejeu que ce programme lit.
type pisteRejeu struct {
	Points []struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"points"`
}

type documentRejeu struct {
	MatchID string       `json:"matchId"`
	Tracks  []pisteRejeu `json:"tracks"`
}

// EntreeCarte : les positions retenues pour une carte, et de quoi les auditer.
type EntreeCarte struct {
	Carte     string       `json:"carte,omitempty"`
	Matchs    []string     `json:"matchs"`
	PasM      float64      `json:"pasM"`
	MinMatchs int          `json:"minMatchs,omitempty"`
	MinOccur  int          `json:"minOccurrences,omitempty"`
	Ecartees  int          `json:"cellulesEcartees,omitempty"`
	Brutes    int          `json:"positionsBrutes"`
	Positions [][2]float64 `json:"positions"`
}

// Catalogue : le fichier versionne.
type Catalogue struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Source        string                 `json:"source"`
	Maps          map[string]EntreeCarte `json:"maps"`
}

func main() {
	var cle, carte, slug, sortie string
	var pas float64
	var minMatchs, minOccur int
	flag.StringVar(&cle, "cle", "", "identifiant de carte (cle du catalogue de fonds)")
	flag.StringVar(&carte, "carte", "", "nom affiche de la carte, pour l audit")
	flag.StringVar(&slug, "title", "halo_infinite", "slug du titre")
	flag.StringVar(&sortie, "out", "", "fichier de sortie ; vide = chemin du PathResolver")
	flag.Float64Var(&pas, "pas", 0.5, "pas de decimation, en metres")
	flag.IntVar(&minMatchs, "min-matchs", 1, "nombre minimal de matchs distincts par cellule")
	flag.IntVar(&minOccur, "min-occurrences", 1, "nombre minimal de passages par cellule")
	flag.Parse()

	if cle == "" || flag.NArg() == 0 {
		slog.Error("mappos: --cle et au moins un fichier de rejeu sont obligatoires")
		os.Exit(2)
	}
	if pas <= 0 {
		slog.Error("mappos: le pas de decimation doit etre strictement positif", "pas", pas)
		os.Exit(2)
	}

	entree, err := agrege(flag.Args(), pas, minMatchs, minOccur)
	if err != nil {
		slog.Error("mappos: agregation impossible", "err", err)
		os.Exit(1)
	}
	entree.Carte = carte

	chemin := sortie
	if chemin == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			slog.Error("mappos: racine du depot introuvable", "err", err)
			os.Exit(1)
		}
		chemin = title.NewPathResolver(root).MapPlayedPositionsPath(slug)
	}
	if err := ecrit(chemin, cle, entree); err != nil {
		slog.Error("mappos: ecriture impossible", "err", err, "path", chemin)
		os.Exit(1)
	}
	slog.Info("mappos: catalogue ecrit", "path", chemin, "carte", cle,
		"matchs", len(entree.Matchs), "brutes", entree.Brutes, "retenues", len(entree.Positions),
		"ecartees", entree.Ecartees, "minMatchs", minMatchs, "minOccurrences", minOccur)
}

// agrege lit les rejeux et decime les positions sur une grille de `pas` metres.
func agrege(fichiers []string, pas float64, minMatchs, minOccur int) (EntreeCarte, error) {
	e := EntreeCarte{PasM: pas, MinMatchs: minMatchs, MinOccur: minOccur}
	type compte struct {
		occurrences int
		matchs      map[string]bool
	}
	vues := map[[2]int]*compte{}
	for _, f := range fichiers {
		brut, err := os.ReadFile(f)
		if err != nil {
			return e, fmt.Errorf("lecture %s : %w", f, err)
		}
		var doc documentRejeu
		if err := json.Unmarshal(brut, &doc); err != nil {
			return e, fmt.Errorf("analyse %s : %w", f, err)
		}
		e.Matchs = append(e.Matchs, doc.MatchID)
		for _, t := range doc.Tracks {
			for _, p := range t.Points {
				if math.IsNaN(p.X) || math.IsNaN(p.Y) {
					continue
				}
				e.Brutes++
				k := [2]int{int(math.Round(p.X / pas)), int(math.Round(p.Y / pas))}
				c := vues[k]
				if c == nil {
					c = &compte{matchs: map[string]bool{}}
					vues[k] = c
				}
				c.occurrences++
				c.matchs[doc.MatchID] = true
			}
		}
	}
	cles := make([][2]int, 0, len(vues))
	for k, c := range vues {
		if c.occurrences < minOccur || len(c.matchs) < minMatchs {
			e.Ecartees++
			continue
		}
		cles = append(cles, k)
	}
	// Tri stable : le fichier doit etre identique d'une execution a l'autre, sinon le diff
	// versionne devient illisible et le catalogue bouge sans que rien n'ait change.
	sort.Slice(cles, func(i, j int) bool {
		if cles[i][0] != cles[j][0] {
			return cles[i][0] < cles[j][0]
		}
		return cles[i][1] < cles[j][1]
	})
	e.Positions = make([][2]float64, 0, len(cles))
	for _, k := range cles {
		e.Positions = append(e.Positions, [2]float64{float64(k[0]) * pas, float64(k[1]) * pas})
	}
	sort.Strings(e.Matchs)
	return e, nil
}

// ecrit fusionne l'entree dans le catalogue existant, sans toucher aux autres cartes.
func ecrit(chemin, cle string, e EntreeCarte) error {
	cat := Catalogue{SchemaVersion: 1, Maps: map[string]EntreeCarte{}}
	cat.Source = "cmd/mappos-build — positions des joueurs decodees des films (data/cache/replays)"
	if brut, err := os.ReadFile(chemin); err == nil {
		if err := json.Unmarshal(brut, &cat); err != nil {
			return fmt.Errorf("catalogue existant illisible : %w", err)
		}
		if cat.Maps == nil {
			cat.Maps = map[string]EntreeCarte{}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	cat.Maps[cle] = e
	brut, err := json.MarshalIndent(cat, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(chemin, append(brut, '\n'), 0o644)
}
