package main

// cmd_backfill_replay.go — sous-commande `levelup backfill-replay`.
//
// ELLE CONSTRUIT LES ARTEFACTS DE REJEU 2D de tous les films du cache local
// (data/cache/replays/{title}/{short8}.json), par la LIBRAIRIE internal/replaybuild —
// jamais en exécutant un CLI (règle admin_actions : pas de spawn, conflit DuckDB
// mono-process ; et un exec perdrait le verrou process filmdec).
//
// # ELLE NE PASSE PAS PAR LE RÉGLAGE « OÙ SE CONSTRUIT UN REJEU », ET C'EST VOULU
//
// Le réglage `replay_build_location` (local / worker / off) arbitre les chemins de
// SERVICE : le fil de l'eau post-sync et l'action admin. Cette commande est un outil
// d'OPÉRATEUR — celui qui la tape a déjà décidé où il construit, sur SA machine, avec
// SES films en cache. La soumettre au réglage lui ferait mettre en file un rattrapage de
// 951 matchs dont aucun ouvrier n'a les films (le master plan §1 est explicite : « un
// ouvrier neuf ne doit pas retélécharger 23 Go pour rattraper l'historique »), ou refuser
// de tourner sur un poste réglé en « worker ». Comportement direct, donc, et inchangé.
//
// # ELLE EST 100 % HORS LIGNE — NI RÉSEAU, NI TOKENS, NI CDN
//
// La source est le cache film sur disque (filmcache) ; la seule base touchée est le
// registre partagé, en LECTURE COURTE au démarrage (le handle est relâché AVANT le
// premier décodage — la passe dure des heures, on ne tient pas un lock DuckDB pendant
// des heures).
//
// # ELLE EST REPRENABLE, ET LA CLÉ EST replay.SchemaVersion
//
// Un artefact présent qui porte la version de schéma courante est sauté ; toute version
// antérieure se lit « à re-cuire » (cf. document.go). `--force` re-cuit tout.
//
// # ELLE PASSE LES GROS FILMS EN DERNIER (même règle que backfill-killsource)
//
// Le coût croît avec le nombre de chunks : trier par coût croissant produit la
// quasi-totalité de la valeur avant les films les plus chers — une interruption laisse un
// résultat presque complet.
//
// # « CARTE HORS CATALOGUE » EST UN ÉCHEC VOULU, COMPTÉ À PART
//
// Les cartes Forge n'ont pas de bornes de déquantification (leur canevas n'est pas la
// carte) : ces films ne sont PAS constructibles aujourd'hui, et les compter comme des
// erreurs noierait les vraies erreurs de décodage.
//
// Usage :
//
//	levelup backfill-replay --dry-run              # le plan de passe, aucune écriture
//	levelup backfill-replay --limit 25             # pilote : les 25 films les moins chers
//	levelup backfill-replay                        # tout (~8 h, artefacts ~2 Go)
//	levelup backfill-replay --force                # re-cuit même les artefacts à jour
//	levelup backfill-replay --only-existing         # re-cuit UNIQUEMENT les artefacts déjà là
//
// # `--only-existing` EST LA PASSE D'APRÈS UN BUMP DE SCHÉMA
//
// Une montée de `replay.SchemaVersion` périme d'un coup TOUS les artefacts : une passe
// ordinaire repartirait alors sur les 951 films du cache (des heures) là où l'on veut
// seulement remettre à niveau ce qui est déjà servi. Ce drapeau borne la passe aux matchs
// qui ont DÉJÀ un artefact sur disque, quelle que soit sa version.
//
// Le cache de films se résout par `--cache`, sinon `LEVELUP_LEGACY_FILM_CACHE_DIR`,
// sinon `<repo>/data/cache` (même règle que backfill-killsource).

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
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// replayCandidat : un film du cache joint à son match du registre.
type replayCandidat struct {
	matchID  string
	mapNames []string // candidats du plus fiable au moins fiable (asset EN, puis brut)
	chunks   int
}

