// Package sync — backfill_registry_names.go : CONVERGENCE des noms d'assets (playlist, carte,
// paire, variante) de match_registry vers les traductions de metadata.asset_translations.
//
// UNE SEULE MÉCANIQUE pour trois appelants : le balayage périodique des noms d'assets
// (wire.ResolveUnresolvedAssetNames, juste après la résolution des traductions), l'action admin
// « backfill registry names » et la sous-commande `levelup backfill-registry-names` (réparation
// des données existantes, avec `--dry-run`).
//
// CE QUI EST RÉÉCRIT : une colonne de nom NULL ou égale à son identifiant (le nom recopié par la
// sync quand la traduction manquait, ou vidé par un ancien outil de réparation). Le nom inscrit
// est la traduction en-US (même source que EnrichRegistryFromMetadata au sync primaire). Pour la
// paire sans traduction, le nom est CONSTRUIT « {variante} on {carte} » (constructPairName, la
// même règle qu'au sync) à partir des noms effectifs de la variante et de la carte. Sans source,
// la colonne reste en l'état et est comptée (Scanned - Fixed).
//
// CE QUI N'EST JAMAIS TOUCHÉ : un vrai nom (garde SQL du persister). La catégorie de mode
// (`mode_category`) suit le nom de la paire écrit : elle est recalculée dans le même UPDATE
// (règle canonique `analysis/modelabel`), tant qu'aucun index ne couvre la colonne.
//
// ÉCRITURE : persist.RegistryNamesPersister, un match à la fois, gardé, idempotent.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/modelabel"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
)

// BackfillRegistryStats : compteurs d'une convergence, PAR MATCH (une colonne d'un match = 1).
// XScanned = matchs dont la colonne est NULL ou égale à l'identifiant ; XFixed = matchs dont la
// colonne a été réécrite (en simulation : le serait). XScanned - XFixed = sans source de nom.
type BackfillRegistryStats struct {
	DryRun           bool
	PlaylistsScanned int
	PlaylistsFixed   int
	MapsScanned      int
	MapsFixed        int
	PairsScanned     int
	PairsFixed       int
	PairsConstructed int // part de PairsFixed obtenue par construction « {variante} on {carte} »
	VariantsScanned  int
	VariantsFixed    int
	Matches          int // matchs dont au moins une colonne a été réécrite (en simulation : le serait)
	Errors           int // matchs dont l'écriture a échoué (journalisés)
}

// Total retourne la somme des colonnes réécrites (ou qui le seraient en simulation).
func (s BackfillRegistryStats) Total() int {
	return s.PlaylistsFixed + s.MapsFixed + s.PairsFixed + s.VariantsFixed
}

// RegistryNamesOptions paramètre une convergence. DryRun : planifier et compter, sans écrire.
type RegistryNamesOptions struct {
	DryRun bool
}

// registryNameRow : un match candidat, ses identifiants et noms bruts.
type registryNameRow struct {
	matchID                                  string
	playlistID, mapID, pairID, variantID     sql.NullString
	playlistName, mapName, pairName, varName sql.NullString
}

// qRegistryNameCandidates : les matchs dont au moins une colonne de nom est NULL ou égale à son
// identifiant (identifiant renseigné). Lecture seule.
const qRegistryNameCandidates = `
SELECT match_id, playlist_id, playlist_name, map_id, map_name,
       pair_id, pair_name, game_variant_id, game_variant_name
FROM match_registry
WHERE (playlist_id <> '' AND (playlist_name IS NULL OR playlist_name = playlist_id))
   OR (map_id <> '' AND (map_name IS NULL OR map_name = map_id))
   OR (pair_id <> '' AND (pair_name IS NULL OR pair_name = pair_id))
   OR (game_variant_id <> '' AND (game_variant_name IS NULL OR game_variant_name = game_variant_id))
ORDER BY match_id`

