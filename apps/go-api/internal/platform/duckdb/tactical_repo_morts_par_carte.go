package duckdb

// tactical_repo_morts_par_carte.go — MES MORTS, GROUPEES PAR CARTE.
//
// # POURQUOI UNE LECTURE A PART, ET NON `KillPositions` SANS CARTE
//
// `KillPositions` REFUSE une carte vide, et ce refus reste entier : sa sortie est une
// GRILLE, qui n'a de sens que carte par carte (cf. sa doc). Cette lecture-ci rend des
// points DEJA GROUPES PAR CARTE — chaque groupe devient sa propre grille. Elle ne leve donc
// aucune garde : elle repond a une autre question.
//
// # ELLE EST BEAUCOUP PLUS ETROITE QUE `KillPositions`
//
// Trois restrictions cumulees, et c'est ce qui la rend tenable sur l'ecran d'ENTREE de
// l'onglet :
//
//	une seule face      la VICTIME, et c'est MOI (`e.victim_xuid = ?`) — la vignette
//	                    montre « ou je meurs », pas les positions de tous les joueurs ;
//	une seule colonne   la position de la victime, pas les quatre coordonnees ;
//	le perimetre        la liste blanche de match_id de la barre L2, comme partout ailleurs.
//
// # MEMES GARDES D'ATTRIBUTION QUE `KillPositions`
//
// `e.publishable` (une passe non publiable est juste en agregat et fausse ligne a ligne) et
// `HAVING count(*) = 1` (un double kill au meme (tueur, instant) ne permet pas d'attribuer
// la position a une victime). Une vignette est une lecture PAR LIGNE comme les autres.
//
// LA VICTIME SE FILTRE APRES LE REGROUPEMENT, ET C'EST LE POINT DELICAT : la poser dans le
// WHERE interne ferait tomber la garde du double kill. Le groupe ne verrait plus qu'UNE
// ligne — la mienne — et `count(*) = 1` conclurait a une attribution certaine sur un
// instant qui en porte deux. Le cas est couvert par le test m6 du corpus.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// QTacticalMortsParCarte : %s = la table des positions au coup fatal
// (`positionsAtKill`, source unique du nom). Le token Campagne TERMINE le WHERE de la
// sous-requete, comme partout cote lecture.
const QTacticalMortsParCarte = `
SELECT map_id, match_id, victim_x, victim_y FROM (
  SELECT mr.map_id                     AS map_id,
         kp.match_id                   AS match_id,
         COALESCE(min(e.victim_xuid), '') AS victim_xuid,
         min(kp.victim_x)              AS victim_x,
         min(kp.victim_y)              AS victim_y
  FROM %s kp
  JOIN match_kill_events_latest e
      ON e.match_id = kp.match_id
     AND e.feed_killer_xuid = kp.killer_xuid
     AND e.time_ms = kp.time_ms
  JOIN match_registry mr ON mr.match_id = kp.match_id
  JOIN match_participants mp ON mp.match_id = mr.match_id AND mp.xuid = ?
  WHERE e.publishable
    AND kp.victim_x IS NOT NULL AND kp.victim_y IS NOT NULL
    AND mr.map_id IS NOT NULL AND mr.map_id <> ''` + clausePvEExclu + campaignExclusionToken

// MortsParCarte rend MES morts localisees du perimetre, groupees par carte.
//
// UN XUID VIDE EST UN REFUS, jamais un balayage : la lecture est bornee par le joueur,
// comme les trois autres du meme lecteur.
func (r *TacticalRepo) MortsParCarte(ctx context.Context, q domain.TacticalQuery) (map[string][]domain.PositionSample, error) {
	if q.PlayerXUID == "" {
		return nil, fmt.Errorf("TacticalRepo.MortsParCarte: xuid vide")
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "TacticalRepo.MortsParCarte: shared reader", "err", err)
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	perim, perimArgs := clausePerimetre(q)
	// L'ORDRE DES ARGUMENTS SUIT L'ORDRE TEXTUEL DES `?` : le joueur (jointure du
	// participant), le perimetre, puis LA VICTIME — filtree APRES le regroupement.
	args := append([]any{q.PlayerXUID}, perimArgs...)
	args = append(args, q.PlayerXUID)
	query := resolveCampaignExclusion(
		fmt.Sprintf(QTacticalMortsParCarte, positionsAtKill), r.pdb.TitleSlug, "mr") + perim +
		"\n  GROUP BY mr.map_id, kp.match_id, kp.killer_xuid, kp.time_ms\n" +
		"  HAVING count(*) = 1\n) WHERE victim_xuid = ?"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, r.degrader(ctx, "MortsParCarte", err)
	}
	out := make(map[string][]domain.PositionSample)
	err = scanRows(ctx, rows, "TacticalRepo.MortsParCarte", func(sc rowScanner) error {
		var mapID string
		var p domain.PositionSample
		if err := sc.Scan(&mapID, &p.MatchID, &p.X, &p.Y); err != nil {
			return err
		}
		out[mapID] = append(out[mapID], p)
		return nil
	})
	return out, err
}
