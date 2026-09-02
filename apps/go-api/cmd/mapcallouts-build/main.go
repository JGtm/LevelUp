// cmd/mapcallouts-build — produit le CATALOGUE DES CALLOUTS (zones nommées) et l'écrit
// dans data/titles/{slug}/reference/map_callouts.json (PathResolver.MapCalloutsPath).
//
// DEUX PASSES, DEUX SOURCES, UN SEUL FICHIER — et un seul écrivain, ce binaire.
//
// PASSE NATIVE (cartes intégrées, clé = module installé). EXIGE LE JEU INSTALLÉ :
//
//	deploy/ds/levels/multi/<carte>/<carte>-rtx-new.module
//	  -> tag levl (internal/himap.ReadModuleCallouts : offsets documentés au champ près)
//	  -> jointure des libellés FR/EN par (carte, volume_index) sur callouts_i18n.csv
//	     (copie VERSIONNÉE — data/titles/{slug}/reference/callouts_i18n.csv : 816/816
//	     résolus par string_id, on ne re-extrait PAS uslg)
//	  -> classement grandes/fines par recouvrement (classify.go, étalonné sur le POC)
//	  -> Ridgeline : polygones remplacés par le dump DÉCOUPÉ versionné (decoupe.go)
//
// INVARIANTS MESURÉS, et la passe ÉCHOUE s'ils bougent : 22 cartes avec zones, 816 zones,
// libellé résolu 816/816, string_id du CSV égal à celui du tag. Un catalogue partiel ne
// s'écrit pas.
//
// PASSE FORGE (cartes communautaires, clé = map_id). N'EXIGE PAS LE JEU — elle lit les
// `map.mvar` téléchargés (forge.go, forge_fetch.go). Voir ces fichiers pour la chaîne et
// pour la règle de publication d'une carte.
//
// LES DEUX PASSES SONT INDÉPENDANTES ET NE S'ÉCRASENT PAS : une reconstruction native
// CONSERVE la section Forge du catalogue existant, et `--forge-only` conserve la section
// native. Ce n'est pas une politesse — les deux sources n'ont pas les mêmes prérequis (le
// jeu d'un côté, le réseau de l'autre), et perdre l'une en refaisant l'autre serait la
// façon la plus simple de vider le catalogue par accident.
//
// Usage :
//
//	CGO_ENABLED=1 go run ./cmd/mapcallouts-build                       # passe native seule
//	CGO_ENABLED=1 go run ./cmd/mapcallouts-build --forge-only --forge-fetch
//	                                                                   # passe Forge seule
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

// deployVariant : la variante sur laquelle la table levl a été établie et mesurée
// (callouts_all.py). La racine est résolue par himap.LevelsDir, jamais écrite en dur.
const deployVariant = "ds"

// Invariants du corpus natif (mesure callouts_all.py + CSV, 2026-08).
const (
	attenduCartes = 22
	attenduZones  = 816
)

// inventaireParDefaut : l'inventaire UGC versionné dont la passe Forge tire sa liste.
var inventaireParDefaut = []string{".ai", "V7.5", "cartes", "inventaire_rotation_ugc_2026-08-27.json"}

// cacheVariantesParDefaut : le cache local des `.mvar`, HORS DÉPÔT (.gitignore).
var cacheVariantesParDefaut = []string{".ai", "V7.5", "dumps", "mapvar"}

// options porte les chemins et les bascules résolus, pour ne pas promener dix paramètres.
type options struct {
	levels      string
	titleSlug   string
	i18nPath    string
	decoupePath string
	outPath     string
	inventaire  string
	cacheForge  string
	forgeOnly   bool
	forgeFetch  bool
	delaiFetch  time.Duration
	res         *title.PathResolver
}

func main() {
	opts := lisOptions()
	labels, labelsParSID, err := chargeLibelles(opts.i18nPath)
	if err != nil {
		slog.Error("libellés", "err", err)
		os.Exit(1)
	}
	cat := chargeCatalogueExistant(opts.outPath, opts.titleSlug)
	if !opts.forgeOnly {
		passeNative(&cat, opts, labels)
	}
	passeForge(&cat, opts, labelsParSID)
	ecritCatalogue(cat, opts.outPath)
}

