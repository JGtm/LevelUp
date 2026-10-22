package duckdb

// citations_cumul.go — LE CUMUL D'UNE CITATION ARRÊTÉ À UN MATCH.
//
// Une tuile de match (Home) et l'onglet citations de la vue match affichent l'anneau de
// progression d'une citation À CE MATCH : `analysis.ComputeTierProgression(cumulative, delta,
// tiers)` attend le total APRÈS ce match (`cumulative - delta` = le total d'avant). Le total
// d'aujourd'hui n'est pas ce cumul : un match ancien y verrait la progression de maintenant, et
// toute citation maîtrisée depuis passerait pour « maîtrisée pendant ce match » sur chaque match
// récent dont le gain dépasse l'écart au dernier palier.
//
// POURQUOI DEUX BASES ET UN CUMUL EN GO. `match_citations` (base joueur) ne porte pas l'heure de
// début du match : elle vit dans `match_registry` (base partagée). Les deux connexions ne se
// joignent pas (ADR 0016) : la série de la citation est lue sur la base joueur, l'heure de ses
// matchs sur le lecteur partagé, et `analysis.CitationCumulAfterMatch` somme dans l'ordre.
//
// ADR 0036 (lectures par périmètre). I1 : aucun nom de joueur lu. I2 : la vue
// `match_citations_latest` se lit SANS borne de matchs, et c'est voulu — un cumul est un préfixe
// de toute la série de la citation, aucune liste de matchs ne le borne ; la vue est celle d'UN
// joueur (quelques milliers de lignes), pas une table partagée de millions de lignes, et la
// requête remplacée la balayait déjà en entier (SUM(value) GROUP BY citation). Les heures sont
// lues en une requête par semi-jointure sur `match_registry` (table), pas sous une fenêtre.
// I4 : une lecture par requête HTTP (le lot de matchs de la Home passe en un seul appel).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
)

// citationRowScan porte une ligne (match, citation, gain) lue sur la base joueur.
type citationRowScan struct {
	matchID string
	norm    string
	delta   int
}

// loadCitationCumulAfterMatch rend, pour chaque (match, citation) de l'historique du joueur sur
// les citations `norms`, le total de la citation arrêté à ce match inclus (ordre chronologique,
// départage par match_id). Toute erreur de lecture est rendue : un cumul faux est pire qu'une
// citation absente, l'appelant décide de la dégradation après l'avoir journalisée.
func loadCitationCumulAfterMatch(ctx context.Context, pdb *PlayerDB, norms []string) (map[analysis.CitationMatchKey]int, error) {
	if len(norms) == 0 {
		return map[analysis.CitationMatchKey]int{}, nil
	}
	history, err := loadCitationHistory(ctx, pdb, norms)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return map[analysis.CitationMatchKey]int{}, nil
	}
	ids := make([]string, 0, len(history))
	seen := make(map[string]struct{}, len(history))
	for _, h := range history {
		if _, ok := seen[h.matchID]; !ok {
			seen[h.matchID] = struct{}{}
			ids = append(ids, h.matchID)
		}
	}
	starts, err := loadCitationHistoryStartTimes(ctx, pdb, ids)
	if err != nil {
		return nil, err
	}
	entries := make([]analysis.CitationHistoryEntry, len(history))
	for i, h := range history {
		entries[i] = analysis.CitationHistoryEntry{
			MatchID: h.matchID, Norm: h.norm, Value: h.delta, StartTime: starts[h.matchID],
		}
	}
	return analysis.CitationCumulAfterMatch(entries), nil
}

// loadCitationHistory lit sur la base joueur la série (match, citation, valeur) des citations
// `norms` (Q41b).
func loadCitationHistory(ctx context.Context, pdb *PlayerDB, norms []string) ([]citationRowScan, error) {
	placeholders := make([]string, len(norms))
	args := make([]interface{}, len(norms))
	for i, n := range norms {
		placeholders[i] = "?"
		args[i] = n
	}
	query := fmt.Sprintf(Q41bCitationHistoryTemplate, strings.Join(placeholders, ", "))
	rows, err := pdb.ReadDB().QueryRecovered(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("historique des citations: %w", err)
	}
	defer rows.Close()
	var out []citationRowScan
	for rows.Next() {
		var r citationRowScan
		if err := rows.Scan(&r.matchID, &r.norm, &r.delta); err != nil {
			return nil, fmt.Errorf("historique des citations, scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("historique des citations: %w", err)
	}
	return out, nil
}

// loadCitationHistoryStartTimes lit sur le lecteur partagé l'heure de début canonique des
// matchs `ids` (Q41c). Un match absent du registre n'a pas d'entrée dans la carte rendue : il
// sera daté « inconnu » par le cumul.
func loadCitationHistoryStartTimes(ctx context.Context, pdb *PlayerDB, ids []string) (map[string]time.Time, error) {
	if pdb.SharedReader == nil && pdb.Shared == nil {
		return nil, fmt.Errorf("heures des matchs de l'historique: lecteur partagé indisponible")
	}
	sharedDB, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("heures des matchs de l'historique: %w", err)
	}
	defer release()

	pred, arg := clauseListeParJointure("match_id", ids)
	rows, err := sharedDB.QueryContext(ctx, Q41cCitationHistoryStartTimes+pred, arg)
	if err != nil {
		return nil, fmt.Errorf("heures des matchs de l'historique: %w", err)
	}
	defer rows.Close()
	out := make(map[string]time.Time, len(ids))
	for rows.Next() {
		var (
			matchID string
			start   sql.NullTime
		)
		if err := rows.Scan(&matchID, &start); err != nil {
			return nil, fmt.Errorf("heures des matchs de l'historique, scan: %w", err)
		}
		// Début NULL : traité comme absent (daté « inconnu » par le cumul), la lecture continue.
		if start.Valid {
			out[matchID] = start.Time
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("heures des matchs de l'historique: %w", err)
	}
	return out, nil
}
