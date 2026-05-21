// Package duckdb — csr_thresholds_repo_test.go : couverture lookup season → threshold.
package duckdb

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
)

// helper : crée une metadata DB temp + applique les migrations metadata.
func openTempMetadataDBWithThresholds(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	if err := migration.RunForDB(db.SQLDb(), migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(TargetMetadata): %v", err)
	}
	return db
}

func TestCSRThresholdsRepo_Get_S13_ReturnsFive(t *testing.T) {
	t.Parallel()
	db := openTempMetadataDBWithThresholds(t)
	repo := NewCSRThresholdsRepo(db)
	if got := repo.Get(context.Background(), "CsrSeason13-1"); got != 5 {
		t.Errorf("Get(CsrSeason13-1) = %d, want 5", got)
	}
}

func TestCSRThresholdsRepo_Get_S2_ReturnsTen(t *testing.T) {
	t.Parallel()
	db := openTempMetadataDBWithThresholds(t)
	repo := NewCSRThresholdsRepo(db)
	if got := repo.Get(context.Background(), "CsrSeason2"); got != 10 {
		t.Errorf("Get(CsrSeason2) = %d, want 10", got)
	}
}

func TestCSRThresholdsRepo_Get_S3_ReturnsFive_PivotSeason(t *testing.T) {
	t.Parallel()
	// S3 = première saison où Halo a baissé à 5. Test explicite du pivot historique.
	db := openTempMetadataDBWithThresholds(t)
	repo := NewCSRThresholdsRepo(db)
	if got := repo.Get(context.Background(), "CsrSeason3-1"); got != 5 {
		t.Errorf("Get(CsrSeason3-1) = %d, want 5 (S3 = pivot seuil 10→5)", got)
	}
}

func TestCSRThresholdsRepo_Get_UnknownSeason_ReturnsDefault(t *testing.T) {
	t.Parallel()
	db := openTempMetadataDBWithThresholds(t)
	repo := NewCSRThresholdsRepo(db)
	if got := repo.Get(context.Background(), "CsrSeason99-1"); got != CSRPlacementThresholdDefault {
		t.Errorf("Get(unknown) = %d, want %d", got, CSRPlacementThresholdDefault)
	}
}

func TestCSRThresholdsRepo_Get_EmptySeasonID_ReturnsDefault(t *testing.T) {
	t.Parallel()
	db := openTempMetadataDBWithThresholds(t)
	repo := NewCSRThresholdsRepo(db)
	if got := repo.Get(context.Background(), ""); got != CSRPlacementThresholdDefault {
		t.Errorf("Get(\"\") = %d, want %d", got, CSRPlacementThresholdDefault)
	}
}

func TestCSRThresholdsRepo_Get_NilMetadata_ReturnsDefault(t *testing.T) {
	t.Parallel()
	repo := NewCSRThresholdsRepo(nil)
	if got := repo.Get(context.Background(), "CsrSeason13-1"); got != CSRPlacementThresholdDefault {
		t.Errorf("Get with nil metadata = %d, want %d", got, CSRPlacementThresholdDefault)
	}
}

func TestCSRThresholdsRepo_Get_NilRepo_ReturnsDefault(t *testing.T) {
	t.Parallel()
	var repo *CSRThresholdsRepo
	if got := repo.Get(context.Background(), "CsrSeason13-1"); got != CSRPlacementThresholdDefault {
		t.Errorf("Get on nil repo = %d, want %d", got, CSRPlacementThresholdDefault)
	}
}

func TestCSRThresholdsRepo_Get_CacheHit_AvoidSecondQuery(t *testing.T) {
	t.Parallel()
	db := openTempMetadataDBWithThresholds(t)
	repo := NewCSRThresholdsRepo(db)
	ctx := context.Background()
	// 1er lookup → query DB + cache set
	if got := repo.Get(ctx, "CsrSeason8-1"); got != 5 {
		t.Fatalf("first Get = %d", got)
	}
	// Vérification présence cache (accès interne sous lock)
	repo.mu.RLock()
	v, ok := repo.cache["CsrSeason8-1"]
	repo.mu.RUnlock()
	if !ok || v != 5 {
		t.Errorf("cache miss after Get : cache[%q]=%d ok=%v", "CsrSeason8-1", v, ok)
	}
	// 2e lookup retourne la valeur cache (smoke check, vraie validation = trace SQL)
	if got := repo.Get(ctx, "CsrSeason8-1"); got != 5 {
		t.Errorf("second Get = %d, want 5", got)
	}
}

func TestCSRThresholdsRepo_DefaultMatchesSyncConstant(t *testing.T) {
	t.Parallel()
	// Garde-fou : la constante duckdb.CSRPlacementThresholdDefault DOIT
	// rester en sync avec sync.CSRPlacementThresholdDefault (dupliquée
	// pour éviter un import cycle). Si on change l'une, on change l'autre.
	if CSRPlacementThresholdDefault != 5 {
		t.Errorf("CSRPlacementThresholdDefault (duckdb) = %d ; attendu 5 (S3+ Halo).", CSRPlacementThresholdDefault)
	}
}
