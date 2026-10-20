// Package sync — convergence_report.go : exposition lecture seule du backlog
// de convergence pour le dashboard monitoring admin.
//
// Réutilise les sélecteurs privés de convergence.go (FindMatchesMissingData
// bitmask-aware + diff enrichment + PSA jamais tentés) SANS rien exécuter :
// c'est un comptage, le travail lui-même reste porté par le cycle de sync.
package sync

import (
	"context"
	"database/sql"
)

// ConvergenceHorizon réexporte la borne de sélection par cycle : les compteurs
// PSA/events de ConvergenceBacklog sont PLAFONNÉS à cette valeur
// (count == ConvergenceHorizon ⇒ afficher « N+ » côté UI).
const ConvergenceHorizon = convergenceHorizon

// ConvergenceBacklogCounts agrège le backlog d'enrichissement d'un joueur.
//
// L'AXE « WEAPONS » A ÉTÉ RETIRÉ le 2026-09-01 (lot arme-source-unique, étape A3) :
// il comptait le retard de l'étape 1.55, supprimée avec son producteur. Le retard de
// film est désormais celui de l'étape 1.57 et il se lit en expvar
// (`killsource_postsync_backlog_restant`) — jauge GLOBALE, pas par joueur, parce que le
// décodage de film ne se sélectionne pas par joueur mais par match.
type ConvergenceBacklogCounts struct {
	// MissingEnrichment : matchs présents en shared.match_participants sans
	// row player_match_enrichment (non plafonné — diff complet).
	MissingEnrichment int
	// MissingPSA / MissingEvents : plafonnés à ConvergenceHorizon
	// (sélections bornées ORDER BY récence).
	MissingPSA    int
	MissingEvents int
}

// Total retourne la somme des backlogs (indicateur « ce joueur a du travail
// de convergence en attente »).
func (c ConvergenceBacklogCounts) Total() int {
	return c.MissingEnrichment + c.MissingPSA + c.MissingEvents
}

// ConvergenceBacklog compte le backlog de convergence d'un joueur (lectures
// seules, best-effort : les erreurs SQL sont loggées par les sélecteurs et
// produisent des compteurs à 0 — même sémantique que hasConvergenceBacklog).
// Nil-safe : DBs nil ou xuid vide → compteurs à zéro.
func ConvergenceBacklog(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) ConvergenceBacklogCounts {
	if playerDB == nil || sharedDB == nil || xuid == "" {
		return ConvergenceBacklogCounts{}
	}
	return ConvergenceBacklogCounts{
		MissingEnrichment: countSharedMatchesMissingEnrichment(ctx, playerDB, sharedDB, xuid),
		MissingPSA:        len(selectMatchesMissingPSA(ctx, playerDB)),
		MissingEvents:     len(selectMatchesMissingEvents(ctx, playerDB, sharedDB, xuid)),
	}
}
