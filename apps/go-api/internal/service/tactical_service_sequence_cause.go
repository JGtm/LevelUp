package service

// tactical_service_sequence_cause.go — POURQUOI UNE SEQUENCE MANQUE (vue D de l'onglet Tactique).
//
// Une cause « donnees » (port.ErrSequence*) n'est rendue qu'apres la verification d'ouvrabilite du
// match : l'appelant l'a joue, lui dire que son rejeu n'est pas cuit ne revele rien. Les refus
// (match non joue, forme, capability) gardent `port.ErrReplayNotAvailable` / ErrCapabilityNotSupported.
//
// « ANALYSE EN COURS » SUIT LA REGLE DE `analysis_pending_parts`, sans seconde copie : c'est le
// resolveur partage (`AnalysisPendingResolver`) qui decide, et la sequence ne regarde que SA cause,
// `replay` (le rejeu 2D manquant alors que le film n'est pas declare perdu), pas `kill_source` ni
// `citations`.

import (
	"context"
	"errors"
	"slices"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// WithAnalysisPending injecte le resolveur de l'indicateur « analyse en cours » (nil : une
// sequence sans artefact est « film non analyse », jamais « en cours »). Chainable.
func (s *TacticalService) WithAnalysisPending(r *AnalysisPendingResolver) *TacticalService {
	s.analysisPending = r
	return s
}

// causeArtefactAbsent dit pourquoi le match n'a pas d'artefact de rejeu : son analyse est en cours
// (cause `replay` du resolveur partage) ou, a defaut, il n'est pas analyse.
func (s *TacticalService) causeArtefactAbsent(ctx context.Context, matchID string) error {
	absent := ReplayLookup{Known: true, Has: func(string) bool { return false }}
	parts := s.analysisPending.Resolve(ctx, []string{matchID}, absent)[matchID]
	if slices.Contains(parts, domain.AnalysisPartReplay) {
		return port.ErrSequenceAnalyseEnCours
	}
	return port.ErrSequenceFilmNonAnalyse
}

// causeDeLaSequence journalise l'indisponibilite d'une sequence et rend l'erreur telle quelle. Un
// refus est journalise en Warn (une demande hors du vocabulaire du joueur), une cause en Info (etat
// nominal d'un parc dont les films ne sont pas tous cuits).
func (s *TacticalService) causeDeLaSequence(ctx context.Context, req domain.TacticalSequenceRequest, cause error) error {
	if errors.Is(cause, port.ErrReplayNotAvailable) {
		s.logger.WarnContext(ctx, "tactique: sequence refusee",
			"player", s.xuid, "match_id", req.MatchID, "map_id", req.MapID)
		return cause
	}
	s.logger.InfoContext(ctx, "tactique: sequence indisponible",
		"player", s.xuid, "match_id", req.MatchID, "cause", cause.Error())
	return cause
}
