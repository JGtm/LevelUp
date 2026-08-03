// Package duckdb — halo5_match_events_source.go : reconstruction du KILL-FEED Halo 5
// d'un match depuis le substrat LOCAL synchronisé, sans appel API live (démo : aucun
// token). Le kill-feed natif h5 est persisté sur 3 tables alignées par
// (match_id, killer_xuid, time_ms) :
//   - killer_victim_pairs : 1 ligne par kill (killer→victim gamertags + time_ms) ;
//   - weapon_kills         : 1 ligne par kill (weapon_id natif + time_ms) ;
//   - kill_positions       : 1 ligne par kill (positions monde killer/victim).
//
// La jointure produit la timeline complète « qui tue qui, avec quelle arme, quand,
// où ». Identités déjà ANONYMISÉES côté seed (DemoPlayer / Player N). Le `kind` du
// kill (melee/ground-pound…) et le headshot ne sont PAS persistés par kill → on pose
// KillKindWeapon par défaut (dégradation propre vs la voie live qui les porte).
//
// Satisfait STRUCTURELLEMENT halo_5.MatchEventsLocalSource (retour canonical, aucun
// import du package halo_5 → pas de cycle ; parité Halo5MatchHistorySource).
package halo5

import (
	"context"
	"database/sql"
	"strconv"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb"
)

// Halo5MatchEventsSource lit le kill-feed local d'un match via le SharedReader h5.
type Halo5MatchEventsSource struct {
	shared duckdb.SharedReader
}

// NewHalo5MatchEventsSource construit la source liée au SharedReader du titre h5.
func NewHalo5MatchEventsSource(shared duckdb.SharedReader) *Halo5MatchEventsSource {
	return &Halo5MatchEventsSource{shared: shared}
}

// h5KillFeedQuery — kills d'un match, ordonnés par instant. v_weapon_kills = dernière
// génération append-only par (match_id, xuid) ; LEFT JOIN tolère un kill sans arme /
// sans position (event partiel) sans le perdre.
//
// BASCULE DU 2026-08-03 : source canonique `match_kill_events_latest`. Halo 5 y arrive par la
// reprise dédupliquée de son kill-feed natif (268 337 lignes, 2 754 matchs — zéro doublon, à la
// différence d'Infinite). Le gain ici n'est donc pas le dédoublonnage : c'est que les bots
// cessent d'être une CHAÎNE VIDE. `killer_victim_pairs` code les bots Halo 5 en `”`, forme que
// rien ne distingue d'un joueur ; la canonique les porte en NULL, et les jointures
// `v_weapon_kills` / `kill_positions` cessent d'apparier sur une chaîne vide partagée par tous
// les bots du match.
const h5KillFeedQuery = `
SELECT kvp.time_ms,
       kvp.feed_killer_gamertag, kvp.victim_gamertag,
       wk.weapon_id,
       kp.killer_x, kp.killer_y, kp.killer_z,
       kp.victim_x, kp.victim_y, kp.victim_z
FROM match_kill_events_latest kvp
LEFT JOIN v_weapon_kills wk
       ON wk.match_id = kvp.match_id AND wk.xuid = kvp.feed_killer_xuid AND wk.time_ms = kvp.time_ms
LEFT JOIN kill_positions kp
       ON kp.match_id = kvp.match_id AND kp.killer_xuid = kvp.feed_killer_xuid AND kp.time_ms = kvp.time_ms
WHERE kvp.match_id = ?
ORDER BY kvp.time_ms, kvp.feed_killer_gamertag, kvp.victim_gamertag`

// GetMatchEvents reconstruit la timeline de kills d'un match. Respecte opts.Types
// (si MatchEventKill n'est pas demandé, retourne une timeline vide). Best-effort :
// erreur de lecture → remontée au caller (qui dégrade vers le live le cas échéant).
func (s *Halo5MatchEventsSource) GetMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	tl := &canonical.MatchEventTimeline{MatchID: matchID}
	if !opts.Wants(canonical.MatchEventKill) {
		return tl, nil // seuls les kills sont reconstruits depuis ce substrat
	}

	db, release, err := s.shared.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rows, err := db.QueryContext(ctx, h5KillFeedQuery, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			timeMs                 int
			killerGT, victimGT     sql.NullString
			weaponID               sql.NullInt64
			kx, ky, kz, vx, vy, vz sql.NullFloat64
		)
		if err := rows.Scan(&timeMs, &killerGT, &victimGT, &weaponID,
			&kx, &ky, &kz, &vx, &vy, &vz); err != nil {
			return nil, err
		}
		ev := canonical.MatchEvent{
			Type:      canonical.MatchEventKill,
			TimeMs:    timeMs,
			Kind:      canonical.KillKindWeapon, // mécanique par kill non persistée (défaut)
			Killer:    h5LocalIdentity(killerGT),
			Victim:    h5LocalIdentity(victimGT),
			Weapon:    h5LocalWeaponRef(weaponID),
			KillerLoc: h5LocalVec3(kx, ky, kz),
			VictimLoc: h5LocalVec3(vx, vy, vz),
		}
		tl.Events = append(tl.Events, ev)
	}
	return tl, rows.Err()
}

// h5LocalIdentity construit une PlayerIdentity gamertag-keyée (h5). nil si vide
// (kill d'environnement → killer nil).
func h5LocalIdentity(gt sql.NullString) *canonical.PlayerIdentity {
	if !gt.Valid || gt.String == "" {
		return nil
	}
	return &canonical.PlayerIdentity{Gamertag: gt.String}
}

// h5LocalWeaponRef construit l'AssetReference d'arme depuis le weapon_id natif (=
// StockId). nil si absent. Label résolu en aval (parité h5WeaponRef live).
func h5LocalWeaponRef(id sql.NullInt64) *canonical.AssetReference {
	if !id.Valid || id.Int64 == 0 {
		return nil
	}
	return &canonical.AssetReference{Kind: "weapon", ID: strconv.FormatInt(id.Int64, 10)}
}

// h5LocalVec3 construit une position monde. nil si une coordonnée manque.
func h5LocalVec3(x, y, z sql.NullFloat64) *canonical.Vec3 {
	if !x.Valid || !y.Valid || !z.Valid {
		return nil
	}
	return &canonical.Vec3{X: x.Float64, Y: y.Float64, Z: z.Float64}
}
