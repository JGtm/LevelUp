package scheduler

// data_health_psa_index.go — garde périodique « index ART de personal_score_awards
// désynchronisé » (volet 2 du chantier note de perf, 2026-08-28).
//
// CONTEXTE. Le 2026-08-27, les 4 player DB locales portaient deux index ART
// incohérents sur personal_score_awards (idx_psa_match, idx_psa_category) : un
// lookup servi par l'index rendait MOINS de lignes qu'un scan séquentiel de la
// même table (jusqu'à 5,7 % des lignes invisibles).
//
// L'issue upstream qui décrit EXACTEMENT ce symptôme est duckdb/duckdb#23645
// (« Failed to delete all rows from index », TOUJOURS OUVERTE, reproduite en
// 1.5.5 — la version que nous embarquons ; un commentaire y documente le même
// canari « comptage filtré < GROUP BY »). Ce n'est PAS #23046, que CLAUDE.md cite
// et qui porte sur une corruption de tas en 1.5.0 (découverte notée au rapport,
// non traitée ici). La cause n'a PAS pu être reproduite localement sur 1.5.5
// (12 scénarios, cf. RAPPORT_VOLET2_INDEX_PSA.md) : on ne sait donc pas si le
// vecteur est éteint. Faute de cause, on pose la DÉTECTION.
//
// CE QUE CETTE GARDE FAIT — ET NE FAIT PAS.
//   - DÉTECTE et ALERTE (slog + compteurs + jauge expvar). Le signal remonte au
//     panneau monitoring via DataHealthCheckResult.WarningsTotal.
//   - NE RÉPARE JAMAIS. Une réparation automatique et silencieuse d'index sur une
//     base dont la corruption n'est pas comprise est plus dangereuse que le
//     symptôme : la réparation (DROP + CREATE INDEX) masquerait la récurrence et
//     empêcherait de mesurer la fréquence réelle. La réparation reste MANUELLE et
//     tracée : `go run ./cmd/repair_psa_index -repair` (serveur arrêté).
//     Divergence assumée avec l'auto-heal LUSR voisin : celui-ci REJOUE un calcul
//     déterministe (aucune donnée détruite), ici on toucherait à des structures
//     de stockage d'une base suspecte.
//
// COÛT BORNÉ. Le contrôle échantillonne psaIndexSampleKeys clés distinctes
// (réservoir, tirage différent à chaque cycle → la couverture s'accumule sur les
// cycles successifs) : 1 scan groupé de la table + N lookups indexés. Mesuré à
// ~30-60 ms par player DB sur un corpus de 1 100 matchs, soit < 0,3 s pour les 4
// joueurs — négligeable devant le cycle data-health (24 h).
//
// TITLE-AGNOSTIC. Aucun `slug == "..."` : la garde boucle sur les titres du
// registre et sonde la PRÉSENCE de la table dans chaque player DB. Aucune clé de
// `capabilities.toml` ne décrit `personal_score_awards` (la plus proche,
// analytics.career_xp_estimate, a une autre sémantique) ; la présence de la table
// est donc le prédicat structurel correct — un titre qui n'écrit pas d'awards est
// skippé silencieusement, exactement comme les autres sondes de ce package.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	// psaIndexSampleKeys : nombre de clés match_id distinctes contrôlées par
	// player DB et par cycle. Borne le coût ; le tirage étant aléatoire, les
	// cycles successifs balaient progressivement le corpus.
	psaIndexSampleKeys = 200
	// psaIndexProbeTimeout : garde-fou par player DB (une sonde qui traîne ne doit
	// pas immobiliser le cycle data-health).
	psaIndexProbeTimeout = 60 * time.Second
	// psaIndexGauge : jauge expvar publiée à chaque cycle COMPLET (clés en écart).
	psaIndexGauge = "data_health_psa_index_desync_keys"
)

// errPSATableAbsent : la player DB ne porte pas personal_score_awards (titre qui
// n'écrit pas d'awards, DB neuve). Skip silencieux — ce n'est pas une anomalie.
var errPSATableAbsent = errors.New("personal_score_awards absente")

