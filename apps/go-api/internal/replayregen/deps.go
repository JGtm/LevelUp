package replayregen

// deps.go — ce que la file demande au serveur : `Deps` et ses rythmes par défaut, le drapeau de reprise du lot
// (`PendingStore`) et le magasin des demandes explicites (`DemandStore`).

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// PendingStore : le drapeau persisté « la file était active » (réglages, `replay_regen_pending`).
type PendingStore interface {
	ReplayRegenPending() (bool, error)
	SetReplayRegenPending(pending bool) error
}

// Deps : tout ce que la file demande au serveur.
type Deps struct {
	Paths *title.PathResolver
	// Titles : les titres dont les rejeux se régénèrent (ceux qui déclarent la capacité de rejeu).
	Titles func() []string
	// FilmPresent : le film du match est dans l'archive locale ; sans lui il n'y a rien à rejouer.
	FilmPresent func(titleSlug, matchID string) bool
	// RefusedByRule rend, pour un titre, l'ensemble des matchs (forme courte, celle du nom du fichier) que la cuisson
	// refuse PAR RÈGLE : un mode que le rejeu ne gère pas (replaybuild.ErrModeNonGere, lu au registre par le même
	// prédicat que la cuisson). Le décompte les écarte d'emblée au lieu d'attendre leur premier refus de cuisson ; une
	// erreur est journalisée et le décompte retombe sur le comportement d'avant (refus au premier essai). nil = aucun.
	RefusedByRule func(ctx context.Context, titleSlug string) (map[string]struct{}, error)
	// Rebuild reconstruit et range le rejeu d'UN match, hors du processus (même chemin que l'action admin). Il
	// enveloppe ErrBusy quand un autre décodage tient la machine.
	Rebuild func(ctx context.Context, titleSlug, matchID string) error
	Pending PendingStore
	// HeavyAllowed : la porte du mode jeu (gameproc.Mode.HeavyWorkAllowed), consultée avant chaque film, avant
	// chaque nouvel essai d'un film occupé et avant chaque décompte.
	HeavyAllowed func() bool
	// SyncRunning : un cycle de synchronisation est en cours. La file lui CÈDE LA PLACE : entre deux films elle
	// attend la fin du cycle (le décodage de l'étape kill source et la cuisson après synchro prennent le même
	// verrou solo, et se reportent au cycle suivant quand la file le tient). nil = jamais.
	SyncRunning func() bool
	// FetchFilm télécharge et archive le film du match d'une demande explicite (jeton du joueur de la demande), ou
	// constate qu'il est déjà à l'archive. Il rend domain.ErrReplayFilmGone quand le film n'existe plus,
	// domain.ErrReplayNoToken sans jeton utilisable, finalise.ErrFilmNonFinalise quand le film n'est pas fini et
	// domain.ErrReplayFilmNotReady quand le film est introuvable mais le match vient de finir (échecs réessayables).
	// nil : le film est supposé déjà à l'archive (la reconstruction échoue sinon).
	FetchFilm func(ctx context.Context, d domain.ReplayGenerationDemand) error
	// Revalidate relit l'état du match d'une demande juste avant son téléchargement et rend l'état que la page
	// servirait pour ce match (domain.ReplayGenerationGeneratable : la demande se sert ; tout autre état la clôt avec
	// lui, sans rien télécharger). Une erreur clôt la demande en échec réessayable. nil = la demande est servie
	// telle quelle.
	Revalidate func(ctx context.Context, d domain.ReplayGenerationDemand) (domain.ReplayGenerationStatus, error)
	// ReleaseFilm retire le film d'un match dont la construction vient de se clore (rejeu rangé, ou échec définitif)
	// quand l'archive des films est désactivée. Un échec réessayable ou un arrêt garde le film. nil = le film reste.
	ReleaseFilm func(ctx context.Context, titleSlug, matchID string)
	// RasterStale dit si le fichier tactique d'un rejeu rangé est absent ou d'un autre format. Il ne se consulte que
	// pour un rejeu dont le film n'est plus à l'archive (celui dont le film est là se refait en entier). nil = aucun
	// fichier tactique à refaire sans film.
	RasterStale func(ctx context.Context, titleSlug, matchID string) bool
	// RebuildRaster refait le fichier tactique d'un match depuis son rejeu rangé : aucun film lu, rien de décodé, le
	// rejeu lui-même n'est pas touché. nil = aucun fichier tactique à refaire sans film.
	RebuildRaster func(ctx context.Context, titleSlug, matchID string) error
	// Demands persiste les demandes explicites en vol (reprise à l'ouverture). nil = rien n'est persisté.
	Demands DemandStore
	// MaxDemands : plafond de demandes explicites en vol ; zéro = [DefaultMaxDemands].
	MaxDemands int
	// Rythmes ; zéro = valeur de production.
	GateEvery      time.Duration
	BusyRetryEvery time.Duration
	BusyRetries    int
}

const (
	defaultGateEvery      = 15 * time.Second
	defaultBusyRetryEvery = 20 * time.Second
	defaultBusyRetries    = 30
)

func orDefault(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}

func (d Deps) gateEvery() time.Duration { return orDefault(d.GateEvery, defaultGateEvery) }
func (d Deps) busyEvery() time.Duration { return orDefault(d.BusyRetryEvery, defaultBusyRetryEvery) }
func (d Deps) busyRetries() int {
	if d.BusyRetries > 0 {
		return d.BusyRetries
	}
	return defaultBusyRetries
}

// DemandStore : le magasin des demandes explicites en vol. Il ne porte aucun secret.
type DemandStore interface {
	// LoadDemands rend les demandes persistées, dans l'ordre de la file ; un magasin vide rend nil.
	LoadDemands(ctx context.Context) ([]domain.ReplayGenerationDemand, error)
	// SaveDemands remplace le contenu du magasin par la liste donnée.
	SaveDemands(ctx context.Context, demands []domain.ReplayGenerationDemand) error
}

const (
	// DefaultMaxDemands : demandes explicites en vol au plus (dix films : quelques minutes de travail et quelques
	// centaines de Mo de films).
	DefaultMaxDemands = 10
	// maxResults : issues terminales gardées en mémoire (la plus ancienne sort).
	maxResults = 50
)

func (d Deps) maxDemands() int {
	if d.MaxDemands > 0 {
		return d.MaxDemands
	}
	return DefaultMaxDemands
}
