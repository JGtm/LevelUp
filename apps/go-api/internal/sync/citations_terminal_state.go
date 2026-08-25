// Package sync — citations_terminal_state.go : état terminal des citations pour
// les matchs dont les events de film n'arriveront JAMAIS.
//
// Contexte (constaté en prod le 2026-08-25) : la règle Phase 4 de citations.go
// laisse candidat tout match à 0 citation dont match_registry.events_loaded est
// faux, au motif que « les events finiront par arriver ». Pour un match ANNULÉ
// par les serveurs, ils n'arrivent jamais : le film est une coquille (chunk non
// vide mais 0 event extractible) et le match était donc re-sélectionné,
// re-calculé et re-rejeté à chaque cycle de sync — des dizaines de passes par
// jour, leurs WARN collatéraux, et une boucle perpétuelle de plus à chaque
// nouveau match annulé.
//
// Arbitrage : au-delà d'un ÂGE seuil, l'absence d'events cesse d'être un retard
// et devient un état TERMINAL — le jeton "_processed" est posé, le match sort du
// pool de selectMatchesForCitations(force=false). Ce n'est pas une impasse : la
// branche force=true (recompute) sélectionne tous les matchs sans consulter
// match_citations, donc un match jetonné dont les events arriveraient plus tard
// reste rattrapable par un recompute.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
)

// citationsTerminalNoEventsAge — âge au-delà duquel un match à 0 citation dont
// les events ne sont pas chargés est déclaré en état terminal.
//
// 7 jours : un film Theater est publié dans les heures qui suivent la fin du
// match, ou jamais. Passé une semaine, il ne reste que deux causes possibles —
// match annulé par les serveurs (film-coquille) ou film indisponible — et
// l'attente est perpétuelle dans les deux cas. Le seuil est volontairement large
// devant le délai réel d'arrivée (heures) : une panne d'API ou un arrêt du
// watcher de plusieurs jours ne doit pas jetonner un match encore récupérable.
const citationsTerminalNoEventsAge = 7 * 24 * time.Hour

// matchAge retourne l'âge du match (maintenant − début du match) lu depuis
// shared.match_registry.
//
// Le début du match passe par le fragment timezone CANONIQUE (règle CLAUDE.md
// n°8, analysis.SQLStartTimeCanonical) — jamais start_time brut : les imports
// OpenSpartan portent un start_time naïf décalé, et un âge calculé dessus
// jetonnerait (ou refuserait de jetonner) au mauvais moment.
//
// Retourne une ERREUR — jamais un âge par défaut — dès que l'âge est
// indéterminable : sharedDB nil, match absent du registre, colonnes illisibles,
// timestamp NULL. Le caller doit alors choisir l'échec sûr.
func matchAge(ctx context.Context, sharedDB *sql.DB, matchID string) (time.Duration, error) {
	if sharedDB == nil {
		return 0, errors.New("matchAge: sharedDB nil")
	}
	q := `SELECT ` + analysis.SQLStartTimeCanonical("") + ` FROM match_registry WHERE match_id = ?`
	var start sql.NullTime
	if err := sharedDB.QueryRowContext(ctx, q, matchID).Scan(&start); err != nil {
		return 0, fmt.Errorf("matchAge %s: %w", matchID, err)
	}
	if !start.Valid {
		return 0, fmt.Errorf("matchAge %s: début de match canonique NULL", matchID)
	}
	return time.Since(start.Time), nil
}

// isCitationsTerminalNoEvents décide si un match à 0 citation dont les events ne
// sont PAS chargés doit malgré tout recevoir le jeton "_processed".
//
// Appelée UNIQUEMENT dans la branche rare (0 delta ET events absents), par
// court-circuit du && côté BackfillMatchCitations : le chemin chaud ne paie
// aucune requête supplémentaire.
//
// Tempérament symétrique de isEventsLoaded, mais d'échec sûr INVERSE :
// isEventsLoaded répond true quand elle ne sait pas (poser le jeton = legacy) ;
// ici, ne pas savoir doit laisser le match candidat — un cycle de plus coûte
// infiniment moins qu'un match jetonné à tort, dont les citations resteraient
// vides jusqu'au prochain recompute force=true. Toute lecture d'âge en échec est
// donc loguée en WARN et retourne false.
func isCitationsTerminalNoEvents(ctx context.Context, sharedDB *sql.DB, matchID string) bool {
	age, err := matchAge(ctx, sharedDB, matchID)
	if err != nil {
		slog.WarnContext(ctx, "citations: âge du match illisible — match laissé candidat (0 delta, events absents)",
			"match_id", matchID, "err", err)
		return false
	}
	if age < citationsTerminalNoEventsAge {
		return false
	}
	slog.InfoContext(ctx, "citations: état terminal — jeton _processed posé",
		"match_id", matchID,
		"age_days", int(age.Hours()/24),
		"seuil_jours", int(citationsTerminalNoEventsAge.Hours()/24),
		"raison", "events jamais arrivés — état terminal")
	return true
}
