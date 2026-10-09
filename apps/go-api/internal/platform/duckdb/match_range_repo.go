// Package duckdb — match_range_repo.go : la portée de frag de TOUT UN LOBBY, par match
// (port.MatchRangeRepository ; plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md,
// lot N2, réserve R3).
//
// # UN SEUL CÔTÉ, ET C'EST LE POINT
//
// `WeaponRangeRepo.loadBothSides` rend deux côtés : tueur et victime. Sans filtre de joueur,
// ces deux côtés porteraient EXACTEMENT LES MÊMES FRAGS — chaque frag du scope est à la fois
// « un frag de quelqu'un » et « une mort de quelqu'un d'autre » — et chaque mesure compterait
// double dans la médiane du lobby. Cette lecture-ci ne prend donc que le côté TUEUR : la
// question posée est « à quelle distance CE JOUEUR frague », et le tueur est celui que
// `kill_positions` clé.
//
// # LA JOINTURE N'EST PAS RECOPIÉE
//
// Elle vit dans kill_measured.go et nulle part ailleurs (garde-rail
// kill_measured_guard_test.go) ; ce fichier compose sa clause de portée par le MÊME
// `buildWeaponRangeQuery`, avec `AllPlayers` qui en retire le filtre de joueur.
//
// # LE DÉNOMINATEUR EST UNE SECONDE REQUÊTE, PAS UNE ESTIMATION
//
// Les frags dont les positions manquent sont par définition absents de la jointure : on ne
// peut pas les compter dedans. Le compte des frags PUBLIABLES du scope se lit donc sur la
// vue canonique seule — trois lignes de SQL, le même scope de match_id, aucune jointure.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

var _ port.MatchRangeRepository = (*WeaponRangeRepo)(nil)

// matchRangeKillsTotalSQL compte les frags PUBLIABLES des matchs du scope — le dénominateur
// de la couverture. `publishable` et `feed_killer_xuid IS NOT NULL` sont les MÊMES gardes
// que la requête mesurée : un dénominateur plus large que son numérateur n'est pas une
// couverture, c'est un chiffre qui fait peur sans rien dire.
const matchRangeKillsTotalSQL = `
SELECT count(*)
FROM ` + KillEventsCanonicalTable + `
WHERE match_id IN (%s)
  AND publishable
  AND feed_killer_xuid IS NOT NULL`

// LoadMatchRangeKills rend les frags mesurés de TOUS les joueurs des matchs du scope, du
// seul côté tueur, avec le nombre de frags publiables de ces mêmes matchs.
//
// Best-effort sur le SEUL dénominateur : une erreur sur le compte est journalisée et rend
// KillsTotal=0 plutôt que de faire échouer la lecture des frags. L'appelant qui publie une
// couverture sait qu'un dénominateur nul veut dire « inconnue » — il ne divise pas par zéro.
func (r *WeaponRangeRepo) LoadMatchRangeKills(
	ctx context.Context, slug string, filters port.WeaponRangeFilters,
) (port.MatchRangeRead, error) {
	const scope = "LoadMatchRangeKills"
	if err := filters.Validate(); err != nil {
		return port.MatchRangeRead{}, fmt.Errorf("WeaponRangeRepo.%s: %w", scope, err)
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "WeaponRangeRepo: no classifier for title, nothing to load",
			"scope", scope, "slug", slug)
		return port.MatchRangeRead{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, weaponRangeQueryTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "WeaponRangeRepo: shared reader unavailable",
			"scope", scope, "slug", slug, "err", err)
		return port.MatchRangeRead{}, fmt.Errorf("WeaponRangeRepo.%s: shared reader: %w", scope, err)
	}
	defer release()

	q, args := buildWeaponRangeQuery(positionsAtKill, weaponRangeKillerColumn, filters)
	measured, err := queryMeasuredKills(ctx, db, q, args, scope)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "WeaponRangeRepo: positions table missing",
				"scope", scope, "slug", slug, "table", string(positionsAtKill))
			return port.MatchRangeRead{}, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "WeaponRangeRepo: query failed", "scope", scope,
			"slug", slug, "err", err)
		return port.MatchRangeRead{}, fmt.Errorf("WeaponRangeRepo.%s: %w", scope, err)
	}

	out := port.MatchRangeRead{
		Kills:      r.toMeasuredKills(ctx, measured, analysis.SideKiller, scope),
		KillsTotal: r.countScopeKills(ctx, db, filters.MatchIDs, scope),
	}
	slog.DebugContext(ctx, "WeaponRangeRepo: lobby range read", "scope", scope, "slug", slug,
		"matchs", len(filters.MatchIDs), "frags_mesures", len(out.Kills),
		"frags_publiables", out.KillsTotal)
	return out, nil
}

// countScopeKills lit le dénominateur de la couverture. Erreur journalisée puis 0 rendu :
// une couverture inconnue est un état, pas une panne de la page.
func (r *WeaponRangeRepo) countScopeKills(
	ctx context.Context, db *sql.DB, matchIDs []string, scope string,
) int {
	args := make([]any, 0, len(matchIDs))
	for _, id := range matchIDs {
		args = append(args, id)
	}
	var total int
	q := fmt.Sprintf(matchRangeKillsTotalSQL, Placeholders(len(matchIDs)))
	if err := db.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		slog.WarnContext(ctx, "WeaponRangeRepo: scope kills count failed, coverage unknown",
			"scope", scope, "err", err)
		return 0
	}
	return total
}
