// cmd/mapfond-build — cuit les FONDS DE CARTE : l'image vue du dessus de chaque carte,
// reconstruite depuis les triangles du maillage de rendu de son .module, et son calage.
//
// POURQUOI CE BINAIRE EXISTE, ET POURQUOI IL N'EST PAS `mapstruct-build`. Les deux produisent
// un asset de carte, mais ils n'ont ni la même entrée, ni la même sortie, ni le même
// consommateur :
//
//	mapstruct-build   map_quant_bounds.json -> map_structure/{module}.json  (AABB, aucune forme)
//	mapfond-build     map_objectives.json   -> map_backgrounds/{module}.png (triangles, image)
//
// Le fond de carte a besoin des ANCRES d'objectifs — pour cadrer, pour déduire le niveau joué,
// et pour se juger — que le catalogue de bornes ne porte pas. Fondre les deux orchestrations
// dans un même binaire mêlerait deux chaînes sans rien mutualiser.
//
// LICENCE — CONTRAINTE STRUCTURANTE : la chaîne passe par la brique `halo-infinite-map-images`
// (`himap`), puis par `himodule` et `ooz` de la brique `halo-infinite-files-reader`, **GPLv3**. Ce
// binaire est de l'outillage HORS LIGNE ; l'application lit l'asset figé, elle ne le fabrique
// jamais. `TestServeurNeLinkePasDeCodeGPL` (cmd/server/licence_gplv3_test.go) interdit à cmd/server d'en
// dépendre.
//
// TROIS GESTES, et eux seuls (chaque option de la ligne de commande se rattache à l'un d'eux, voir
// options.go) :
//
//  1. CUIRE des cartes : l'image et le sidecar de chaque carte retenue.
//     CGO_ENABLED=1 go run ./cmd/mapfond-build --maps "ridgeline,catalyst"
//  2. REFAIRE LE PLAN SANS RECUIRE : le plan des murs et l'image plan des sidecars déjà publiés,
//     le fond, le calage, les statistiques et la date de génération restant ceux de la cuisson.
//     CGO_ENABLED=1 go run ./cmd/mapfond-build --plan-seul --maps "ridgeline"
//  3. PRODUIRE LE TABLEAU AVANT / APRÈS : une ligne TSV par carte traitée, à côté d'une cuisson ou
//     d'un plan seul ; --avant-dir porte les sidecars d'avant, pour comparer les ancres au sol.
//     CGO_ENABLED=1 go run ./cmd/mapfond-build --maps "ridgeline" --rapport-tsv rapport.tsv --avant-dir DIR
//
// --maps vide = toutes les cartes du catalogue d'objectifs installées localement. Une carte
// NATIVE se désigne par son dossier installé (`ridgeline`), une carte FORGE par son map_id
// UGC (clé de sa déclaration `himap.CartesForge` et de son fond publié).
//
// --avis lit l'avis du user carte par carte sur le croisement fond / maillage de navigation (voir
// avis.go) : le rognage au maillage s'arme dans la forme que l'avis donne, une carte « publiee »
// n'est pas recuite.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

func main() {
	os.Exit(execute(context.Background(), os.Args[1:], os.Stderr))
}

// execute lit la ligne de commande, la valide, prépare l'environnement et lance le geste demandé.
// Il rend le code de sortie : 0, 1 pour un échec de la cuisson ou une option refusée, 2 pour une
// erreur de syntaxe (le paquet flag l'a déjà écrite).
func execute(ctx context.Context, args []string, sortie io.Writer) int {
	opts, err := litOptions(args, sortie)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		return 2
	}
	if err := opts.valide(); err != nil {
		slog.ErrorContext(ctx, "options", "err", err)
		return 1
	}
	env, err := opts.environnement()
	if err != nil {
		slog.ErrorContext(ctx, "préparation", "err", err)
		return 1
	}
	if opts.planSeul {
		return executePlanSeul(ctx, env, opts.rapports)
	}
	return cuit(ctx, env, opts)
}

// environnement résout ce que les options désignent : racines, catalogues, réglages, avis.
func (o options) environnement() (*environnement, error) {
	env, err := prepare(o.titleSlug, o.outDir)
	if err != nil {
		return nil, err
	}
	env.style = himap.StyleFond(o.style)
	env.filtre = filtreDe(o.maps)
	env.echelle = o.mpp
	env.sansNatives, env.sansForge = !o.natives, !o.forge
	if o.avis != "" {
		if env.avis, err = chargeAvis(strings.Split(o.avis, ",")...); err != nil {
			return nil, fmt.Errorf("avis %q : %w", o.avis, err)
		}
	}
	return env, nil
}

