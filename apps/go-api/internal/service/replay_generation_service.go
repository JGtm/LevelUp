package service

// replay_generation_service.go — la génération à la demande du rejeu d'UN match qui n'en a pas (placement local).
// Le service CONTRÔLE (capacité du titre, match connu, mode géré, film absent, jeton, plafond) et rend un état en
// codes ; il ne télécharge ni ne décode rien : la file de `replayregen` exécute. Une instance par requête, liée au
// joueur de la page (celui que la garde de propriété a admis) : la demande porte SON xuid, jamais un autre.
//
// Chaque refus et chaque échec se journalise avant d'être rendu (slog, clés titleSlug / match_id / player).

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// ReplayGenerationPlayer : le joueur de la page auquel la demande est liée.
type ReplayGenerationPlayer struct {
	Slug     string
	XUID     string
	Gamertag string
}

// ReplayGenerationService implémente port.ReplayGeneration.
type ReplayGenerationService struct {
	caps   games.CapabilityMap
	queue  port.ReplayGenerationQueue
	probe  port.ReplayMatchProbe
	token  port.ReplayTokenPresence
	player ReplayGenerationPlayer
}

var _ port.ReplayGeneration = (*ReplayGenerationService)(nil)

// NewReplayGenerationService assemble le service. caps est la CapabilityMap du titre (passée, jamais résolue ici) ;
// token dit si le joueur a un jeton au magasin (nil = aucun).
func NewReplayGenerationService(caps games.CapabilityMap, queue port.ReplayGenerationQueue,
	probe port.ReplayMatchProbe, token port.ReplayTokenPresence, player ReplayGenerationPlayer) *ReplayGenerationService {
	return &ReplayGenerationService{caps: caps, queue: queue, probe: probe, token: token, player: player}
}

// GenerationStatus rend l'état du match, sans rien lancer.
//
// Ordre d'évaluation : capacité du titre (sinon games.ErrCapabilityNotSupported) ; demande en file ou en cours ;
// rejeu rangé (`ready`) ; issue gardée par la file (échec, film indisponible, non pris en charge) ; match inconnu ;
// mode non géré ; bit « film absent » ; film non archivé sans jeton ; sinon `generatable`.
func (s *ReplayGenerationService) GenerationStatus(ctx context.Context, titleSlug, matchID string) (domain.ReplayGenerationStatus, error) {
	st, _, err := s.evaluate(ctx, titleSlug, matchID)
	return st, err
}

// RequestGeneration met le match en file quand il est générable (ou en échec réessayable). Un match déjà rangé, en
// file ou en cours rend son état sans rien ajouter (created=false) ; les autres refus se rendent par les erreurs de
// domain : ErrReplayFilmUnavailable, ErrReplayGenerationNotSupported (raison portée par
// ReplayGenerationNotSupportedError), ErrReplayGenerationQueueFull.
func (s *ReplayGenerationService) RequestGeneration(ctx context.Context, titleSlug, matchID string) (domain.ReplayGenerationStatus, bool, error) {
	st, probe, err := s.evaluate(ctx, titleSlug, matchID)
	if err != nil {
		return domain.ReplayGenerationStatus{}, false, err
	}
	switch {
	case st.State == domain.ReplayGenerationReady, st.State == domain.ReplayGenerationQueued,
		st.State == domain.ReplayGenerationRunning:
		return st, false, nil
	case st.State == domain.ReplayGenerationFilmUnavailable:
		s.logRefusal(ctx, titleSlug, matchID, st)
		return st, false, domain.ErrReplayFilmUnavailable
	case st.State == domain.ReplayGenerationNotSupported, st.State == domain.ReplayGenerationFailed && !st.Retryable:
		s.logRefusal(ctx, titleSlug, matchID, st)
		return st, false, &domain.ReplayGenerationNotSupportedError{Reason: st.Reason}
	}
	fullID := probe.FullID
	if fullID == "" {
		fullID = matchID
	}
	queued, created, err := s.queue.Enqueue(domain.ReplayGenerationDemand{
		TitleSlug: titleSlug, MatchID: fullID, PlayerSlug: s.player.Slug, XUID: s.player.XUID,
		Gamertag: s.player.Gamertag, RequestedAt: time.Now().UTC(),
	})
	if err != nil {
		level := slog.LevelError
		if errors.Is(err, domain.ErrReplayGenerationQueueFull) {
			level = slog.LevelWarn
		}
		slog.Log(ctx, level, "replay_generation: demande refusée par la file",
			"titleSlug", titleSlug, "match_id", matchID, "player", s.player.Slug, "err", err)
		return domain.ReplayGenerationStatus{}, false, err
	}
	return queued, created, nil
}

