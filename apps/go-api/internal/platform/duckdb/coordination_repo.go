// Package duckdb — coordination_repo.go : les APPUIS d'un SCOPE DE MATCHS, pour le bloc
// « Coordination » des pages Sessions et Séries temporelles (lot N1, décisions D22).
//
// ─── REQUÊTE VOISINE DE Q21d, AUTRE MAILLE ────────────────────────────────────────────
//
// Q21d (match_view_repo_assist_pairs.go) lit UN match et rend ses paires nommées. Celle-ci
// lit une LISTE BLANCHE de matchs et rend, par match, le couple (assistant, tueur crédité)
// et son compte. Deux différences qui découlent de la maille :
//
//	AUCUN GAMERTAG            le bloc compte, il ne nomme personne — les noms de film
//	                          périmés afficheraient un joueur sous deux orthographes.
//	L'ASSISTANT VIDE SORT     Q21d écarte `assist_xuid IS NULL` parce qu'une PAIRE ne sait
//	                          pas représenter « mesuré, pas d'assistant ». Ici cet état est
//	                          le DÉNOMINATEUR « mes frags mesurés » : sans lui, « on me
//	                          prépare » se diviserait par les seuls frags déjà appuyés et
//	                          vaudrait 100 % pour tout le monde.
//
// ─── LES DEUX MÊMES PORTES DE MESURE QUE Q21d ─────────────────────────────────────────
//
//	publishable    une ligne nomme DEUX joueurs : lecture ligne à ligne, pas cumul anonyme ;
//	assist_known   sans lui on compterait des « pas d'assistant » jamais observés.
//
// Une mort dont l'assistance n'est pas mesurée n'a AUCUNE ligne ici — elle n'entre dans
// aucun dénominateur. C'est la doctrine des trois états de domain/assist_pairs.go, portée
// à l'échelle d'un scope.
//
// ─── AUCUN SCAN ───────────────────────────────────────────────────────────────────────
//
// La lecture est bornée par la liste blanche de match_id résolue en amont (le même
// périmètre que le reste de la page). Liste vide = aucune requête.
package duckdb

import (
	"context"
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
// `assist_xuid` NULL devient la chaîne VIDE côté Go — l'état MESURÉ « personne n'a
// assisté ». Un tueur sans xuid (bot non rattaché) est écarté : il ne peut être ni moi ni
// un coéquipier identifié, et l'agréger fusionnerait tous les bots en un joueur fantôme.
const QCoordinationAppuis = `
SELECT
    match_id,
    COALESCE(assist_xuid, '') AS assist_xuid,
    feed_killer_xuid,
    COUNT(*)                  AS nombre
FROM ` + KillEventsCanonicalTable + `
WHERE match_id IN (%s)
  AND publishable
  AND assist_known
  AND feed_killer_xuid IS NOT NULL
GROUP BY match_id, assist_xuid, feed_killer_xuid
ORDER BY match_id, assist_xuid, feed_killer_xuid`

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
		var a domain.CoordinationAppuiRow
		if err := sc.Scan(&a.MatchID, &a.AssistXUID, &a.KillerXUID, &a.Nombre); err != nil {
			return err
		}
		out = append(out, a)
		return nil
	})
	return out, err
}
