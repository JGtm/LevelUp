package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// HistoryProfileSource liste les profils suivis (couples joueur x titre) ; *config.AppConfig la satisfait.
type HistoryProfileSource interface {
	LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error)
}

// TitleNamer rend le nom d'un titre tel que le registre le déclare.
type TitleNamer func(titleSlug string) string

// HistorySyncService compose l'état de l'historique d'un joueur sur chacun de ses titres suivis : ce que
// la base a rangé (port.HistoryProgressReader), la dernière passe et le calendrier de l'ordonnanceur
// (port.SyncActivity). Lecture seule.
type HistorySyncService struct {
	profiles  HistoryProfileSource
	progress  port.HistoryProgressReader
	activity  port.SyncActivity
	titleName TitleNamer
}

var _ port.HistorySyncService = (*HistorySyncService)(nil)

// NewHistorySyncService construit le service ; titleName peut être nil (le nom du titre est alors son
// identifiant).
func NewHistorySyncService(profiles HistoryProfileSource, progress port.HistoryProgressReader, activity port.SyncActivity, titleName TitleNamer) *HistorySyncService {
	return &HistorySyncService{profiles: profiles, progress: progress, activity: activity, titleName: titleName}
}

// HistorySyncStatus rend l'état de l'historique du joueur, un élément par titre suivi (ordre des
// identifiants de titre), libellés dans la langue de la requête. Une lecture en échec sur un titre n'échoue
// pas la réponse : ce titre est « inconnu ». Un joueur sans titre suivi rend une liste vide.
func (s *HistorySyncService) HistorySyncStatus(ctx context.Context, playerSlug string) (*domain.HistorySyncStatus, error) {
	profiles, err := s.profiles.LoadPlayers()
	if err != nil {
		return nil, fmt.Errorf("history sync status: profils: %w", err)
	}
	loc := i18n.FromContext(ctx)
	out := &domain.HistorySyncStatus{
		AutoSyncEnabled: s.activity.AutoSyncEnabled(),
		Titles:          []domain.TitleHistoryStatus{},
	}
	if next := s.activity.NextCycleAt(); !next.IsZero() {
		next = next.UTC()
		out.NextCycleAt = &next
	}
	for _, p := range profiles {
		if p.PlayerSlug != playerSlug || p.AuthOnly {
			continue
		}
		out.Titles = append(out.Titles, s.titleStatus(ctx, loc, p))
	}
	sort.Slice(out.Titles, func(i, j int) bool { return out.Titles[i].TitleSlug < out.Titles[j].TitleSlug })
	return out, nil
}

// titleStatus compose l'état d'un profil : lecture de la base, puis dernière passe de l'ordonnanceur.
func (s *HistorySyncService) titleStatus(ctx context.Context, loc i18n.Locale, p domain.PlayerSummary) domain.TitleHistoryStatus {
	slug := p.TitleSlug
	if slug == "" {
		slug = titlePkg.DefaultSlug
	}
	st := domain.TitleHistoryStatus{TitleSlug: slug, TitleName: slug}
	if s.titleName != nil {
		if name := s.titleName(slug); name != "" {
			st.TitleName = name
		}
	}

	state := domain.HistoryStateUnknown
	progress, err := s.progress.ReadHistoryProgress(ctx, slug, p.Gamertag, p.XUID)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "history sync status: lecture de l'état de l'historique échouée",
			"title_slug", slug, "gamertag", p.Gamertag, "err", err)
	case progress.Complete:
		state = domain.HistoryStateComplete
	default:
		state = domain.HistoryStateCatchingUp
	}
	if err == nil {
		st.MatchCount = progress.MatchCount
		if !progress.OldestAt.IsZero() {
			oldest := progress.OldestAt.UTC()
			st.OldestMatchAt = &oldest
		}
	}
	if !p.SyncEnabled {
		state = domain.HistoryStatePaused
	}
	st.State, st.StateLabel = state, stateLabel(loc, state)
	st.FilmsPending = s.activity.FilmsPending(slug)

	if pass, ok := s.activity.LastPass(slug, p.Gamertag); ok {
		st.LastPass = &domain.HistoryPass{
			At:           pass.At.UTC().Truncate(time.Second),
			MatchesAdded: pass.MatchesInserted,
			Status:       string(pass.Status),
			StatusLabel:  passLabel(loc, pass.Status),
		}
	}
	return st
}

// stateLabel rend le libellé d'un état de l'historique dans la langue demandée (catalogue sync.toml).
func stateLabel(loc i18n.Locale, state string) string {
	switch state {
	case domain.HistoryStateComplete:
		return i18n.Msg(loc, "sync.history.state.complete", nil)
	case domain.HistoryStateCatchingUp:
		return i18n.Msg(loc, "sync.history.state.catching_up", nil)
	case domain.HistoryStatePaused:
		return i18n.Msg(loc, "sync.history.state.paused", nil)
	default:
		return i18n.Msg(loc, "sync.history.state.unknown", nil)
	}
}

// passLabel rend le libellé de l'issue d'une passe dans la langue demandée (catalogue sync.toml).
func passLabel(loc i18n.Locale, status port.SyncPassStatus) string {
	switch status {
	case port.SyncPassOK:
		return i18n.Msg(loc, "sync.history.pass.ok", nil)
	case port.SyncPassPartial:
		return i18n.Msg(loc, "sync.history.pass.partial", nil)
	case port.SyncPassFailed:
		return i18n.Msg(loc, "sync.history.pass.failed", nil)
	default:
		return i18n.Msg(loc, "sync.history.pass.skipped", nil)
	}
}