// lisOptions déclare les drapeaux et résout les chemins par défaut.
func lisOptions() options {
	var o options
	flag.StringVar(&o.levels, "levels", "", "racine des dossiers de cartes (.module) ; vide = installation détectée")
	flag.StringVar(&o.titleSlug, "title", title.DefaultSlug, "slug du titre")
	flag.StringVar(&o.i18nPath, "i18n", "", "CSV des libellés (défaut : reference/callouts_i18n.csv du titre)")
	flag.StringVar(&o.decoupePath, "decoupe", "", "dump découpé de Ridgeline (défaut : .ai/V7.5/dumps/callout_zones_ridgeline_clipped.json)")
	flag.StringVar(&o.outPath, "out", "", "fichier de sortie (défaut : PathResolver.MapCalloutsPath)")
	flag.StringVar(&o.inventaire, "forge-inventaire", "", "inventaire UGC versionné (défaut : .ai/V7.5/cartes/inventaire_rotation_ugc_2026-08-27.json)")
	flag.StringVar(&o.cacheForge, "forge-cache", "", "cache local des .mvar (défaut : .ai/V7.5/dumps/mapvar)")
	flag.BoolVar(&o.forgeOnly, "forge-only", false, "ne pas reconstruire la partie native (n'exige pas le jeu installé)")
	flag.BoolVar(&o.forgeFetch, "forge-fetch", false, "télécharger les map.mvar manquants (stockage blob UGC, sans jeton)")
	ms := flag.Int("forge-rate-ms", 300, "délai entre deux téléchargements (politesse)")
	flag.Parse()

	root, err := title.FindRepoRoot()
	if err != nil {
		slog.Error("racine repo", "err", err)
		os.Exit(1)
	}
	o.res = title.NewPathResolver(root)
	o.delaiFetch = time.Duration(*ms) * time.Millisecond
	if o.levels == "" && !o.forgeOnly {
		dir, lerr := himap.LevelsDir(deployVariant)
		if lerr != nil {
			slog.Error("installation du jeu", "err", lerr)
			os.Exit(1)
		}
		o.levels = dir
		slog.Info("installation détectée", "levels", dir)
	}
	if o.outPath == "" {
		o.outPath = o.res.MapCalloutsPath(o.titleSlug)
	}
	if o.i18nPath == "" {
		o.i18nPath = filepath.Join(filepath.Dir(o.res.MapCalloutsPath(o.titleSlug)), "callouts_i18n.csv")
	}
	if o.decoupePath == "" {
		o.decoupePath = filepath.Join(root, ".ai", "V7.5", "dumps", "callout_zones_ridgeline_clipped.json")
	}
	if o.inventaire == "" {
		o.inventaire = filepath.Join(append([]string{root}, inventaireParDefaut...)...)
	}
	if o.cacheForge == "" {
		o.cacheForge = filepath.Join(append([]string{root}, cacheVariantesParDefaut...)...)
	}
	return o
}

// chargeCatalogueExistant relit le catalogue déjà sur disque. Son absence est le cas d'une
// PREMIÈRE construction, pas une erreur ; son illisibilité, elle, arrête tout — écraser un
// fichier qu'on n'a pas su lire perdrait la passe qu'on ne rejoue pas.
func chargeCatalogueExistant(path, titleSlug string) replay.MapCalloutsCatalog {
	vide := replay.MapCalloutsCatalog{
		SchemaVersion: replay.MapCalloutsSchemaVersion,
		TitleSlug:     titleSlug,
		Maps:          map[string]replay.MapCalloutsEntry{},
		Brut:          map[string][]replay.CalloutBrutZone{},
	}
	if _, err := os.Stat(path); err != nil {
		return vide
	}
	cat, err := replay.LoadMapCallouts(path)
	if err != nil {
		slog.Error("catalogue existant illisible — rien écrit", "err", err, "path", path)
		os.Exit(1)
	}
	if cat.Maps == nil {
		cat.Maps = map[string]replay.MapCalloutsEntry{}
	}
	if cat.Brut == nil {
		cat.Brut = map[string][]replay.CalloutBrutZone{}
	}
	cat.TitleSlug = titleSlug
	return *cat
}

