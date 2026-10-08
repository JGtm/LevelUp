// Package duckdb — squad_vehicle_repo.go : la ressource « véhicules » du bloc Emprise (plan
// `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3, décisions D5, D8, D9, D10 ; ADR 0036).
//
// UN CHARGEMENT PAR REQUÊTE, TROIS LECTURES BORNÉES PAR LA LISTE DES MATCHS :
//
//	prises   `match_vehicle_takes_latest` (ADR 0026 : la vue seulement), `match_id` lié en UNE
//	         constante `VARCHAR[]` que DuckDB pousse sous la fenêtre de la vue (I2). La vue rend la
//	         DERNIÈRE PASSE ENTIÈRE de chaque match : lignes de prise + ligne `match` (couverture).
//	frags    `match_kill_events_latest`, pour les SEULS matchs dont la passe est mesurée, a apparié
//	         ses frags et en compte (les autres n'ont aucun frag à répartir par camp). Le tueur du
//	         kill-feed, la source de dégât MESURÉE, traduite en clé de registre par le classificateur
//	         du titre, puis en classe par le MÊME résolveur que la Répartition des frags
//	         (`resolveWeaponKeyDimensions`) : une seule définition de « frag de classe véhicule »
//	         (`domain.IsEngineFragClass`, D5). Les écrasements, sans clé de registre, sortent du
//	         compte, comme dans la Répartition. Le camp du tueur vient de `match_participants`.
//	camp     le camp du joueur de la page dans chaque match (`match_participants`, table ordinaire,
//	         semi-jointure).
//
// Jamais `v_gamertag_lookup` (I1) : aucun nom n'est lu ici.
//
// Table absente : games.ErrCapabilityNotSupported. Sans classificateur (titre sans
// `film.kill_source`), les prises se lisent et aucun match n'est « événements lus » : le rendement
// n'existe pas, la ressource reste mesurée.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// SquadVehicleRepo lit la ressource véhicules sur le SharedReader du joueur.
type SquadVehicleRepo struct {
	pdb        *PlayerDB
	classifier port.KillSourceClassifier
}

// NewSquadVehicleRepo construit le repo. classifier peut être nil (aucun frag par camp).
func NewSquadVehicleRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *SquadVehicleRepo {
	return &SquadVehicleRepo{pdb: pdb, classifier: classifier}
}

const squadVehicleQueryTimeout = 15 * time.Second

// qSquadVehicleTakes : le prédicat de liste est posé par LoadVehicleUsage (%s). Ordre total :
// la ligne `match` d'abord (`match` < `take`), puis camp, joueur, famille.
const qSquadVehicleTakes = `
SELECT v.match_id, v.row_kind, v.camp, v.xuid, v.family, v.takes, v.aboard_ms, v.episodes,
       v.proximity_episodes, v.frags, v.measured, v.unmeasured_reason, v.doc_schema,
       v.episodes_read, v.episodes_unnamed, v.episodes_no_camp, v.frags_read, v.frags_reason,
       v.frags_total, v.frags_unmatched
FROM match_vehicle_takes_latest v
WHERE %s
ORDER BY v.match_id, v.row_kind, v.camp, v.xuid, v.family`

// qSquadVehicleFrags : les frags mesurés des matchs demandés, par (match, camp du tueur, source).
const qSquadVehicleFrags = `
SELECT k.match_id, p.team_id, k.source_tag, COUNT(*)::INTEGER AS frags
FROM match_kill_events_latest k
LEFT JOIN match_participants p ON p.match_id = k.match_id AND p.xuid = k.feed_killer_xuid
WHERE %s AND k.source_tag IS NOT NULL AND k.feed_killer_xuid IS NOT NULL
GROUP BY k.match_id, p.team_id, k.source_tag
ORDER BY k.match_id, p.team_id, k.source_tag`

