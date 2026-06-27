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
}

// SeedDemoResult résume l'exécution.
type SeedDemoResult struct {
	OutDir         string
	MatchIDs       []string
	MetadataCopied bool
	SharedRows     map[string]int // table → rows insérées
	PlayerRows     map[string]int
	MediaCopied    int
	ConfigsWritten bool
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
	{name: "xuid_aliases", where: "xuid IN (SELECT DISTINCT xuid FROM match_participants WHERE match_id IN (%s))"},
	{name: "match_csrs", where: matchIDInClause, appendOnly: true},
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

// SeedDemo exécute le pipeline complet.
func SeedDemo(ctx context.Context, opts SeedDemoOptions) (SeedDemoResult, error) {
	start := time.Now()
	res := SeedDemoResult{OutDir: opts.OutDir, SharedRows: map[string]int{}, PlayerRows: map[string]int{}}

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

	// 1. Corpus = matchs récents (solo inclus, pour l'historique/home du DemoPlayer)
	// UNION les 3 dernières sessions en escouade (pour la page Escouade). Dédupliqué.
	// L'union garantit que le DemoPlayer garde un historique solo riche tout en
	// disposant d'un vrai corpus 3-stack pour les coéquipiers.
	recentMatches, rErr := selectRecentMatchIDs(ctx, opts.SourceSharedDB, opts.SourceXUID, opts.MaxMatches)
	if rErr != nil {
		slog.WarnContext(ctx, "seed-demo: matchs récents indisponibles", "err", rErr)
	}
	squadMatches, sErr := selectSquadSessionCorpus(ctx, opts.SourcePlayerDB, DefaultSquadSessions)
	if sErr != nil {
		slog.WarnContext(ctx, "seed-demo: corpus squad indisponible", "err", sErr)
	}
	// Matchs classés récents : font apparaître les playlists classées (+ CSR) dans
	// recent_playlist_ranks (le corpus solo+squad est 100% Partie rapide non classé).
	rankedMatches, rkErr := selectRecentRankedMatchIDs(ctx, opts.SourceSharedDB, opts.SourceXUID, DefaultRankedMatches)
	if rkErr != nil {
		slog.WarnContext(ctx, "seed-demo: matchs classés indisponibles", "err", rkErr)
	}
	matchIDs := unionMatchIDs(recentMatches, squadMatches, rankedMatches)
	if len(matchIDs) == 0 {
		return res, fmt.Errorf("seed-demo: aucun match trouvé pour xuid=%s dans %s",
			opts.SourceXUID, opts.SourceSharedDB)
	}
	res.MatchIDs = matchIDs
	slog.InfoContext(ctx, "seed-demo: corpus sélectionné",
		"total", len(matchIDs), "recent", len(recentMatches), "squad", len(squadMatches), "ranked", len(rankedMatches))

	// 1b. Roster démo : source + coéquipiers principaux (classés sur le corpus
	// escouade) + autres participants, mappés vers des identités démo stables.
	roster, err := buildDemoRoster(ctx, opts.SourceSharedDB, squadMatches, matchIDs, opts.SourceXUID, 2)
	if err != nil {
		return res, fmt.Errorf("seed-demo: build roster: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: roster démo construit", "participants", len(roster))

	// 2. Copie metadata.duckdb
	outMeta := filepath.Join(titleOut, "warehouse", "metadata.duckdb")
	if err := copyMetadataFile(opts.SourceMetaDB, outMeta); err != nil {
		return res, fmt.Errorf("seed-demo: copy metadata: %w", err)
	}
	res.MetadataCopied = true
	slog.InfoContext(ctx, "seed-demo: metadata copiée", "out", outMeta)

	// 3. Extraction shared
	outShared := filepath.Join(titleOut, "warehouse", "shared_matches_v2.duckdb")
	sharedRows, err := extractSharedTables(ctx, opts.SourceSharedDB, outShared, matchIDs,
		opts.SourceXUID, opts.DemoXUID)
	if err != nil {
		return res, fmt.Errorf("seed-demo: extract shared: %w", err)
	}
	res.SharedRows = sharedRows
	slog.InfoContext(ctx, "seed-demo: shared extraite", "out", outShared, "rows", sharedRows)

	// 3b. Anonymisation universelle : remappe TOUS les xuid + gamertag du shared
	// (source → DemoPlayer, coéquipiers → DemoPlayer2/3, autres → Player N) pour
	// que les médailles/arme/frags du DemoPlayer s'affichent ET qu'aucun vrai
	// gamertag ne fuite (page Escouade, kill-feed, Face-à-face).
	if err := anonymizeSharedUniversal(ctx, outShared, roster); err != nil {
		return res, fmt.Errorf("seed-demo: anonymisation universelle: %w", err)
	}
	slog.InfoContext(ctx, "seed-demo: anonymisation universelle appliquée", "mapped", len(roster))

	// 4. Migration shared (une fois) : reconstruit les tables append-only +
	// vues _latest. Idempotent.
	if err := applyMigrationsOnPath(outShared, migration.TargetShared); err != nil {
		return res, fmt.Errorf("seed-demo: migrate shared: %w", err)
	}

	// 5. Seed des player DB du roster : DemoPlayer (source) + DemoPlayer2/3
	// (coéquipiers principaux), chacune filtrée sur le corpus + anonymisée vers son
	// xuid démo. Les coéquipiers donnent leurs vraies stats (perf, LUSR) à la page
	// Escouade. Les player DB coéquipiers sont résolues depuis db_profiles par xuid.
	mains := rosterMains(roster)
	var seeded []seededDemoPlayer
	for i, m := range mains {
		srcPlayerDB := opts.SourcePlayerDB
		if i > 0 {
			// Title-aware : on veut la player DB du coéquipier POUR LE TITRE courant
			// (ses stats H5 quand on seede H5), pas une autre (il peut exister sous
			// plusieurs titres). Fallback tous-titres si absent du titre courant.
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
				return res, fmt.Errorf("seed-demo: extract player principal: %w", perr)
			}
			slog.WarnContext(ctx, "seed-demo: extract player coéquipier échoué, skip",
				"gamertag", m.DemoGamertag, "err", perr)
			continue
		}
		if err := applyMigrationsOnPath(outP, migration.TargetPlayer); err != nil {
			return res, fmt.Errorf("seed-demo: migrate player %s: %w", demoDir, err)
		}
		// Identité Spartan : empruntée pour le DemoPlayer principal (le joueur
		// source n'a pas de customization propre) ; les coéquipiers ont déjà la
		// leur (career_progression réelle copiée).
		if i == 0 {
			res.PlayerRows = rows
			if opts.SourcePlayersDir != "" {
				if err := borrowDemoIdentity(ctx, outP, opts.SourcePlayersDir, m.DemoXUID); err != nil {
					slog.WarnContext(ctx, "seed-demo: emprunt identité échoué (non bloquant)", "err", err)
				}
			}
		}
		seeded = append(seeded, seededDemoPlayer{Dir: demoDir, XUID: m.DemoXUID, Gamertag: m.DemoGamertag})
		slog.InfoContext(ctx, "seed-demo: player seedée", "dir", demoDir, "gamertag", m.DemoGamertag, "rows", rows)
	}

	// 6. Médias (DemoPlayer principal). Copie les vraies vidéos HLS "Halo Infinite" du
	// joueur source (data/media/{gamertag}/{hls,thumbs}) vers la démo + media_files au
	// schéma canonique shared_social. player_slug = SLUG de route du main (la page Média
	// filtre par auteur = slug courant). Attribution grossière aux matchs du corpus.
	if opts.IncludeMedia {
		mediaDir := filepath.Join(titleOut, "players", DefaultDemoGamertag, "media")
		outSocial := filepath.Join(titleOut, "warehouse", "shared_social.duckdb")
		srcMediaDir := filepath.Join(opts.RepoRoot, "data", "media", opts.SourceLabel)
		// shared_social SOURCE (prod) : même dossier warehouse que shared_matches_v2.
		// Porte les associations média→match RÉELLES (vraie carte de chaque vidéo).
		srcSocialDB := filepath.Join(filepath.Dir(opts.SourceSharedDB), "shared_social.duckdb")
		mediaCount, mediaErr := extractDemoMedia(ctx, srcMediaDir, opts.SourceSharedDB, srcSocialDB,
			outSocial, mediaDir, matchIDs, DefaultDemoMainSlug, opts.MaxMedia)
		if mediaErr != nil {
			slog.WarnContext(ctx, "seed-demo: extraction média partielle", "err", mediaErr, "copied", mediaCount)
		}
		res.MediaCopied = mediaCount
	}

	res.SeededPlayers = seeded

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
	)
	return res, nil
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

// copyMetadataFile copie src→dst (file copy bit-à-bit, équivalent shutil.copy2).
// Crée le répertoire parent si nécessaire.
func copyMetadataFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
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
	// Supprimer dst existant pour idempotence (DuckDB échoue sur ATTACH si fichier
	// existe avec schéma différent).
	_ = os.Remove(dstPath)

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
	_ = os.Remove(dstPath)

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
		SELECT * REPLACE ('%s' AS xuid, CURRENT_TIMESTAMP AS recorded_at)
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
