// Package service — match_range_block.go : LE PRODUCTEUR UNIQUE du bloc « portée des
// engagements » par (match, joueur) (plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md,
// lots N2 puis U).
//
// # POURQUOI UN PRODUCTEUR, ET PAS UNE MÉTHODE PAR PAGE
//
// Trois scopes le consomment aujourd'hui : les matchs de la session affichée, ceux de la
// session comparée, la PÉRIODE DE RÉFÉRENCE de la colonne de session (lot U), plus la
// fenêtre de la page Séries temporelles. Quatre montages, une seule définition de « médiane
// du lobby » et une seule règle de dégradation — un second constructeur aurait donné deux
// couvertures pour la même mesure au premier correctif (même leçon que
// `coordination_block.go`).
//
// # BEST-EFFORT, TOUJOURS
//
// Repo non câblé (titre sans décodeur de film), scope vide, lecture en échec, aucun match
// décodé : le bloc est nil et la page se rend. L'erreur est JOURNALISÉE avant la
// dégradation ; une capacité absente se journalise en Debug, ce n'est pas une panne.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// matchRangeQuery est la demande d'UN scope. Struct plutôt que cinq arguments adjacents,
// et surtout parce que `Publish` et `PublishOrder` se lisent ensemble ou pas du tout.
type matchRangeQuery struct {
	Repo      port.MatchRangeRepository
	TitleSlug string
	// Matches est le scope, DANS L'ORDRE DE SORTIE (le plus ancien d'abord).
	Matches []analysis.MatchRangeMatch
	// Publish nomme les joueurs publiés (xuid -> gamertag) ; PublishOrder fixe leur ordre.
	Publish      map[string]string
	PublishOrder []string
	// Scope nomme le montage dans les journaux (« session », « reference », « timeseries »).
	Scope string
}

// buildMatchRangeBlock lit les frags mesurés du scope et en rend le bloc publié. nil dans
// tous les cas où il n'y a rien à dire — jamais un bloc vide, qui se lirait comme une
// mesure à zéro.
func buildMatchRangeBlock(ctx context.Context, q matchRangeQuery) *domain.MatchRangeBlock {
	if q.Repo == nil || len(q.Matches) == 0 || len(q.Publish) == 0 {
		return nil
	}
	ids := make([]string, 0, len(q.Matches))
	for _, m := range q.Matches {
		ids = append(ids, m.MatchID)
	}
	read, err := q.Repo.LoadMatchRangeKills(ctx, q.TitleSlug, port.WeaponRangeFilters{
		MatchIDs:   ids,
		AllPlayers: true,
	})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			// Titre sans positions par kill : une absence de capacité, pas une panne.
			slog.DebugContext(ctx, "range block unsupported for title",
				"slug", q.TitleSlug, "scope", q.Scope, "match_count", len(ids))
			return nil
		}
		slog.ErrorContext(ctx, "range block load failed",
			"err", err, "scope", q.Scope, "match_count", len(ids))
		return nil
	}
	profiles := analysis.MatchRangeProfiles(analysis.MatchRangeInput{
		Kills:        read.Kills,
		Matches:      q.Matches,
		Publish:      q.Publish,
		PublishOrder: q.PublishOrder,
	})
	if len(profiles) == 0 {
		slog.InfoContext(ctx, "range block empty, section omitted",
			"scope", q.Scope, "match_count", len(ids),
			"cause", "aucun match du scope ne porte de frag mesure")
		return nil
	}
	return &domain.MatchRangeBlock{
		Profiles:      profiles,
		KillsMeasured: len(read.Kills),
		KillsTotal:    read.KillsTotal,
	}
}
