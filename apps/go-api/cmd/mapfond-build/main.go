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
// LICENCE — CONTRAINTE STRUCTURANTE : la chaîne passe par internal/himap -> internal/himodule
// -> internal/ooz, **GPLv3**. Ce binaire est de l'outillage HORS LIGNE ; l'application lit
// l'asset figé, elle ne le fabrique jamais. `licence_test.go` interdit à cmd/server d'en
// dépendre.
//
// Usage :
//
//	CGO_ENABLED=1 go run ./cmd/mapfond-build [--maps "ridgeline,catalyst"] [--title slug]
//	                                         [--out-dir DIR] [--style jeu] [--rapport FILE]
//	                                         [--natives=false] [--forge=false]
//
// --maps vide = toutes les cartes du catalogue d'objectifs installées localement. Une carte
// NATIVE se désigne par son dossier installé (`ridgeline`), une carte FORGE par son map_id
// UGC (clé de sa déclaration `himap.CartesForge` et de son fond publié).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

func main() {
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	maps := flag.String("maps", "", "clés de carte séparées par des virgules — dossier installé (native) "+
		"ou map_id UGC (Forge) ; vide = tout ce qui est installé")
	outDir := flag.String("out-dir", "", "répertoire de sortie (défaut : PathResolver.MapBackgroundDir)")
	style := flag.String("style", string(himap.StyleFondParDefaut), "habillage : "+stylesConnus())
	mpp := flag.Float64("mpp", 0, "cote du pixel en metres (0 = "+fmt.Sprintf("%.4f", himap.EchelleFondCarte)+", celle de production)")
	rapport := flag.String("rapport", "", "écrit un rapport Markdown à ce chemin")
	natives := flag.Bool("natives", true, "cuire les cartes cuites dans un module")
	forge := flag.Bool("forge", true, "cuire les cartes Forge (posées depuis leur .mvar)")
	flag.Parse()

	ctx := context.Background()
	if !himap.StyleFondValide(himap.StyleFond(*style)) {
		slog.ErrorContext(ctx, "habillage inconnu", "style", *style, "connus", stylesConnus())
		os.Exit(1)
	}
	env, err := prepare(*titleSlug, *outDir)
	if err != nil {
		slog.ErrorContext(ctx, "préparation", "err", err)
		os.Exit(1)
	}
	env.style = himap.StyleFond(*style)
	env.filtre = filtreDe(*maps)
	env.echelle = *mpp

	var bilans []bilanAsset
	if *natives {
		bilans = append(bilans, env.cuitNatives(ctx)...)
	}
	if *forge {
		bilans = append(bilans, env.cuitForge(ctx)...)
	}
	sort.Slice(bilans, func(i, j int) bool { return bilans[i].Cle < bilans[j].Cle })

	echecs := resume(ctx, bilans, env)
	if *rapport != "" {
		if err := ecritRapport(*rapport, bilans, env); err != nil {
			slog.ErrorContext(ctx, "rapport", "err", err, "path", *rapport)
			echecs++
		}
	}
	if echecs > 0 {
		os.Exit(1)
	}
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
	// echelle : cote du pixel en metres. Zero = celle de production (`EchelleFondCarte`).
	// Reglage PAR CARTE au sens du gate du 26/08 — ici il vaut pour toute la cuisson demandee.
	echelle float64
	filtre  map[string]bool
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
	}, nil
}

// nomsAffichesParModule lit le lien nom affiché -> module du catalogue de bornes.
//
// Son absence n'est PAS fatale : les noms sont un confort de lecture, pas une entrée de la
// chaîne. Elle est signalée, jamais avalée — un catalogue absent qui passe en silence est le
// piège que ce chantier paie depuis le début.
func nomsAffichesParModule(chemin string) map[string][]string {
	out := map[string][]string{}
	cat, err := filmdec.LoadMapQuantCatalog(chemin)
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
	echecs, degrades, ancres, avecSol, nonCuisinables := 0, 0, 0, 0, 0
	var octets int64
	for _, b := range bilans {
		switch {
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
		"cartes", len(bilans)-echecs-nonCuisinables, "échecs", echecs,
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
