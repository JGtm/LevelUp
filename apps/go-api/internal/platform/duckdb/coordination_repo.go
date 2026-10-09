// Package duckdb — coordination_repo.go : les APPUIS d'un SCOPE DE MATCHS, pour le bloc
// « Coordination » des pages Sessions et Séries temporelles (lot N1, décisions D22).
//
// ─── REQUÊTE VOISINE DE Q21d, AUTRE MAILLE ────────────────────────────────────────────
//
// Q21d (match_view_repo_assist_pairs.go) lit UN match et rend ses paires nommées. Celle-ci
// lit une LISTE BLANCHE de matchs et rend, par match, le couple (assistant, tueur crédité)
// et son compte. Deux différences qui découlent de la maille :
//
//	AUCUN NOM AFFICHÉ         le bloc compte, il ne nomme personne. Le gamertag d'un
//	                          assistant sans xuid (bot, joueur non résolu) sort SEULEMENT
//	                          pour dire qu'il existe : il compte comme les autres.
//	L'ABSENCE D'ASSISTANT SORT Q21d écarte les morts sans assistant parce qu'une PAIRE ne sait
//	                          pas représenter « mesuré, pas d'assistant ». Ici cet état fait
//	                          partie des frags du joueur lus par le film, plancher de la base
//	                          officielle de « on me prépare ».
//	LES BOTS COMPTENT         un tueur sans xuid (bot) sort avec un xuid NULL : son camp se
//	                          déduit de l'assistant, ou de la victime quand l'assistant est
//	                          lui aussi sans xuid (analysis/coordination).
//
// ─── LES DEUX MÊMES PORTES DE MESURE QUE Q21d ─────────────────────────────────────────
//
//	publishable    une ligne nomme DEUX joueurs : lecture ligne à ligne, pas cumul anonyme ;
//	assist_known   sans lui on compterait des « pas d'assistant » jamais observés.
//
// Une mort dont l'assistance n'est pas mesurée n'a AUCUNE ligne ici. Elle reste pourtant
// dans la base de « on me prépare », qui est la feuille de match (QCoordinationFragsOfficiels,
// règle des bases de domain/relation_assists.go).
//
// ─── AUCUN SCAN ───────────────────────────────────────────────────────────────────────
//
// La lecture est bornée par la liste blanche de match_id résolue en amont (le même
// périmètre que le reste de la page). Liste vide = aucune requête.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// coordinationReadTimeout borne la lecture. Même ordre que la lecture tactique : le scope
// d'une page de séries temporelles porte sur des centaines de matchs.
const coordinationReadTimeout = 30 * time.Second

// QCoordinationAppuis : par match, le couple (assistant, tueur crédité) et son compte.
//
// Colonnes : match_id, assist_xuid, assist_gamertag (seulement quand assist_xuid est NULL :
// l'assistant sans xuid est nommé par le film), feed_killer_xuid (NULL = tueur sans xuid,
// un bot), victim_xuid (seulement quand assistant ET tueur sont sans xuid : la victime
// range alors l'appui), nombre. Les NULL deviennent des chaînes vides côté Go ; aucune
// identité n'en est fusionnée, puisque le bloc compte sans nommer.
const QCoordinationAppuis = `
SELECT
    kv.match_id,
    kv.assist_xuid,
    CASE WHEN kv.assist_xuid IS NULL THEN kv.assist_gamertag END                               AS assist_gamertag,
    kv.feed_killer_xuid,
    CASE WHEN kv.assist_xuid IS NULL AND kv.feed_killer_xuid IS NULL THEN kv.victim_xuid END   AS victim_xuid,
    COUNT(*)                                                                                   AS nombre
FROM ` + KillEventsCanonicalTable + ` kv
WHERE kv.match_id IN (%s)
  AND kv.publishable
  AND kv.assist_known
GROUP BY ALL
ORDER BY ALL`

