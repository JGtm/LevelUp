package main

// Témoins de la passe FORGE propres à la CLI : la lecture de l'inventaire UGC et le compte des
// variantes absentes. La chaîne elle-même (jointure des libellés, zone muette, règle de
// publication) est témoignée là où elle vit : internal/mapcatalog/callouts_entree_test.go.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/mapcatalog"
)

// TestChargeInventaireNeGardeQueLesCartesForgeAvecVariante — l'inventaire porte aussi les
// cartes natives et des entrées sans map.mvar : elles n'ont rien à faire dans la passe.
func TestChargeInventaireNeGardeQueLesCartesForgeAvecVariante(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inv.json")
	body := `{"schema_version":1,"cartes":[
	 {"map_id":"aaa","nom":"Native","famille":"native","mvar":["map.mvar"],"blob_prefix":"https://x/"},
	 {"map_id":"bbb","nom":"SansVariante","famille":"forge","mvar":["fo11_blank.mvar"],"blob_prefix":"https://x/"},
	 {"map_id":"ccc","nom":"Bonne","famille":"forge","mvar":["fo11_blank.mvar","map.mvar"],"blob_prefix":"https://x/"}]}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cibles, err := chargeInventaire(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cibles) != 1 || cibles[0].MapID != "ccc" {
		t.Fatalf("cibles = %+v, attendu la seule carte Forge portant map.mvar", cibles)
	}
}

// TestChargeInventaireVideEstUneErreur — un inventaire sans cible est une CONFIGURATION
// cassée (mauvais fichier, schéma changé), pas un corpus vide : il doit se voir.
func TestChargeInventaireVideEstUneErreur(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inv.json")
	if err := os.WriteFile(p, []byte(`{"schema_version":1,"cartes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := chargeInventaire(p); err == nil {
		t.Error("inventaire sans carte Forge : attendu une erreur, obtenu nil")
	}
}

// TestConstruitPasseForgeCompteLesVariantesAbsentes — un cache vide n'est pas une panne :
// chaque carte manquante est comptée et nommée, aucune n'entre au catalogue.
func TestConstruitPasseForgeCompteLesVariantesAbsentes(t *testing.T) {
	stats := nouvellesStats()
	out := construitPasseForge([]carteUGC{
		{MapID: "aaa", Nom: "Absente1"}, {MapID: "bbb", Nom: "Absente2"},
	}, t.TempDir(), mapcatalog.Lexique{}, stats)
	if len(out) != 0 {
		t.Errorf("cartes publiées = %d, attendu 0", len(out))
	}
	if len(stats.Illisibles) != 2 || stats.Cartes != 0 {
		t.Errorf("stats = %d illisibles / %d cartes, attendu 2 / 0", len(stats.Illisibles), stats.Cartes)
	}
}
