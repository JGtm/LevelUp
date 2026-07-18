package livesync

// kill_kind_backfill.go — backfill de la MÉCANIQUE de kill Halo 5 (weapon_kills.kill_kind)
// sur les matchs DÉJÀ collectés AVANT la capture kill_kind (colonne NULL sur tout
// l'historique). Même chemin que events_backfill_surfaces (re-fetch /events PAR MATCH,
// serveur arrêté, best-effort reprenable) mais surface distincte : on re-dérive
// weapon_kills COMPLET (tous les kills du match, avec kill_kind) et on l'INSÈRE en
// NOUVELLE GÉNÉRATION (persist.PersistWeaponKillsNewGeneration → nextval).
//
// PIÈGE APPEND-ONLY (critique) : v_weapon_kills ne conserve, par (match_id, xuid), que la
// génération MAX. Ré-insérer un SOUS-ENSEMBLE des kills d'un couple (p.ex. le seul bucket
// « Spartan ») créerait une nouvelle génération INCOMPLÈTE qui SUPPLANTERAIT l'ancienne
// complète = perte de kills. MapKillEvents produit TOUS les kills du match (tous xuids) →
// la nouvelle génération est complète par couple. INSERT-only, zéro DELETE/UPDATE →
// ART-safe (ADR 0026). Un couple présent dans l'ancienne génération mais absent du
// re-fetch garde sa génération (clés indépendantes dans la vue) — aucune perte.
//
// Idempotence : h5KillKind renvoie TOUJOURS une valeur non vide (défaut weapon) → après
// backfill, la génération courante porte kill_kind NOT NULL et le match sort de la
// sélection (matchesMissingKillKind). Une 2e passe le saute.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/halo_5/ingest"
	"levelup/go-api/internal/persist"
)

// KillKindBackfillStats résume une passe de backfill kill_kind.
type KillKindBackfillStats struct {
	Matches  int // candidats (génération courante avec kill_kind NULL)
	Updated  int // matchs ré-insérés en nouvelle génération
	Empty    int // re-fetch sans kill exploitable (aucune ligne re-dérivée)
	FetchErr int // timelines indisponibles (skip, retry ultérieur)
	WriteErr int // écritures échouées (match sauté)
	KillRows int // weapon_kills ré-insérés
}

// RunKillKindBackfill re-dérive weapon_kills (avec kill_kind) pour les matchs H5 dont la
// génération courante porte kill_kind NULL (collectés avant la capture), et les ré-insère
// en NOUVELLE GÉNÉRATION. shared : RW single-writer (serveur arrêté). fetch injecté
// (prod = halo5.FetchCanonicalEvents sur une CaptureSource). titleSlug pilote l'exclusion
// Campagne (read-side canonique). maxMatches<=0 = tous. Best-effort par match : un fetch
// ou une écriture KO saute le match (compté), sans interrompre la passe — relancer reprend
// les matchs encore en kill_kind NULL.
func RunKillKindBackfill(ctx context.Context, shared *sql.DB, fetch FetchEventsFunc, titleSlug string, maxMatches int, logger *slog.Logger) (KillKindBackfillStats, error) {
	if logger == nil {
		logger = slog.Default()
	}
	resolveXUID, err := loadGamertagXUIDResolver(ctx, shared)
	if err != nil {
		return KillKindBackfillStats{}, fmt.Errorf("backfill kill_kind: résolveur xuid: %w", err)
	}
	ids, err := matchesMissingKillKind(ctx, shared, titleSlug, maxMatches)
	if err != nil {
		return KillKindBackfillStats{}, fmt.Errorf("backfill kill_kind: énumération: %w", err)
	}

	stats := KillKindBackfillStats{Matches: len(ids)}
	for i, id := range ids {
		events, ferr := fetch(ctx, id)
		if ferr != nil {
			stats.FetchErr++
			logger.WarnContext(ctx, "backfill kill_kind: timeline indisponible (skip)", "match_id", id, "err", ferr)
			continue
		}
		// MapKillEvents rend TOUS les kills du match (tous xuids), chacun porteur de
		// kill_kind (ev.Kind). Contrat anti-perte de PersistWeaponKillsNewGeneration :
		// génération complète par couple.
		_, weapons := ingest.MapKillEvents(id, events, resolveXUID)
		if len(weapons) == 0 {
			stats.Empty++
			logger.WarnContext(ctx, "backfill kill_kind: aucun kill re-dérivé (skip)", "match_id", id)
			continue
		}
		if werr := persist.PersistWeaponKillsNewGeneration(ctx, shared, weapons); werr != nil {
			stats.WriteErr++
			logger.WarnContext(ctx, "backfill kill_kind: écriture échouée (match sauté)", "match_id", id, "err", werr)
			continue
		}
		stats.Updated++
		stats.KillRows += len(weapons)
		if (i+1)%200 == 0 {
			logger.InfoContext(ctx, "backfill kill_kind: progression",
				"fait", i+1, "total", len(ids), "maj", stats.Updated, "kills", stats.KillRows)
		}
	}
	logger.InfoContext(ctx, "backfill kill_kind: terminé",
		"matchs", stats.Matches, "maj", stats.Updated, "vides", stats.Empty,
		"fetch_err", stats.FetchErr, "write_err", stats.WriteErr, "kill_rows", stats.KillRows)
	return stats, nil
}

// matchesMissingKillKind : match_ids H5 dont la génération COURANTE de weapon_kills porte
// au moins une ligne kill_kind NULL (collectés avant la capture kill_kind). Récents
// d'abord. La lecture passe par v_weapon_kills (génération MAX) → après backfill la
// nouvelle génération porte kill_kind et le match sort de la sélection (idempotent).
// Exclut la Campagne via l'exclusion read-side canonique (analysis, par game_variant_id :
// match_registry n'a PAS de game_mode — seul le game_variant_id discrimine, cf.
// analysis/campaign_exclusion.go). Warzone n'a aucun discriminant de schéma (non masqué
// nulle part côté lecture) ; un éventuel match Warzone résiduel re-dérivé reste INSERT-only
// ART-safe. maxMatches<=0 = tous.
func matchesMissingKillKind(ctx context.Context, shared *sql.DB, titleSlug string, maxMatches int) ([]string, error) {
	q := `SELECT r.match_id FROM match_registry r
	      WHERE EXISTS (SELECT 1 FROM v_weapon_kills w WHERE w.match_id = r.match_id AND w.kill_kind IS NULL)` +
		analysis.SQLExcludeCampaignVariants(titleSlug, "r") +
		` ORDER BY COALESCE(r.start_time_utc, r.start_time) DESC`
	if maxMatches > 0 {
		q += fmt.Sprintf(" LIMIT %d", maxMatches)
	}
	rows, err := shared.QueryContext(ctx, q)
	if err != nil {
		return nil, err
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
