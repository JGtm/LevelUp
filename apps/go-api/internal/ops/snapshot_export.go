// Package ops — snapshot_export.go : primitives bas-niveau d'export Parquet d'un cut.
//
// Modèle FULL RE-EXPORT : à chaque cut on réécrit l'intégralité des faits/dérivés des
// matchs ready dans une nouvelle version. Pas de partition par-batch → pas de
// compaction. Un fichier Parquet par table shared ; un par (table dérivée, joueur).
// Tout COPY filtre sur l'ensemble des matchs `snapshot_ready_at IS NOT NULL`.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sharedSnapshotTables : faits bruts immuables shared inclus dans chaque snapshot
// (tables de base, toutes porteuses d'une colonne match_id). Réutilisé par la lecture
// per-joueur (OpenSnapshotForPlayer).
var sharedSnapshotTables = []string{
	"match_registry",
	"match_participants",
	"medals_earned",
	"highlight_events",
	"killer_victim_pairs",
}

// sharedSnapshotViewExports : relations shared COLLAPSED (vues append-only) exportées
// match-keyed sous un nom de fichier dédié — on fige le RÉSULTAT de la vue (dernière
// génération / dernier written_at) plutôt que de répliquer la logique QUALIFY côté
// lecture. La lecture recrée une vue passthrough portant le nom live.
var sharedSnapshotViewExports = []struct{ source, dest, view string }{
	{"v_weapon_kills", "weapon_kills", "v_weapon_kills"},
	{"match_csrs_latest", "match_csrs", "match_csrs_latest"},
}

// sharedSnapshotGlobalTables : relations shared NON match-keyed (clé xuid) exportées en
// ENTIER (petites, globales) — requises par v_gamertag_lookup au moment de la lecture.
var sharedSnapshotGlobalTables = []string{"xuid_aliases"}

// relationExists : la table OU la vue `name` existe-t-elle (information_schema.tables
// couvre les deux dans DuckDB) ? Garde les COPY contre une relation absente (schéma
// shared partiel / fixture minimale).
func relationExists(ctx context.Context, e interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, name string) bool {
	var n int
	if err := e.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, name).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// derivedSnapshotSpec : un dérivé ancré (player DB) à exporter par joueur. La requête
// filtre sur les matchs ready du joueur (snapshot_ready_at IS NOT NULL dans sa PME).
type derivedSnapshotSpec struct {
	name  string
	query string
}

// derivedSnapshotSpecs : les 3 dérivés ancrés lus depuis les vues `_latest` du joueur.
// skill_rank restreint à LUSR (CSR exclu : non versionné dans ce snapshot) ; les deux
// tables sans snapshot_ready_at sont filtrées par jointure sur la PME ready.
var derivedSnapshotSpecs = []derivedSnapshotSpec{
	{"player_match_enrichment", `SELECT * FROM player_match_enrichment_latest WHERE snapshot_ready_at IS NOT NULL`},
	{"match_skill_rank", `SELECT s.* FROM match_skill_rank_latest s
		WHERE s.rating_type = 'LUSR'
		  AND s.match_id IN (SELECT match_id FROM player_match_enrichment_latest WHERE snapshot_ready_at IS NOT NULL)`},
	{"match_citations", `SELECT c.* FROM match_citations_latest c
		WHERE c.match_id IN (SELECT match_id FROM player_match_enrichment_latest WHERE snapshot_ready_at IS NOT NULL)`},
}

// execer abstrait *sql.DB et *sql.Conn (même signature ExecContext) → un COPY peut
// s'exécuter sur une connexion épinglée (faits shared + table temp) ou sur le pool
// (dérivés player).
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// sqlQuote échappe une valeur en littéral chaîne SQL (chemins de destination COPY).
func sqlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// sanitizeSnapshotFilename neutralise les séparateurs de chemin d'un gamertag pour en
// faire un nom de fichier sûr (les gamertags Xbox tolèrent l'espace, jamais / ni \).
func sanitizeSnapshotFilename(gt string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return r.Replace(gt)
}

// createReadyIDTemp (re)crée la table temp _snap_ready sur la connexion épinglée et y
// insère les match_id ready par lots. CREATE OR REPLACE : robuste si la connexion
// physique est réutilisée depuis le pool (la temp DuckDB survit au retour au pool).
func createReadyIDTemp(ctx context.Context, conn *sql.Conn, ids []string) error {
	if _, err := conn.ExecContext(ctx, `CREATE OR REPLACE TEMP TABLE _snap_ready (match_id VARCHAR)`); err != nil {
		return fmt.Errorf("create temp ready: %w", err)
	}
	const chunk = 500
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		var b strings.Builder
		b.WriteString("INSERT INTO _snap_ready VALUES ")
		args := make([]any, 0, end-i)
		for j := i; j < end; j++ {
			if j > i {
				b.WriteByte(',')
			}
			b.WriteString("(?)")
			args = append(args, ids[j])
		}
		if _, err := conn.ExecContext(ctx, b.String(), args...); err != nil {
			return fmt.Errorf("insert ready ids: %w", err)
		}
	}
	return nil
}

