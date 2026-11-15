package port

// openspartan_import_store.go — ce que les services d'import OpenSpartan demandent au stockage : écrire
// dans la base partagée (matchs, événements marquants, alias de l'importeur) et, après l'import, amorcer
// et recalculer ce qui dérive des matchs importés dans la base du joueur. Aucun type SQL ici : les
// implémentations (paquet `importstore`) tiennent les handles, les baux et le SQL.

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
)

// OpenSpartanMatchCSR est une ligne de CSR par match, extraite du payload de compétence d'un match
// importé, à écrire dans la base partagée.
type OpenSpartanMatchCSR struct {
	MatchID                     string
	XUID                        string
	RatingType                  string
	RatingValue                 *float64
	Tier                        string
	SubTier                     int
	TierLabel                   string
	RatingDelta                 *float64
	MeasurementMatchesRemaining int
	SeasonID                    string
}

// OpenSpartanAlias propose le gamertag d'un joueur croisé dans un match importé ; un nom déjà connu n'est
// jamais réécrit.
type OpenSpartanAlias struct {
	XUID     string
	Gamertag string
	LastSeen time.Time
}

// OpenSpartanMatchWrite est tout ce qu'un match importé écrit dans la base partagée, en une seule
// écriture atomique et idempotente (un match déjà présent n'est pas réécrit).
type OpenSpartanMatchWrite struct {
	Registry     domain.MatchRegistryRow
	Participants []domain.MatchParticipantRow
	Medals       []domain.MedalRow
	CSRs         []OpenSpartanMatchCSR
	Aliases      []OpenSpartanAlias
}

// OpenSpartanSharedStore écrit dans la base partagée pendant un import. Valable entre
// [OpenSpartanSharedAcquirer.AcquireShared] et l'appel de la fonction de libération.
type OpenSpartanSharedStore interface {
	// PersistMatch écrit un match, INSERT-only (ADR 0019/0026).
	PersistMatch(ctx context.Context, w OpenSpartanMatchWrite) error
	// InsertHighlightEvents écrit les événements marquants d'un match ; rend le nombre de lignes écrites.
	InsertHighlightEvents(ctx context.Context, matchID string, events []highlightevent.HighlightEvent) (int, error)
	// UpsertOwnerAlias met à jour l'alias de l'importeur.
	UpsertOwnerAlias(ctx context.Context, xuid, gamertag string) error
}

// OpenSpartanSharedAcquirer prend l'écriture de la base partagée, coordonnée avec la synchronisation et les
// lecteurs.
type OpenSpartanSharedAcquirer interface {
	// AcquireShared rend le magasin d'écriture et la fonction qui le libère (à appeler une fois).
	AcquireShared(ctx context.Context) (OpenSpartanSharedStore, func(), error)
}

// OpenSpartanPostImportPaths : les bases du titre sur lesquelles le recalcul d'après-import travaille.
type OpenSpartanPostImportPaths struct {
	Player   string // stats.duckdb du joueur, créée si absente
	Shared   string // base partagée des matchs
	Metadata string // référentiels du titre
	PVE      string // base des matchs Firefight, lue en lecture seule (absente : étape dégradée)
}

// OpenSpartanStageError nomme l'étape d'un recalcul qui a échoué lorsque ce n'est pas l'étape appelée
// (la prise d'une base avant le calcul, par exemple) : le service range l'échec sous ce nom.
type OpenSpartanStageError struct {
	Stage string
	Err   error
}

// Error rend le texte de la cause, tel que la base ou le fournisseur l'a produit.
func (e *OpenSpartanStageError) Error() string { return e.Err.Error() }

// Unwrap rend la cause.
func (e *OpenSpartanStageError) Unwrap() error { return e.Err }

// OpenSpartanEnrichmentFailure : l'amorçage de la ligne d'un match a échoué.
type OpenSpartanEnrichmentFailure struct {
	MatchID string
	Err     error
}

// OpenSpartanPostImportStore ouvre la base du joueur pour le recalcul d'après-import.
type OpenSpartanPostImportStore interface {
	// Open ouvre (et crée au besoin) la base du joueur ; l'appelant ferme la session.
	Open(ctx context.Context, paths OpenSpartanPostImportPaths) (OpenSpartanPostImportSession, error)
}

// OpenSpartanPostImportSession porte les étapes de recalcul d'un joueur sur ses matchs importés. Chaque
// étape prend elle-même les bases partagées qu'elle écrit, le temps de son travail. Une étape qui échoue
// à prendre une base rend une [OpenSpartanStageError].
type OpenSpartanPostImportSession interface {
	// Close ferme la base du joueur.
	Close() error
	// PrimeEnrichment amorce la ligne de base des matchs qui n'en ont pas, sans s'arrêter à la première
	// erreur : failures porte un échec par match, err l'échec de la lecture préalable.
	PrimeEnrichment(ctx context.Context, matchIDs []string) (failures []OpenSpartanEnrichmentFailure, err error)
	// ProjectCSR projette le CSR par match du joueur de la base partagée vers sa base ; rend le nombre de lignes.
	ProjectCSR(ctx context.Context, xuid string) (int, error)
	// RecomputeLUSR rejoue la note LUSR sur tout l'historique du joueur ; rend le nombre de matchs.
	RecomputeLUSR(ctx context.Context, xuid string) (int, error)
	// ComputePerfScores calcule les notes de performance ; force recalcule celles qui existent déjà.
	ComputePerfScores(ctx context.Context, xuid string, force bool) (int, error)
	// BackfillCitations recalcule les citations des matchs donnés.
	BackfillCitations(ctx context.Context, xuid string, matchIDs []string) error
}
