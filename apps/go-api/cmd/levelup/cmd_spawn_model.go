package main

// cmd_spawn_model.go — sous-commande `levelup spawn-model` : la MESURE et l'AJUSTEMENT du
// modele de reapparition (analysis/tactical, reapparition*.go).
//
//	levelup spawn-model evaluate   # les mesures : par mode, mort contre reapparition
//	levelup spawn-model fit        # ecrit le fichier de poids versionne (reference/spawn_model.json)
//
// # CE QU'ELLE LIT
//
// Les ARTEFACTS de rejeu (`*.json.gz` du dossier d'artefacts du titre), un a la fois, par la
// meme projection que le sidecar de raster : aucun film, aucun decodage. La carte et la variante
// de chaque match viennent du registre de la base partagee, ouverte EN LECTURE par
// `OpenReadForQuery` ; si un autre processus la tient en ecriture, la commande le dit et
// s'arrete (elle ne force rien). Aucune base n'est ecrite, aucun sidecar n'est touche.
//
// # CE QUE LES MESURES ECARTENT, ET COMMENT ON LE VOIT
//
// Le bilan de lecture compte, par cause, les artefacts sans registre, hors familles, illisibles,
// les documents sans lecture du film (aucune reapparition), les reapparitions lues au film et celles
// sans equipe connue.

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// Sous-commandes de `spawn-model`.
const (
	sousCommandeEvaluate = "evaluate"
	sousCommandeFit      = "fit"
)

// Seuils de matchs par carte, selon ce que la sous-commande fait de la carte. `evaluate` mesure
// avec celui de la mesure de depart ; `fit` en prend un plus bas pour que les familles clairsemees
// (Bases, Roi de la colline) aient des cartes. `--matchs-min-carte` le remplace.
const (
	matchsMinCarteMesure     = 8
	matchsMinCarteAjustement = 3
)

// Sections de `evaluate`.
const (
	sectionParMode = "m3"
	sectionInstant = "m4"
	// sectionPoids : relit le fichier de poids tel que le service le lit, sans lire le parc.
	sectionPoids = "poids"
)

// spawnModelOptions : les reglages communs aux deux sous-commandes.
type spawnModelOptions struct {
	titleSlug string
	sections  string
	famille   string
	limite    int
	variantes bool
	// matchsMinCarte : une carte vue dans moins de matchs n'est pas mesuree ; 0 = le seuil de la
	// sous-commande (matchsMinCarteMesure ou matchsMinCarteAjustement).
	matchsMinCarte int
	// sortie : fit ecrit ici au lieu du fichier de reference ; obligatoire pour un ajustement partiel.
	sortie string
	// table range les variantes du titre dans leurs familles de modes.
	table *mappings.SpawnFamilySet
}

// lireOptionsSpawnModel lit les options d'une sous-commande (evaluate | fit).
func lireOptionsSpawnModel(sous string, args []string) (spawnModelOptions, error) {
	fs := flag.NewFlagSet("spawn-model "+sous, flag.ExitOnError)
	o := spawnModelOptions{}
	fs.StringVar(&o.titleSlug, "title", "halo_infinite", "slug du titre")
	fs.StringVar(&o.sections, "sections", "m3,m4", "evaluate : sections a produire (m3 par mode, m4 mort contre reapparition)")
	fs.StringVar(&o.famille, "famille", "", "restreint a une famille de modes (celles de spawn_families.toml)")
	fs.IntVar(&o.limite, "limit", 0, "borne le nombre d'artefacts lus (0 = tous)")
	fs.BoolVar(&o.variantes, "variantes", false, "liste les variantes du parc par famille puis s'arrete")
	fs.IntVar(&o.matchsMinCarte, "matchs-min-carte", 0,
		fmt.Sprintf("matchs minimum par carte pour qu'elle soit retenue (defaut : %d pour evaluate, %d pour fit)",
			matchsMinCarteMesure, matchsMinCarteAjustement))
	fs.StringVar(&o.sortie, "sortie", "", "fit : fichier de poids a ecrire (obligatoire avec --famille ou --limit)")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if o.matchsMinCarte == 0 {
		o.matchsMinCarte = matchsMinCarteMesure
		if sous == sousCommandeFit {
			o.matchsMinCarte = matchsMinCarteAjustement
		}
	}
	return o, nil
}

func runSpawnModel(cfg *config.AppConfig, args []string) error {
	if len(args) == 0 || (args[0] != sousCommandeEvaluate && args[0] != sousCommandeFit) {
		return fmt.Errorf("sous-commande attendue : evaluate | fit (voir `levelup help`)")
	}
	o, err := lireOptionsSpawnModel(args[0], args[1:])
	if err != nil {
		return err
	}
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapFilmSpawnForecast) {
		fmt.Printf("le titre %s ne declare pas %s : aucune mesure (passe vide)\n",
			o.titleSlug, string(games.CapFilmSpawnForecast))
		return nil
	}
	if o.table, err = famillesDuTitre(cfg, o.titleSlug); err != nil {
		return err
	}
	ctx := context.Background()
	switch {
	case args[0] == sousCommandeEvaluate && o.sections == sectionPoids:
		return imprimerPoidsEcrits(ctx, cfg, o)
	case args[0] == sousCommandeFit:
		err = ajusterEtEcrire(ctx, cfg, o)
	default:
		err = evaluerLeParc(ctx, cfg, o)
	}
	// `--variantes` liste les variantes puis s'arrete : une sortie normale pour les deux sous-commandes.
	if errors.Is(err, errArretVariantes) {
		return nil
	}
	return err
}

