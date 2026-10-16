package duckdb

// title_asset_drawer_loader_absent_test.go — la metadata d'un titre additionnel peut ne pas
// porter tous ses référentiels (titre jamais synchronisé, base copiée avant une migration, démo
// sans ce titre). Les chargeurs sont « best-effort : table absente/vide -> vide » : l'ABSENCE
// d'une table n'est donc pas une erreur et ne journalise pas ERROR (lot L8-7 du plan sécurité du
// 2026-10-15 : cinq ERROR « duckdb: query failed » au boot sur la metadata Halo 5 de la démo,
// une par table : maps_catalog, weapon_labels, medal_definitions, team_colors,
// csr_designations). Une table PRÉSENTE dont la lecture échoue reste une vraie erreur.

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// erreurHandler retient les enregistrements de niveau ERROR.
type erreurHandler struct {
	mu   *sync.Mutex
	errs *[]string
}

func (h erreurHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h erreurHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		var sb strings.Builder
		sb.WriteString(r.Message)
		r.Attrs(func(a slog.Attr) bool {
			sb.WriteString(" " + a.Key + "=" + a.Value.String())
			return true
		})
		h.mu.Lock()
		*h.errs = append(*h.errs, sb.String())
		h.mu.Unlock()
	}
	return nil
}
func (h erreurHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h erreurHandler) WithGroup(string) slog.Handler      { return h }

// capturerErreurs remplace le logger par défaut le temps du test et rend les ERROR vus.
func capturerErreurs(t *testing.T) *[]string {
	t.Helper()
	errs := &[]string{}
	prev := slog.Default()
	slog.SetDefault(slog.New(erreurHandler{mu: &sync.Mutex{}, errs: errs}))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return errs
}

func TestTitleAssetLoaders_TablesAbsentes_PasDErreur(t *testing.T) {
	db := openAssetMemDB(t) // metadata vide : aucune des cinq tables
	ctx := context.Background()
	errs := capturerErreurs(t)

	maps, weapons, medals := LoadTitleAssetDrawerData(ctx, db, "halo_5")
	if len(maps)+len(weapons)+len(medals) != 0 {
		t.Errorf("metadata vide : maps=%d weapons=%d medals=%d, attendu 0", len(maps), len(weapons), len(medals))
	}
	if got := LoadTeamColorNames(ctx, db); len(got) != 0 {
		t.Errorf("team_colors absente : %d entrée(s), attendu 0", len(got))
	}
	if got := LoadCSRBadgeMap(ctx, db); len(got) != 0 {
		t.Errorf("csr_designations absente : %d entrée(s), attendu 0", len(got))
	}
	if len(*errs) != 0 {
		t.Errorf("%d ERROR journalisée(s) pour une absence attendue :\n  %s", len(*errs), strings.Join(*errs, "\n  "))
	}
}

func TestTitleAssetLoaders_TablePartielle_ChargeCeQuiExiste(t *testing.T) {
	db := openAssetMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE weapon_labels (weapon_id UBIGINT, name_en VARCHAR, icon_url VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO weapon_labels VALUES (7, 'Magnum', 'https://cdn.example/magnum.png')`); err != nil {
		t.Fatal(err)
	}
	errs := capturerErreurs(t)

	_, weapons, _ := LoadTitleAssetDrawerData(ctx, db, "halo_5")
	if len(weapons) != 1 || weapons[0].NameEN != "Magnum" || weapons[0].ImageURL == "" {
		t.Fatalf("weapons = %+v, attendu l'arme présente", weapons)
	}
	if len(*errs) != 0 {
		t.Errorf("ERROR journalisée(s) alors que seules des tables sont absentes :\n  %s", strings.Join(*errs, "\n  "))
	}
}

func TestTitleAssetLoaders_TablePresenteEnEchec_ResteUneErreur(t *testing.T) {
	db := openAssetMemDB(t)
	ctx := context.Background()
	// team_colors existe mais sans la colonne lue : la lecture échoue pour de vrai.
	if _, err := db.Exec(ctx, `CREATE TABLE team_colors (team_id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	errs := capturerErreurs(t)

	_ = LoadTeamColorNames(ctx, db)
	if len(*errs) == 0 {
		t.Error("une table présente dont la lecture échoue doit rester journalisée en ERROR")
	}
}