// evaluate lit l'état du match et ce que la base et le disque en disent (la sonde, rendue pour la demande).
func (s *ReplayGenerationService) evaluate(ctx context.Context, titleSlug, matchID string) (
	domain.ReplayGenerationStatus, port.ReplayMatchProbeResult, error) {
	if !s.caps.Has(games.CapFilmReplayArtifact) {
		slog.WarnContext(ctx, "replay_generation: titre sans capacité de rejeu",
			"titleSlug", titleSlug, "match_id", matchID, "player", s.player.Slug)
		return domain.ReplayGenerationStatus{}, port.ReplayMatchProbeResult{}, games.ErrCapabilityNotSupported
	}
	known, haveKnown := s.queueStatus(titleSlug, matchID)
	if haveKnown && (known.State == domain.ReplayGenerationQueued || known.State == domain.ReplayGenerationRunning) {
		return known, port.ReplayMatchProbeResult{}, nil
	}
	probe, err := s.probe.ProbeReplayMatch(ctx, titleSlug, matchID)
	if err != nil {
		slog.ErrorContext(ctx, "replay_generation: lecture de l'état du match impossible",
			"titleSlug", titleSlug, "match_id", matchID, "player", s.player.Slug, "err", err)
		return domain.ReplayGenerationStatus{}, probe, err
	}
	if probe.ArtifactPresent {
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationReady}, probe, nil
	}
	if haveKnown {
		return known, probe, nil
	}
	return s.statusFromProbe(probe), probe, nil
}

// queueStatus : ce que la file sait du match ; la file absente ne sait rien.
func (s *ReplayGenerationService) queueStatus(titleSlug, matchID string) (domain.ReplayGenerationStatus, bool) {
	if s.queue == nil {
		return domain.ReplayGenerationStatus{}, false
	}
	return s.queue.DemandStatus(titleSlug, matchID)
}

// statusFromProbe : l'état d'un match que la file ne connaît pas, d'après la base et le disque.
func (s *ReplayGenerationService) statusFromProbe(p port.ReplayMatchProbeResult) domain.ReplayGenerationStatus {
	notSupported := func(reason string) domain.ReplayGenerationStatus {
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationNotSupported, Reason: reason}
	}
	switch {
	case !p.Known:
		return notSupported(domain.ReplayGenReasonMatchUnknown)
	case p.UnmanagedMode:
		return notSupported(domain.ReplayGenReasonModeUnmanaged)
	case p.FilmAbsent:
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationFilmUnavailable}
	case !p.FilmArchived && (s.token == nil || !s.token(s.player.XUID)):
		return notSupported(domain.ReplayGenReasonNoToken)
	}
	return domain.ReplayGenerationStatus{State: domain.ReplayGenerationGeneratable}
}

// logRefusal journalise le refus d'une demande (état non générable).
func (s *ReplayGenerationService) logRefusal(ctx context.Context, titleSlug, matchID string, st domain.ReplayGenerationStatus) {
	slog.InfoContext(ctx, "replay_generation: demande refusée",
		"titleSlug", titleSlug, "match_id", matchID, "player", s.player.Slug, "state", st.State, "reason", st.Reason)
}
