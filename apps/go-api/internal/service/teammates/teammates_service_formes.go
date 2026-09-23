// Package teammates — teammates_service_formes.go : LE BLOC « FORMES RETENUES »
// de l'onglet Synergies (artefact 2ec1b8eb, lot D2 du 2026-09-13).
//
// MÊME SCOPE ET MÊMES AMIS QUE LE BLOC D'ÉQUIPEMENT (teammates_service_usage.go) :
// `filteredMatches` — les matchs du joueur principal après période, cascade et
// sessions, la même population que Options/MatchHistory/TotalMatches — et les
// coéquipiers SÉLECTIONNÉS comme escouade. Deux blocs de la même page qui
// répondraient sur deux scopes différents seraient deux vérités.
//
// L'IDENTITÉ D'AFFICHAGE D'UN MATCH (heure, mode, carte) vient de l'historique
// DÉJÀ construit par la page : la ré-interroger aurait fait deux libellés
// possibles du même match. Un match absent de l'historique garde son heure (le
// canonical la porte) et perd ses libellés — l'écran montre alors l'heure seule,
// jamais un « inconnu ».
package teammates

import (
	"context"

	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// WithSquadFormes injecte les deux sources du bloc et la racine du dépôt (le
// catalogue d'armes du titre s'y lit à la requête). Câblé gated par les mêmes
// capabilities que les blocs qu'il prolonge (film.usage_summary pour l'usage,
// les stats d'objectif pour les colonnes) ; nil ⇒ bloc servi avec Available=false
// et raison machine, ou sans ses cartes d'objectif.
func (s *TeammatesService) WithSquadFormes(
	usage port.SquadFormesUsageRepository, objectives port.SquadFormesObjectiveRepository, repoRoot string,
) *TeammatesService {
	s.formesUsageRepo = usage
	s.formesObjectiveRepo = objectives
	s.repoRoot = repoRoot
	return s
}

// loadSquadFormes publie le bloc sur le scope filtré de la page.
func (s *TeammatesService) loadSquadFormes(
	ctx context.Context, playerXUID string, filteredMatches []legacymatch.SynthesisMatchRow,
	history []domain.SquadMatchHistoryRow, req domain.TeammatesQueryRequest,
) *domain.SquadFormesBlock {
	defer timing.FromContext(ctx).Section("squad_formes")()
	return squadagg.BuildSquadFormesBlock(ctx, squadagg.SquadFormesQuery{
		Repo:              s.formesUsageRepo,
		Objectives:        s.formesObjectiveRepo,
		PlayerXUID:        playerXUID,
		MainGamertag:      s.gamertag,
		Metas:             formesMatchMetas(filteredMatches, history),
		SelectedGamertags: req.SelectedGamertags,
		RepoRoot:          s.repoRoot,
		TitleSlug:         s.titleSlug,
		Locale:            req.Locale,
	})
}

// formesMatchMetas — le scope dans l'ordre de la page, chaque match nommé par
// l'historique quand celui-ci le porte.
func formesMatchMetas(
	rows []legacymatch.SynthesisMatchRow, history []domain.SquadMatchHistoryRow,
) []squadformes.MatchMeta {
	byID := make(map[string]*domain.SquadMatchHistoryRow, len(history))
	for i := range history {
		byID[history[i].MatchID] = &history[i]
	}
	out := make([]squadformes.MatchMeta, 0, len(rows))
	for _, r := range rows {
		meta := squadformes.MatchMeta{
			MatchID:   r.MatchID,
			StartTime: r.StartTime.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if h := byID[r.MatchID]; h != nil {
			meta.ModeLabel, meta.MapLabel = h.ModeUI, h.MapUI
			if h.StartTime != "" {
				meta.StartTime = h.StartTime
			}
		}
		out = append(out, meta)
	}
	return out
}
