// Package ops — seed_demo.go : génération du jeu de données démo pour demo.levelup.info.
//
// Portage de scripts/prepare_demo_data.py (supprimé au commit c03707aa lors
// du nettoyage Python legacy). Extrait N matchs récents d'un joueur source,
// anonymise son xuid en "0000000000000000", recrée les vues V6, et écrit les
// configs db_profiles/app_settings.
//
// Utilisation :
//
//	docker compose run --rm levelup levelup seed-demo --gamertag JGtm \
//	    --max-matches 50 --service-tag SPTA --out data/demo
//
// Pipeline :
//  1. Résoudre source_xuid depuis db_profiles.json
//  2. Sélectionner les N match_ids les plus récents (ORDER BY start_time DESC)
//  3. Copier metadata.duckdb intégralement (référentiels neutres)
//  4. Extraire shared_matches_v2.duckdb filtré + anonymisé
//  5. Extraire player stats.duckdb filtré + anonymisé
//  6. Extraire médias (5 max) + media_registry.json
//  7. Écrire db_profiles.json + app_settings.json
//
// Idempotent : un 2e run écrase les fichiers de sortie.
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// SeedDemoOptions configure la génération démo.
type SeedDemoOptions struct {
	// Sources (résolues depuis db_profiles.json si vides via ResolveFromProfiles).
	SourcePlayerDB string // ex: data/players/JGtm/stats.duckdb
	SourceSharedDB string // ex: data/warehouse/shared_matches_v2.duckdb
	SourceMetaDB   string // ex: data/warehouse/metadata.duckdb
	SourceXUID     string // xuid réel du joueur source (lu depuis profiles si vide)

	// Cible et anonymisation.
	OutDir       string // ex: data/demo
	MaxMatches   int    // 50 par défaut
	DemoGamertag string // "DEMO" par défaut
	DemoXUID     string // "0000000000000000" par défaut
	SourceLabel  string // gamertag source affiché en log (ex: "JGtm")
	ServiceTag   string // ex: "SPTA" — profile_service_tag dans app_settings
	IncludeMedia bool   // true par défaut côté CLI
	MaxMedia     int    // 5 par défaut

	// SourcePlayersDir : racine des player DBs (data/titles/{slug}/players) où
	// chercher une identité Spartan à emprunter pour la démo (emblem/banner/
	// backdrop/adornment). Vide → emprunt d'identité désactivé.
	SourcePlayersDir string

	// ProfilesPath + RepoRoot : pour seeder les player DB des coéquipiers
	// principaux (DemoPlayer2/3). ProfilesPath = db_profiles.json source (résout
	// xuid→db_path) ; RepoRoot = racine pour rendre ces chemins absolus. Vides →
	// démo mono-joueur (pas de player DB coéquipier).
	ProfilesPath string
	RepoRoot     string

	// TitleSlug : titre seedé (défaut → title.DefaultSlug). Détermine le
	// sous-répertoire de sortie : le titre par défaut écrit au layout PLAT legacy
	// (OutDir/warehouse, OutDir/players/DEMO — byte-identique à la démo mono-titre) ;
	// un titre additionnel écrit sous OutDir/titles/{slug}/ (miroir du PathResolver
	// prod). Cf. demoTitleSubdir.
	TitleSlug string
	// SkipConfigs : ne pas écrire db_profiles.json/app_settings.json à la fin (true
	// quand l'orchestrateur multi-titre les écrit une seule fois, en v3, après tous
	// les titres). Faux (défaut) → écrit les configs mono-titre v2.1 comme avant.
	SkipConfigs bool
	// IgnoreManifest : forcer la sélection dynamique même si un manifeste figé existe
	// (config/demo/<gamertag>/<slug>.json). Faux (défaut) → manifeste auto-détecté et
	// utilisé s'il est présent (corpus reproductible).
	IgnoreManifest bool
}

// SeedDemoResult résume l'exécution.
type SeedDemoResult struct {
	OutDir         string
	MatchIDs       []string
	MetadataCopied bool
	SharedRows     map[string]int // table → rows insérées
	PlayerRows     map[string]int
	// PrestigeRows : lignes Prestige/Progression générées (player DB + shared_social),
	// par table. Vide si le corpus démo ne contient aucun match du joueur démo.
	PrestigeRows   map[string]int
	MediaCopied    int
	ConfigsWritten bool
	Frozen         bool // true si le corpus vient d'un manifeste figé (vs sélection dynamique)
	Duration       time.Duration
	// SeededPlayers : player DB démo effectivement seedées pour ce titre (DemoPlayer
	// + coéquipiers). Agrégées par l'orchestrateur multi-titre pour écrire db_profiles
	// v3 (une entrée par couple titre × gamertag démo).
	SeededPlayers []seededDemoPlayer
}

// Constantes par défaut.
const (
	DefaultDemoXUID     = "0000000000000000"
	DefaultDemoGamertag = "DEMO"
	DefaultMaxMatches   = 50
	DefaultMaxMedia     = 5
)

// matchIDInClause est la clause WHERE templated pour filtrer les tables shared/player
// par match_ids. Factorisé pour éviter la duplication (9 occurrences dans les tables maps).
const matchIDInClause = "match_id IN (%s)"

// Table player principale, référencée aussi par anonymizeXUIDInTables (4 occurrences).
const playerEnrichmentTable = "player_match_enrichment"

// Tables shared/player référencées plus de 3 fois dans seed_demo + tests.
const (
	tableMatchRegistry     = "match_registry"
	tableMatchParticipants = "match_participants"
)

// extractTable décrit une table à extraire d'une DB source vers la DB démo.
//
// appendOnly : la table source est en schéma append-only (colonnes techniques
// id + written_at + vue <table>_latest). On l'extrait SANS id/written_at (forme
// "pré-migration") pour que RunForDB la reconstruise append-only et (re)crée la
// vue _latest à jour — sinon la migration skip (check `id` déjà présent) et la
// vue n'est jamais créée dans la DB démo (cf. incident 2026-06-05 : home 500
// sur match_csrs.written_at / player_csr_snapshots_latest).
type extractTable struct {
	name       string
	where      string
	appendOnly bool
}