// replayBackfillOptions : les réglages de la passe.
type replayBackfillOptions struct {
	titleSlug string
	cacheDir  string
	limit     int
	force     bool
	dryRun    bool
	// onlyExisting borne la passe aux matchs qui portent DÉJÀ un artefact sur disque.
	onlyExisting bool
}

// replayBackfillReport : le rapport par catégories exigé par le gate du lot 6.
type replayBackfillReport struct {
	construits    int
	dejaAJour     int
	horsCatalogue int // échec VOULU (cartes Forge sans bornes), compté À PART
	horsRegistre  int // film en cache sans ligne match_registry
	sansArtefact  int // écarté par --only-existing : aucun artefact sur disque
	erreurs       int
}

func runBackfillReplay(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-replay", flag.ExitOnError)
	o := replayBackfillOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.StringVar(&o.cacheDir, "cache", "", "racine du cache de films (defaut : LEVELUP_LEGACY_FILM_CACHE_DIR puis <repo>/data/cache)")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de films construits (0 = tous) — les moins chers d abord")
	fs.BoolVar(&o.force, "force", false, "re-cuire meme les artefacts deja a la version de schema courante")
	fs.BoolVar(&o.dryRun, "dry-run", false, "afficher le plan de passe sans rien construire")
	fs.BoolVar(&o.onlyExisting, "only-existing", false,
		"ne traiter que les matchs qui ont deja un artefact sur disque (passe d apres un bump de schema)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	cacheRoot := resoudreCacheFilms(cfg, o.cacheDir)
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)

	candidats, horsRegistre, err := replaysACuire(ctx, cfg, pr, cacheRoot, o)
	if err != nil {
		return err
	}
	report := replayBackfillReport{horsRegistre: horsRegistre}

	// Le filtre « déjà à jour » se fait AVANT le tri/limit : un pilote --limit 25 doit
	// livrer 25 constructions réelles, pas 25 skips.
	aFaire := make([]replayCandidat, 0, len(candidats))
	for _, c := range candidats {
		path := pr.ReplayArtifactPath(o.titleSlug, c.matchID)
		if o.onlyExisting {
			if _, err := os.Stat(path); err != nil {
				report.sansArtefact++
				continue
			}
		}
		if !o.force && replaybuild.ArtifactUpToDate(path) {
			report.dejaAJour++
			continue
		}
		aFaire = append(aFaire, c)
	}
	// LES GROS EN DERNIER. À coût égal, l'identifiant départage — une passe doit être
	// reproductible, y compris dans son ordre.
	sort.Slice(aFaire, func(i, j int) bool {
		if aFaire[i].chunks != aFaire[j].chunks {
			return aFaire[i].chunks < aFaire[j].chunks
		}
		return aFaire[i].matchID < aFaire[j].matchID
	})
	if o.limit > 0 && len(aFaire) > o.limit {
		aFaire = aFaire[:o.limit]
	}

	fmt.Printf("films a construire : %d (cache %s, %d deja a jour, %d hors registre, %d sans artefact)\n",
		len(aFaire), cacheRoot, report.dejaAJour, report.horsRegistre, report.sansArtefact)
	if o.dryRun {
		afficherPlanReplay(aFaire)
		return nil
	}
	if len(aFaire) == 0 {
		afficherRapportReplay(report)
		return nil
	}

	builder, err := replaybuild.NewBuilder(cfg.RepoRoot, o.titleSlug)
	if err != nil {
		return err
	}
	// LES FAITS SE LISENT EN UNE SEULE OUVERTURE, AVANT LA BOUCLE : le decodage d'un film dure
	// des secondes a des minutes, et tenir un handle shared pendant une passe de masse est
	// exactement ce que le modele mono-process interdit.
	faits := chargerFaitsReplay(ctx, pr, o.titleSlug, aFaire)
	debut := time.Now()
	for i, c := range aFaire {
		filmDir := filmcache.ChunkDir(cacheRoot, titlePkg.FilmShortMatchID(c.matchID))
		out, berr := builder.BuildMatch(c.matchID, c.mapNames, filmDir, faits[c.matchID])
		switch {
		case berr == nil:
			report.construits++
			fmt.Printf("  [%d/%d] %s : %d tracks, %d octets (%s)\n",
				i+1, len(aFaire), c.matchID, out.Tracks, out.Bytes, out.Module)
		case errors.Is(berr, replaybuild.ErrMapNotInCatalog):
			report.horsCatalogue++
			fmt.Printf("  [%d/%d] %s : carte hors catalogue (%v) — echec voulu\n",
				i+1, len(aFaire), c.matchID, c.mapNames)
		default:
			report.erreurs++
			fmt.Printf("  [%d/%d] %s : ERREUR %v\n", i+1, len(aFaire), c.matchID, berr)
		}
	}
	fmt.Printf("passe terminee en %s\n", time.Since(debut).Round(time.Second))
	afficherRapportReplay(report)
	return nil
}

