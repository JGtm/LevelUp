package citationsfilm

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// SelectMatchesFilmPlusRecent rend les matchs du joueur dont les citations ont été calculées AVANT
// la dernière dérivation de leur film (résumé d'usage). Les citations du film (« Jouets éducatifs »,
// « Presque, ça ne compte pas ») lisent ce résumé ; il arrive souvent APRÈS l'étape citations — film
// cuit par un ouvrier, rattrapage des dérivés, `levelup backfill-usage-summary` — et le post-sync ne
// reprend jamais un match qui a déjà des citations. Le recalcul est COMPLET (la vue
// `match_citations_latest` ne sert que la dernière génération d'un match).
//
// Comparaison d'horodatages UTC écrits par les deux tables (`written_at`). Un titre sans film (vue
// absente sur la base partagée) ne rend rien.
func SelectMatchesFilmPlusRecent(ctx context.Context, playerDB, sharedDB *sql.DB) ([]string, error) {
	calculees, err := citationsWrittenAt(ctx, playerDB)
	if err != nil || len(calculees) == 0 {
		return nil, err
	}
	rows, err := sharedDB.QueryContext(ctx, `SELECT match_id, MAX(written_at) FROM match_usage_films_latest GROUP BY match_id`)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			slog.DebugContext(ctx, "citations post-sync: résumé d'usage absent (titre sans film)", "err", err)
			return nil, nil
		}
		return nil, fmt.Errorf("citations: dérivations du film: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		var derive time.Time
		if err := rows.Scan(&id, &derive); err != nil {
			return nil, fmt.Errorf("citations: scan des dérivations du film: %w", err)
		}
		if calcule, ok := calculees[id]; ok && derive.After(calcule) {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

// citationsWrittenAt : pour chaque match du joueur, l'instant d'écriture de sa génération courante.
func citationsWrittenAt(ctx context.Context, playerDB *sql.DB) (map[string]time.Time, error) {
	rows, err := playerDB.QueryContext(ctx, `SELECT match_id, MAX(written_at) FROM match_citations_latest GROUP BY match_id`)
	if err != nil {
		return nil, fmt.Errorf("citations: générations courantes: %w", err)
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("citations: scan des générations: %w", err)
		}
		out[id] = at
	}
	return out, rows.Err()
}

// UnionMatchIDs concatène deux listes sans doublon, dans l'ordre d'arrivée.
func UnionMatchIDs(a, b []string) []string {
	vus := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, l := range [][]string{a, b} {
		for _, id := range l {
			if !vus[id] {
				vus[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}