// cuit est le geste « cuire des cartes » : les natives puis les Forge retenues, le bilan, et le
// rapport TSV s'il est demandé. Rend le code de sortie.
func cuit(ctx context.Context, env *environnement, o options) int {
	var bilans []bilanAsset
	if o.natives {
		bilans = append(bilans, env.cuitNatives(ctx)...)
	}
	if o.forge {
		bilans = append(bilans, env.cuitForge(ctx)...)
	}
	sort.Slice(bilans, func(i, j int) bool { return bilans[i].Cle < bilans[j].Cle })

	echecs := resume(ctx, bilans, env)
	if o.rapports.tsv != "" {
		if err := ecritRapportTSV(o.rapports.tsv, o.rapports.avantDir, bilans, env); err != nil {
			slog.ErrorContext(ctx, "rapport tsv", "err", err, "path", o.rapports.tsv)
			echecs++
		}
	}
	if echecs > 0 {
		return 1
	}
	return 0
}

// environnement porte ce qui est résolu une fois pour toutes.
type environnement struct {
	titleSlug string
	repoRoot  string
	sortieDir string
	racineJeu string
	catalogue *replay.MapObjectivesCatalog
	style     himap.StyleFond
	// reglages : les choix PAR CARTE (habillage, echelle), lus en donnee. Jamais nil.
	reglages *reglagesFond
	// callouts : les zones nommees par carte. Nil si le catalogue est absent — la mesure hors
	// zones est alors muette, jamais un zero silencieux.
	callouts *replay.MapCalloutsCatalog
	// positions : les positions reellement jouees par carte, oracle du rognage le plus fort du
	// chantier. Nil si le catalogue est absent — chaque carte qui le demande le dira alors.
	positions map[string][]himap.PositionJouee
	// echelle : cote du pixel en metres. Zero = celle de production (`EchelleFondCarte`).
	// Reglage PAR CARTE au sens du gate du 26/08 — ici il vaut pour toute la cuisson demandee.
	echelle float64
	filtre  map[string]bool
	// avis : l'avis du user sur le croisement fond / maillage (`-avis`). Nil sans l'option.
	avis *avisFond
	// sansNatives et sansForge : `-natives=false` et `-forge=false`.
	sansNatives, sansForge bool
	// nomsAffiches : module installé -> noms de carte affichés, lus dans
	// map_quant_bounds.json. C'est LA table déclarée du lien nom affiché -> module ; on ne
	// la recopie pas, on la lit.
	nomsAffiches map[string][]string
	// levelsDir et liens servent la résolution nom de carte -> dossier installé quand les
	// jetons de nom échouent (cf. resolution.go).
	levelsDir   string
	liens       map[string]string
	nonInstalle []string
}

func prepare(titleSlug, outDir string) (*environnement, error) {
	root, err := title.FindRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("racine du dépôt : %w", err)
	}
	racine, err := himap.DeployRoot()
	if err != nil {
		return nil, err
	}
	res := title.NewPathResolver(root)
	cat, err := replay.LoadMapObjectives(res.MapObjectivesPath(titleSlug))
	if err != nil {
		return nil, err
	}
	dir := outDir
	if dir == "" {
		dir = res.MapBackgroundDir(titleSlug)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("répertoire de sortie : %w", err)
	}
	levels, err := himap.LevelsDir("pc")
	if err != nil {
		return nil, err
	}
	reg, err := chargeReglages(res.MapFondReglagesPath(titleSlug))
	if err != nil {
		return nil, err
	}
	return &environnement{
		titleSlug: titleSlug, repoRoot: root, sortieDir: dir,
		racineJeu: racine, catalogue: cat,
		nomsAffiches: nomsAffichesParModule(res.MapQuantBoundsPath(titleSlug)),
		levelsDir:    levels,
		liens:        liensDuDepot(root, levels),
		reglages:     reg,
		callouts:     chargeCallouts(res.MapCalloutsPath(titleSlug)),
		positions:    chargePositionsJouees(res.MapPlayedPositionsPath(titleSlug)),
	}, nil
}

// nomsAffichesParModule lit le lien nom affiché -> module du catalogue de bornes.
//
// Son absence n'est PAS fatale : les noms sont un confort de lecture, pas une entrée de la
// chaîne. Elle est signalée, jamais avalée — un catalogue absent qui passe en silence est le
// piège que ce chantier paie depuis le début.
func nomsAffichesParModule(chemin string) map[string][]string {
	out := map[string][]string{}
	cat, err := decfilm.LoadMapQuantCatalog(chemin)
	if err != nil {
		slog.Warn("catalogue de bornes illisible — les assets sortiront sans nom affiché",
			"err", err, "path", chemin)
		return out
	}
	for nom, e := range cat.Maps {
		out[e.Module] = append(out[e.Module], nom)
	}
	for _, noms := range out {
		sort.Strings(noms)
	}
	return out
}

// filtreDe rend l'ensemble des clés demandées, ou nil pour « tout ».
func filtreDe(list string) map[string]bool {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	out := map[string]bool{}
	for _, raw := range strings.Split(list, ",") {
		if k := strings.TrimSpace(raw); k != "" {
			out[k] = true
		}
	}
	return out
}

