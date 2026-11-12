package importstore

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
)

// importSource étiquette les lots d'écriture et le writer pris par un import OpenSpartan.
const importSource = "openspartan_import"

// SharedAcquirer prend le writer de la base partagée pour un import OpenSpartan.
type SharedAcquirer struct {
	provider     sharedprovider.Provider
	sharedDBPath string
}

var _ port.OpenSpartanSharedAcquirer = (*SharedAcquirer)(nil)

// NewSharedAcquirer lie l'acquisition au fournisseur de la base partagée (coordonné avec la
// synchronisation et les lecteurs). provider nil : repli sur le bail applicatif et l'ouverture directe de
// sharedDBPath.
func NewSharedAcquirer(provider sharedprovider.Provider, sharedDBPath string) *SharedAcquirer {
	return &SharedAcquirer{provider: provider, sharedDBPath: sharedDBPath}
}

// AcquireShared prend le writer partagé, étiqueté pour les diagnostics ; la fonction rendue le libère.
func (a *SharedAcquirer) AcquireShared(ctx context.Context) (port.OpenSpartanSharedStore, func(), error) {
	db, release, err := sync.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, importSource), a.provider, a.sharedDBPath)
	if err != nil {
		return nil, nil, err
	}
	return &sharedStore{db: db}, release, nil
}

// sharedStore écrit dans le handle partagé pris par AcquireShared.
type sharedStore struct {
	db *sql.DB
}

// PersistMatch écrit registre, participants, médailles, CSR par match et alias de tiers d'un match dans UNE
// transaction INSERT-only (persist.SharedPersister, ADR 0019/0026, anti-ART #23645) : jamais d'ON CONFLICT
// DO UPDATE, un match déjà présent n'est pas réécrit.
func (s *sharedStore) PersistMatch(ctx context.Context, w port.OpenSpartanMatchWrite) error {
	builder := persist.NewBatchBuilder(title.DefaultSlug, "", "", importSource)
	builder.SetMatch(&w.Registry)
	if len(w.Participants) > 0 {
		builder.AddParticipants(w.Participants)
	}
	if len(w.Medals) > 0 {
		builder.AddMedals(w.Medals)
	}
	if len(w.CSRs) > 0 {
		inserts := make([]persist.MatchCSRInsert, 0, len(w.CSRs))
		for _, c := range w.CSRs {
			inserts = append(inserts, sync.SharedCSRRowToMatchCSRInsert(sync.SharedMatchCSRRow{
				MatchID:                     c.MatchID,
				XUID:                        c.XUID,
				RatingType:                  c.RatingType,
				RatingValue:                 c.RatingValue,
				Tier:                        c.Tier,
				SubTier:                     c.SubTier,
				TierLabel:                   c.TierLabel,
				RatingDelta:                 c.RatingDelta,
				MeasurementMatchesRemaining: c.MeasurementMatchesRemaining,
				SeasonID:                    c.SeasonID,
			}))
		}
		builder.AddMatchCSRs(inserts)
	}
	if len(w.Aliases) > 0 {
		aliases := make([]persist.XUIDAliasInsert, 0, len(w.Aliases))
		for _, a := range w.Aliases {
			aliases = append(aliases, persist.XUIDAliasInsert{XUID: a.XUID, Gamertag: a.Gamertag, LastSeen: a.LastSeen})
		}
		builder.AddXUIDAliases(aliases)
	}
	return persist.NewSharedPersister(s.db).Persist(ctx, builder.Build())
}

// InsertHighlightEvents écrit les événements marquants d'un match (INSERT OR IGNORE).
func (s *sharedStore) InsertHighlightEvents(ctx context.Context, matchID string, events []highlightevent.HighlightEvent) (int, error) {
	return sync.InsertHighlightEvents(ctx, s.db, matchID, events)
}

// UpsertOwnerAlias met à jour l'alias de l'importeur (SELECT puis UPDATE ou INSERT, jamais ON CONFLICT).
func (s *sharedStore) UpsertOwnerAlias(ctx context.Context, xuid, gamertag string) error {
	return sync.UpsertXUIDAlias(ctx, s.db, xuid, gamertag)
}