// BackfillRegistryNames fait converger les noms d'assets de match_registry vers les traductions
// de metadata (cf. en-tête). sharedDB doit porter le writer de shared_matches_v2 hors
// simulation. metadataDB nil : aucune source de nom — en écriture, avertissement et stats vides ;
// en simulation, les candidats sont comptés (XScanned), tous sans source (XFixed = 0).
func BackfillRegistryNames(ctx context.Context, sharedDB, metadataDB *sql.DB,
	opts RegistryNamesOptions) (BackfillRegistryStats, error) {
	stats := BackfillRegistryStats{DryRun: opts.DryRun}
	if metadataDB == nil && !opts.DryRun {
		slog.WarnContext(ctx, "BackfillRegistryNames: metadata DB nil — aucune source de nom, rien écrit")
		return stats, nil
	}
	if sharedDB == nil {
		return stats, fmt.Errorf("BackfillRegistryNames: sharedDB nil")
	}
	rows, err := loadRegistryNameCandidates(ctx, sharedDB)
	if err != nil {
		return stats, err
	}
	if metadataDB == nil {
		countCandidatesWithoutSource(rows, &stats)
		slog.WarnContext(ctx, "BackfillRegistryNames: metadata DB nil — simulation : candidats comptés, tous sans source",
			"candidats", len(rows))
		return stats, nil
	}
	names := newTranslationCache(metadataDB)
	persister := persist.NewRegistryNamesPersister(sharedDB)
	withCategory, err := modeCategoryWritable(ctx, sharedDB)
	if err != nil {
		return stats, err
	}
	for _, row := range rows {
		writes, construite := planRegistryNames(ctx, row, names, &stats)
		if !withCategory {
			dropModeCategories(writes)
		}
		if len(writes) == 0 {
			continue
		}
		if opts.DryRun {
			stats.Matches++
			countRegistryWrites(&stats, kindsOf(writes), construite)
			continue
		}
		ecrits, err := persister.WriteMatchNames(ctx, row.matchID, writes)
		if err != nil {
			stats.Errors++
			slog.ErrorContext(ctx, "BackfillRegistryNames: écriture échouée",
				"match_id", row.matchID, "err", err)
			continue
		}
		if len(ecrits) > 0 {
			stats.Matches++
		}
		countRegistryWrites(&stats, ecrits, construite)
	}
	slog.InfoContext(ctx, "BackfillRegistryNames: convergence terminée",
		"dry_run", opts.DryRun, "candidats", len(rows), "colonnes", stats.Total(),
		"paires_construites", stats.PairsConstructed, "errors", stats.Errors)
	return stats, nil
}

