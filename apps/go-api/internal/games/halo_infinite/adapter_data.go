// Package halo_infinite — implémentation des adapters multi-titres pour HI.
//
// Phase B du plan : ce package wrappe les repos existants
// (internal/platform/duckdb/*) sans réécrire leurs requêtes. La migration
// physique des queries arrivera en Phase F (cleanup) une fois la bascule
// endpoint par endpoint validée par golden parity.
//
// Le data adapter utilise des interfaces minimales (CareerSource, etc.) pour
// rester découplé des structs concrets et permettre un mock simple en tests
// unitaires.
package halo_infinite

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// CareerSource est l'interface minimale consommée par le data adapter pour
// la lecture de la progression carrière. Implémentée par
// internal/platform/duckdb/CareerRepo.
type CareerSource interface {
	GetLatestRank(ctx context.Context) (*domain.CareerRankData, error)
	GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error)
}

// DataAdapter est l'implémentation HI de games.TitleDataAdapter.
//
// Phase B : seules les méthodes nécessaires aux endpoints prévus en Phase C
// sont câblées. Les autres retournent ErrCapabilityNotSupported pour rendre
// les absences explicites côté caller (cf. plan §5.7).
type DataAdapter struct {
	career CareerSource
	logger *slog.Logger
}

// NewDataAdapter construit un data adapter HI.
//
// career peut être nil si le titre est déclaré sans capability career — dans
// ce cas, LoadCareerSnapshot retourne ErrCapabilityNotSupported.
func NewDataAdapter(career CareerSource, logger *slog.Logger) *DataAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &DataAdapter{career: career, logger: logger}
}

// TitleSlug retourne le slug HI canonique.
func (a *DataAdapter) TitleSlug() string { return titlePkg.DefaultSlug }

// Capabilities décrit l'état des capabilities HI exposées par cet adapter.
//
// La map est dérivée de HALO_INFINITE_CAPABILITY_MAP.md. Un appel à une
// capability "not_exposed" retourne ErrCapabilityNotSupported.
func (a *DataAdapter) Capabilities() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapMatchHistory:       games.CapSupported,
		games.CapMatchDetailCore:    games.CapSupported,
		games.CapMatchSkillSnapshot: games.CapDegraded,
		games.CapCareerProgression:  capCareer(a.career),
		games.CapPveFirefight:       games.CapSupported,
		games.CapTimeseries:         games.CapNotExposed, // sortira en Phase C
	}
}

func capCareer(c CareerSource) games.CapabilityStatus {
	if c == nil {
		return games.CapNotExposed
	}
	return games.CapSupported
}

// LoadMatchSummaries n'est pas câblée en Phase B. Elle remontera en Phase C.
func (a *DataAdapter) LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadMatchDetail n'est pas câblée en Phase B.
func (a *DataAdapter) LoadMatchDetail(ctx context.Context, matchID string) (*canonical.MatchDetail, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadPlayerStats n'est pas câblée en Phase B.
func (a *DataAdapter) LoadPlayerStats(ctx context.Context, xuid string, scope canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadCareerSnapshot wrappe CareerSource.GetLatestRank et projette le résultat
// vers le canonique services.
//
// Comportement :
//   - career source nil → ErrCapabilityNotSupported ;
//   - aucune entrée trouvée (sql.ErrNoRows) → CareerSnapshot vide, pas d'erreur ;
//   - autre erreur → propagée avec contexte.
func (a *DataAdapter) LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	if a.career == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", games.CapCareerProgression,
		)
		return nil, games.ErrCapabilityNotSupported
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	row, err := a.career.GetLatestRank(ctx)
	if err != nil {
		if isNoRowsErr(err) {
			return &canonical.CareerSnapshot{
				Player: canonical.PlayerIdentity{XUID: xuid},
			}, nil
		}
		return nil, err
	}

	return projectCareerSnapshot(xuid, row), nil
}

// LoadEncounters wrappe CareerSource.GetEncounters et projette le résultat
// vers le canonique services.
//
// Comportement :
//   - career source nil → ErrCapabilityNotSupported ;
//   - aucune entrée trouvée → slice vide, pas d'erreur ;
//   - autre erreur → propagée avec contexte.
//
// L'argument xuid est ignoré : le CareerRepo HI résout déjà l'identité du
// joueur courant via son PlayerDB. Le paramètre est conservé dans la
// signature canonique pour permettre à un futur titre B de s'en servir.
func (a *DataAdapter) LoadEncounters(ctx context.Context, xuid string) ([]canonical.EncounterRow, error) {
	if a.career == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "career.encounters",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	rows, err := a.career.GetEncounters(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]canonical.EncounterRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectEncounterRow(r))
	}
	return out, nil
}

func projectEncounterRow(r domain.EncounterRawRow) canonical.EncounterRow {
	row := canonical.EncounterRow{
		Identity:   canonical.PlayerIdentity{XUID: r.XUID, Gamertag: r.Gamertag},
		MatchCount: r.MatchCount,
		AsTeammate: r.AsTeammate,
		AsEnemy:    r.AsEnemy,
	}
	if r.AvgKDA != nil {
		v := *r.AvgKDA
		row.AvgKDA = &v
	}
	return row
}

// LoadTimeseries n'est pas câblée en Phase B.
func (a *DataAdapter) LoadTimeseries(ctx context.Context, xuid string, query canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, games.ErrCapabilityNotSupported
}

// projectCareerSnapshot transforme un domain.CareerRankData en canonical.
// Garde la conversion isolée pour tester la projection sans toucher la DB.
func projectCareerSnapshot(xuid string, row *domain.CareerRankData) *canonical.CareerSnapshot {
	if row == nil {
		return &canonical.CareerSnapshot{Player: canonical.PlayerIdentity{XUID: xuid}}
	}

	xp := row.CurrentXP
	snap := &canonical.CareerSnapshot{
		Player:    canonical.PlayerIdentity{XUID: xuid},
		CurrentXP: &xp,
	}
	if row.XPForNextRank != nil {
		v := *row.XPForNextRank
		snap.XPForNextRank = &v
	}
	if row.RankLabel != nil || row.RankName != nil {
		snap.CurrentRank = &canonical.AssetReference{
			Kind:         "career_rank",
			ID:           rankID(row),
			DefaultLabel: stringDeref(row.RankName, stringDeref(row.RankLabel, "")),
		}
	}
	return snap
}

func rankID(row *domain.CareerRankData) string {
	if row.RankLabel != nil {
		return *row.RankLabel
	}
	if row.RankName != nil {
		return *row.RankName
	}
	return ""
}

func stringDeref(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// isNoRowsErr détecte les "pas de résultat" sans coupler le package au driver.
// La string match est volontairement permissive : sql.ErrNoRows s'imprime ainsi.
func isNoRowsErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, errSentinelNoRows) || matchesNoRowsString(err.Error())
}

// errSentinelNoRows est une sentinel locale comparable.
// On accepte aussi sql.ErrNoRows via duck-typing string pour ne pas importer
// database/sql dans games/halo_infinite (le projet utilise duckdb-go directement).
var errSentinelNoRows = errors.New("sql: no rows in result set")

func matchesNoRowsString(s string) bool {
	return s == "sql: no rows in result set"
}
