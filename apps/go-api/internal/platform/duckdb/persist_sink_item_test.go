// Package duckdb — persist_sink_item_test.go : tests pour UpsertItemDefinition.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/migration"
)

// openItemTestDB ouvre une metadata.duckdb en mémoire avec les migrations appliquées.
func openItemTestDB(t *testing.T) *DB {
	t.Helper()
	return openBattlePassTestDB(t, "metadata_item.duckdb", migration.TargetMetadata)
}

func sinkForMeta(t *testing.T, db *DB) *PersistSink {
	t.Helper()
	return &PersistSink{MetaPath: db.Path()}
}

// ---------------------------------------------------------------------------
// itemDefLocalizedText
// ---------------------------------------------------------------------------

func TestItemDefLocalizedText_String(t *testing.T) {
	if got := itemDefLocalizedText("Hello", "fr-FR"); got != "Hello" {
		t.Errorf("got %q, want %q", got, "Hello")
	}
}

func TestItemDefLocalizedText_Nil(t *testing.T) {
	if got := itemDefLocalizedText(nil, "fr-FR"); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestItemDefLocalizedText_MapWithTranslations(t *testing.T) {
	v := map[string]any{
		"translations": map[string]any{
			"fr-FR": "Revêtement Noctua",
			"en-US": "Noctua Coating",
		},
	}
	if got := itemDefLocalizedText(v, "fr-FR"); got != "Revêtement Noctua" {
		t.Errorf("got %q, want %q", got, "Revêtement Noctua")
	}
	if got := itemDefLocalizedText(v, "en-US"); got != "Noctua Coating" {
		t.Errorf("got %q, want %q", got, "Noctua Coating")
	}
}

func TestItemDefLocalizedText_MapWithValueFallback(t *testing.T) {
	v := map[string]any{"value": "Some Title", "status": "Resolved"}
	if got := itemDefLocalizedText(v, "fr-FR"); got != "Some Title" {
		t.Errorf("got %q, want %q", got, "Some Title")
	}
}

func TestItemDefLocalizedText_MapMissingLang_FallsBackToOtherLang(t *testing.T) {
	v := map[string]any{
		"translations": map[string]any{
			"en-US": "English Title",
		},
	}
	// fr-FR absent → fallback vers en-US dans la liste de préférence
	if got := itemDefLocalizedText(v, "fr-FR"); got != "English Title" {
		t.Errorf("got %q, want %q (fallback en-US)", got, "English Title")
	}
}

// ---------------------------------------------------------------------------
// UpsertItemDefinition — comportement de base
// ---------------------------------------------------------------------------

func sampleItemJSON(quality, itemType, displayPath, titleFR, titleEN string) []byte {
	return []byte(`{
		"CommonData": {
			"Title": {
				"translations": {
					"fr-FR": "` + titleFR + `",
					"en-US": "` + titleEN + `"
				}
			},
			"Description": {
				"translations": {
					"fr-FR": "Description FR",
					"en-US": "Description EN"
				}
			},
			"Quality": "` + quality + `",
			"Type": "` + itemType + `",
			"DisplayPath": {
				"Media": {
					"MediaUrl": {
						"Path": "` + displayPath + `"
					}
				}
			}
		}
	}`)
}

func TestUpsertItemDefinition_InsertsStructuredData(t *testing.T) {
	ctx := context.Background()
	db := openItemTestDB(t)
	sink := sinkForMeta(t, db)

	raw := sampleItemJSON("Legendary", "ArmorCoating", "progression/items/coat.png", "Revêtement Légendaire", "Legendary Coating")

	if err := sink.UpsertItemDefinition(ctx, "Inventory/Coat-01.json", raw); err != nil {
		t.Fatalf("UpsertItemDefinition: %v", err)
	}

	// Vérifier battlepass_item_definitions
	var quality, itemType, displayPath string
	if err := db.QueryRow(ctx, `
		SELECT quality, item_type, display_path
		FROM battlepass_item_definitions
		WHERE inventory_item_path = 'Inventory/Coat-01.json'`).
		Scan(&quality, &itemType, &displayPath); err != nil {
		t.Fatalf("SELECT battlepass_item_definitions: %v", err)
	}
	if quality != "Legendary" {
		t.Errorf("quality = %q, want %q", quality, "Legendary")
	}
	if itemType != "ArmorCoating" {
		t.Errorf("item_type = %q, want %q", itemType, "ArmorCoating")
	}
	if displayPath != "progression/items/coat.png" {
		t.Errorf("display_path = %q, want %q", displayPath, "progression/items/coat.png")
	}

	// Vérifier battlepass_item_translations fr-FR
	var titleFR string
	if err := db.QueryRow(ctx, `
		SELECT title FROM battlepass_item_translations
		WHERE inventory_item_path = 'Inventory/Coat-01.json' AND lang = 'fr-FR'`).
		Scan(&titleFR); err != nil {
		t.Fatalf("SELECT translations fr-FR: %v", err)
	}
	if titleFR != "Revêtement Légendaire" {
		t.Errorf("title fr-FR = %q, want %q", titleFR, "Revêtement Légendaire")
	}

	// Vérifier battlepass_item_translations en-US
	var titleEN string
	if err := db.QueryRow(ctx, `
		SELECT title FROM battlepass_item_translations
		WHERE inventory_item_path = 'Inventory/Coat-01.json' AND lang = 'en-US'`).
		Scan(&titleEN); err != nil {
		t.Fatalf("SELECT translations en-US: %v", err)
	}
	if titleEN != "Legendary Coating" {
		t.Errorf("title en-US = %q, want %q", titleEN, "Legendary Coating")
	}
}

func TestUpsertItemDefinition_Idempotent(t *testing.T) {
	ctx := context.Background()
	db := openItemTestDB(t)
	sink := sinkForMeta(t, db)

	raw := sampleItemJSON("Epic", "WeaponCharm", "progression/items/charm.png", "Amulette", "Charm")

	for i := range 3 {
		if err := sink.UpsertItemDefinition(ctx, "Inventory/Charm-01.json", raw); err != nil {
			t.Fatalf("UpsertItemDefinition call %d: %v", i+1, err)
		}
	}

	var count int
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM battlepass_item_definitions
		WHERE inventory_item_path = 'Inventory/Charm-01.json'`).Scan(&count); err != nil {
		t.Fatalf("COUNT battlepass_item_definitions: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (idempotent)", count)
	}

	var translationCount int
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM battlepass_item_translations
		WHERE inventory_item_path = 'Inventory/Charm-01.json'`).Scan(&translationCount); err != nil {
		t.Fatalf("COUNT battlepass_item_translations: %v", err)
	}
	if translationCount != 2 { // fr-FR + en-US
		t.Errorf("translation count = %d, want 2", translationCount)
	}
}

