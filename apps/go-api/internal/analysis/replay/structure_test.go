package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTempStructure(t *testing.T, ms MapStructure) string {
	t.Helper()
	blob, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	p := filepath.Join(t.TempDir(), "ridgeline.json")
	if err := os.WriteFile(p, blob, 0o644); err != nil {
		t.Fatalf("écriture: %v", err)
	}
	return p
}

func TestLoadMapStructure(t *testing.T) {
	p := writeTempStructure(t, MapStructure{
		SchemaVersion: MapStructureSchemaVersion,
		Module:        "ridgeline",
		Surfaces:      []Surface{{X0: -1, Y0: -2, X1: 3, Y1: 4, Z: 1.5, ZB: 0}},
	})
	ms, err := LoadMapStructure(p)
	if err != nil {
		t.Fatalf("LoadMapStructure: %v", err)
	}
	if ms.Module != "ridgeline" || len(ms.Surfaces) != 1 {
		t.Fatalf("document inattendu: %+v", ms)
	}
	if got := ms.Surfaces[0].Area(); got != 24 {
		t.Fatalf("aire = %v, attendu 24", got)
	}
}

// Une face inférieure à exactement 0 doit SURVIVRE à l'aller-retour JSON : un omitempty sur
// ZB la relirait comme 0 par défaut — invisible ici, mais il déplacerait d'un étage toute
// surface dont la face haute serait, elle, omise. Le test fige l'absence d'omitempty.
func TestSurfaceZeroAltitudeSurvivesJSON(t *testing.T) {
	blob, err := json.Marshal(Surface{X0: 1, Y0: 1, X1: 2, Y1: 2, Z: 0, ZB: 0})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("désérialisation: %v", err)
	}
	for _, k := range []string{"z", "zb"} {
		if _, ok := back[k]; !ok {
			t.Fatalf("champ %q omis du JSON (%s) — une altitude nulle serait perdue", k, blob)
		}
	}
}

func TestLoadMapStructureRejectsOtherSchemaVersion(t *testing.T) {
	p := writeTempStructure(t, MapStructure{SchemaVersion: MapStructureSchemaVersion + 1})
	if _, err := LoadMapStructure(p); err == nil {
		t.Fatal("une version de schéma inconnue doit être refusée")
	}
}

func TestSurfaceBounds(t *testing.T) {
	if surfaceBounds(nil) != nil {
		t.Fatal("aucune surface -> pas de bornes")
	}
	b := surfaceBounds([]Surface{
		{X0: 0, Y0: 0, X1: 2, Y1: 2},
		{X0: -5, Y0: 1, X1: -1, Y1: 9},
	})
	if b == nil || b.MinX != -5 || b.MinY != 0 || b.MaxX != 2 || b.MaxY != 9 {
		t.Fatalf("bornes inattendues: %+v", b)
	}
}

// L'ajout de la structure ne doit PAS incrémenter SchemaVersion : les champs sont
// optionnels (omitempty), un client existant qui les ignore reste correct.
func TestStructureIsOptionalInDocument(t *testing.T) {
	doc := ReplayDocument{SchemaVersion: SchemaVersion}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("désérialisation: %v", err)
	}
	for _, k := range []string{"structure", "structureBounds"} {
		if _, ok := back[k]; ok {
			t.Fatalf("champ %q présent sur un document sans structure — omitempty manquant", k)
		}
	}
	if SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d : l'ajout de champs optionnels ne doit pas l'incrémenter", SchemaVersion)
	}
}