// copyToParquet exécute `COPY (<query>) TO <destFile>` (zstd niv. 9) en créant le
// répertoire parent au besoin, et retourne le nombre de lignes écrites.
func copyToParquet(ctx context.Context, e execer, query, destFile string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", filepath.Dir(destFile), err)
	}
	stmt := fmt.Sprintf("COPY (%s) TO %s (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL 9)",
		query, sqlQuote(destFile))
	res, err := e.ExecContext(ctx, stmt)
	if err != nil {
		return 0, fmt.Errorf("copy parquet %s: %w", destFile, err)
	}
	n, _ := res.RowsAffected()
	if n < 0 {
		n = 0
	}
	return n, nil
}

// partitionInfoFor stat + checksum un fichier Parquet produit pour le manifest.
func partitionInfoFor(versionDir, relPath, table, player string, rowCount int64) (PartitionInfo, error) {
	full := filepath.Join(versionDir, relPath)
	fi, err := os.Stat(full)
	if err != nil {
		return PartitionInfo{}, fmt.Errorf("stat partition %s: %w", relPath, err)
	}
	sum, err := sha256File(full)
	if err != nil {
		return PartitionInfo{}, fmt.Errorf("sha256 partition %s: %w", relPath, err)
	}
	return PartitionInfo{
		Table: table, Player: player, RelPath: filepath.ToSlash(relPath),
		RowCount: rowCount, SizeBytes: fi.Size(), SHA256: sum,
	}, nil
}

// exportSharedFacts ouvre shared RO, épingle une connexion, matérialise l'ensemble
// ready en table temp, et COPY les 5 faits bruts filtrés. Une connexion unique pour
// que la table temp soit visible par tous les COPY.
func exportSharedFacts(ctx context.Context, opener SharedReadOpener, versionDir string, readyIDs []string) ([]PartitionInfo, error) {
	db, release, err := opener.OpenSharedRO(ctx)
	if err != nil {
		return nil, fmt.Errorf("ouverture shared RO: %w", err)
	}
	defer release()
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("conn shared: %w", err)
	}
	defer conn.Close() //nolint:errcheck
	if err := createReadyIDTemp(ctx, conn, readyIDs); err != nil {
		return nil, err
	}
	var parts []PartitionInfo
	// Tables de base + vues collapsed : match-keyed, filtrées au set ready.
	matchKeyed := make([]struct{ source, dest string }, 0, len(sharedSnapshotTables)+len(sharedSnapshotViewExports))
	for _, tbl := range sharedSnapshotTables {
		matchKeyed = append(matchKeyed, struct{ source, dest string }{tbl, tbl})
	}
	for _, ve := range sharedSnapshotViewExports {
		matchKeyed = append(matchKeyed, struct{ source, dest string }{ve.source, ve.dest})
	}
	for _, mk := range matchKeyed {
		if !relationExists(ctx, conn, mk.source) {
			continue // relation absente (schéma partiel) → la lecture dégradera vers live
		}
		rel := filepath.Join("shared", mk.dest+".parquet")
		q := fmt.Sprintf("SELECT s.* FROM %s s WHERE s.match_id IN (SELECT match_id FROM _snap_ready)", mk.source)
		n, err := copyToParquet(ctx, conn, q, filepath.Join(versionDir, rel))
		if err != nil {
			return nil, err
		}
		pi, err := partitionInfoFor(versionDir, rel, mk.dest, "", n)
		if err != nil {
			return nil, err
		}
		parts = append(parts, pi)
	}
	// Tables globales (non match-keyed) : exportées en entier.
	for _, tbl := range sharedSnapshotGlobalTables {
		if !relationExists(ctx, conn, tbl) {
			continue
		}
		rel := filepath.Join("shared", tbl+".parquet")
		n, err := copyToParquet(ctx, conn, fmt.Sprintf("SELECT * FROM %s", tbl), filepath.Join(versionDir, rel))
		if err != nil {
			return nil, err
		}
		pi, err := partitionInfoFor(versionDir, rel, tbl, "", n)
		if err != nil {
			return nil, err
		}
		parts = append(parts, pi)
	}
	return parts, nil
}

// exportOnePlayerDerived COPY les 3 dérivés ancrés d'un joueur (vues _latest), filtrés
// sur ses matchs ready. Exécuté sur le pool de la player DB (pas de table temp ⇒ pas
// besoin d'épingler une connexion).
func exportOnePlayerDerived(ctx context.Context, db *sql.DB, versionDir, gamertag string) ([]PartitionInfo, error) {
	safe := sanitizeSnapshotFilename(gamertag)
	var parts []PartitionInfo
	for _, spec := range derivedSnapshotSpecs {
		rel := filepath.Join("derived", spec.name, safe+".parquet")
		n, err := copyToParquet(ctx, db, spec.query, filepath.Join(versionDir, rel))
		if err != nil {
			return nil, err
		}
		pi, err := partitionInfoFor(versionDir, rel, spec.name, gamertag, n)
		if err != nil {
			return nil, err
		}
		parts = append(parts, pi)
	}
	return parts, nil
}
