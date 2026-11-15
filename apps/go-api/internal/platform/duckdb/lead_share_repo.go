// Package duckdb — lead_share_repo.go : la MAINMISE d'un match (temps passé en tête du score), lue
// de la passe retenue de chaque match (ADR 0036).
//
// UN CHARGEMENT PAR REQUÊTE, DEUX LECTURES BORNÉES PAR LA LISTE DES MATCHS :
//
//	passes   `match_lead_share_latest` (ADR 0026 : la vue seulement), `match_id` lié en UNE constante
//	         `VARCHAR[]` que DuckDB pousse sous la fenêtre de la vue. La vue rend la DERNIÈRE PASSE
//	         ENTIÈRE de chaque match ; seules les passes mesurées sont lues.
//	camp     le camp du joueur de la page dans chaque match (`match_participants`, table ordinaire,
//	         semi-jointure) — le même lecteur que la ressource véhicules.
//
// Jamais `v_gamertag_lookup` : aucun nom n'est lu ici. Table absente :
// games.ErrCapabilityNotSupported.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/games"
)

// LeadShareRepo lit la Mainmise sur le SharedReader du joueur.
type LeadShareRepo struct {
	pdb *PlayerDB
}

// NewLeadShareRepo construit le repo.
func NewLeadShareRepo(pdb *PlayerDB) *LeadShareRepo {
	return &LeadShareRepo{pdb: pdb}
}

const leadShareQueryTimeout = 15 * time.Second

// leadShareKindCamp : la valeur de `row_kind` d'une ligne de camp (persist.LeadShareRowKindCamp).
const leadShareKindCamp = "camp"

// qLeadSharePasses : le prédicat de liste est posé par LoadLeadShare (%s). La ligne `match` porte la
// durée, les retournements et les camps muets ; chaque ligne `camp` le temps en tête d'un camp.
const qLeadSharePasses = `
SELECT l.match_id, l.row_kind, l.camp, l.lead_ms, l.measured_ms, l.lead_changes, l.silent_camps
FROM match_lead_share_latest l
WHERE %s AND l.measured
ORDER BY l.match_id, l.row_kind, l.camp`

// LoadLeadShare rend, pour chaque match MESURÉ de `matchIDs`, la Mainmise de l'équipe de
// `playerXUID`. Liste vide ou joueur inconnu : rien à lire. Un match dont le camp du joueur n'est
// pas connu n'est pas rendu.
func (r *LeadShareRepo) LoadLeadShare(
	ctx context.Context, matchIDs []string, playerXUID string,
) (map[string]leadshare.PlayerShare, error) {
	out := map[string]leadshare.PlayerShare{}
	if len(matchIDs) == 0 || playerXUID == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, leadShareQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("LeadShareRepo: shared reader: %w", err)
	}
	defer release()

	passes, err := r.lirePasses(ctx, db, matchIDs)
	if err != nil {
		return out, err
	}
	camps := map[string]int{}
	if err := lireCampDuJoueur(ctx, db, pdbTitleSlug(r.pdb), matchIDs, playerXUID, camps); err != nil {
		return out, err
	}
	for id, m := range passes {
		camp, ok := camps[id]
		if !ok {
			continue
		}
		if s, lisible := leadshare.ShareOfCamp(*m, camp); lisible {
			out[id] = s
		}
	}
	return out, nil
}

// lirePasses range les lignes de la vue par match.
func (r *LeadShareRepo) lirePasses(ctx context.Context, db *sql.DB, ids []string) (map[string]*leadshare.MatchRead, error) {
	par, arg := clauseListeMatchs("l.match_id", ids)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(qLeadSharePasses, par), arg)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "LeadShareRepo: table de la mainmise absente",
				"table", "match_lead_share_latest", "err", err)
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("LeadShareRepo: query passes: %w", err)
	}
	defer rows.Close()
	passes := map[string]*leadshare.MatchRead{}
	for rows.Next() {
		var id, kind string
		var camp, changes, silent int
		var lead, measured int64
		if err := rows.Scan(&id, &kind, &camp, &lead, &measured, &changes, &silent); err != nil {
			return nil, fmt.Errorf("LeadShareRepo: scan passes: %w", err)
		}
		m := passes[id]
		if m == nil {
			m = &leadshare.MatchRead{LeadMS: map[int]int64{}}
			passes[id] = m
		}
		m.MeasuredMS, m.LeadChanges, m.SilentCamps = measured, changes, silent
		if kind == leadShareKindCamp {
			m.LeadMS[camp] = lead
		}
	}
	return passes, rows.Err()
}
