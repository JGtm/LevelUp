//go:build cgo

// report.go — contrôles LECTURE SEULE du lot 4 : tables avant/après du recompute.
//
// Chaque player DB est ouverte en access_mode=read_only, la shared y est ATTACHée
// en `sh` (READ_ONLY) pour les jointures outcome/pair_name. Les prédicats sur la
// shared forcent le scan (`col || ” = ?`) : c'est le même parti que
// recompute_after_art_rebuild.go et diag_perfsim — un contrôle d'intégrité ne doit
// pas dépendre d'un index ART dont la fiabilité est précisément l'objet du doute.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
)

// sectionsFor rend les requêtes de contrôle d'un joueur, dans l'ordre du rapport.
// Le xuid est passé en paramètre lié (?), jamais concaténé.
func sectionsFor() []struct {
	heading string
	query   string
	nArgs   int
} {
	return []struct {
		heading string
		query   string
		nArgs   int
	}{
		{
			heading: "1. Chaînes de performance (player_match_enrichment_latest)",
			query: `SELECT COALESCE(performance_chain,'(null)') AS chaine,
			               COUNT(*) AS lignes,
			               COUNT(performance_score) AS notees,
			               ROUND(median(performance_score),2) AS mediane,
			               ROUND(quantile_cont(performance_score,0.10),2) AS p10,
			               ROUND(quantile_cont(performance_score,0.90),2) AS p90
			        FROM player_match_enrichment_latest
			        GROUP BY 1 ORDER BY 1`,
		},
		{
			heading: "2. Contrôles stricts (attendus : ranked_restante=0, notes_sur_dnf=0)",
			query: `SELECT
			  (SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL) AS notees_total,
			  (SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_chain='ranked') AS ranked_restante,
			  (SELECT COUNT(*) FROM player_match_enrichment_latest e
			     JOIN sh.match_participants p ON p.match_id || '' = e.match_id
			    WHERE p.xuid || '' = ? AND e.performance_score IS NOT NULL AND p.outcome = 4) AS notes_sur_dnf,
			  (SELECT COUNT(*) FROM sh.match_participants p
			    WHERE p.xuid || '' = ? AND p.outcome = 4) AS dnf_total`,
			nArgs: 2,
		},
		{
			heading: "3. LUSR — match_skill_rank_latest par type et groupe",
			query: `SELECT rating_type, COALESCE(playlist_group,'(null)') AS groupe, COUNT(*) AS n
			        FROM match_skill_rank_latest GROUP BY 1,2 ORDER BY 1,2`,
		},
		{
			heading: "4. Volumes bruts (append-only vs vue)",
			query: `SELECT 'player_match_enrichment' AS tabl, COUNT(*) AS n FROM player_match_enrichment
			        UNION ALL SELECT 'player_match_enrichment_latest', COUNT(*) FROM player_match_enrichment_latest
			        UNION ALL SELECT 'match_skill_rank', COUNT(*) FROM match_skill_rank
			        UNION ALL SELECT 'match_skill_rank_latest', COUNT(*) FROM match_skill_rank_latest`,
		},
	}
}

// runReport imprime les tables de contrôle des joueurs demandés.
func runReport(ctx context.Context, env runEnv) error {
	fmt.Printf("=== CONTRÔLES PERF/LUSR [%s] — %s ===\n\n",
		env.label, time.Now().Format(time.RFC3339))

	return forEachPlayerRO(ctx, env, func(gt, xuid string, db *sql.DB) error {
		fmt.Printf("---------- %s (xuid=%s) ----------\n", gt, xuid)
		for _, s := range sectionsFor() {
			args := make([]any, s.nArgs)
			for i := range args {
				args[i] = xuid
			}
			fmt.Printf("\n%s\n", s.heading)
			if err := printQuery(ctx, db, s.query, args...); err != nil {
				return fmt.Errorf("%s / %s: %w", gt, s.heading, err)
			}
		}
		fmt.Println()
		return nil
	})
}

// runSQL exécute une requête ad hoc par joueur. Le jeton {{xuid}} y est remplacé
// par le xuid résolu, et `sh` désigne la shared attachée en lecture seule.
// Le fragment timezone canonique est disponible via {{start_time}}.
func runSQL(ctx context.Context, env runEnv, query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("mode sql : -sql est vide")
	}
	return forEachPlayerRO(ctx, env, func(gt, xuid string, db *sql.DB) error {
		q := strings.ReplaceAll(query, "{{xuid}}", "'"+xuid+"'")
		q = strings.ReplaceAll(q, "{{start_time}}", analysis.SQLStartTimeCanonical("r"))
		fmt.Printf("---------- %s (xuid=%s) ----------\n", gt, xuid)
		if err := printQuery(ctx, db, q); err != nil {
			return fmt.Errorf("%s: %w", gt, err)
		}
		fmt.Println()
		return nil
	})
}

// forEachPlayerRO ouvre la shared en RO une fois, puis chaque player DB en RO avec
// la shared attachée, et applique fn. Tout est fermé avant le joueur suivant : le
// modèle mono-process n'aime pas les handles qui traînent.
func forEachPlayerRO(ctx context.Context, env runEnv, fn func(gt, xuid string, db *sql.DB) error) error {
	sharedPath := env.paths.SharedDBPath(titleSlug)
	xuids, err := resolveAllXUIDs(ctx, sharedPath, env.players)
	if err != nil {
		return err
	}

	for _, gt := range env.players {
		db, err := openDB(env.paths.PlayerDBPath(titleSlug, gt), true)
		if err != nil {
			return err
		}
		if err := attachShared(ctx, db, sharedPath); err != nil {
			db.Close()
			return fmt.Errorf("attach shared pour %s: %w", gt, err)
		}
		err = fn(gt, xuids[gt], db)
		db.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// resolveAllXUIDs résout les xuids puis REFERME la shared : elle sera ensuite
// ATTACHée par chaque player DB, et deux handles concurrents sur le même fichier
// n'apportent rien qu'un risque de configuration divergente.
func resolveAllXUIDs(ctx context.Context, sharedPath string, players []string) (map[string]string, error) {
	shared, err := openDB(sharedPath, true)
	if err != nil {
		return nil, err
	}
	defer shared.Close()

	xuids := make(map[string]string, len(players))
	for _, gt := range players {
		x, err := resolveXUID(ctx, shared, gt)
		if err != nil {
			return nil, err
		}
		xuids[gt] = x
	}
	return xuids, nil
}

// attachShared attache la shared en lecture seule sous le nom `sh`.
func attachShared(ctx context.Context, db *sql.DB, sharedPath string) error {
	escaped := strings.ReplaceAll(sharedPath, "'", "''")
	_, err := db.ExecContext(ctx, "ATTACH '"+escaped+"' AS sh (READ_ONLY)")
	return err
}

// printQuery exécute une requête et imprime le résultat en TSV (en-tête + lignes).
// Le jeu de résultats est entièrement consommé avant tout retour : le pool RO est
// borné à une connexion.
func printQuery(ctx context.Context, db *sql.DB, query string, args ...any) error {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("columns: %w", err)
	}
	fmt.Println(strings.Join(cols, "\t"))

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	n := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		cells := make([]string, len(cols))
		for i, v := range vals {
			cells[i] = render(v)
		}
		fmt.Println(strings.Join(cells, "\t"))
		n++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}
	fmt.Printf("(%d lignes)\n", n)
	return nil
}

func render(v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04")
	default:
		return fmt.Sprintf("%v", t)
	}
}
