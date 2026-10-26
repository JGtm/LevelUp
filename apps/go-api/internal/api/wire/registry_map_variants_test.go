package wire

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
)

// Les suffixes sont ceux du titre du joueur : un autre titre, ou un registre sans table, n'en a pas.
// Sans catalogue de cartes intégrées (racine du dépôt vide), aucun résolveur de clé de module.
func TestMapGroupRulesFor(t *testing.T) {
	reg := NewServiceRegistry(&config.AppConfig{}, nil).WithMapVariantSuffixes(map[string][]string{
		"halo_infinite": {"- Ranked", "Heavies"},
	})

	got := reg.mapGroupRulesFor(&duckdb.PlayerDB{TitleSlug: "halo_infinite"})
	if want := []string{"- Ranked", "Heavies"}; !reflect.DeepEqual(got.Suffixes, want) {
		t.Errorf("suffixes de halo_infinite = %v, attendu %v", got.Suffixes, want)
	}
	if got.Modules != nil {
		t.Error("sans catalogue des bornes, le titre n'a pas de résolveur de clé de module")
	}
	if got := reg.mapGroupRulesFor(&duckdb.PlayerDB{TitleSlug: "halo_5"}); got.Active() {
		t.Errorf("halo_5 ne déclare aucune variante : reçu %+v", got)
	}
	vide := NewServiceRegistry(&config.AppConfig{}, nil)
	if got := vide.mapGroupRulesFor(&duckdb.PlayerDB{TitleSlug: "halo_infinite"}); got.Active() {
		t.Errorf("registre sans table : aucune règle attendue, reçu %+v", got)
	}
}