// LoadVehicleUsage rend les prises, la couverture des passes, les frags d'engin par camp et le camp
// du joueur sur les `matchIDs`. Liste vide : rien à lire. Table absente :
// games.ErrCapabilityNotSupported.
func (r *SquadVehicleRepo) LoadVehicleUsage(
	ctx context.Context, matchIDs []string, playerXUID string,
) (squademprise.VehicleRead, error) {
	out := squademprise.VehicleRead{EventsRead: map[string]bool{}, PlayerTeam: map[string]int{}}
	if len(matchIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, squadVehicleQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("SquadVehicleRepo: shared reader: %w", err)
	}
	defer release()

	if err := r.lirePrises(ctx, db, matchIDs, &out); err != nil {
		return out, err
	}
	if err := r.lireFrags(ctx, db, &out); err != nil {
		return out, err
	}
	return out, lireCampDuJoueur(ctx, db, pdbTitleSlug(r.pdb), matchIDs, playerXUID, out.PlayerTeam)
}

// lirePrises lit la dernière passe entière de chaque match : la ligne `match` et les prises.
func (r *SquadVehicleRepo) lirePrises(ctx context.Context, db *sql.DB, ids []string, out *squademprise.VehicleRead) error {
	par, arg := clauseListeMatchs("v.match_id", ids)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(qSquadVehicleTakes, par), arg)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "SquadVehicleRepo: table des prises absente",
				"table", "match_vehicle_takes_latest", "err", err)
			return games.ErrCapabilityNotSupported
		}
		return fmt.Errorf("SquadVehicleRepo: query prises: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := scanVehicleLine(rows, out); err != nil {
			return err
		}
	}
	return rows.Err()
}

// scanVehicleLine range une ligne lue : `match` -> passe, `take` -> prise.
func scanVehicleLine(rows *sql.Rows, out *squademprise.VehicleRead) error {
	var (
		kind                         string
		row                          squademprise.VehicleRow
		pass                         squademprise.VehiclePass
		measured, fragsRead          bool
		aboard                       int64
		episodes, proximity, frags   int
		takes                        int
		docSchema, read, unnamed, nc int
		total, unmatched             int
	)
	if err := rows.Scan(&row.MatchID, &kind, &row.Camp, &row.XUID, &row.Family, &takes, &aboard,
		&episodes, &proximity, &frags, &measured, &pass.Reason, &docSchema, &read, &unnamed, &nc,
		&fragsRead, &pass.FragsReason, &total, &unmatched); err != nil {
		return fmt.Errorf("SquadVehicleRepo: scan: %w", err)
	}
	if kind == "match" {
		pass.MatchID, pass.Measured, pass.DocSchema = row.MatchID, measured, docSchema
		pass.EpisodesRead, pass.EpisodesUnnamed, pass.EpisodesNoCamp = read, unnamed, nc
		pass.FragsRead, pass.FragsTotal, pass.FragsUnmatched = fragsRead, total, unmatched
		out.Passes = append(out.Passes, pass)
		return nil
	}
	row.Takes, row.AboardMS, row.Episodes, row.ProximityEpisodes, row.Frags = takes, aboard, episodes, proximity, frags
	out.Rows = append(out.Rows, row)
	return nil
}

// matchsAvecFrags : les matchs dont les frags d'engin se répartissent par camp — passe mesurée, frags
// appariés, au moins un frag.
func matchsAvecFrags(passes []squademprise.VehiclePass) []string {
	var ids []string
	for _, p := range passes {
		if p.Measured && p.FragsRead && p.FragsTotal > 0 {
			ids = append(ids, p.MatchID)
		}
	}
	return ids
}

