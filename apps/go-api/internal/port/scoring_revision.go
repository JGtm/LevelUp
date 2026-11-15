package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// RevisionStep nomme une étape du passage de démarrage par joueur ; chaque étape a sa propre révision
// rangée dans `sync_meta`, réussit ou échoue indépendamment des autres.
type RevisionStep string

const (
	// RevisionStepPlayerDB : l'accès à la base du joueur, avant toute étape (fichier illisible).
	RevisionStepPlayerDB RevisionStep = "base_joueur"
	// RevisionStepKPI : rejeu des notes de performance et du LUSR après un changement de formule.
	RevisionStepKPI RevisionStep = "kpi"
	// RevisionStepCampaigns : recalage du départ des campagnes de l'axe Objectif après un changement
	// de sa mesure.
	RevisionStepCampaigns RevisionStep = "campagnes"
)

// RevisionOutcome est l'issue d'une étape pour un joueur : recalcul effectué (Applied), à jour (ni
// Applied ni Err) ou échec (Err). Une étape en échec garde sa révision rangée et est retentée au
// démarrage suivant ; les étapes réussies du même joueur ne le sont pas.
type RevisionOutcome struct {
	Step    RevisionStep
	Applied bool
	Err     error
}

// ScoringRevisionApplier : ce dont le passage de démarrage des révisions de mesure a besoin — les
// profils d'un titre et l'application des révisions courantes à l'un d'eux (rejeu des KPI après un
// changement de formule, recalage du départ des campagnes de l'axe Objectif). Implémenté au câblage du
// serveur (recalculateur de match sync.MatchRecomputer, service de campagnes).
type ScoringRevisionApplier interface {
	// FollowedPlayers rend les profils du titre.
	FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error)
	// ApplyForPlayer applique chaque étape dont la révision rangée diffère de la révision courante et
	// rend l'issue de chacune, dans l'ordre. Un profil sans base joueur n'a aucune étape : liste vide.
	ApplyForPlayer(ctx context.Context, titleSlug string, player domain.PlayerSummary) []RevisionOutcome
}