// psaIndexReport : résultat du contrôle d'UNE player DB.
type psaIndexReport struct {
	KeysSampled   int // clés match_id effectivement contrôlées
	KeysDiverging int // clés dont le lookup indexé ne rend pas le compte du scan
	RowsMissing   int // total de lignes invisibles au lookup indexé (scan - indexé)
}

// scanPSAIndexDesync compare, sur un échantillon de clés match_id, le comptage
// par SCAN FORCÉ au comptage par LOOKUP INDEXÉ. Même principe que le `-dry-run`
// de cmd/repair_psa_index :
//   - référence : `GROUP BY match_id || ”` — la clé est une EXPRESSION, qu'aucun
//     index ART ne peut servir, donc scan séquentiel garanti ;
//   - contrôle : `WHERE match_id = ?` — colonne NUE, servie par idx_psa_match.
//
// Les clés NULL sont exclues (`col = NULL` ne matche jamais → faux écart).
// Tout écart = index désynchronisé de la table.
func scanPSAIndexDesync(ctx context.Context, db *sql.DB, sampleKeys int) (psaIndexReport, error) {
	var rep psaIndexReport
	var present int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'personal_score_awards'`,
	).Scan(&present); err != nil {
		return rep, fmt.Errorf("présence de la table: %w", err)
	}
	if present == 0 {
		return rep, errPSATableAbsent
	}

	refs, err := psaSampleKeyCounts(ctx, db, sampleKeys)
	if err != nil {
		return rep, err
	}
	if len(refs) == 0 {
		return rep, nil
	}

	stmt, err := db.PrepareContext(ctx,
		`SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?`)
	if err != nil {
		return rep, fmt.Errorf("préparation du lookup indexé: %w", err)
	}
	defer stmt.Close() //nolint:errcheck // statement de lecture

	return comparePSACounts(refs, func(key string) (int, error) {
		var indexed int
		if err := stmt.QueryRowContext(ctx, key).Scan(&indexed); err != nil {
			return 0, fmt.Errorf("lookup indexé (clé %s): %w", key, err)
		}
		return indexed, nil
	})
}

// comparePSACounts est la RÈGLE DE DÉTECTION isolée de tout SQL : pour chaque clé,
// le comptage indexé doit égaler le comptage de référence. Toute inégalité compte
// une clé en écart ; seul un DÉFICIT (indexé < scan) alimente RowsMissing — un
// excédent serait une autre pathologie (doublons d'index) et ne doit pas être
// masqué par une soustraction négative.
func comparePSACounts(refs map[string]int, lookup func(string) (int, error)) (psaIndexReport, error) {
	rep := psaIndexReport{KeysSampled: len(refs)}
	for key, scanned := range refs {
		indexed, err := lookup(key)
		if err != nil {
			return rep, err
		}
		if indexed == scanned {
			continue
		}
		// NB : le déficit est calculé AVANT l'incrément — pas par goût, mais parce
		// que la séquence `x++` suivie de `if a > b {` est le motif interdit par le
		// ratchet archlint F2 (balayage « plus longue série »). Faux positif ici :
		// aucune série n'est mesurée. Réordonner coûte moins qu'une exemption.
		if scanned > indexed {
			rep.RowsMissing += scanned - indexed
		}
		rep.KeysDiverging++
	}
	return rep, nil
}

// psaSampleKeyCounts tire au sort sampleKeys clés match_id distinctes et retourne
// leur cardinalité de RÉFÉRENCE (scan forcé). Le réservoir n'est pas graine : le
// tirage change à chaque cycle, la couverture s'accumule.
func psaSampleKeyCounts(ctx context.Context, db *sql.DB, sampleKeys int) (map[string]int, error) {
	query := fmt.Sprintf(`
		SELECT k, n FROM (
			SELECT match_id || '' AS k, COUNT(*) AS n
			FROM personal_score_awards
			WHERE match_id IS NOT NULL
			GROUP BY 1
		) USING SAMPLE %d ROWS (reservoir)`, sampleKeys)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("scan de référence: %w", err)
	}
	defer rows.Close() //nolint:errcheck // lecture

	out := make(map[string]int, sampleKeys)
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, fmt.Errorf("scan de référence (ligne): %w", err)
		}
		out[k] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("itération du scan de référence: %w", err)
	}
	return out, nil
}

// auditTitlePSAIndex contrôle les player DB d'UN titre et AGRÈGE dans res.
// Best-effort, sémantique « unmeasured ≠ sain » alignée sur auditTitleLUSRGaps.
func (s *HealthScheduler) auditTitlePSAIndex(ctx context.Context, pr *titlePkg.PathResolver, slug string, res *DataHealthCheckResult) {
	entries, err := os.ReadDir(pr.PlayersRootDir(slug))
	if err != nil {
		slog.WarnContext(ctx, "data_health: répertoire joueurs illisible — index PSA non mesuré",
			"titleSlug", slug, "err", err)
		res.PSAIndexPlayersUnmeasured++
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		playerPath := pr.PlayerDBPath(slug, e.Name())
		if _, statErr := os.Stat(playerPath); statErr != nil {
			continue
		}
		s.auditPlayerPSAIndex(ctx, slug, e.Name(), playerPath, res)
	}
}

// auditPlayerPSAIndex contrôle UNE player DB. Lecture via OpenReadForQuery
// (jamais OpenReadOnly forcé) : si un sync tient déjà le fichier en RW dans ce
// process, le handle en cache est réutilisé au lieu d'échouer sur
// « different configuration » (ADR 0013/0016).
func (s *HealthScheduler) auditPlayerPSAIndex(ctx context.Context, slug, gamertag, playerPath string, res *DataHealthCheckResult) {
	sqlDB, release, err := duckdb.OpenReadForQuery(playerPath)
	if err != nil {
		slog.WarnContext(ctx, "data_health: player DB inouvrable pour contrôle index PSA (non mesuré)",
			"titleSlug", slug, "gamertag", gamertag, "err", err)
		res.PSAIndexPlayersUnmeasured++
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, psaIndexProbeTimeout)
	rep, probeErr := scanPSAIndexDesync(probeCtx, sqlDB, psaIndexSampleKeys)
	cancel()
	release()

	if errors.Is(probeErr, errPSATableAbsent) {
		slog.DebugContext(ctx, "data_health: personal_score_awards absente — skip contrôle index",
			"titleSlug", slug, "gamertag", gamertag)
		return
	}
	if probeErr != nil {
		slog.WarnContext(ctx, "data_health: contrôle index PSA échoué",
			"titleSlug", slug, "gamertag", gamertag, "err", probeErr)
		res.ProbeErrors++
		res.PSAIndexPlayersUnmeasured++
		return
	}

	res.PSAIndexPlayersScanned++
	if rep.KeysDiverging == 0 {
		return
	}
	res.PSAIndexDesyncPlayers++
	res.PSAIndexDesyncKeys += rep.KeysDiverging
	res.PSAIndexRowsMissing += rep.RowsMissing
	// ALERTE, PAS DE RÉPARATION (cf. en-tête du fichier) : la remédiation est
	// manuelle et tracée, `go run ./cmd/repair_psa_index -repair` serveur arrêté.
	slog.ErrorContext(ctx, "data_health: index personal_score_awards DÉSYNCHRONISÉ (lookup indexé < scan) — réparation MANUELLE requise",
		"titleSlug", slug, "gamertag", gamertag,
		"keys_sampled", rep.KeysSampled,
		"keys_diverging", rep.KeysDiverging,
		"rows_missing", rep.RowsMissing,
		"remediation", "go run ./cmd/repair_psa_index -repair (serveur arrêté)")
}

// publishPSAIndexGaugeIfComplete republie la jauge expvar — SEULEMENT si le
// contrôle a été COMPLET. Un contrôle partiel (player DB tenue RW, sonde en
// échec) sous-compte : republier éteindrait le signal à tort (« unmeasured ≠
// sain », même invariant que la jauge LUSR).
func publishPSAIndexGaugeIfComplete(ctx context.Context, res *DataHealthCheckResult) {
	if res.PSAIndexPlayersUnmeasured == 0 {
		observability.SetInt(psaIndexGauge, int64(res.PSAIndexDesyncKeys))
		return
	}
	slog.WarnContext(ctx, "data_health: jauge index PSA non mise à jour (contrôle partiel)",
		"psa_index_players_unmeasured", res.PSAIndexPlayersUnmeasured,
		"psa_index_desync_keys_scanned", res.PSAIndexDesyncKeys)
}