func (e *environnement) retenue(cle string) bool {
	return e.filtre == nil || e.filtre[cle]
}

// resoutModule rend le `.module` d'une entrée du catalogue : d'abord par les jetons de nom,
// puis par le dépôt de variantes quand le nom ne dit rien (cf. resolution.go).
func (e *environnement) resoutModule(moduleCatalogue string) (string, bool) {
	if chemin, ok := himap.ChercheModuleInstalle(moduleCatalogue); ok {
		return chemin, true
	}
	m := moduleDuCatalogue(e.liens, e.levelsDir, moduleCatalogue)
	if m == "" {
		return "", false
	}
	return filepath.Join(e.levelsDir, m, m+"-rtx-new.module"), true
}

// noteNonInstalle enregistre un module du catalogue absent de l'installation, une seule fois.
func (e *environnement) noteNonInstalle(module string) {
	for _, m := range e.nonInstalle {
		if m == module {
			return
		}
	}
	e.nonInstalle = append(e.nonInstalle, module)
}

func stylesConnus() string {
	noms := make([]string, 0, len(himap.StylesFond))
	for _, s := range himap.StylesFond {
		noms = append(noms, string(s))
	}
	return strings.Join(noms, "|")
}

// resume journalise le bilan global et rend le nombre d'échecs.
func resume(ctx context.Context, bilans []bilanAsset, env *environnement) int {
	echecs, degrades, ancres, avecSol, nonCuisinables, sautees := 0, 0, 0, 0, 0, 0
	var octets int64
	for _, b := range bilans {
		switch {
		case b.laissee():
			sautees++
			continue
		case b.NonCuisinable:
			nonCuisinables++
			continue
		case b.Err != nil:
			echecs++
			continue
		}
		octets += b.OctetsPNG
		ancres += b.Bilan.AncresDansLeCadre
		avecSol += b.Bilan.AncresAvecSol
		if len(b.Bilan.Degradations) > 0 {
			degrades++
		}
	}
	if len(env.nonInstalle) > 0 {
		slog.InfoContext(ctx, "cartes du catalogue non installées localement",
			"n", len(env.nonInstalle), "modules", strings.Join(env.nonInstalle, ", "))
	}
	slog.InfoContext(ctx, "cuisson terminée",
		"cartes", len(bilans)-echecs-nonCuisinables-sautees, "sautées", sautees, "échecs", echecs,
		"nonCuisinables", nonCuisinables, "dégradées", degrades,
		"ancresAvecSol", fmt.Sprintf("%d/%d", avecSol, ancres),
		"octets", octets, "style", string(env.style), "reglagesParCarte", len(env.reglages.Cartes),
		"sortie", env.sortieDir)
	return echecs
}

// echelleEffective rend l'echelle vraiment appliquee : celle demandee, ou celle de production.
// Le rapport de cuisson la publie — un tableau qui annoncerait 0,0920 m/px alors que la
// cuisson en a utilise une autre serait un faux temoin.
func (e *environnement) echelleEffective() float64 {
	if e.echelle > 0 {
		return e.echelle
	}
	return himap.EchelleFondCarte
}

// chargeCallouts lit le catalogue des zones nommées. ABSENT ou illisible n'arrête pas la
// cuisson — mais c'est SIGNALÉ : une mesure muette qu'on prendrait pour un zéro serait pire
// que pas de mesure du tout.
func chargeCallouts(chemin string) *replay.MapCalloutsCatalog {
	cat, err := replay.LoadMapCallouts(chemin)
	if err != nil {
		slog.Warn("mapfond: catalogue de callouts indisponible — mesure hors zones muette",
			"err", err, "path", chemin)
		return nil
	}
	return cat
}

// chargePositionsJouees lit le catalogue des positions jouées. Son absence n'est pas une erreur :
// aucune carte n'a alors ce levier disponible, et `positionsJoueesDe` le dira carte par carte.
func chargePositionsJouees(chemin string) map[string][]himap.PositionJouee {
	brut, err := os.ReadFile(chemin)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("mapfond: catalogue des positions jouees illisible", "err", err, "path", chemin)
		}
		return nil
	}
	var cat struct {
		Maps map[string]struct {
			Positions [][2]float64 `json:"positions"`
		} `json:"maps"`
	}
	if err := json.Unmarshal(brut, &cat); err != nil {
		slog.Warn("mapfond: catalogue des positions jouees invalide", "err", err, "path", chemin)
		return nil
	}
	out := make(map[string][]himap.PositionJouee, len(cat.Maps))
	for cle, m := range cat.Maps {
		pts := make([]himap.PositionJouee, 0, len(m.Positions))
		for _, p := range m.Positions {
			pts = append(pts, himap.PositionJouee{X: p[0], Y: p[1]})
		}
		out[cle] = pts
	}
	slog.Info("mapfond: catalogue des positions jouees charge", "cartes", len(out), "path", chemin)
	return out
}