// QCoordinationFragsOfficiels : les frags du joueur par match, tels que la feuille de match
// les compte — la base de « on me prépare ». ?1 = xuid du joueur, puis les match_id.
const QCoordinationFragsOfficiels = `
SELECT match_id, CAST(MAX(kills) AS BIGINT) AS kills
FROM match_participants
WHERE xuid = ?
  AND match_id IN (%s)
  AND kills IS NOT NULL
GROUP BY match_id`

// CoordinationRepo implémente port.CoordinationRepository.
type CoordinationRepo struct {
	pdb *PlayerDB
}

// NewCoordinationRepo crée un CoordinationRepo lié à un PlayerDB.
func NewCoordinationRepo(pdb *PlayerDB) *CoordinationRepo {
	return &CoordinationRepo{pdb: pdb}
}

// LoadAppuis rend les lignes d'appui MESURÉES des matchs demandés.
//
// DÉGRADATION GRACIEUSE, JAMAIS MUETTE : lecteur indisponible ou table absente d'une base
// non migrée rendent zéro ligne et une trace. Le service publie alors un bloc dont le
// versant appui a des dénominateurs vides — la couverture le dit — plutôt que d'échouer
// la page. Un titre sans décodeur de film n'est pas une panne.
func (r *CoordinationRepo) LoadAppuis(
	ctx context.Context, matchIDs []string,
) ([]domain.CoordinationAppuiRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, coordinationReadTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "coordination: appuis indisponibles (shared reader)",
			"err", err, "match_count", len(matchIDs))
		return nil, nil
	}
	defer release()

	q := fmt.Sprintf(QCoordinationAppuis, Placeholders(len(matchIDs)))
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "coordination: appuis indisponibles (requete)",
			"err", err, "match_count", len(matchIDs))
		return nil, nil
	}
	out := make([]domain.CoordinationAppuiRow, 0, len(matchIDs)*8)
	err = scanRows(ctx, rows, "CoordinationRepo.LoadAppuis", func(sc rowScanner) error {
		var (
			a                    domain.CoordinationAppuiRow
			ax, agt, kx, victime sql.NullString
		)
		if err := sc.Scan(&a.MatchID, &ax, &agt, &kx, &victime, &a.Nombre); err != nil {
			return err
		}
		a.AssistXUID, a.AssistGamertag, a.KillerXUID, a.VictimXUID = ax.String, agt.String, kx.String, victime.String
		out = append(out, a)
		return nil
	})
	return out, err
}

// LoadFragsOfficiels rend les frags du joueur `playerXUID` par match, d'après la feuille de
// match. Un match où il n'a pas de compte est absent de la map. Même dégradation que
// LoadAppuis : lecteur indisponible ou requête en échec rendent une map vide et une trace —
// la base retombe alors sur les frags lus par le film.
func (r *CoordinationRepo) LoadFragsOfficiels(
	ctx context.Context, playerXUID string, matchIDs []string,
) (map[string]int, error) {
	out := make(map[string]int, len(matchIDs))
	if len(matchIDs) == 0 || playerXUID == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, coordinationReadTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "coordination: frags officiels indisponibles (shared reader)",
			"err", err, "match_count", len(matchIDs))
		return out, nil
	}
	defer release()

	q := fmt.Sprintf(QCoordinationFragsOfficiels, Placeholders(len(matchIDs)))
	rows, err := db.QueryContext(ctx, q, append([]any{playerXUID}, ToAnySlice(matchIDs)...)...)
	if err != nil {
		slog.WarnContext(ctx, "coordination: frags officiels indisponibles (requete)",
			"err", err, "match_count", len(matchIDs))
		return out, nil
	}
	err = scanRows(ctx, rows, "CoordinationRepo.LoadFragsOfficiels", func(sc rowScanner) error {
		var (
			matchID string
			kills   int
		)
		if err := sc.Scan(&matchID, &kills); err != nil {
			return err
		}
		out[matchID] = kills
		return nil
	})
	return out, err
}
