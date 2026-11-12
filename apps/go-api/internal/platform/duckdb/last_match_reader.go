package duckdb

// last_match_reader.go : le dernier match persisté de chaque joueur suivi d'un titre, derrière
// `port.LastMatchReader`.

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/port"
)

var _ port.LastMatchReader = LastMatchReader{}

// ErrLastMatchRead : la lecture groupée du dernier match a échoué (base absente, requête, ligne ou
// itération) ; la cause est journalisée par le lecteur.
var ErrLastMatchRead = errors.New("lecture du dernier match impossible")

// LastMatchReader lit le dernier match de chaque joueur dans la base partagée d'un titre.
type LastMatchReader struct {
	// Log reçoit l'avertissement de chaque échec de lecture.
	Log *slog.Logger
}

// LastMatchByXUID lit, en UNE requête groupée par titre, le dernier match persisté de chaque joueur
// (timestamp canonique, règle CLAUDE.md n°8 — jamais start_time brut). Lit la base partagée du titre par
// OpenReadForQuery (réutilise le handle en cache RO/RW s'il est tenu — B-swap-safe, jamais OpenReadOnly
// forcé). Ne résout AUCUNE player-DB : un profil auth_only (sans match) est simplement absent du résultat, et
// AUCUNE DB n'est créée pour lui (E3). Best-effort : base absente ou illisible → ErrLastMatchRead, jamais de
// panic.
func (r LastMatchReader) LastMatchByXUID(ctx context.Context, sharedDBPath, titleSlug string, xuids []string) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	if len(xuids) == 0 {
		return out, nil
	}
	sqlDB, release, err := OpenReadForQuery(sharedDBPath)
	if err != nil {
		r.Log.WarnContext(ctx, "admin_freshness: shared reader indisponible",
			"title", titleSlug, "err", err)
		return out, ErrLastMatchRead
	}
	defer release()

	placeholders := make([]string, len(xuids))
	args := make([]any, len(xuids))
	for i, x := range xuids {
		placeholders[i] = "?"
		args[i] = x
	}
	q := `SELECT mp.xuid, MAX(` + analysis.SQLStartTimeCanonical("mr") + `)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid IN (` + strings.Join(placeholders, ", ") + `)
		GROUP BY mp.xuid`
	rows, err := sqlDB.QueryContext(ctx, q, args...)
	if err != nil {
		r.Log.WarnContext(ctx, "admin_freshness: requête groupée dernier match échouée",
			"title", titleSlug, "err", err)
		return out, ErrLastMatchRead
	}
	defer rows.Close()
	for rows.Next() {
		var xuid string
		var last sql.NullTime
		if err := rows.Scan(&xuid, &last); err != nil {
			r.Log.WarnContext(ctx, "admin_freshness: scan dernier match",
				"title", titleSlug, "err", err)
			return out, ErrLastMatchRead
		}
		if last.Valid {
			out[xuid] = last.Time
		}
	}
	if err := rows.Err(); err != nil {
		r.Log.WarnContext(ctx, "admin_freshness: itération dernier match",
			"title", titleSlug, "err", err)
		return out, ErrLastMatchRead
	}
	return out, nil
}