// replaysACuire joint le cache film au registre : pour chaque film du cache, le match et
// ses identités de carte candidates. LA LECTURE DE BASE EST COURTE : les deux handles
// (shared RO, metadata RO) sont relâchés au retour, AVANT tout décodage.
func replaysACuire(
	ctx context.Context, cfg *config.AppConfig, pr *titlePkg.PathResolver, cacheRoot string, o replayBackfillOptions,
) ([]replayCandidat, int, error) {
	shorts, err := filmcache.ListShortIDs(cacheRoot)
	if err != nil {
		return nil, 0, err
	}
	if len(shorts) == 0 {
		return nil, 0, fmt.Errorf("aucun film dans le cache %s — cette passe est HORS LIGNE, elle n'a pas d'autre source", cacheRoot)
	}

	registre, err := registreParShort(ctx, cfg, pr, o.titleSlug)
	if err != nil {
		return nil, 0, err
	}

	horsRegistre := 0
	out := make([]replayCandidat, 0, len(shorts))
	for _, short := range shorts {
		entry, ok := registre[short]
		if !ok {
			horsRegistre++
			continue
		}
		n, ok := compterChunks(cacheRoot, short)
		if !ok {
			continue // manifeste illisible/vide : rien à décoder
		}
		out = append(out, replayCandidat{matchID: entry.matchID, mapNames: entry.mapNames, chunks: n})
	}
	return out, horsRegistre, nil
}

// registreEntry : un match du registre, avec ses identités de carte candidates.
type registreEntry struct {
	matchID  string
	mapNames []string
}

// registreParShort lit match_registry (RO) et metadata.asset_translations (RO,
// best-effort) et indexe par forme courte de match_id.
//
// L'ordre des candidats est celui de ReplayMapRepo : nom d'asset EN (résolu, fiable même
// quand map_name porte un UUID brut) PUIS map_name brut. metadata peut être tenue RW par
// un serveur qui tourne : son ouverture en échec DÉGRADE (candidat brut seul), jamais ne
// bloque la passe.
func registreParShort(ctx context.Context, cfg *config.AppConfig, pr *titlePkg.PathResolver, titleSlug string) (map[string]registreEntry, error) {
	sharedPath := pr.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("open shared RO (%s): %w (serveur en ecriture ? reessayer)", sharedPath, err)
	}
	defer release()

	enParMapID := nomsAssetsEN(ctx, pr, titleSlug)

	rows, err := db.QueryContext(ctx, `SELECT match_id, map_name, map_id FROM match_registry ORDER BY match_id`)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]registreEntry{}
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		if err := rows.Scan(&id, &rawName, &mapID); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		var names []string
		if en := enParMapID[strings.TrimSpace(mapID.String)]; en != "" {
			names = append(names, en)
		}
		if raw := strings.TrimSpace(rawName.String); raw != "" {
			names = append(names, raw)
		}
		out[titlePkg.FilmShortMatchID(id)] = registreEntry{matchID: id, mapNames: names}
	}
	return out, rows.Err()
}

