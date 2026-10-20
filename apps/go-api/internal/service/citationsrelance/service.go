// Package citationsrelance recalcule les citations du film d'un match pour chacun de ses joueurs
// suivis DÈS que les dérivations de ce film sont écrites (cuisson locale, dépôt d'un ouvrier,
// rattrapage des dérivés), au lieu d'attendre le prochain cycle de synchronisation de chacun.
//
// LE TRAVAIL NE SE FAIT JAMAIS SUR LE CHEMIN QUI LE DEMANDE. [Service.Soumettre] pose les matchs
// dans une file bornée et rend la main (le dépôt d'un ouvrier est une requête HTTP) ; une seule
// goroutine ([Service.Run], câblée au boot sur le contexte des tâches de fond) les traite un par
// un. File pleine ou service arrêté : la demande est comptée perdue — le post-sync de chaque
// joueur recalcule par la même règle, rien n'est perdu au-delà d'un cycle.
package citationsrelance

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// FileCap borne les matchs en attente. Une session de jeu en livre quelques-uns par heure ; un
// rattrapage des dérivés en livre quelques dizaines par cycle au plus.
const FileCap = 256

// Compteurs expvar (ADR 0009), par titre.
const (
	CompteurRecalculees  = "citations_relance_recalculees_total"
	CompteurAJour        = "citations_relance_a_jour_total"
	CompteurJoueurOccupe = "citations_relance_joueur_occupe_total"
	CompteurErreurs      = "citations_relance_erreurs_total"
	CompteurPerdues      = "citations_relance_perdues_total"
)

type demande struct {
	titleSlug, matchID string
}

// Service : la file et son ouvrier.
type Service struct {
	r      port.FilmCitationsRecomputer
	file   chan demande
	mu     sync.Mutex
	enFile map[demande]bool
}

// New construit le service ; [Service.Run] le fait travailler.
func New(r port.FilmCitationsRecomputer) *Service {
	return &Service{r: r, file: make(chan demande, FileCap), enFile: map[demande]bool{}}
}

// Soumettre pose les matchs dans la file, sans jamais attendre. Un match déjà en attente n'est
// pas posé deux fois. Forme de replayartifacts.NotifieurDerivations.
func (s *Service) Soumettre(ctx context.Context, titleSlug string, matchIDs []string) {
	for _, id := range matchIDs {
		d := demande{titleSlug: titleSlug, matchID: id}
		s.mu.Lock()
		if s.enFile[d] {
			s.mu.Unlock()
			continue
		}
		select {
		case s.file <- d:
			s.enFile[d] = true
		default:
			observability.IncCounterT(titleSlug, CompteurPerdues)
			slog.WarnContext(ctx, "citations relance: file pleine, match laissé au post-sync des joueurs",
				"match_id", id, "titleSlug", titleSlug, "capacite", FileCap)
		}
		s.mu.Unlock()
	}
}

// Run traite la file jusqu'à l'annulation du contexte.
func (s *Service) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-s.file:
			s.mu.Lock()
			delete(s.enFile, d)
			s.mu.Unlock()
			s.traiter(ctx, d)
		}
	}
}

// traiter relance les citations d'un match pour chacun de ses participants suivis.
func (s *Service) traiter(ctx context.Context, d demande) {
	suivis, err := s.r.FollowedPlayers(d.titleSlug)
	if err != nil {
		observability.IncCounterT(d.titleSlug, CompteurErreurs)
		slog.WarnContext(ctx, "citations relance: profils suivis illisibles", "err", err,
			"match_id", d.matchID, "titleSlug", d.titleSlug)
		return
	}
	participants, err := s.r.Participants(ctx, d.titleSlug, d.matchID)
	if err != nil {
		observability.IncCounterT(d.titleSlug, CompteurErreurs)
		slog.WarnContext(ctx, "citations relance: participants illisibles", "err", err,
			"match_id", d.matchID, "titleSlug", d.titleSlug)
		return
	}
	for _, p := range joueursSuivisDuMatch(suivis, participants) {
		if ctx.Err() != nil {
			return
		}
		s.relancerJoueur(ctx, d, p)
	}
}

// relancerJoueur recalcule pour un joueur et compte l'issue.
func (s *Service) relancerJoueur(ctx context.Context, d demande, p domain.PlayerSummary) {
	issue, err := s.r.RecomputeForPlayer(ctx, d.titleSlug, p, d.matchID)
	switch {
	case err != nil:
		observability.IncCounterT(d.titleSlug, CompteurErreurs)
		slog.WarnContext(ctx, "citations relance: recalcul échoué, laissé au post-sync du joueur",
			"err", err, "match_id", d.matchID, "player", p.Gamertag, "titleSlug", d.titleSlug)
	case issue == domain.CitationsRelanceRecalculee:
		observability.IncCounterT(d.titleSlug, CompteurRecalculees)
		slog.InfoContext(ctx, "citations relance: citations du film recalculées",
			"match_id", d.matchID, "player", p.Gamertag, "titleSlug", d.titleSlug)
	case issue == domain.CitationsRelanceJoueurOccupe:
		observability.IncCounterT(d.titleSlug, CompteurJoueurOccupe)
		slog.InfoContext(ctx, "citations relance: base du joueur tenue, son post-sync recalculera",
			"match_id", d.matchID, "player", p.Gamertag, "titleSlug", d.titleSlug)
	default:
		observability.IncCounterT(d.titleSlug, CompteurAJour)
	}
}

// joueursSuivisDuMatch : les profils suivis qui ont participé au match, une fois chacun.
func joueursSuivisDuMatch(suivis []domain.PlayerSummary, participants []string) []domain.PlayerSummary {
	present := make(map[string]bool, len(participants))
	for _, x := range participants {
		present[x] = true
	}
	var out []domain.PlayerSummary
	vus := map[string]bool{}
	for _, p := range domain.SyncablePlayers(suivis) {
		if p.XUID == "" || !present[p.XUID] || vus[p.XUID] {
			continue
		}
		vus[p.XUID] = true
		out = append(out, p)
	}
	return out
}
