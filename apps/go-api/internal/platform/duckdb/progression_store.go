package duckdb

// progression_store.go : les lectures de progression et de campagne d'UN joueur, derrière les ports
// `port.ProgressionStore` et `port.CampaignScope`. Chaque méthode délègue au dépôt de sa table (vues
// `_latest` pour les tables append-only) ; aucun SQL ici.

import (
	"context"

	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// PlayerProgressionStore lit la progression d'un joueur dans ses bases (stats du joueur, référentiel
// metadata, records dans la base sociale partagée).
type PlayerProgressionStore struct {
	pdb *PlayerDB
}

var _ port.ProgressionStore = (*PlayerProgressionStore)(nil)

// NewPlayerProgressionStore lie le dépôt au joueur de `pdb` ; pdb.Player, pdb.Metadata et pdb.XUID doivent
// être renseignés.
func NewPlayerProgressionStore(pdb *PlayerDB) *PlayerProgressionStore {
	return &PlayerProgressionStore{pdb: pdb}
}

// ListStreaks rend les séries du joueur pour le titre.
func (s *PlayerProgressionStore) ListStreaks(ctx context.Context, titleSlug string) ([]streaks.Streak, error) {
	return NewStreaksRepo(s.pdb.Player).List(ctx, s.pdb.XUID, titleSlug)
}

// ListPersonalBests rend les records personnels courants du joueur.
func (s *PlayerProgressionStore) ListPersonalBests(ctx context.Context) ([]records.PersonalRecord, error) {
	return NewPersonalRecordsRepo(s.pdb).ListByXUID(ctx, s.pdb.XUID)
}

// ListRecordHistory rend les `limit` records battus les plus récents du joueur pour le titre.
func (s *PlayerProgressionStore) ListRecordHistory(ctx context.Context, titleSlug string, limit int) ([]records.RecordHistory, error) {
	return NewRecordHistoryRepo(s.pdb.Player).ListRecent(ctx, s.pdb.XUID, titleSlug, limit)
}

// ListMilestoneCatalog rend le catalogue des jalons du titre (référentiel metadata).
func (s *PlayerProgressionStore) ListMilestoneCatalog(ctx context.Context, titleSlug string) ([]milestones.CatalogEntry, error) {
	return NewMilestoneCatalogRepo(s.pdb.Metadata).ListByTitle(ctx, titleSlug)
}

// ListMilestonesEarned rend les jalons franchis par le joueur pour le titre.
func (s *PlayerProgressionStore) ListMilestonesEarned(ctx context.Context, titleSlug string) ([]milestones.Earned, error) {
	return NewMilestoneEarnedRepo(s.pdb.Player).ListByUser(ctx, s.pdb.XUID, titleSlug)
}

// NewCampaignScope rend les dépôts de campagne du joueur de `pdb`.
func NewCampaignScope(pdb *PlayerDB) port.CampaignScope {
	return port.CampaignScope{
		XUID:    pdb.XUID,
		Repo:    NewCampaignRepo(pdb.Player),
		Samples: NewCampaignSampleProvider(pdb),
	}
}