// loadRegistryNameCandidates lit les matchs candidats (qRegistryNameCandidates).
func loadRegistryNameCandidates(ctx context.Context, sharedDB *sql.DB) ([]registryNameRow, error) {
	rs, err := sharedDB.QueryContext(ctx, qRegistryNameCandidates)
	if err != nil {
		return nil, fmt.Errorf("BackfillRegistryNames: lecture des candidats: %w", err)
	}
	defer rs.Close()
	var out []registryNameRow
	for rs.Next() {
		var r registryNameRow
		if err := rs.Scan(&r.matchID, &r.playlistID, &r.playlistName, &r.mapID, &r.mapName,
			&r.pairID, &r.pairName, &r.variantID, &r.varName); err != nil {
			return nil, fmt.Errorf("BackfillRegistryNames: scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rs.Err(); err != nil {
		return nil, fmt.Errorf("BackfillRegistryNames: itération: %w", err)
	}
	return out, nil
}

// planRegistryNames calcule les écritures d'un match et compte ses colonnes candidates
// (XScanned). Rend aussi si la paire planifiée est construite (et non traduite).
func planRegistryNames(ctx context.Context, row registryNameRow, names *translationCache,
	stats *BackfillRegistryStats) ([]persist.RegistryNameWrite, bool) {
	var writes []persist.RegistryNameWrite
	// effectif : le nom après convergence (planifié s'il y en a un, sinon l'existant).
	effectif := func(kind string, id, name sql.NullString, scanned *int) string {
		if !nameNeedsConvergence(id, name) {
			return name.String
		}
		*scanned++
		if n := names.lookup(ctx, kind, id.String); n != "" {
			writes = append(writes, persist.RegistryNameWrite{Kind: kind, Name: n})
			return n
		}
		return name.String
	}
	effectif(games.AssetKindPlaylist, row.playlistID, row.playlistName, &stats.PlaylistsScanned)
	carte := effectif(games.AssetKindMap, row.mapID, row.mapName, &stats.MapsScanned)
	variante := effectif(games.AssetKindGameVariant, row.variantID, row.varName, &stats.VariantsScanned)
	if !nameNeedsConvergence(row.pairID, row.pairName) {
		return writes, false
	}
	stats.PairsScanned++
	if n := names.lookup(ctx, games.AssetKindPair, row.pairID.String); n != "" {
		return append(writes, pairNameWrite(n)), false
	}
	if construit, ok := constructPairName(&variante, &row.variantID.String, &carte, &row.mapID.String); ok {
		return append(writes, pairNameWrite(construit)), true
	}
	return writes, false
}

// pairNameWrite : l'écriture d'un nom de paire, avec la catégorie de mode que ce nom donne.
func pairNameWrite(name string) persist.RegistryNameWrite {
	return persist.RegistryNameWrite{
		Kind: games.AssetKindPair, Name: name, ModeCategory: modelabel.InferCategory(name),
	}
}

// qModeCategoryIndexes : les index de match_registry qui couvrent mode_category.
const qModeCategoryIndexes = `SELECT COUNT(*) FROM duckdb_indexes()
	WHERE table_name = 'match_registry' AND sql ILIKE '%mode_category%'`

// modeCategoryWritable dit si la catégorie peut être réécrite avec le nom : seulement si aucun
// index ne couvre la colonne (un UPDATE d'une colonne indexée est le vecteur du bug ART DuckDB
// #23645). Une base pas encore migrée garde sa catégorie, avec un avertissement.
func modeCategoryWritable(ctx context.Context, sharedDB *sql.DB) (bool, error) {
	var n int
	if err := sharedDB.QueryRowContext(ctx, qModeCategoryIndexes).Scan(&n); err != nil {
		return false, fmt.Errorf("BackfillRegistryNames: index de mode_category: %w", err)
	}
	if n > 0 {
		slog.WarnContext(ctx, "BackfillRegistryNames: mode_category encore indexée — catégorie non réécrite "+
			"(migration shared_recompute_mode_category_v1 à appliquer)", "index", n)
		return false, nil
	}
	return true, nil
}

// dropModeCategories retire la catégorie de chaque écriture planifiée.
func dropModeCategories(writes []persist.RegistryNameWrite) {
	for i := range writes {
		writes[i].ModeCategory = ""
	}
}

// countCandidatesWithoutSource compte, par colonne, les matchs dont le nom est à faire converger
// (XScanned), sans aucune écriture planifiée : simulation sans base de métadonnées.
func countCandidatesWithoutSource(rows []registryNameRow, stats *BackfillRegistryStats) {
	for _, r := range rows {
		for _, c := range []struct {
			id, name sql.NullString
			scanned  *int
		}{
			{r.playlistID, r.playlistName, &stats.PlaylistsScanned},
			{r.mapID, r.mapName, &stats.MapsScanned},
			{r.pairID, r.pairName, &stats.PairsScanned},
			{r.variantID, r.varName, &stats.VariantsScanned},
		} {
			if nameNeedsConvergence(c.id, c.name) {
				*c.scanned++
			}
		}
	}
}

// nameNeedsConvergence : identifiant renseigné et nom NULL ou égal à l'identifiant — la même
// condition que qRegistryNameCandidates et que la garde du persister.
func nameNeedsConvergence(id, name sql.NullString) bool {
	if !id.Valid || id.String == "" {
		return false
	}
	return !name.Valid || name.String == id.String
}

// countRegistryWrites ajoute aux XFixed les genres écrits d'un match.
func countRegistryWrites(stats *BackfillRegistryStats, kinds []string, paireConstruite bool) {
	for _, k := range kinds {
		switch k {
		case games.AssetKindPlaylist:
			stats.PlaylistsFixed++
		case games.AssetKindMap:
			stats.MapsFixed++
		case games.AssetKindPair:
			stats.PairsFixed++
			if paireConstruite {
				stats.PairsConstructed++
			}
		case games.AssetKindGameVariant:
			stats.VariantsFixed++
		}
	}
}

func kindsOf(writes []persist.RegistryNameWrite) []string {
	out := make([]string, len(writes))
	for i, w := range writes {
		out[i] = w.Kind
	}
	return out
}

// translationCache mémoïse lookupAssetCanonicalEN par (genre, identifiant) : un même asset
// revient sur des centaines de matchs. Une lecture en échec est journalisée et vaut « pas de
// traduction » (l'asset reste compté sans source).
type translationCache struct {
	db    *sql.DB
	names map[string]string
}

func newTranslationCache(db *sql.DB) *translationCache {
	return &translationCache{db: db, names: map[string]string{}}
}

func (c *translationCache) lookup(ctx context.Context, kind, id string) string {
	key := kind + "|" + id
	if n, ok := c.names[key]; ok {
		return n
	}
	n, err := lookupAssetCanonicalEN(ctx, c.db, kind, id)
	if err != nil {
		slog.WarnContext(ctx, "BackfillRegistryNames: lecture de traduction échouée",
			"asset_type", kind, "asset_id", id, "err", err)
		n = ""
	}
	if n == id {
		n = "" // une « traduction » égale à l'identifiant n'est pas un nom
	}
	c.names[key] = n
	return n
}
