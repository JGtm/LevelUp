package duckdb

// replay_oracle_repo.go — LES VERITES OFFICIELLES D'UN MATCH QUE LE REJEU NE LIT PAS.
//
// L'oracle du banc de verite (`internal/replayverite`, decision D-2 de
// .ai/V7.5/film_re/BANC_DE_VERITE_CONCEPTION_2026-09-30.md). Il est lu a cote des faits de match
// (`replay_facts_repo.go`) mais ne les rejoint jamais : ce qui passe ici n'est pas une entree de
// cuisson, et ne doit pas en devenir une.
//
// LECTURE SEULE ET COURTE, sur un handle deja ouvert par l'appelant (`OpenReadForQuery` cote
// `levelup replay-facts-export --oracle`) : deux `SELECT` bornes au match, aucune ecriture.
// Les stats d'objectif se lisent sur la vue `match_objective_stats_latest` UNIQUEMENT (table
// append-only, regle ART n 2 : une lecture brute servirait des lignes perimees).

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

var _ port.ReplayOracleRepo = (*ReplayOracleRepo)(nil)

// ReplayOracleRepo lit l'oracle d'un match (participants + stats d'objectif).
type ReplayOracleRepo struct {
	shared *sql.DB
}

// NewReplayOracleRepo cree le repo sur une connexion shared DEJA ouverte ; il n'ouvre ni ne ferme
// rien.
func NewReplayOracleRepo(shared *sql.DB) *ReplayOracleRepo {
	return &ReplayOracleRepo{shared: shared}
}

// colonnesHorsStat : les colonnes de `match_objective_stats_latest` qui IDENTIFIENT une ligne et
// ne sont pas des stats — tout le reste de la vue (un `SELECT *` sur une table large qui s'etend
// par ALTER) est une stat d'objectif.
var colonnesHorsStat = map[string]bool{"id": true, "match_id": true, colonneXUID: true, "written_at": true}

// colonneXUID : la colonne qui porte le joueur d une ligne de stats d objectif.
const colonneXUID = "xuid"

// OracleForMatch rend l'oracle du match. Un match sans participant rend un oracle VIDE sans
// erreur : l'appelant decide s'il l'ecrit.
func (r *ReplayOracleRepo) OracleForMatch(ctx context.Context, matchID string) (domain.MatchOracle, error) {
	out := domain.MatchOracle{MatchID: matchID}
	if r == nil || r.shared == nil {
		return out, nil
	}
	players, err := r.participants(ctx, matchID)
	if err != nil {
		return out, err
	}
	stats, err := r.objectiveStats(ctx, matchID)
	if err != nil {
		return out, err
	}
	for i := range players {
		if s, ok := stats[players[i].XUID]; ok {
			players[i].Objectives = s
		}
	}
	out.Players = players
	return out, nil
}

// participants lit les colonnes officielles de la ligne de match qui ne sont PAS des entrees de la
// cuisson. Les NULL restent nil (cf. domain.MatchPlayerOracle).
func (r *ReplayOracleRepo) participants(ctx context.Context, matchID string) ([]domain.MatchPlayerOracle, error) {
	rows, err := r.shared.QueryContext(ctx,
		`SELECT xuid, personal_score, score, shots_fired, shots_hit, headshot_kills, melee_kills,
		        grenade_kills, power_weapon_kills, time_played_seconds,
		        present_at_beginning, present_at_completion
		 FROM match_participants WHERE match_id = ? ORDER BY xuid`, matchID)
	if err != nil {
		return nil, fmt.Errorf("oracle de rejeu : lecture match_participants : %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.MatchPlayerOracle
	for rows.Next() {
		var xuid string
		var ent [9]sql.NullInt64
		var debut, fin sql.NullBool
		if err := rows.Scan(&xuid, &ent[0], &ent[1], &ent[2], &ent[3], &ent[4], &ent[5], &ent[6],
			&ent[7], &ent[8], &debut, &fin); err != nil {
			return nil, fmt.Errorf("oracle de rejeu : lecture match_participants (scan) : %w", err)
		}
		if xuid == "" {
			continue
		}
		out = append(out, domain.MatchPlayerOracle{
			XUID: xuid, PersonalScore: entier(ent[0]), Score: entier(ent[1]),
			ShotsFired: entier(ent[2]), ShotsHit: entier(ent[3]), HeadshotKills: entier(ent[4]),
			MeleeKills: entier(ent[5]), GrenadeKills: entier(ent[6]), PowerWeaponKills: entier(ent[7]),
			TimePlayedSeconds: entier(ent[8]), PresentAtBeginning: booleen(debut),
			PresentAtCompletion: booleen(fin),
		})
	}
	return out, rows.Err()
}

// objectiveStats lit, par xuid, les colonnes NON NULLES de la vue `_latest`, par leur nom.
func (r *ReplayOracleRepo) objectiveStats(ctx context.Context, matchID string) (map[string]map[string]float64, error) {
	rows, err := r.shared.QueryContext(ctx,
		`SELECT * FROM match_objective_stats_latest WHERE match_id = ?`, matchID)
	if err != nil {
		return nil, fmt.Errorf("oracle de rejeu : lecture match_objective_stats_latest : %w", err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("oracle de rejeu : colonnes des stats d'objectif : %w", err)
	}
	out := map[string]map[string]float64{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("oracle de rejeu : lecture des stats d'objectif (scan) : %w", err)
		}
		xuid, stats := statsDeLaLigne(cols, vals)
		if xuid != "" && len(stats) > 0 {
			out[xuid] = stats
		}
	}
	return out, rows.Err()
}

// statsDeLaLigne separe le xuid des stats numeriques non nulles d'une ligne.
func statsDeLaLigne(cols []string, vals []any) (string, map[string]float64) {
	xuid := ""
	stats := map[string]float64{}
	for i, c := range cols {
		if c == colonneXUID {
			if s, ok := vals[i].(string); ok {
				xuid = s
			}
			continue
		}
		if colonnesHorsStat[c] {
			continue
		}
		if v, ok := nombre(vals[i]); ok {
			stats[c] = v
		}
	}
	return xuid, stats
}

// nombre convertit une valeur scannee en float64 ; nil (NULL) et les non-nombres rendent faux.
func nombre(v any) (float64, bool) {
	switch x := v.(type) {
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	default:
		return 0, false
	}
}

func entier(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func booleen(b sql.NullBool) *bool {
	if !b.Valid {
		return nil
	}
	v := b.Bool
	return &v
}