// donneesDuParc ouvre le registre en lecture, liste les artefacts et lit le parc. La poignee de
// lecture est rendue pour les zones, avec sa liberation.
func donneesDuParc(ctx context.Context, cfg *config.AppConfig, o spawnModelOptions,
	familles map[string]bool) (*jeuReapparitions, *sql.DB, func(), error) {
	pr := cfg.PathResolver()
	db, release, err := duckdb.OpenReadForQuery(pr.SharedDBPath(o.titleSlug))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ouverture en lecture de la base partagee (tenue en ecriture par "+
			"un autre processus ? arreter le serveur) : %w", err)
	}
	registre, err := registreDesMatchs(ctx, db, o.table)
	if err != nil {
		release()
		return nil, nil, nil, err
	}
	shorts, err := artefactsDuTitre(ctx, cfg, o.titleSlug)
	if err != nil {
		release()
		return nil, nil, nil, err
	}
	if o.variantes {
		fmt.Println(strings.Join(variantesDuParc(registre, shorts), "\n"))
		release()
		return nil, nil, nil, errArretVariantes
	}
	slog.InfoContext(ctx, "spawn-model : lecture du parc", "artefacts", len(shorts),
		"registre", len(registre), "familles", len(familles))
	jeu := lireLeParc(ctx, cfg, optionsLecture{
		titleSlug: o.titleSlug, familles: familles, limite: o.limite,
		registre: registre, artefactsS: shorts,
	})
	return jeu, db, release, nil
}

// errArretVariantes : la sortie normale de `--variantes`, qui n'est pas une erreur.
var errArretVariantes = errors.New("arret apres la liste des variantes")

// famillesDemandees rend l'ensemble des familles a lire.
func famillesDemandees(o spawnModelOptions) (map[string]bool, error) {
	out := map[string]bool{}
	for _, f := range o.table.Families() {
		if o.famille == "" || o.famille == f {
			out[f] = true
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("famille inconnue %q (attendu : %s)", o.famille, strings.Join(o.table.Families(), ", "))
	}
	return out, nil
}

func sectionActive(o spawnModelOptions, s string) bool {
	for _, x := range strings.Split(o.sections, ",") {
		if strings.TrimSpace(x) == s {
			return true
		}
	}
	return false
}

// evaluerLeParc lit le parc une fois et produit les sections demandees.
func evaluerLeParc(ctx context.Context, cfg *config.AppConfig, o spawnModelOptions) error {
	familles, err := famillesDemandees(o)
	if err != nil {
		return err
	}
	jeu, db, release, err := donneesDuParc(ctx, cfg, o, familles)
	if err != nil {
		return err
	}
	defer release()
	imprimerBilan(os.Stdout, jeu.bilan)
	r := rapporteur{ctx: ctx, cfg: cfg, o: o, jeu: jeu, db: db, familles: familles}
	if sectionActive(o, sectionParMode) {
		r.parMode()
	}
	if sectionActive(o, sectionInstant) {
		r.instant()
	}
	return nil
}

// imprimerPoidsEcrits relit le fichier de poids par la lecture du service (repli journalise et
// compte quand il manque) et l'imprime, puis dit quel jeu de poids le service retiendrait pour
// chaque famille du titre et pour un match sans famille.
func imprimerPoidsEcrits(ctx context.Context, cfg *config.AppConfig, o spawnModelOptions) error {
	p, ok := service.ChargerPoidsReapparition(ctx, o.titleSlug, cfg.PathResolver().SpawnModelPath(o.titleSlug))
	if !ok {
		return fmt.Errorf("aucun fichier de poids exploitable pour %s (voir le journal)", o.titleSlug)
	}
	fmt.Printf("poids du %s : %s\n\n", p.Date, p.Definition)
	fmt.Println("| famille | reapparitions | matchs | a 8 m, 3 premiers : hasard | modele | exact 3 premiers |")
	fmt.Println("|---|---|---|---|---|---|")
	noms := make([]string, 0, len(p.Familles))
	for nom := range p.Familles {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		f := p.Familles[nom]
		fmt.Printf("| %s | %d | %d | %.1f | %.1f | %.1f |\n", nom, f.Reapparitions, f.Matchs,
			f.Scores.Rayon3Hasard, f.Scores.Rayon3, f.Scores.Top3)
	}
	fmt.Println("\n| famille du match | jeu de poids retenu |\n|---|---|")
	for _, fam := range append(o.table.Families(), "") {
		retenu, _, ok := tactical.PoidsPourFamille(p, fam)
		if !ok {
			retenu = "(aucun : pas de prevision)"
		}
		if fam == "" {
			fam = "(sans famille)"
		}
		fmt.Printf("| %s | %s |\n", fam, retenu)
	}
	return nil
}