func TestUpsertItemDefinition_HashChanges_InvalidatesOldEntry(t *testing.T) {
	ctx := context.Background()
	db := openItemTestDB(t)
	sink := sinkForMeta(t, db)

	rawV1 := sampleItemJSON("Common", "SpartanEmblem", "progression/emblem-v1.png", "Emblème", "Emblem")
	rawV2 := sampleItemJSON("Rare", "SpartanEmblem", "progression/emblem-v2.png", "Emblème Rare", "Rare Emblem")

	if err := sink.UpsertItemDefinition(ctx, "Inventory/Emblem-01.json", rawV1); err != nil {
		t.Fatalf("UpsertItemDefinition v1: %v", err)
	}
	if err := sink.UpsertItemDefinition(ctx, "Inventory/Emblem-01.json", rawV2); err != nil {
		t.Fatalf("UpsertItemDefinition v2: %v", err)
	}

	// L'ancienne entrée doit être is_current=FALSE
	var oldIsCurrent bool
	if err := db.QueryRow(ctx, `
		SELECT is_current FROM battlepass_item_definitions
		WHERE inventory_item_path = 'Inventory/Emblem-01.json' AND quality = 'Common'`).
		Scan(&oldIsCurrent); err != nil {
		t.Fatalf("SELECT old entry: %v", err)
	}
	if oldIsCurrent {
		t.Error("ancienne entrée devrait avoir is_current=FALSE")
	}

	// La nouvelle entrée doit être is_current=TRUE
	var newQuality string
	var newIsCurrent bool
	if err := db.QueryRow(ctx, `
		SELECT quality, is_current FROM battlepass_item_definitions
		WHERE inventory_item_path = 'Inventory/Emblem-01.json' AND quality = 'Rare'`).
		Scan(&newQuality, &newIsCurrent); err != nil {
		t.Fatalf("SELECT new entry: %v", err)
	}
	if !newIsCurrent {
		t.Error("nouvelle entrée devrait avoir is_current=TRUE")
	}
}

func TestUpsertItemDefinition_FallbackToItemTypeField(t *testing.T) {
	// Quand "Type" est absent mais "ItemType" est présent (ancien format GameCMS)
	ctx := context.Background()
	db := openItemTestDB(t)
	sink := sinkForMeta(t, db)

	raw := []byte(`{
		"CommonData": {
			"Title": {"value": "Old Format Item"},
			"Quality": "Common",
			"ItemType": "WeaponCoating",
			"DisplayPath": {"Media": {"MediaUrl": {"Path": "progression/weapon.png"}}}
		}
	}`)

	if err := sink.UpsertItemDefinition(ctx, "Inventory/OldFormat.json", raw); err != nil {
		t.Fatalf("UpsertItemDefinition: %v", err)
	}

	var itemType string
	if err := db.QueryRow(ctx, `
		SELECT item_type FROM battlepass_item_definitions
		WHERE inventory_item_path = 'Inventory/OldFormat.json'`).Scan(&itemType); err != nil {
		t.Fatalf("SELECT item_type: %v", err)
	}
	if itemType != "WeaponCoating" {
		t.Errorf("item_type = %q, want %q (fallback ItemType)", itemType, "WeaponCoating")
	}
}

func TestUpsertItemDefinition_EmptyOptionalFields_DoesNotFail(t *testing.T) {
	ctx := context.Background()
	db := openItemTestDB(t)
	sink := sinkForMeta(t, db)

	// Payload minimal — pas de Quality, Type, DisplayPath, ni traductions
	raw := []byte(`{"CommonData": {}}`)

	if err := sink.UpsertItemDefinition(ctx, "Inventory/Minimal.json", raw); err != nil {
		t.Fatalf("UpsertItemDefinition minimal: %v", err)
	}

	var count int
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM battlepass_item_definitions
		WHERE inventory_item_path = 'Inventory/Minimal.json'`).Scan(&count); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
