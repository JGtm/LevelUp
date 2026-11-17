package wire

// registry_replay_generation_film.go — le film d'un match demandé à la génération à la demande : l'archive d'abord,
// puis le téléchargement avec le jeton DU JOUEUR DE LA DEMANDE, l'archivage, le marqueur « film absent » quand le
// film n'existe plus, et le retrait du film après la reconstruction quand l'archive est désactivée.
//
// LE JETON. Un seul appel, `tokencache.ResolveFreshPlayerTokens(ctx, d.XUID)`, paramétré par le xuid porté par la
// demande (celui du profil de la page, admis par la garde de propriété). Aucun autre jeton ne sert : ni le pool, ni le
// premier jeton valide, ni une boucle sur les profils (garde archlint `replay_generation_token_source_test.go`). Un film
// déjà complet à l'archive ne demande aucun jeton. Les jetons ne servent qu'au client de ce film ; aucune trace n'en
// est gardée dans la file, l'état ni le journal, et aucun message ne conseille de recapturer.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth/tokencache"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/filmfetch"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/sync/replayartifacts"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

const (
	// replayGenerationFilmRPS : débit des requêtes du téléchargement d'un film (un seul film à la fois).
	replayGenerationFilmRPS = 3
	// replayGenerationAbsentLeaseBudget : attente maximale de l'écrivain partagé pour poser le marqueur « film
	// absent », une fois par film expiré, depuis la goroutine de la file (jamais sous une requête).
	replayGenerationAbsentLeaseBudget = 8 * time.Second
)

// filmDownloader : la seule opération réseau du film, rendue par *haloadapter.Client.
type filmDownloader interface {
	GetFilmChunks(ctx context.Context, matchID string) ([]filmfetch.FilmChunk, bool, error)
}

// filmDemandFetcher : le téléchargement du film d'une demande, avec ses dépendances PASSÉES (jamais résolues ici)
// pour que les tests l'exercent sans réseau ni base.
type filmDemandFetcher struct {
	archiveRoot string
	// complete dit si le film est complet à l'archive (filmcache.FilmComplet).
	complete func(root, short string) (bool, error)
	// ensurePlayer rend le joueur de la demande résolu au pool du serveur (le rafraîchissement de jeton le cherche
	// par xuid). nil = rien à faire.
	ensurePlayer func(ctx context.Context, d domain.ReplayGenerationDemand) error
	// tokens rend les jetons frais DU joueur xuid : tokencache.ResolveFreshPlayerTokens en production.
	tokens func(ctx context.Context, xuid string) (*domain.HaloTokens, error)
	// newClient construit le client de téléchargement d'un film.
	newClient func(tokens *domain.HaloTokens) filmDownloader
	// write archive le film (filmarchive.Write).
	write func(ctx context.Context, root, short string, chunks []filmarchive.WriteChunk) error
	// markAbsent pose le bit « film absent » du match (killcollector.MarquerFilmAbsent).
	markAbsent func(ctx context.Context, matchID string) error
}

// fetch télécharge et archive le film de la demande, ou constate qu'il est déjà complet à l'archive.
func (f *filmDemandFetcher) fetch(ctx context.Context, d domain.ReplayGenerationDemand) error {
	short := title.FilmShortMatchID(d.MatchID)
	complete, err := f.complete(f.archiveRoot, short)
	if err != nil {
		slog.WarnContext(ctx, "replay_generation: état du film à l'archive illisible, film retéléchargé",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "err", err)
	}
	if complete {
		return nil
	}
	if f.ensurePlayer != nil {
		if err := f.ensurePlayer(ctx, d); err != nil {
			slog.ErrorContext(ctx, "replay_generation: profil de la demande introuvable",
				"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "err", err)
			return fmt.Errorf("profil de la demande: %w", err)
		}
	}
	client, err := f.clientFor(ctx, d)
	if err != nil {
		return err
	}
	chunks, found, err := client.GetFilmChunks(ctx, d.MatchID)
	switch {
	case errors.Is(err, finalise.ErrFilmNonFinalise):
		return fmt.Errorf("film non finalisé: %w", err)
	case err != nil && filmfetch.IsFilmGoneErr(err):
		return f.filmGone(ctx, d, err)
	case err != nil:
		return fmt.Errorf("téléchargement du film: %w", err)
	case !found || len(chunks) == 0:
		return f.filmGone(ctx, d, nil)
	}
	wc := make([]filmarchive.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmarchive.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS, Data: c.Data,
		})
	}
	if err := f.write(ctx, f.archiveRoot, short, wc); err != nil {
		slog.ErrorContext(ctx, "replay_generation: film non archivé",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "err", err)
		return fmt.Errorf("archivage du film: %w", err)
	}
	return nil
}

