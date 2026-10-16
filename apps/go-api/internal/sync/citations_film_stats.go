package sync

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
)

// Clés de `CitationContext.Stats` tirées du résumé d'usage du film (shared.match_usage_players_latest,
// vue append-only — ADR 0026). Elles sont le stat_name des citations qui les lisent.
const (
	// filmStatPadPickups : armes prises sur socle par le joueur (« Jouets éducatifs »).
	filmStatPadPickups = "pad_pickups"
	// filmStatShieldlessKills : frags bouclier à 0 suivis de la survie (« Presque, ça ne compte pas »).
	filmStatShieldlessKills = "shieldless_kills"
)

// loadFilmStats charge les grandeurs du résumé d'usage du film pour (match, joueur). Best-effort,
// comme loadObjectiveStats : nil sans ligne (match sans film dérivé, ou joueur sans usage) ; une
// colonne NULL (frags sans bouclier non mesurés) n'ouvre aucune clé. Un titre sans film (vue
// absente) se dit en DEBUG ; toute autre erreur SQL en WARN — les citations restent alors à 0.
func loadFilmStats(ctx context.Context, sharedDB *sql.DB, matchID, xuid string) map[string]float64 {
	if sharedDB == nil {
		return nil
	}
	var pads sql.NullInt64
	var shieldless sql.NullInt64
	err := sharedDB.QueryRowContext(ctx, `SELECT pad_pickups, shieldless_kills
		FROM match_usage_players_latest WHERE match_id = ? AND xuid = ? LIMIT 1`, matchID, xuid).
		Scan(&pads, &shieldless)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			slog.DebugContext(ctx, "citations: résumé d'usage absent (titre sans film)", "match_id", matchID, "err", err)
		} else {
			slog.WarnContext(ctx, "citations: résumé d'usage illisible, citations du film à 0", "match_id", matchID, "err", err)
		}
		return nil
	}
	out := make(map[string]float64, 2)
	if pads.Valid {
		out[filmStatPadPickups] = float64(pads.Int64)
	}
	if shieldless.Valid {
		out[filmStatShieldlessKills] = float64(shieldless.Int64)
	}
	return out
}