// Tables shared à extraire avec leur clause WHERE (les ? sont les match_ids
// formatés en littéraux SQL via formatIDsLiteral — DuckDB ne supporte pas
// les placeholders dans CREATE TABLE AS SELECT).
//
// xuid_aliases : SELECT * WHERE xuid IN (xuids des match_participants).
var sharedTablesWhere = []extractTable{
	{name: tableMatchRegistry, where: matchIDInClause},
	{name: tableMatchParticipants, where: matchIDInClause},
	{name: "medals_earned", where: matchIDInClause},
	{name: "highlight_events", where: matchIDInClause},
	{name: "weapon_kills", where: matchIDInClause},
	{name: "killer_victim_pairs", where: matchIDInClause},
	// match_kill_events : le kill-feed canonique, écrit en parallèle de la table ci-dessus
	// depuis le 2026-08-02 (double écriture datée). L'omettre livrerait une démo dont les
	// surfaces bâties sur la table canonique sont vides — panne silencieuse, visible seulement
	// à l'écran, exactement le mode d'échec que la ligne du dessus a déjà connu.
	//
	// appendOnly:FALSE, pour la raison de match_objective_stats plus bas : la table est créée
	// DIRECTEMENT en forme append-only (jamais convertie par ApplyAppendOnlyRebuild), donc
	// ses colonnes techniques doivent voyager — la vue _latest trie sur written_at et id.
	{name: "match_kill_events", where: matchIDInClause},
	{name: "xuid_aliases", where: "xuid IN (SELECT DISTINCT xuid FROM match_participants WHERE match_id IN (%s))"},
	{name: "match_csrs", where: matchIDInClause, appendOnly: true},
	// match_objective_stats : stats objectifs par joueur/match (CTF, Zones, Oddball,
	// Stockpile, Extraction, VIP). Sans elle, toutes les surfaces « objectifs » de la
	// démo sont vides (scoreboard MatchView Q12 LEFT JOIN match_objective_stats_latest).
	//
	// appendOnly:FALSE — et c'est DÉLIBÉRÉ, contrairement à match_csrs juste au-dessus.
	// Le motif `* EXCLUDE (id, written_at)` ne vaut QUE pour les tables converties par
	// migration.ApplyAppendOnlyRebuild (match_skill_rank, player_csr_snapshots,
	// match_csrs, pve_match_stats, …), dont la migration DÉTECTE l'absence de `id` et
	// reconstruit. match_objective_stats, elle, est créée DIRECTEMENT en forme
	// append-only (steps_shared_objective_stats.go) avec un `CREATE TABLE IF NOT EXISTS` :
	// sur une table déjà présente elle est un no-op, donc les colonnes techniques ne
	// seraient JAMAIS rajoutées — et la vue _latest, dont le QUALIFY trie
	// `written_at DESC, id DESC`, échouerait au binder et ferait échouer tout le seed.
	// On copie donc la table TELLE QUELLE (id + written_at inclus) : la vue _latest
	// recréée par applyMigrationsOnPath déduplique alors exactement comme en prod.
	{name: "match_objective_stats", where: matchIDInClause},
	// Tables Halo 5-spécifiques (absentes côté Infinite → extraction best-effort, la
	// table source manquante est ignorée par extractSharedTables). Toutes filtrées
	// par match_id et non append-only (cf. probe schéma 2026-06-27).
	{name: "match_commendations", where: matchIDInClause}, // commendations natives par match (xuid)
	{name: "kill_positions", where: matchIDInClause},      // positions monde du kill (killer_xuid)
	{name: "weapon_accuracy", where: matchIDInClause},     // précision par arme par match (xuid)
}

// Tables player à extraire. sessions/career_progression/player_csr_snapshots
// copiés intégralement (pas filtrés par match_id car données globales joueur).
var playerTablesWhere = []extractTable{
	{name: playerEnrichmentTable, where: matchIDInClause},
	{name: "match_citations", where: matchIDInClause},
	{name: "sessions", where: "1=1"},
	{name: "career_progression", where: "1=1"},
	{name: "sync_meta", where: "key NOT IN ('msal_token_cache')"},
	{name: "match_skill_rank", where: matchIDInClause, appendOnly: true},
	{name: "player_csr_snapshots", where: "1=1", appendOnly: true},
	{name: "battlepass_snapshots", where: "1=1"},
}

// extractSelectExpr retourne l'expression SELECT pour une table extraite.
// Les tables append-only sont extraites sans leurs colonnes techniques id /
// written_at afin que les migrations canoniques (RunForDB) les reconstruisent.
func extractSelectExpr(appendOnly bool) string {
	if appendOnly {
		return "* EXCLUDE (id, written_at)"
	}
	return "*"
}

// applyMigrationsOnPath ouvre la DB DuckDB à dbPath et applique les migrations
// canoniques du target. Utilisé après l'extraction démo pour reconstruire les
// tables append-only et créer les vues _latest (match_csrs_latest,
// match_skill_rank_latest, player_csr_snapshots_latest, …) à jour.
func applyMigrationsOnPath(dbPath string, target migration.TargetDB) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open %s for migrations: %w", dbPath, err)
	}
	defer db.Close()
	if err := migration.RunForDB(db, target); err != nil {
		return fmt.Errorf("migrations %s (%s): %w", target, dbPath, err)
	}
	return nil
}

// applyTitleMigrationsOnPath applique les migrations d'un TITRE (RunForTitleDB) — pour
// les schémas dont les RACINES sont enregistrées PAR TITRE et non globalement. C'est le
// cas de shared_social : `create_base_shared_social_schema` (table media_files,
// media_match_associations) vit dans les steps du titre (halo_infinite/migrations) et
// n'est PAS exécutée par RunForDB (global) → une shared_social démo migrée via RunForDB
// n'a pas de media_files (insert démo échoue). On force donc le set de migrations du
// titre par défaut (schéma shared_social title-agnostique).
func applyTitleMigrationsOnPath(dbPath, slug string, target migration.TargetDB) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open %s for title migrations: %w", dbPath, err)
	}
	defer db.Close()
	if err := migration.RunForTitleDB(db, slug, target); err != nil {
		return fmt.Errorf("title migrations %s/%s (%s): %w", slug, target, dbPath, err)
	}
	return nil
}

