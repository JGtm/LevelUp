package duckdb_test

import (
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

func makeAsset(id, hash string) duckdb.AssetEntry {
	return duckdb.AssetEntry{
		TitleID:     "halo_infinite",
		AssetID:     id,
		ContentHash: hash,
	}
}

// TestAssetDiff_NewDetected vérifie qu'un asset absent localement est classé New.
func TestAssetDiff_NewDetected(t *testing.T) {
	local := []duckdb.AssetEntry{}
	incoming := []duckdb.AssetEntry{makeAsset("asset-1", "abc123")}

	report := duckdb.ComputeAssetDiff(local, incoming)

	if len(report.New) != 1 {
		t.Errorf("New: attendu 1, obtenu %d", len(report.New))
	}
	if len(report.Modified) != 0 {
		t.Errorf("Modified: attendu 0, obtenu %d", len(report.Modified))
	}
	if report.Unchanged != 0 {
		t.Errorf("Unchanged: attendu 0, obtenu %d", report.Unchanged)
	}
}

// TestAssetDiff_ModifiedDetected vérifie qu'un asset avec hash différent est classé Modified.
func TestAssetDiff_ModifiedDetected(t *testing.T) {
	local := []duckdb.AssetEntry{makeAsset("asset-1", "old-hash")}
	incoming := []duckdb.AssetEntry{makeAsset("asset-1", "new-hash")}

	report := duckdb.ComputeAssetDiff(local, incoming)

	if len(report.Modified) != 1 {
		t.Errorf("Modified: attendu 1, obtenu %d", len(report.Modified))
	}
	if len(report.New) != 0 {
		t.Errorf("New: attendu 0, obtenu %d", len(report.New))
	}
}

// TestAssetDiff_UnchangedNoWrite vérifie qu'un asset identique est classé Unchanged.
func TestAssetDiff_UnchangedNoWrite(t *testing.T) {
	local := []duckdb.AssetEntry{makeAsset("asset-1", "same-hash")}
	incoming := []duckdb.AssetEntry{makeAsset("asset-1", "same-hash")}

	report := duckdb.ComputeAssetDiff(local, incoming)

	if report.Unchanged != 1 {
		t.Errorf("Unchanged: attendu 1, obtenu %d", report.Unchanged)
	}
	if len(report.New) != 0 || len(report.Modified) != 0 {
		t.Error("Unchanged: New et Modified doivent être vides")
	}
}

// TestAssetDiff_Mixed vérifie un scénario mixte : 1 nouveau, 1 modifié, 1 inchangé.
func TestAssetDiff_Mixed(t *testing.T) {
	local := []duckdb.AssetEntry{
		makeAsset("asset-existing", "hash-old"),
		makeAsset("asset-same", "hash-stable"),
	}
	incoming := []duckdb.AssetEntry{
		makeAsset("asset-new", "hash-fresh"),
		makeAsset("asset-existing", "hash-updated"),
		makeAsset("asset-same", "hash-stable"),
	}

	report := duckdb.ComputeAssetDiff(local, incoming)

	if len(report.New) != 1 {
		t.Errorf("New: attendu 1, obtenu %d", len(report.New))
	}
	if len(report.Modified) != 1 {
		t.Errorf("Modified: attendu 1, obtenu %d", len(report.Modified))
	}
	if report.Unchanged != 1 {
		t.Errorf("Unchanged: attendu 1, obtenu %d", report.Unchanged)
	}
}
