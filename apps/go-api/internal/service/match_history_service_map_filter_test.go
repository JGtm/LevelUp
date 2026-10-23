package service

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

func rowCarte(id, fr, en string, filtre *string) domain.MatchHistoryRawRow {
	return domain.MatchHistoryRawRow{MatchID: id, MapNameFR: &fr, MapName: &en, MapFilterLabel: filtre}
}

func strp(s string) *string { return &s }

// La variante est proposée sous la carte de base, et choisir la base ramène les deux.
func TestFilterByExplorerMapNames_VarianteSousLaBase(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		rowCarte("m1", "Thunderhead", "Thunderhead", nil),
		rowCarte("m2", "Du lourd sur Thunderhead", "Thunderhead Heavies", strp("Thunderhead")),
		rowCarte("m3", "Aquarius", "Aquarius", nil),
	}

	_, _, maps, _ := computeExplorerAvailableOptions(rows)
	if want := []string{"Aquarius", "Thunderhead"}; !reflect.DeepEqual(maps, want) {
		t.Errorf("options de cartes = %v, attendu %v", maps, want)
	}

	got := filterByExplorerMapNames(rows, []string{"Thunderhead"})
	if len(got) != 2 || got[0].MatchID != "m1" || got[1].MatchID != "m2" {
		t.Errorf("la carte de base doit ramener ses variantes : %+v", got)
	}
}

// Le lien de la Tactique porte le libellé de la vignette de la variante : il ne ramène que celle-ci.
func TestFilterByExplorerMapNames_LienDeVignetteDeLaVariante(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		rowCarte("m1", "Thunderhead", "Thunderhead", nil),
		rowCarte("m2", "Du lourd sur Thunderhead", "Thunderhead Heavies", strp("Thunderhead")),
	}
	got := filterByExplorerMapNames(rows, []string{"Du lourd sur Thunderhead"})
	if len(got) != 1 || got[0].MatchID != "m2" {
		t.Errorf("le libellé affiché de la variante doit rester sélectionnable : %+v", got)
	}
}

func TestExplorerMapFilterLabel_SansBaseLibelleAffiche(t *testing.T) {
	r := rowCarte("m1", "Site Alpha", "Alpha Site", nil)
	if got := explorerMapFilterLabel(r); got != "Site Alpha" {
		t.Errorf("sans carte de base : libellé affiché attendu, reçu %q", got)
	}
	r.MapFilterLabel = strp("")
	if got := explorerMapFilterLabel(r); got != "Site Alpha" {
		t.Errorf("libellé de base vide : libellé affiché attendu, reçu %q", got)
	}
}