// SeedDemo exécute le pipeline complet.
func SeedDemo(ctx context.Context, opts SeedDemoOptions) (SeedDemoResult, error) {
	start := time.Now()
	res := SeedDemoResult{
		OutDir:       opts.OutDir,
		SharedRows:   map[string]int{},
		PlayerRows:   map[string]int{},
		PrestigeRows: map[string]int{},
	}

	if err := validateSeedDemoOpts(&opts); err != nil {
		return res, fmt.Errorf("seed-demo: %w", err)
	}

	// titleOut : sous-répertoire de sortie title-scopé. Titre par défaut → OutDir
	// plat (byte-identique mono-titre) ; titre additionnel → OutDir/titles/{slug}/.
	titleOut := demoTitleSubdir(opts.OutDir, opts.TitleSlug)

	slog.InfoContext(ctx, "seed-demo: démarrage",
		"title_slug", opts.TitleSlug,
		"source_gamertag", opts.SourceLabel,
		"source_xuid", opts.SourceXUID,
		"max_matches", opts.MaxMatches,
		"out_dir", titleOut,
		"include_media", opts.IncludeMedia,
	)

	// Phases 0+1+1b : corpus (figé via manifeste ou dynamique) + roster démo.
	matchIDs, roster, frozen, err := resolveDemoCorpusAndRoster(ctx, opts)
	res.Frozen = frozen
	if err != nil {
		return res, err
	}
	res.MatchIDs = matchIDs

	// Phases 2-4 : warehouse démo (metadata copiée + shared extrait + anonymisé + migré).
	metaCopied, sharedRows, err := buildDemoWarehouse(ctx, opts, titleOut, matchIDs, roster)
	res.MetadataCopied = metaCopied
	if err != nil {
		return res, err
	}
	res.SharedRows = sharedRows

	// 5. Player DB du roster (DemoPlayer principal + coéquipiers principaux).
	seeded, playerRows, err := seedDemoPlayerDBs(ctx, opts, titleOut, matchIDs, roster)
	if err != nil {
		return res, err
	}
	res.PlayerRows = playerRows
	res.SeededPlayers = seeded

	// 5b. shared_social démo : reconstruite AVANT les phases qui y écrivent (médias
	// puis Prestige). Un seul propriétaire du cycle de vie du fichier → un reseed ne
	// peut pas empiler deux générations de lignes.
	outSocial := filepath.Join(titleOut, "warehouse", "shared_social.duckdb")
	if err := rebuildDemoSharedSocial(ctx, outSocial); err != nil {
		return res, fmt.Errorf("seed-demo: %w", err)
	}

	// 6. Médias (DemoPlayer principal).
	if opts.IncludeMedia {
		mediaCount, mediaErr := seedDemoMediaFiles(ctx, opts, titleOut, matchIDs)
		if mediaErr != nil {
			slog.WarnContext(ctx, "seed-demo: extraction média partielle", "err", mediaErr, "copied", mediaCount)
		}
		res.MediaCopied = mediaCount
	}

	// 6b. Échantillons Prestige (arcs, objectifs, points de progression, séries,
	// records, jalons, escouade) dérivés du corpus démo. Sans cette phase les pages
	// Prestige/Ascension sont accessibles (prestige_enabled=true) mais vides.
	prestigeRows, prestigeErr := seedDemoPrestige(ctx, titleOut, opts.TitleSlug, matchIDs)
	if prestigeErr != nil {
		return res, fmt.Errorf("seed-demo: %w", prestigeErr)
	}
	res.PrestigeRows = prestigeRows

	// 7. Configs (db_profiles avec les 3 profils démo + app_settings). SKIP quand
	// l'orchestrateur multi-titre les écrira une fois (v3) après tous les titres.
	if !opts.SkipConfigs {
		if err := writeDemoConfigsMulti(opts.OutDir, seeded, opts.ServiceTag, res.MediaCopied > 0); err != nil {
			return res, fmt.Errorf("seed-demo: write configs: %w", err)
		}
		res.ConfigsWritten = true
	}

	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "seed-demo: terminé",
		"duration", res.Duration,
		"matches", len(matchIDs),
		"media", res.MediaCopied,
		"prestige_rows", res.PrestigeRows,
	)
	return res, nil
}