// passeNative reconstruit les entrées des cartes intégrées. Elle REMPLACE `Maps` et `Brut`,
// et ne touche jamais `MapsByID`.
func passeNative(cat *replay.MapCalloutsCatalog, opts options, labels libelles) {
	decoupeModule, decoupeZones, err := chargeDecoupe(opts.decoupePath)
	if err != nil {
		slog.Error("dump découpé", "err", err)
		os.Exit(1)
	}
	cat.Maps = map[string]replay.MapCalloutsEntry{}
	cat.Brut = map[string][]replay.CalloutBrutZone{}
	cat.Source = "tag levl de " + opts.levels + " + callouts_i18n.csv (libellés uslg figés)" +
		" ; Ridgeline : polygones du dump découpé versionné" +
		" ; autres cartes à fond publié : découpe sur le masque praticable (internal/mapdecoupe)"
	entries, err := os.ReadDir(opts.levels)
	if err != nil {
		slog.Error("dossier des cartes", "err", err, "levels", opts.levels)
		os.Exit(1)
	}
	totalZones := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n, ok := ingereModule(cat, opts, labels, moduleADecouper{
			module: e.Name(), decoupeModule: decoupeModule, decoupeZones: decoupeZones,
		})
		if ok {
			totalZones += n
		}
	}
	if len(cat.Maps) != attenduCartes || totalZones != attenduZones {
		slog.Error("corpus inattendu — rien écrit",
			"cartes", len(cat.Maps), "zones", totalZones,
			"attendu", fmt.Sprintf("%d cartes / %d zones", attenduCartes, attenduZones))
		os.Exit(1)
	}
}

// moduleADecouper porte le module courant et le choix de découpage qui s'y applique.
type moduleADecouper struct {
	module        string
	decoupeModule string
	decoupeZones  map[int]decoupeZone
}

// ingereModule lit un module installé et ajoute son entrée. Rend le nombre de zones et
// `false` quand le dossier n'est pas une carte à zones (canevas Forge, dossier sans module).
func ingereModule(cat *replay.MapCalloutsCatalog, opts options, labels libelles, m moduleADecouper) (int, bool) {
	modPath := filepath.Join(opts.levels, m.module, m.module+"-rtx-new.module")
	if _, statErr := os.Stat(modPath); statErr != nil {
		return 0, false
	}
	cs, err := himap.ReadModuleCallouts(modPath)
	if err != nil {
		slog.Error("lecture des callouts", "err", err, "module", m.module)
		os.Exit(1)
	}
	if len(cs) == 0 {
		// Canevas Forge et assimilés : ZÉRO zone PAR CONSTRUCTION — pas d'entrée ici.
		// Les zones d'une CARTE Forge, elles, vivent dans son map.mvar (passe Forge).
		return 0, false
	}
	entry, err := construitEntree(m.module, cs, labels)
	if err != nil {
		slog.Error("jointure des libellés", "err", err, "module", m.module)
		os.Exit(1)
	}
	if m.module == m.decoupeModule {
		if err := appliqueDecoupe(&entry, m.decoupeZones); err != nil {
			slog.Error("dump découpé", "err", err, "module", m.module)
			os.Exit(1)
		}
	} else if err := decoupeCarte(&entry, opts.res, opts.titleSlug, cat.Brut); err != nil {
		slog.Error("découpage sur le fond publié", "err", err, "module", m.module)
		os.Exit(1)
	}
	cat.Maps[m.module] = entry
	grandes := 0
	for _, z := range entry.Zones {
		if z.Big {
			grandes++
		}
	}
	slog.Info("carte lue", "module", m.module, "zones", len(entry.Zones),
		"grandes", grandes, "provenance", entry.Provenance)
	return len(entry.Zones), true
}

