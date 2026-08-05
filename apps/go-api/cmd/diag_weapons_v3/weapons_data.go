package main

// weapons_data.go — lecture de la baseline v2 (table weapon_kills GELÉE, RO) et
// des agrégats per-joueur (match_participants.melee_kills/grenade_kills) pour la
// comparaison §4. AUCUNE écriture sur weapon_kills (v2 gelée).

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/platform/duckdb"
)

// v2Kill — une ligne weapon_kills v2, appariée à v3 par (xuid, time_ms).
type v2Kill struct {
	xuid       string
	timeMS     int
	weaponID   *uint64
	confidence string
	path       string
}

// v2KeyOf renvoie la clé d'appariement (xuid, time_ms) d'un kill v2.
func v2KeyOf(k v2Kill) killKey { return killKey{xuid: k.xuid, timeMS: k.timeMS} }

// killKey apparie v2<->v3 par (xuid, time_ms).
type killKey struct {
	xuid   string
	timeMS int
}

// participantAgg — agrégats per-joueur depuis match_participants (melee/grenade
// kills déclarés par l'API). hasCols=false si les colonnes sont absentes du schéma.
type participantAgg struct {
	byXUID  map[string]aggCounts
	hasCols bool
}

// aggCounts — compteurs melee/grenade d'un joueur (API).
type aggCounts struct {
	melee   int
	grenade int
}

// loadV2Baseline lit la baseline v2 weapon_kills du match (lecture seule).
// time_ms, weapon_id, confidence, attribution_path, xuid (cf. tâche couche 5).
func loadV2Baseline(ctx context.Context, db *sql.DB, matchID string) (map[killKey]v2Kill, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid, time_ms, weapon_id, confidence, COALESCE(attribution_path, 'none')
		FROM weapon_kills WHERE match_id = ?`, matchID)
	if err != nil {
		return nil, fmt.Errorf("loadV2Baseline(%s): %w", matchID, err)
	}
	defer rows.Close()

	out := map[killKey]v2Kill{}
	for rows.Next() {
		var (
			k      v2Kill
			weapon duckdb.NullableUBigint
		)
		if err := rows.Scan(&k.xuid, &k.timeMS, &weapon, &k.confidence, &k.path); err != nil {
			continue
		}
		if weapon.Valid {
			v := uint64(weapon.Value.Int64()) //nolint:gosec // bit-preserving reinterpret
			k.weaponID = &v
		}
		out[v2KeyOf(k)] = k
	}
	return out, rows.Err()
}

// loadParticipantAggregates lit melee_kills/grenade_kills par joueur depuis
// match_participants. Si les colonnes n'existent pas (schéma plus ancien), renvoie
// hasCols=false (la comparaison §4.7 est alors notée comme skippée).
func loadParticipantAggregates(ctx context.Context, db *sql.DB, matchID string) (participantAgg, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid, COALESCE(melee_kills, 0), COALESCE(grenade_kills, 0)
		FROM match_participants WHERE match_id = ?`, matchID)
	if err != nil {
		// Colonnes/table absentes → on dégrade proprement (skip la section §4.7).
		return participantAgg{hasCols: false}, nil //nolint:nilerr // dégradation gracieuse documentée
	}
	defer rows.Close()

	agg := participantAgg{byXUID: map[string]aggCounts{}, hasCols: true}
	for rows.Next() {
		var (
			xuid string
			m, g int
		)
		if err := rows.Scan(&xuid, &m, &g); err != nil {
			continue
		}
		agg.byXUID[xuid] = aggCounts{melee: m, grenade: g}
	}
	return agg, rows.Err()
}
