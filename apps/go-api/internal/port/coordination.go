// Package port — coordination.go : le contrat du lecteur d'APPUIS d'un scope, pour le bloc
// « Coordination » des pages Sessions et Séries temporelles.
//
// Fichier séparé de repository_data.go, qui dépasse déjà le seuil des 500 lignes : ce
// paquet range ses contrats par sujet (tactical.go, medals.go, ...), et grossir un
// god-file au motif que le voisin y vit accroîtrait une dette gelée par la baseline
// (CLAUDE.md n 5).
//
// LE JOURNAL DES MORTS N'A PAS DE PORT ICI, ET C'EST VOLONTAIRE : le bloc lit les morts
// par [TacticalRepository.KillEvents], qui rend déjà l'univers (matchs retenus, drapeau
// « mesuré », table des équipes) ET les événements pour une liste blanche de match_id. Un
// second lecteur d'événements aurait donné deux définitions de « match mesuré » libres de
// diverger — le défaut exact que la correction R2 du 2026-09-06 a supprimé sur la page
// Escouade.
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// CoordinationRepository lit les APPUIS mesurés d'une liste blanche de matchs.
//
// Source : `shared.match_kill_events_latest` — vue `_latest` UNIQUEMENT (règle ART n 2,
// jamais la table brute), mêmes portes de mesure que Q21d (`publishable AND assist_known`).
// Implémenté par platform/duckdb.CoordinationRepo.
//
// DÉGRADATION GRACIEUSE : zéro ligne est l'état NOMINAL d'un titre sans décodeur de film
// ou d'un scope dont aucun match n'est décodé — pas une panne. Le service publie alors un
// versant appui à dénominateurs vides, et la couverture du bloc le dit.
type CoordinationRepository interface {
	// LoadAppuis rend, par match et par couple (assistant, tueur crédité), le nombre de
	// morts mesurées correspondantes.
	//
	// UN ASSISTANT VIDE EST UN ÉTAT MESURÉ (« personne n'a assisté »), pas une absence de
	// ligne : c'est lui qui porte le dénominateur « mes frags mesurés ». Liste de matchs
	// vide ⇒ aucune requête, aucune ligne.
	LoadAppuis(ctx context.Context, matchIDs []string) ([]domain.CoordinationAppuiRow, error)
}