// nomsAssetsEN charge la table map_id -> nom EN depuis metadata.asset_translations.
// BEST-EFFORT : metadata absente ou tenue RW par un autre process → map vide + message,
// la passe continue sur les map_name bruts du registre.
func nomsAssetsEN(ctx context.Context, pr *titlePkg.PathResolver, titleSlug string) map[string]string {
	metaPath := pr.MetadataDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(metaPath)
	if err != nil {
		fmt.Printf("metadata illisible (%v) — resolution des noms EN degradee (map_name brut seul)\n", err)
		return nil
	}
	defer release()
	rows, err := db.QueryContext(ctx,
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = 'map' AND lang = 'en-US'`)
	if err != nil {
		fmt.Printf("asset_translations illisible (%v) — resolution des noms EN degradee\n", err)
		return nil
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		out[strings.TrimSpace(id)] = strings.TrimSpace(name)
	}
	return out
}

// afficherPlanReplay : le plan de passe, avec sa queue de films chers en évidence.
func afficherPlanReplay(candidats []replayCandidat) {
	total, gros := 0, 0
	for _, c := range candidats {
		total += c.chunks
		if c.chunks > 50 {
			gros++
		}
	}
	fmt.Printf("  chunks a decoder : %d au total, %d film(s) au-dela de 50 chunks (passes en dernier)\n", total, gros)
	for i, c := range candidats {
		if i >= 5 && i < len(candidats)-3 {
			continue
		}
		if i == 5 {
			fmt.Println("  ...")
		}
		fmt.Printf("  %-40s %3d chunks  %v\n", c.matchID, c.chunks, c.mapNames)
	}
}

// afficherRapportReplay : le rapport final par catégories (gate lot 6 : la couverture se
// publie, « carte hors catalogue » est un échec VOULU compté à part des erreurs).
func afficherRapportReplay(r replayBackfillReport) {
	fmt.Println("rapport de passe :")
	fmt.Printf("  construits           %d\n", r.construits)
	fmt.Printf("  deja a jour          %d\n", r.dejaAJour)
	fmt.Printf("  carte hors catalogue %d (echec voulu : cartes sans bornes, Forge en tete)\n", r.horsCatalogue)
	fmt.Printf("  hors registre        %d (film en cache sans match en base)\n", r.horsRegistre)
	fmt.Printf("  sans artefact        %d (ecartes par --only-existing)\n", r.sansArtefact)
	fmt.Printf("  erreurs de decodage  %d\n", r.erreurs)
}

// chargerFaitsReplay lit, en UNE ouverture RO, ce que la base sait de chaque match du lot :
// les lignes de match (pont d'identite des joueurs), les scores des deux camps (identite des
// camps) et le nom de variante (famille d'objectif).
//
// LA LECTURE EST COURTE ET FERMEE AVANT TOUT DECODAGE, comme celle du registre : une passe de
// masse ne tient jamais un handle shared pendant qu'elle decode. Une base indisponible (serveur
// en ecriture) DEGRADE la passe — les artefacts sortent sans compteurs de joueur ni actions
// d'objectif — plutot que de la bloquer.
func chargerFaitsReplay(ctx context.Context, pr *titlePkg.PathResolver, titleSlug string,
	aFaire []replayCandidat) map[string]port.MatchFacts {
	out := make(map[string]port.MatchFacts, len(aFaire))
	sharedPath := pr.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		fmt.Printf("  faits de match indisponibles (%v) — artefacts sans compteurs de joueur ni actions d'objectif\n", err)
		return out
	}
	defer release()

	repo := duckdb.NewReplayFactsRepo(db)
	manquants := 0
	for _, c := range aFaire {
		facts, ferr := repo.FactsForMatch(ctx, c.matchID)
		if ferr != nil {
			slog.WarnContext(ctx, "backfill-replay: faits de match illisibles",
				"err", ferr, "match_id", c.matchID)
			manquants++
			continue
		}
		if facts.Empty() {
			manquants++
			continue
		}
		out[c.matchID] = facts
	}
	fmt.Printf("faits de match charges : %d/%d (%d sans faits)\n", len(out), len(aFaire), manquants)
	return out
}