// lireFrags lit les frags de classe véhicule par camp du tueur (D5) pour les matchs qui en ont.
func (r *SquadVehicleRepo) lireFrags(ctx context.Context, db *sql.DB, out *squademprise.VehicleRead) error {
	ids := matchsAvecFrags(out.Passes)
	if len(ids) == 0 {
		return nil
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "SquadVehicleRepo: aucun classificateur de source de degat : frags par camp non lus",
			"matchs", len(ids))
		return nil
	}
	par, arg := clauseListeMatchs("k.match_id", ids)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(qSquadVehicleFrags, par), arg)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "SquadVehicleRepo: journal de kills absent : frags par camp non lus", "err", err)
			return nil
		}
		return fmt.Errorf("SquadVehicleRepo: query frags: %w", err)
	}
	defer rows.Close()
	type brut struct {
		match string
		team  *int
		tag   uint32
		n     int
	}
	var lus []brut
	keys := map[string]bool{}
	for rows.Next() {
		var b brut
		var team sql.NullInt64
		if err := rows.Scan(&b.match, &team, &b.tag, &b.n); err != nil {
			return fmt.Errorf("SquadVehicleRepo: scan frags: %w", err)
		}
		if team.Valid {
			v := int(team.Int64)
			b.team = &v
		}
		lus = append(lus, b)
		out.EventsRead[b.match] = true
		if key, ok := r.classifier.KillSourceRegistryKey(b.tag); ok {
			keys[key] = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("SquadVehicleRepo: lignes frags: %w", err)
	}
	classes := classesDesCles(ctx, r.pdb, keys)
	if len(keys) > 0 && len(classes) == 0 {
		// Registre des armes illisible : aucune classe ne se lit, donc aucun frag d'engin ne se
		// reconnait. Ce n'est PAS un zero : les matchs restent « non lus ».
		slog.WarnContext(ctx, "SquadVehicleRepo: registre des armes sans classe pour les sources lues : frags par camp non lus",
			"cles", len(keys), "matchs", len(ids))
		out.EventsRead = map[string]bool{}
		return nil
	}
	for _, b := range lus {
		key, ok := r.classifier.KillSourceRegistryKey(b.tag)
		if !ok || !domain.IsEngineFragClass(classes[key]) {
			continue
		}
		out.Frags = append(out.Frags, squademprise.VehicleFragRow{MatchID: b.match, TeamID: b.team, Frags: b.n})
	}
	return nil
}

// classesDesCles rend la classe de chaque clé de registre, par le résolveur de la Répartition des
// frags (une clé hors registre n'a pas de classe).
func classesDesCles(ctx context.Context, pdb *PlayerDB, keys map[string]bool) map[string]string {
	list := make([]string, 0, len(keys))
	for k := range keys {
		list = append(list, k)
	}
	out := make(map[string]string, len(list))
	for k, meta := range resolveWeaponKeyDimensions(ctx, pdb.Metadata, pdbTitleSlug(pdb), list) {
		out[k] = meta.class
	}
	return out
}

// lireCampDuJoueur lit le camp du joueur de la page dans chaque match.
func lireCampDuJoueur(ctx context.Context, db *sql.DB, titleSlug string, ids []string, xuid string, into map[string]int) error {
	if xuid == "" {
		return nil
	}
	par, arg := clauseListeParJointure("p.match_id", ids)
	// Un match de Campagne n'a pas de camp à comparer : même exclusion que les autres lecteurs des
	// matchs d'un joueur (le périmètre de la page n'en contient déjà aucun).
	// Le camp du joueur de la page, par match (la requête est ICI, non en constante : le garde de la
	// Campagne lit la déclaration entière, exclusion comprise).
	q := fmt.Sprintf(`SELECT p.match_id, p.team_id FROM match_participants p
		WHERE p.xuid = ? AND p.team_id IS NOT NULL AND %s`, par) + excludeCampaignByMatchID(titleSlug, "p.match_id")
	rows, err := db.QueryContext(ctx, q, xuid, arg)
	if err != nil {
		return fmt.Errorf("SquadVehicleRepo: query camps: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var team int
		if err := rows.Scan(&id, &team); err != nil {
			return fmt.Errorf("SquadVehicleRepo: scan camps: %w", err)
		}
		into[id] = team
	}
	return rows.Err()
}