// resolveDemoCorpusAndRoster sélectionne le corpus (figé via manifeste ou dynamique
// « N récents ») puis construit le roster démo (source + coéquipiers principaux + autres
// participants → identités démo stables, anti-fuite). Phases 0+1+1b de SeedDemo (K2d).
// frozen=true si un manifeste figé a été utilisé.
func resolveDemoCorpusAndRoster(ctx context.Context, opts SeedDemoOptions) (matchIDs []string, roster []demoRosterEntry, frozen bool, err error) {
	// 0. Manifeste figé (optionnel) : config/demo/<gamertag>/<slug>.json → corpus reproductible.
	var manifest *DemoManifest
	if !opts.IgnoreManifest && opts.RepoRoot != "" && opts.SourceLabel != "" {
		mp := titlePkg.NewPathResolver(opts.RepoRoot).DemoManifestPath(opts.SourceLabel, opts.TitleSlug)
		m, found, mErr := LoadDemoManifest(mp)
		if mErr != nil {
			return nil, nil, false, fmt.Errorf("seed-demo: %w", mErr)
		}
		if found {
			manifest = m
			frozen = true
			slog.InfoContext(ctx, "seed-demo: manifeste figé chargé", "path", mp, "corpus", len(m.CorpusMatchIDs()))
		}
	}

	// 1. Corpus = solo ∪ squad ∪ ranked ∪ media (ordre solo→squad→ranked→media, dédupliqué).
	var dc dynamicCorpus
	if manifest != nil {
		dc = dynamicCorpus{
			solo:   manifest.Corpus.SoloMatchIDs,
			squad:  manifest.Corpus.SquadMatchIDs,
			ranked: manifest.Corpus.RankedMatchIDs,
			media:  manifest.Corpus.MediaMatchIDs,
		}
	} else {
		dc = selectDynamicCorpus(ctx, opts)
	}
	matchIDs = unionMatchIDs(dc.solo, dc.squad, dc.ranked, dc.media)
	if len(matchIDs) == 0 {
		return nil, nil, frozen, fmt.Errorf("seed-demo: aucun match trouvé pour xuid=%s dans %s",
			opts.SourceXUID, opts.SourceSharedDB)
	}
	slog.InfoContext(ctx, "seed-demo: corpus sélectionné",
		"frozen", manifest != nil, "total", len(matchIDs),
		"solo", len(dc.solo), "squad", len(dc.squad), "ranked", len(dc.ranked), "media", len(dc.media))

	// 1b. Roster démo (dérivé du corpus : figé ⇒ déterministe + couverture xuids réels).
	roster, err = buildDemoRoster(ctx, opts.SourceSharedDB, dc.squad, matchIDs, opts.SourceXUID, 2)
	if err != nil {
		return nil, nil, frozen, fmt.Errorf("seed-demo: build roster: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: roster démo construit", "participants", len(roster))
	return matchIDs, roster, frozen, nil
}

// buildDemoWarehouse copie la metadata, extrait le shared filtré sur le corpus, anonymise
// universellement (xuid+gamertag → identités démo, aucun vrai gamertag ne fuite) puis migre
// le shared (append-only + vues _latest, idempotent). Phases 2-4 de SeedDemo (K2d).
// metaCopied=true dès que la copie metadata a réussi (préservé même si une phase suivante échoue).
func buildDemoWarehouse(ctx context.Context, opts SeedDemoOptions, titleOut string,
	matchIDs []string, roster []demoRosterEntry) (metaCopied bool, sharedRows map[string]int, err error) {
	// 2. Copie metadata.duckdb.
	outMeta := filepath.Join(titleOut, "warehouse", "metadata.duckdb")
	if err = copyMetadataFile(opts.SourceMetaDB, outMeta); err != nil {
		return false, nil, fmt.Errorf("seed-demo: copy metadata: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: metadata copiée", "out", outMeta)

	// 3. Extraction shared.
	outShared := filepath.Join(titleOut, "warehouse", "shared_matches_v2.duckdb")
	sharedRows, err = extractSharedTables(ctx, opts.SourceSharedDB, outShared, matchIDs, opts.SourceXUID, opts.DemoXUID)
	if err != nil {
		return true, nil, fmt.Errorf("seed-demo: extract shared: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: shared extraite", "out", outShared, "rows", sharedRows)

	// 3b. Anonymisation universelle (Escouade, kill-feed, Face-à-face — aucune fuite).
	if err = anonymizeSharedUniversal(ctx, outShared, roster); err != nil {
		return true, sharedRows, fmt.Errorf("seed-demo: anonymisation universelle: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: anonymisation universelle appliquée", "mapped", len(roster))

	// 4. Migration shared (reconstruit tables append-only + vues _latest, idempotent).
	if err = applyMigrationsOnPath(outShared, migration.TargetShared); err != nil {
		return true, sharedRows, fmt.Errorf("seed-demo: migrate shared: %w", err)
	}
	return true, sharedRows, nil
}

// seedDemoPlayerDBs seede les player DB du roster : DemoPlayer principal (source) +
// coéquipiers principaux (DemoPlayer2/3, vraies stats perf/LUSR pour la page Escouade),
// chacune filtrée sur le corpus + anonymisée vers son xuid démo + migrée. Identité Spartan
// empruntée pour le principal (le source n'a pas de customization propre). Phase 5 (K2d).
// playerRows = compteurs du DemoPlayer principal (i==0).
func seedDemoPlayerDBs(ctx context.Context, opts SeedDemoOptions, titleOut string,
	matchIDs []string, roster []demoRosterEntry) (seeded []seededDemoPlayer, playerRows map[string]int, err error) {
	mains := rosterMains(roster)
	for i, m := range mains {
		srcPlayerDB := opts.SourcePlayerDB
		if i > 0 {
			// Title-aware : player DB du coéquipier POUR LE TITRE courant (ses stats H5
			// quand on seede H5). Fallback tous-titres si absent du titre courant.
			rel, found, rerr := resolvePlayerDBByXUIDForTitle(opts.ProfilesPath, opts.TitleSlug, m.SourceXUID)
			if rerr != nil || !found {
				slog.WarnContext(ctx, "seed-demo: player DB coéquipier introuvable pour ce titre, skip",
					"title", opts.TitleSlug, "xuid", m.SourceXUID, "gamertag", m.DemoGamertag, "err", rerr)
				continue
			}
			srcPlayerDB = filepath.Join(opts.RepoRoot, rel)
		}
		demoDir := demoDirForIndex(i)
		outP := filepath.Join(titleOut, "players", demoDir, "stats.duckdb")
		rows, perr := extractPlayerTables(ctx, srcPlayerDB, outP, matchIDs, m.SourceXUID, m.DemoXUID)
		if perr != nil {
			if i == 0 {
				return seeded, playerRows, fmt.Errorf("seed-demo: extract player principal: %w", perr)
			}
			slog.WarnContext(ctx, "seed-demo: extract player coéquipier échoué, skip",
				"gamertag", m.DemoGamertag, "err", perr)
			continue
		}
		if err := applyMigrationsOnPath(outP, migration.TargetPlayer); err != nil {
			return seeded, playerRows, fmt.Errorf("seed-demo: migrate player %s: %w", demoDir, err)
		}
		if i == 0 {
			playerRows = rows
			if opts.SourcePlayersDir != "" {
				if err := borrowDemoIdentity(ctx, outP, opts.SourcePlayersDir, m.DemoXUID); err != nil {
					slog.WarnContext(ctx, "seed-demo: emprunt identité échoué (non bloquant)", "err", err)
				}
			}
		}
		seeded = append(seeded, seededDemoPlayer{Dir: demoDir, XUID: m.DemoXUID, Gamertag: m.DemoGamertag})
		slog.InfoContext(ctx, "seed-demo: player seedée", "dir", demoDir, "gamertag", m.DemoGamertag, "rows", rows)
	}
	return seeded, playerRows, nil
}

// seedDemoMediaFiles extrait les médias du DemoPlayer principal (fichiers → dir média PLAT
// servi par ServeMediaFile ; shared_social title-scopé). Infinite : flux HLS + attribution
// par carte ; titre additionnel : clips mp4 + association réelle indexée. Phase 6 (K2d).
func seedDemoMediaFiles(ctx context.Context, opts SeedDemoOptions, titleOut string, matchIDs []string) (int, error) {
	flatMediaDir := filepath.Join(opts.OutDir, "players", DefaultDemoGamertag, "media")
	outSocial := filepath.Join(titleOut, "warehouse", "shared_social.duckdb")
	// shared_social SOURCE (prod) : même dossier warehouse que shared_matches_v2.
	srcSocialDB := filepath.Join(filepath.Dir(opts.SourceSharedDB), "shared_social.duckdb")
	if opts.TitleSlug == titlePkg.DefaultSlug {
		srcMediaDir := filepath.Join(opts.RepoRoot, "data", "media", opts.SourceLabel)
		return extractDemoMedia(ctx, srcMediaDir, opts.SourceSharedDB, srcSocialDB,
			outSocial, flatMediaDir, matchIDs, DefaultDemoMainSlug, opts.MaxMedia)
	}
	// mediaBaseDir = racine média (parent du dir par gamertag) pour réancrer les file_path
	// relatifs stockés par index-media (ex. "JGtm/clip.mp4" en prod).
	mediaBaseDir := filepath.Join(opts.RepoRoot, "data", "media")
	return extractDemoMediaH5(ctx, srcSocialDB, outSocial, flatMediaDir,
		matchIDs, DefaultDemoMainSlug, opts.MaxMedia, mediaBaseDir)
}

// validateSeedDemoOpts applique les valeurs par défaut + vérifie les chemins source.
func validateSeedDemoOpts(opts *SeedDemoOptions) error {
	if opts.SourcePlayerDB == "" || opts.SourceSharedDB == "" || opts.SourceMetaDB == "" {
		return fmt.Errorf("source paths required (SourcePlayerDB, SourceSharedDB, SourceMetaDB)")
	}
	if opts.SourceXUID == "" {
		return fmt.Errorf("SourceXUID required (use ResolveSourceXUIDFromProfiles to read from db_profiles.json)")
	}
	if opts.OutDir == "" {
		return fmt.Errorf("OutDir required")
	}
	if opts.MaxMatches <= 0 {
		opts.MaxMatches = DefaultMaxMatches
	}
	if opts.DemoXUID == "" {
		opts.DemoXUID = DefaultDemoXUID
	}
	if opts.DemoGamertag == "" {
		opts.DemoGamertag = DefaultDemoGamertag
	}
	if opts.MaxMedia <= 0 {
		opts.MaxMedia = DefaultMaxMedia
	}
	if opts.TitleSlug == "" {
		opts.TitleSlug = titlePkg.DefaultSlug
	}
	for _, p := range []string{opts.SourcePlayerDB, opts.SourceSharedDB, opts.SourceMetaDB} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("source DB introuvable: %s: %w", p, err)
		}
	}
	return nil
}

// profileEntry est la structure d'un profil joueur dans db_profiles.json.
type profileEntry struct {
	XUID   string `json:"xuid"`
	DBPath string `json:"db_path"`
}

// ResolveSourceXUIDFromProfiles lit db_profiles.json et retourne le xuid + le
// chemin player DB pour le gamertag donné.
//
// Supporte 2 formats :
//   - v3.0 multi-titres : profiles.{titleSlug}.{gamertag}.{xuid,db_path} (cherche
//     d'abord dans halo_infinite, puis dans tous les autres titres).
//   - v2.1 legacy : profiles.{gamertag}.{xuid,db_path} (un seul niveau).
func ResolveSourceXUIDFromProfiles(profilesPath, gamertag string) (xuid, playerDBPath string, err error) {
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return "", "", fmt.Errorf("read profiles: %w", err)
	}

	// Tentative v3.0 (nested par titre).
	var docV3 struct {
		Version  string                             `json:"version"`
		Profiles map[string]map[string]profileEntry `json:"profiles"`
	}
	if err := json.Unmarshal(data, &docV3); err == nil && docV3.Version >= "3.0" {
		// Chercher d'abord dans le titre par défaut, puis fallback sur le premier
		// titre qui contient le gamertag.
		if titleProfiles, ok := docV3.Profiles[titlePkg.DefaultSlug]; ok {
			if p, ok := titleProfiles[gamertag]; ok {
				if p.XUID == "" {
					return "", "", fmt.Errorf("profile %q sans xuid", gamertag)
				}
				return p.XUID, p.DBPath, nil
			}
		}
		for _, titleProfiles := range docV3.Profiles {
			if p, ok := titleProfiles[gamertag]; ok {
				if p.XUID == "" {
					return "", "", fmt.Errorf("profile %q sans xuid", gamertag)
				}
				return p.XUID, p.DBPath, nil
			}
		}
		return "", "", fmt.Errorf("gamertag %q introuvable dans %s (v3.0)", gamertag, profilesPath)
	}

	// Fallback v2.1 (flat par gamertag, legacy).
	var docV2 struct {
		Profiles map[string]profileEntry `json:"profiles"`
	}
	if err := json.Unmarshal(data, &docV2); err != nil {
		return "", "", fmt.Errorf("parse profiles: %w", err)
	}
	p, ok := docV2.Profiles[gamertag]
	if !ok {
		return "", "", fmt.Errorf("gamertag %q introuvable dans %s", gamertag, profilesPath)
	}
	if p.XUID == "" {
		return "", "", fmt.Errorf("profile %q sans xuid", gamertag)
	}
	return p.XUID, p.DBPath, nil
}

// selectRecentMatchIDs retourne les N match_ids les plus récents du joueur dans shared.
func selectRecentMatchIDs(ctx context.Context, sharedDBPath, xuid string, limit int) ([]string, error) {
	db, err := sql.Open("duckdb", sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open shared: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id
		FROM match_registry mr
		JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?
		ORDER BY mr.start_time DESC
		LIMIT ?`, xuid, limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// dynamicCorpus regroupe les 4 sous-listes de la sélection démo (solo/squad/ranked/
// media), avant union. Partagé par SeedDemo (chemin dynamique) et EmitDemoManifest.
type dynamicCorpus struct {
	solo, squad, ranked, media []string
}

// selectDynamicCorpus exécute la sélection dynamique « N plus récents » (comportement
// historique). Chaque source est best-effort : un échec logge un warn et laisse la
// sous-liste vide (l'union reste exploitable tant qu'au moins une source répond).
func selectDynamicCorpus(ctx context.Context, opts SeedDemoOptions) dynamicCorpus {
	var dc dynamicCorpus
	if r, err := selectRecentMatchIDs(ctx, opts.SourceSharedDB, opts.SourceXUID, opts.MaxMatches); err != nil {
		slog.WarnContext(ctx, "seed-demo: matchs récents indisponibles", "err", err)
	} else {
		dc.solo = r
	}
	if s, err := selectSquadSessionCorpus(ctx, opts.SourcePlayerDB, DefaultSquadSessions); err != nil {
		slog.WarnContext(ctx, "seed-demo: corpus squad indisponible", "err", err)
	} else {
		dc.squad = s
	}
	if rk, err := selectRecentRankedMatchIDs(ctx, opts.SourceSharedDB, opts.SourceXUID, DefaultRankedMatches); err != nil {
		slog.WarnContext(ctx, "seed-demo: matchs classés indisponibles", "err", err)
	} else {
		dc.ranked = rk
	}
	if opts.IncludeMedia {
		srcSocialDB := filepath.Join(filepath.Dir(opts.SourceSharedDB), "shared_social.duckdb")
		if mm, err := selectMediaAnchoredMatchIDs(ctx, srcSocialDB, opts.MaxMedia); err != nil {
			slog.WarnContext(ctx, "seed-demo: corpus média-ancré indisponible", "err", err)
		} else {
			dc.media = mm
		}
	}
	return dc
}

// copyMetadataFile installe une copie bit-à-bit de src en dst en REMPLAÇANT L'INODE
// de dst : écriture dans un temporaire du même répertoire, puis retrait de dst et
// rename. Crée le répertoire parent si nécessaire.
//
// CONTRAINTE — pourquoi l'inode doit être neuf (bug prouvé en prod, 2026-07-26) :
// jusqu'à ce jour, le seed démo du déploiement s'exécutait pendant que l'ANCIEN
// conteneur `levelup-demo` tournait toujours (il n'était recréé qu'APRÈS le seed) et
// tenait les DuckDB démo ouvertes avec leur verrou. L'implémentation précédente
// ouvrait dst en O_TRUNC : elle tronquait donc un fichier DuckDB qu'un process vivant
// utilisait (la démo servait une base invalide pendant toute la fenêtre de
// déploiement), et l'ATTACH READ_ONLY de la phase Prestige
// (insertDemoMilestonesEarned) échouait ensuite sur « Conflicting lock is held » →
// table milestone_earned démo vide à chaque déploiement. Avec un inode neuf, un
// lecteur qui tient le fichier garde le sien (délié mais intact pour lui) et le
// fichier publié n'est tenu par personne — exactement la stratégie de
// extractSharedTables / extractPlayerTables (retrait avant création).
//
// STATUT depuis le 2026-07-26 : le job deploy-demo de .github/workflows/deploy.yml
// stoppe désormais `levelup-demo` pendant le seed (en miroir de la prod), donc la
// cause première est traitée en amont. Ce remplacement d'inode reste la DÉFENSE EN
// PROFONDEUR — et non plus le seul rempart : `levelup seed-demo` se lance aussi à la
// main et en local, avec un serveur démo debout, et rien dans le code Go ne peut
// vérifier qu'un lecteur n'est pas branché sur le fichier. Ne pas le retirer sous
// prétexte que le workflow s'en charge.
//
// os.Rename n'écrase pas une cible existante sous Windows : dst est retiré d'abord.
// Aucun repli sur une écriture en place n'est admis (ce serait ressusciter le bug),
// donc toute erreur de retrait autre que « n'existe pas » est fatale.
func copyMetadataFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := tmp.Name()
	// Nettoyage sur tout chemin d'erreur ; après un rename réussi le nom n'existe
	// plus et le remove est un no-op.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmp, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	// Sync avant publication : le conteneur démo recréé juste après lit ce fichier.
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}
	// os.CreateTemp crée en 0600 ; la démo lit la metadata sous un autre uid.
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("chmod tmp: %w", err)
	}
	if err := removeDuckDBForFreshWrite(dst); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("rename tmp→dst: %w", err)
	}
	return nil
}

// removeForInodeSwap retire path pour libérer le nom avant un rename. Seul
// « n'existe pas » est toléré : une erreur de retrait signifie que le fichier ne peut
// pas être remplacé par un inode neuf, et le seul repli serait l'écriture en place —
// interdite (cf. copyMetadataFile).
func removeForInodeSwap(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", filepath.Base(path), err)
	}
	return nil
}

// removeDuckDBForFreshWrite libère le NOM d'une base DuckDB avant qu'un inode neuf ne
// le reprenne — que ce soit par création directe (`sql.Open` sur un chemin inexistant)
// ou par rename d'un temporaire. POINT DE PASSAGE UNIQUE de tous les seeds démo :
// aucun `os.Remove` nu sur un chemin `.duckdb` dans seed_demo*.go (garde-rail
// TestSeedDemoRemovesWALWithDB).
//
// Le WAL n'est PAS optionnel. Un `<db>.wal` laissé par la génération précédente
// (conteneur démo tué sans CHECKPOINT, seed interrompu, crash) survit au retrait du
// seul fichier `.duckdb`. Comportement MESURÉ sur DuckDB v1.4
// (TestCopyMetadataFile_ForeignWALNotReplayedIntoPublishedDB) :
//
//   - base publiée par RENAME (copyMetadataFile) + WAL survivant → DuckDB rejoue le
//     WAL SANS ERREUR dans la base neuve : tables et lignes de l'ancienne génération
//     ressuscitent dedans, puis l'ouverture READ_ONLY suivante meurt sur « Failure
//     while replaying WAL file ». C'est le cas prouvé, et il est silencieux ;
//   - base CRÉÉE par DuckDB (extracteurs, générateurs synthétiques) → le WAL
//     préexistant est écarté sans dommage. Le retrait y est une défense en
//     profondeur : le comportement n'est pas documenté, et une extraction avortée
//     entre le retrait et la création laisserait sinon le WAL derrière elle.
//
// C'est la raison — déjà appliquée par rebuildDemoSharedSocial / extractDemoMedia —
// généralisée ici à tous les sites.
//
// Le WAL est retiré EN PREMIER : s'il résiste, on abandonne sans avoir délié la base,
// donc sans laisser derrière soi un couple (base absente, WAL présent) qui est
// exactement l'état corrupteur.
func removeDuckDBForFreshWrite(path string) error {
	if err := removeForInodeSwap(path + ".wal"); err != nil {
		return err
	}
	return removeForInodeSwap(path)
}

// errIsMissingTable signale qu'une erreur DuckDB est due à une table source/cible
// absente (ex. table Halo 5-spécifique manquante côté démo Infinite). Utilisé pour
// rendre l'extraction et l'anonymisation best-effort sur le sur-ensemble de tables
// multi-titre.
func errIsMissingTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "Table with name") // "Catalog Error: Table with name X does not exist"
}

// formatIDsLiteral retourne "'id1', 'id2', ..." pour interpolation SQL.
// DuckDB ne supporte pas les ? dans CREATE TABLE AS SELECT, donc on inline.
// Les match_ids étant des UUID hex contrôlés (pas user input), pas de SQLi.
func formatIDsLiteral(ids []string) string {
	if len(ids) == 0 {
		return "''" // pas de match → clause vide qui ne matche rien
	}
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		// Sanity check : un match_id Halo est UUID hex (a-f, 0-9, -). Refuser toute
		// chaîne avec apostrophe pour défense en profondeur (même si fournis par DB).
		if strings.ContainsAny(id, "'\"\\;") {
			continue
		}
		quoted = append(quoted, "'"+id+"'")
	}
	if len(quoted) == 0 {
		return "''"
	}
	return strings.Join(quoted, ", ")
}

// extractSharedTables crée out_shared en ATTACHant src_shared et en copiant
// les 7 tables filtrées sur match_ids. Anonymise sourceXUID → demoXUID dans
// match_participants et xuid_aliases. Recrée les vues V6 à la fin.
func extractSharedTables(
	ctx context.Context,
	srcPath, dstPath string,
	matchIDs []string,
	sourceXUID, demoXUID string,
) (map[string]int, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	// Supprimer dst (+ son WAL) pour idempotence : DuckDB échoue sur ATTACH si le
	// fichier existe avec un schéma différent, et rejouerait un WAL orphelin contre la
	// base fraîche (cf. removeDuckDBForFreshWrite).
	if err := removeDuckDBForFreshWrite(dstPath); err != nil {
		return nil, err
	}

	dst, err := sql.Open("duckdb", dstPath)
	if err != nil {
		return nil, fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS src (READ_ONLY)", srcPath)); err != nil {
		return nil, fmt.Errorf("attach src: %w", err)
	}
	defer func() { _, _ = dst.ExecContext(ctx, "DETACH src") }()

	idsLit := formatIDsLiteral(matchIDs)
	counts := make(map[string]int, len(sharedTablesWhere))
	for _, t := range sharedTablesWhere {
		where := fmt.Sprintf(t.where, idsLit)
		stmt := fmt.Sprintf(`CREATE TABLE %s AS SELECT %s FROM src.%s WHERE %s`,
			t.name, extractSelectExpr(t.appendOnly), t.name, where)
		if _, err := dst.ExecContext(ctx, stmt); err != nil {
			// Table source absente (ex. match_csrs sur une DB sans données CSR, ou
			// fixture de test minimal) : best-effort comme les corpus squad/ranked.
			// Les tables append-only sont de toute façon (re)créées vides + vue
			// _latest par les migrations canoniques (applyMigrationsOnPath) → on
			// n'avorte pas le seed pour ça.
			if strings.Contains(err.Error(), "does not exist") {
				slog.WarnContext(ctx, "seed-demo: table source absente, ignorée", "table", t.name, "err", err)
				counts[t.name] = 0
				continue
			}
			return counts, fmt.Errorf("extract %s: %w", t.name, err)
		}
		var n int
		if err := dst.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", t.name)).Scan(&n); err != nil {
			return counts, fmt.Errorf("count %s: %w", t.name, err)
		}
		counts[t.name] = n
	}

	// NB : l'anonymisation des xuid/gamertag (source + coéquipiers + autres) est
	// faite par applyUniversalAnonymization() côté SeedDemo, après extraction, via
	// le roster démo — pas ici. (sourceXUID/demoXUID restent dans la signature pour
	// compat appelants/tests, mais ne sont plus utilisés ici.)
	_ = sourceXUID
	_ = demoXUID

	// Recréer les vues V6 (v_gamertag_lookup, v_match_full, v_weapon_kills).
	if err := recreateSharedViews(ctx, dst); err != nil {
		// Non-bloquant : log mais continue (vues optionnelles côté demo).
		slog.WarnContext(ctx, "seed-demo: recreate views partielle", "err", err)
	}
	return counts, nil
}

// extractPlayerTables extrait les tables enrichment/citations/sessions/career/sync_meta/skill_rank.
// Anonymise sourceXUID → demoXUID + met à jour sync_meta.xuid si présent.
func extractPlayerTables(
	ctx context.Context,
	srcPath, dstPath string,
	matchIDs []string,
	sourceXUID, demoXUID string,
) (map[string]int, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	if err := removeDuckDBForFreshWrite(dstPath); err != nil {
		return nil, err
	}

	dst, err := sql.Open("duckdb", dstPath)
	if err != nil {
		return nil, fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS src (READ_ONLY)", srcPath)); err != nil {
		return nil, fmt.Errorf("attach src: %w", err)
	}
	defer func() { _, _ = dst.ExecContext(ctx, "DETACH src") }()

	idsLit := formatIDsLiteral(matchIDs)
	counts := make(map[string]int, len(playerTablesWhere))
	for _, t := range playerTablesWhere {
		where := t.where
		if strings.Contains(where, "%s") {
			where = fmt.Sprintf(where, idsLit)
		}
		stmt := fmt.Sprintf(`CREATE TABLE %s AS SELECT %s FROM src.%s WHERE %s`,
			t.name, extractSelectExpr(t.appendOnly), t.name, where)
		if _, err := dst.ExecContext(ctx, stmt); err != nil {
			// Tolérant : certaines tables peuvent ne pas exister (ex : match_skill_rank
			// sur DB legacy). Log et continue.
			slog.WarnContext(ctx, "seed-demo: extract player table partielle",
				"table", t.name, "err", err)
			counts[t.name] = 0
			continue
		}
		var n int
		if err := dst.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", t.name)).Scan(&n); err != nil {
			return counts, fmt.Errorf("count %s: %w", t.name, err)
		}
		counts[t.name] = n
	}

	// Anonymisation : sourceXUID → demoXUID dans tables avec colonne xuid.
	// Schéma player_match_enrichment / match_skill_rank / match_citations : PAS de xuid
	// (player DB mono-joueur, xuid implicite via path /data/players/{gamertag}/).
	// Seule career_progression a une colonne xuid à anonymiser (cf. steps_player.go:36-51).
	if err := anonymizeXUIDInTables(ctx, dst, sourceXUID, demoXUID,
		[]string{"career_progression", "battlepass_snapshots"}); err != nil {
		// Non bloquant
		slog.WarnContext(ctx, "seed-demo: anonymize player partielle", "err", err)
	}
	// sync_meta.value pour la clé 'xuid' canonique.
	if _, err := dst.ExecContext(ctx,
		`UPDATE sync_meta SET value = ? WHERE key = 'xuid' AND value = ?`,
		demoXUID, sourceXUID); err != nil {
		slog.WarnContext(ctx, "seed-demo: update sync_meta xuid partielle", "err", err)
	}
	return counts, nil
}

// anonymizeXUIDInTables exécute UPDATE ... SET xuid = ? WHERE xuid = ? sur les
// tables fournies. Tolérant : les tables sans colonne xuid sont skip avec warn.
func anonymizeXUIDInTables(
	ctx context.Context,
	db *sql.DB,
	sourceXUID, demoXUID string,
	tables []string,
) error {
	for _, t := range tables {
		stmt := fmt.Sprintf(`UPDATE %s SET xuid = ? WHERE xuid = ?`, t)
		if _, err := db.ExecContext(ctx, stmt, demoXUID, sourceXUID); err != nil {
			return fmt.Errorf("update %s: %w", t, err)
		}
	}
	return nil
}

// recreateSharedViews recrée v_gamertag_lookup, v_match_full, v_weapon_kills
// requis par les pages analytics. Tolère l'absence de tables sources.
func recreateSharedViews(ctx context.Context, db *sql.DB) error {
	// v_gamertag_lookup : DDL partagé via analysis.GamertagLookupViewSQL
	// (SOURCE UNIQUE — même résolveur que le boot et les migrations).
	if _, err := db.ExecContext(ctx, analysis.GamertagLookupViewSQL()); err != nil {
		return fmt.Errorf("v_gamertag_lookup: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`); err != nil {
		return fmt.Errorf("v_match_full: %w", err)
	}
	// v_weapon_kills : tolère table absente. Append-only #23046 (Phase 2) : la vue
	// ne retourne que la dernière génération par (match_id,xuid) — comme la migration
	// shared_append_only_weapon_kills_v1 (le demo copie prod qui porte generation_id).
	_, _ = db.ExecContext(ctx, `
		CREATE OR REPLACE VIEW v_weapon_kills AS
		SELECT * EXCLUDE (rk) FROM (
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id,
			       DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
			FROM weapon_kills)
		WHERE rk = 1`)
	return nil
}

// writeDemoConfigs écrit db_profiles.json + app_settings.json dans outDir.
func writeDemoConfigs(outDir, demoXUID, gamertag, serviceTag string, mediaEnabled bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	// db_profiles.json
	profiles := map[string]any{
		"version":        "2.1",
		"warehouse_path": "data/warehouse",
		"metadata_db":    "data/warehouse/metadata.duckdb",
		"profiles": map[string]any{
			DefaultDemoGamertag: map[string]any{
				"db_path":         "data/players/" + DefaultDemoGamertag + "/stats.duckdb",
				"xuid":            demoXUID,
				"waypoint_player": gamertag,
			},
		},
	}
	if err := writeJSONFile(filepath.Join(outDir, "db_profiles.json"), profiles); err != nil {
		return fmt.Errorf("db_profiles.json: %w", err)
	}
	// app_settings.json
	settings := map[string]any{
		"lang":                                "fr",
		"media_enabled":                       mediaEnabled,
		"spnkr_refresh_on_start":              false,
		"spnkr_refresh_on_manual_refresh":     false,
		"spnkr_refresh_max_matches":           0,
		"spnkr_refresh_with_highlight_events": false,
		"spnkr_refresh_with_backfill":         false,
		"spnkr_refresh_backfill_medals":       false,
		"spnkr_refresh_backfill_events":       false,
		"spnkr_refresh_backfill_skill":        false,
		"discord_notifications_enabled":       false,
		"doppler_enabled":                     false,
		"tailscale_funnel_enabled":            false,
		"profile_assets_download_enabled":     false,
		"profile_api_enabled":                 false,
		"profile_service_tag":                 serviceTag,
		"repository_mode":                     "duckdb",
		"enable_duckdb_analytics":             true,
	}
	if err := writeJSONFile(filepath.Join(outDir, "app_settings.json"), settings); err != nil {
		return fmt.Errorf("app_settings.json: %w", err)
	}
	return nil
}

// demoContainerMediaDir : chemin RUNTIME des fichiers média dans le conteneur
// levelup-demo (LEVELUP_ROOT=/app + mount ./data/demo). ServeMediaFile résout les
// file_path relatifs (= filename) contre ce MediaCapturesBaseDir.
const demoContainerMediaDir = "/app/data/demo/players/" + DefaultDemoGamertag + "/media"

// demoAppSettings retourne le map app_settings.json démo (langue FR, sync OFF…).
func demoAppSettings(serviceTag string, mediaEnabled bool) map[string]any {
	return map[string]any{
		"lang":                                "fr",
		"media_enabled":                       mediaEnabled,
		"media_captures_base_dir":             demoContainerMediaDir,
		"spnkr_refresh_on_start":              false,
		"spnkr_refresh_on_manual_refresh":     false,
		"spnkr_refresh_max_matches":           0,
		"spnkr_refresh_with_highlight_events": false,
		"spnkr_refresh_with_backfill":         false,
		"spnkr_refresh_backfill_medals":       false,
		"spnkr_refresh_backfill_events":       false,
		"spnkr_refresh_backfill_skill":        false,
		"discord_notifications_enabled":       false,
		"doppler_enabled":                     false,
		"tailscale_funnel_enabled":            false,
		"profile_assets_download_enabled":     false,
		"profile_api_enabled":                 false,
		"profile_service_tag":                 serviceTag,
		"repository_mode":                     "duckdb",
		"enable_duckdb_analytics":             true,
		// Active les routes Prestige en démo : sans ça /prestige/me 404 et la section
		// Prestige du Home ne rend pas. Le rang Prestige est servi via fixture read-time
		// (LazyPrestigeService DemoMode) car le DemoPlayer n'a pas d'activité PP réelle.
		"prestige_enabled": true,
	}
}

// writeDemoConfigsMulti écrit db_profiles.json (1 profil par player DB démo
// seedée, clé = gamertag démo) + app_settings.json dans outDir. waypoint_player
// = gamertag démo (jamais le gamertag source réel — évite le "JGtm" affiché).
func writeDemoConfigsMulti(outDir string, seeded []seededDemoPlayer, serviceTag string, mediaEnabled bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	profilesMap := make(map[string]any, len(seeded))
	for _, s := range seeded {
		profilesMap[s.Gamertag] = map[string]any{
			"db_path":         "data/players/" + s.Dir + "/stats.duckdb",
			"xuid":            s.XUID,
			"waypoint_player": s.Gamertag,
		}
	}
	profiles := map[string]any{
		"version":        "2.1",
		"warehouse_path": "data/warehouse",
		"metadata_db":    "data/warehouse/metadata.duckdb",
		"profiles":       profilesMap,
	}
	if err := writeJSONFile(filepath.Join(outDir, "db_profiles.json"), profiles); err != nil {
		return fmt.Errorf("db_profiles.json: %w", err)
	}
	if err := writeJSONFile(filepath.Join(outDir, "app_settings.json"), demoAppSettings(serviceTag, mediaEnabled)); err != nil {
		return fmt.Errorf("app_settings.json: %w", err)
	}
	return nil
}

// borrowDemoIdentity copie une identité Spartan complète (rang, XP, emblem,
// banner, backdrop, adornment, spartan_id) d'un vrai joueur dans la
// career_progression démo, avec un recorded_at frais pour qu'elle soit la plus
// récente (ARG_MAX côté Q26c). Le joueur démo est anonyme et jamais sync'd, donc
// n'a aucune customization propre ; on emprunte une identité figée pour un rendu
// réaliste. Best-effort : no-op si aucun joueur n'a d'identité peuplée.
func borrowDemoIdentity(ctx context.Context, demoPlayerDB, playersDir, demoXUID string) error {
	srcDB := findPlayerWithIdentity(ctx, playersDir)
	if srcDB == "" {
		slog.WarnContext(ctx, "seed-demo: aucune identité Spartan à emprunter (aucun joueur avec emblem+banner)")
		return nil
	}

	db, err := sql.Open("duckdb", demoPlayerDB)
	if err != nil {
		return fmt.Errorf("open demo player DB: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS idsrc (READ_ONLY)", srcDB)); err != nil {
		return fmt.Errorf("attach identity source: %w", err)
	}
	defer func() { _, _ = db.ExecContext(ctx, "DETACH idsrc") }()

	// INSERT BY NAME : robuste à l'ordre/sous-ensemble de colonnes. REPLACE
	// anonymise le xuid et rend la row la plus récente (recorded_at = now).
	stmt := fmt.Sprintf(`
		INSERT INTO career_progression BY NAME
		SELECT * REPLACE ('%s' AS xuid, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS recorded_at)
		FROM idsrc.career_progression
		WHERE NULLIF(TRIM(emblem_image_url), '') IS NOT NULL
		  AND NULLIF(TRIM(banner_image_url), '') IS NOT NULL
		ORDER BY recorded_at DESC
		LIMIT 1`, demoXUID)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("insert borrowed identity: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: identité Spartan empruntée", "source_db", srcDB)
	return nil
}

// findPlayerWithIdentity retourne le chemin du premier player DB (ordre trié,
// hors DEMO) dont career_progression a une row avec emblem + banner non vides.
// Retourne "" si aucun. Tolérant : DB verrouillée / table absente → skip.
func findPlayerWithIdentity(ctx context.Context, playersDir string) string {
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() && e.Name() != DefaultDemoGamertag {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		dbPath := filepath.Join(playersDir, name, "stats.duckdb")
		if _, statErr := os.Stat(dbPath); statErr != nil {
			continue
		}
		if playerHasIdentity(ctx, dbPath) {
			return dbPath
		}
	}
	return ""
}

// playerHasIdentity teste (READ_ONLY) si la career_progression d'un player DB
// contient au moins une row avec emblem + banner non vides.
func playerHasIdentity(ctx context.Context, dbPath string) bool {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return false
	}
	defer db.Close()
	var n int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM career_progression
		WHERE NULLIF(TRIM(emblem_image_url), '') IS NOT NULL
		  AND NULLIF(TRIM(banner_image_url), '') IS NOT NULL`).Scan(&n)
	return err == nil && n > 0
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
