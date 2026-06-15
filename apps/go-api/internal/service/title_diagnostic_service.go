// Package service — title_diagnostic_service.go : rapport de santé d'un titre
// (PMT-14 volet A). Compose la présence des mappings TOML (config) et la réalité
// DB (présence des bases + lignes des tables cœur) via un port.TableInspector
// read-only. Aucune écriture.
package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/feature"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// CapabilitiesProvider expose les capabilities déclarées d'un titre
// (capabilities.toml). Optionnel : si fourni, le diagnostic calcule le DRIFT
// déclaré-vs-réalité (cf. WithCapabilities).
type CapabilitiesProvider interface {
	GetCapabilities(slug string) (*mappings.CapabilityMappingSet, bool)
}

// TitleDiagnosticService produit le rapport de diagnostic d'un titre.
type TitleDiagnosticService struct {
	repoRoot  string
	inspector port.TableInspector
	caps      CapabilitiesProvider // optionnel (drift) ; nil = rapport santé seul
}

// NewTitleDiagnosticService crée le service.
func NewTitleDiagnosticService(repoRoot string, inspector port.TableInspector) *TitleDiagnosticService {
	return &TitleDiagnosticService{repoRoot: repoRoot, inspector: inspector}
}

// WithCapabilities injecte le fournisseur de capabilities déclarées pour activer
// le calcul du drift (déclaré-vs-DB). Sans lui, le rapport reste « santé » seul.
func (s *TitleDiagnosticService) WithCapabilities(caps CapabilitiesProvider) *TitleDiagnosticService {
	s.caps = caps
	return s
}

// featureBackingTable mappe une feature produit vers la table DB censée la
// nourrir (sous-ensemble : seules les features à source unique évidente). Sert
// au drift « data » (feature déclarée available mais table vide/absente).
var featureBackingTable = map[feature.Key]struct{ db, table string }{
	feature.KeyMatchHistory: {db: "shared_matches_v2.duckdb", table: "match_registry"},
	feature.KeyPveStats:     {db: "shared_pve.duckdb", table: "pve_match_stats"},
}

type diagConfigFile struct {
	name     string
	required bool
}

// diagnosticConfigFiles = mappings TOML attendus pour un titre. fields +
// capabilities sont requis (sans eux le titre n'est pas exploitable).
var diagnosticConfigFiles = []diagConfigFile{
	{"fields.toml", true},
	{"capabilities.toml", true},
	{"outcomes.toml", false},
	{"assets.toml", false},
	{"constants.toml", false},
}

// Diagnose produit le rapport pour titleSlug. Best-effort : une DB absente ou
// illisible n'interrompt pas le rapport (statut renseigné par DB/table).
func (s *TitleDiagnosticService) Diagnose(ctx context.Context, titleSlug string) (*domain.TitleDiagnostic, error) {
	pr := titlePkg.NewPathResolver(s.repoRoot)
	report := &domain.TitleDiagnostic{TitleSlug: titleSlug}

	mappingsDir := filepath.Join(s.repoRoot, "config", "titles", titleSlug, "mappings")
	for _, cf := range diagnosticConfigFiles {
		_, err := os.Stat(filepath.Join(mappingsDir, cf.name))
		report.ConfigFiles = append(report.ConfigFiles, domain.ConfigFileStatus{
			Name: cf.name, Present: err == nil, Required: cf.required,
		})
	}

	probes := []struct {
		name   string
		path   string
		tables []string
	}{
		{"metadata.duckdb", pr.MetadataDBPath(titleSlug), []string{"season_calendars", "career_ranks", "asset_translations"}},
		{"shared_matches_v2.duckdb", pr.SharedDBPath(titleSlug), []string{"match_registry", "match_participants"}},
		{"shared_pve.duckdb", pr.SharedPVEDBPath(titleSlug), []string{"pve_match_stats"}},
	}
	for _, p := range probes {
		ds := domain.DatabaseStatus{Name: p.name}
		if _, err := os.Stat(p.path); err != nil {
			report.Databases = append(report.Databases, ds) // Exists=false
			continue
		}
		ds.Exists = true
		for _, tbl := range p.tables {
			n, exists, err := s.inspector.CountRows(ctx, p.path, tbl)
			if err != nil {
				if ds.Error == "" {
					ds.Error = err.Error()
				}
				continue
			}
			ds.Tables = append(ds.Tables, domain.TableStatus{Name: tbl, Exists: exists, Rows: n})
		}
		report.Databases = append(report.Databases, ds)
	}

	if s.caps != nil {
		report.Drifts = s.computeDrifts(ctx, titleSlug, report.Databases)
	}

	dataDrifts, featureDrifts := 0, 0
	for _, d := range report.Drifts {
		switch d.Kind {
		case "data":
			dataDrifts++
		case "feature":
			featureDrifts++
		}
	}
	slog.InfoContext(ctx, "title_diagnostic.report",
		"title", titleSlug,
		"config_files", len(report.ConfigFiles),
		"databases", len(report.Databases),
		"data_drifts", dataDrifts,
		"feature_drifts", featureDrifts,
	)
	return report, nil
}

// computeDrifts compare la config déclarée (capabilities.toml → cascade
// feature-matrix, RÉUTILISE le FeatureChecker games.ComputeFeatureMatrix) à la
// réalité DB pour détecter les écarts (feature available sans données = data
// drift ; feature degraded par enrichissement manquant = feature drift).
func (s *TitleDiagnosticService) computeDrifts(ctx context.Context, slug string, dbs []domain.DatabaseStatus) []domain.TitleDrift {
	set, ok := s.caps.GetCapabilities(slug)
	if !ok {
		return nil // pas de capabilities déclarées → drift non calculable
	}
	cm, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		return nil // capabilities.toml hors-vocabulaire → drift non calculable (déjà loggé ailleurs)
	}
	matrix := games.ComputeFeatureMatrix(cm)

	var drifts []domain.TitleDrift
	for _, fk := range games.AllFeatureKeys() {
		switch matrix[fk] {
		case feature.StatusDegraded:
			drifts = append(drifts, domain.TitleDrift{
				Feature:  string(fk),
				Kind:     "feature",
				Computed: string(feature.StatusDegraded),
				Reason:   "capability primaire déclarée mais un enrichissement manque (surface partielle)",
			})
		case feature.StatusAvailable:
			bt, has := featureBackingTable[fk]
			if !has {
				continue
			}
			rows, present := tableRows(dbs, bt.db, bt.table)
			if present && rows > 0 {
				continue
			}
			slog.WarnContext(ctx, "title_diagnostic.capability_absent",
				"title", slug, "table", bt.table)
			drifts = append(drifts, domain.TitleDrift{
				Feature:  string(fk),
				Kind:     "data",
				Computed: string(feature.StatusAvailable),
				Reason:   "feature déclarée disponible mais " + bt.table + " absente/vide",
			})
		}
	}
	return drifts
}

// tableRows lit le nombre de lignes d'une table dans les bases déjà sondées.
func tableRows(dbs []domain.DatabaseStatus, dbName, table string) (int64, bool) {
	for _, db := range dbs {
		if db.Name != dbName || !db.Exists {
			continue
		}
		for _, t := range db.Tables {
			if t.Name == table {
				return t.Rows, t.Exists
			}
		}
	}
	return 0, false
}