// clientFor obtient les jetons du joueur de la demande, UN SEUL APPEL, et en tire le client de ce film. Sans jeton
// utilisable (aucun, erreur, jeton vide) : domain.ErrReplayNoToken, cause journalisée, aucun conseil de recapture.
func (f *filmDemandFetcher) clientFor(ctx context.Context, d domain.ReplayGenerationDemand) (filmDownloader, error) {
	tokens, err := f.tokens(ctx, d.XUID)
	if err != nil || tokens == nil || tokens.SpartanToken == "" {
		slog.WarnContext(ctx, "replay_generation: jeton du joueur indisponible",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "xuid", d.XUID, "err", err)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrReplayNoToken, err)
		}
		return nil, domain.ErrReplayNoToken
	}
	return f.newClient(tokens), nil
}

// filmGone constate qu'un film n'existe plus (404/410 ou manifeste sans morceau) : pose le marqueur « film absent » et
// rend domain.ErrReplayFilmGone. Un échec du marqueur se journalise ; l'état reste « film indisponible » pour ce
// processus.
func (f *filmDemandFetcher) filmGone(ctx context.Context, d domain.ReplayGenerationDemand, cause error) error {
	if err := f.markAbsent(ctx, d.MatchID); err != nil {
		slog.WarnContext(ctx, "replay_generation: marqueur film absent non posé",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "err", err)
	}
	if cause != nil {
		return fmt.Errorf("%w: %w", domain.ErrReplayFilmGone, cause)
	}
	return domain.ErrReplayFilmGone
}

// fetchFilmForDemand est la fonction branchée sur replayregen.Deps.FetchFilm.
func (r *ServiceRegistry) fetchFilmForDemand(ctx context.Context, d domain.ReplayGenerationDemand) error {
	paths := r.cfg.PathResolver()
	return (&filmDemandFetcher{
		archiveRoot: paths.FilmArchiveDir(),
		complete:    filmcache.FilmComplet,
		ensurePlayer: func(ctx context.Context, d domain.ReplayGenerationDemand) error {
			_, err := r.resolve(ctxkeys.WithTitleSlug(ctx, d.TitleSlug), d.PlayerSlug)
			return err
		},
		tokens: tokencache.ResolveFreshPlayerTokens,
		newClient: func(t *domain.HaloTokens) filmDownloader {
			return syncpkg.NewHaloAPIClient(t.SpartanToken, t.ClearanceToken, replayGenerationFilmRPS)
		},
		write: filmarchive.Write,
		markAbsent: func(ctx context.Context, matchID string) error {
			acquire := syncpkg.NewSharedWriterAcquirer(r.sharedProviderForTitle(d.TitleSlug),
				paths.SharedDBPath(d.TitleSlug), "replay_generation_film_absent", replayGenerationAbsentLeaseBudget)
			return killcollector.MarquerFilmAbsent(ctx, persist.SharedWriterFn(acquire), matchID)
		},
	}).fetch(ctx, d)
}

// releaseFilmAfterBuild retire le film d'un match dont le rejeu vient d'être rangé quand le réglage « archiver les
// films » est faux (même règle que la cuisson). Un échec de retrait se journalise : le film reste.
func (r *ServiceRegistry) releaseFilmAfterBuild(ctx context.Context, titleSlug, matchID string) {
	retirerFilmSansArchive(ctx, r.cfg.PathResolver().FilmArchiveDir(), titleSlug, matchID, r.archiveFilmsEnabled(ctx))
}

// retirerFilmSansArchive retire le film du match de l'archive quand le réglage « archiver les films » est faux.
func retirerFilmSansArchive(ctx context.Context, archiveRoot, titleSlug, matchID string, archive bool) {
	if archive {
		return
	}
	if err := filmarchive.Supprimer(archiveRoot, title.FilmShortMatchID(matchID)); err != nil {
		slog.WarnContext(ctx, "replay_generation: film généré non retiré (archive désactivée)",
			"titleSlug", titleSlug, "match_id", matchID, "err", err)
		return
	}
	observability.IncCounter(replayartifacts.CompteurFilmsRetiresSansArchive)
}

// archiveFilmsEnabled lit le réglage « archiver les films » (oui par défaut : sans réglages, le film reste).
func (r *ServiceRegistry) archiveFilmsEnabled(ctx context.Context) bool {
	if r.settingsStore == nil {
		return true
	}
	s, err := r.settingsStore.Load()
	if err != nil {
		slog.WarnContext(ctx, "replay_generation: réglages illisibles, film conservé", "err", err)
		return true
	}
	return s == nil || s.ArchiveFilms
}