// passeForge reconstruit la section `maps_by_id` à partir des variantes en cache.
//
// Elle ne s'exécute QUE si on la demande (`--forge-only` ou `--forge-fetch`) : sans cela,
// la section existante est conservée telle quelle. Une passe Forge qui repartirait d'un
// cache vide effacerait silencieusement des cartes déjà extraites.
func passeForge(cat *replay.MapCalloutsCatalog, opts options, labels libellesParStringID) {
	if !opts.forgeOnly && !opts.forgeFetch {
		return
	}
	cibles, err := chargeInventaire(opts.inventaire)
	if err != nil {
		slog.Error("inventaire UGC", "err", err)
		os.Exit(1)
	}
	if opts.forgeFetch {
		pris, echecs := telechargeVariantes(context.Background(), cibles, opts.cacheForge, opts.delaiFetch)
		slog.Info("variantes téléchargées", "nouvelles", pris, "echecs", echecs, "cache", opts.cacheForge)
	}
	stats := nouvellesStats()
	cat.MapsByID = construitPasseForge(cibles, opts.cacheForge, labels, stats)
	if len(cat.MapsByID) == 0 {
		slog.Error("passe Forge : aucune carte publiable — rien écrit",
			"cibles", len(cibles), "cache", opts.cacheForge)
		os.Exit(1)
	}
	journaliseStats(stats, len(cibles))
}

// journaliseStats rend la MESURE de la passe Forge — des chiffres, pas une impression.
func journaliseStats(s *forgeStats, cibles int) {
	slog.Info("passe Forge : couverture",
		"cartes_visees", cibles, "cartes_publiees", s.Cartes,
		"zones", s.Zones, "zones_nommees", s.ZonesNommees,
		"string_id_distincts", len(s.SidsDistincts), "string_id_resolus", len(s.SidsResolus),
		"sans_zone", len(s.SansZone), "sans_aucun_libelle", len(s.SansLibelle),
		"variantes_absentes_ou_illisibles", len(s.Illisibles))
	for _, n := range s.SansZone {
		slog.Info("carte Forge sans zone nommée", "nom", n)
	}
	for _, n := range s.SansLibelle {
		slog.Info("carte Forge écartée : aucune zone nommable", "nom", n)
	}
	for _, n := range s.Illisibles {
		slog.Info("carte Forge non traitée", "detail", n)
	}
}

// ecritCatalogue sérialise et écrit le fichier de référence.
func ecritCatalogue(cat replay.MapCalloutsCatalog, outPath string) {
	blob, err := json.MarshalIndent(cat, "", " ")
	if err != nil {
		slog.Error("sérialisation", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		slog.Error("création du répertoire", "err", err, "path", outPath)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, append(blob, '\n'), 0o644); err != nil {
		slog.Error("écriture", "err", err, "path", outPath)
		os.Exit(1)
	}
	zones := 0
	for _, e := range cat.Maps {
		zones += len(e.Zones)
	}
	zonesForge := 0
	for _, e := range cat.MapsByID {
		zonesForge += len(e.Zones)
	}
	slog.Info("catalogue écrit", "path", outPath,
		"cartes_natives", len(cat.Maps), "zones_natives", zones,
		"cartes_forge", len(cat.MapsByID), "zones_forge", zonesForge)
}

// construitEntree joint les libellés et classe les zones d'une carte intégrée.
func construitEntree(module string, cs []himap.Callout, labels libelles) (replay.MapCalloutsEntry, error) {
	var shaped []shapedPoly
	for _, c := range cs {
		if c.HasShape && len(c.Polygon) >= 3 {
			shaped = append(shaped, shapedPoly{vi: c.VolumeIndex, poly: c.Polygon})
		}
	}
	big := classifyBig(shaped)

	entry := replay.MapCalloutsEntry{
		Module:     module,
		Provenance: replay.CalloutsProvenanceBrut,
		Zones:      make([]replay.CalloutZone, 0, len(cs)),
	}
	for _, c := range cs {
		lbl, err := labels.resolve(module, c)
		if err != nil {
			return replay.MapCalloutsEntry{}, err
		}
		z := replay.CalloutZone{
			VolumeIndex: c.VolumeIndex,
			Name:        c.Name,
			EN:          lbl.en,
			FR:          lbl.fr,
			X:           c.Pos[0],
			Y:           c.Pos[1],
			Z:           c.Pos[2],
			ZBottom:     c.ZBottom(),
			ZTop:        c.ZTop(),
			Big:         big[c.VolumeIndex],
		}
		if c.HasShape {
			z.Polygon = c.Polygon
		}
		entry.Zones = append(entry.Zones, z)
	}
	sort.Slice(entry.Zones, func(i, j int) bool {
		return entry.Zones[i].VolumeIndex < entry.Zones[j].VolumeIndex
	})
	return entry, nil
}
