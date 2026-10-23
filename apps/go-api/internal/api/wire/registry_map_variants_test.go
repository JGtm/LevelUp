package wire

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
)

// Les suffixes sont ceux du titre du joueur : un autre titre, ou un registre sans table, n'en a pas.
func TestMapVariantSuffixesFor(t *testing.T) {
	reg := NewServiceRegistry(&config.AppConfig{}, nil).WithMapVariantSuffixes(map[string][]string{
		"halo_infinite": {"- Ranked", "Heavies"},
	})

	got := reg.mapVariantSuffixesFor(&duckdb.PlayerDB{TitleSlug: "halo_infinite"})
	if want := []string{"- Ranked", "Heavies"}; !reflect.DeepEqual(got, want) {
		t.Errorf("suffixes de halo_infinite = %v, attendu %v", got, want)
	}
	if got := reg.mapVariantSuffixesFor(&duckdb.PlayerDB{TitleSlug: "halo_5"}); got != nil {
		t.Errorf("halo_5 ne déclare aucune variante : reçu %v", got)
	}
	vide := NewServiceRegistry(&config.AppConfig{}, nil)
	if got := vide.mapVariantSuffixesFor(&duckdb.PlayerDB{TitleSlug: "halo_infinite"}); got != nil {
		t.Errorf("registre sans table : aucune variante attendue, reçu %v", got)
	}
}
