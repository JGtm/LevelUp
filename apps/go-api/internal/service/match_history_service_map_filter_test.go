package service

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

func rowCarte(id, fr, en string, base i18n.Text) domain.MatchHistoryRawRow {
	return domain.MatchHistoryRawRow{MatchID: id, MapName: i18n.FromColumns(en, fr), MapFilterNames: base}
}

// baseThunderhead : la carte de base d'une variante, nommée dans chaque langue.
func baseThunderhead() i18n.Text {
	return i18n.Text{i18n.Default: "Thunderhead", i18n.Fallback: "Thunderhead"}
}

// La variante est proposée sous la carte de base, et choisir la base ramène les deux.
func TestFilterByExplorerMapNames_VarianteSousLaBase(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		rowCarte("m1", "Thunderhead", "Thunderhead", nil),
		rowCarte("m2", "Du lourd sur Thunderhead", "Thunderhead Heavies", baseThunderhead()),
		rowCarte("m3", "Aquarius", "Aquarius", nil),
	}

	_, _, maps, _ := computeExplorerAvailableOptions(rows, i18n.Default)
	want := []domain.LabelValue{{Label: "Aquarius", Value: "Aquarius"}, {Label: "Thunderhead", Value: "Thunderhead"}}
	if !reflect.DeepEqual(maps, want) {
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
		rowCarte("m2", "Du lourd sur Thunderhead", "Thunderhead Heavies", baseThunderhead()),
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
	r.MapFilterNames = i18n.Text{i18n.Default: ""}
	if got := explorerMapFilterLabel(r); got != "Site Alpha" {
		t.Errorf("nom de base vide : libellé affiché attendu, reçu %q", got)
	}
}

// Le filtre `?maps=` porte le nom de la langue par défaut : l'option rend ce nom pour VALEUR et le nom de
// la langue de la requête pour LIBELLÉ, et la sélection ramène les mêmes matchs en fr et en en.
func TestExplorerMapOptions_LibelleSuitLaLangueValeurNonEtLeFiltreTient(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		rowCarte("m1", "Ruelles", "Streets", nil),
		rowCarte("m2", "Du lourd sur Ruelles", "Streets Heavies", i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"}),
		rowCarte("m3", "Écluse", "Lock", nil),
	}
	_, _, mapsFR, _ := computeExplorerAvailableOptions(rows, i18n.Default)
	_, _, mapsEN, _ := computeExplorerAvailableOptions(rows, i18n.Fallback)
	wantFR := []domain.LabelValue{{Label: "Ruelles", Value: "Ruelles"}, {Label: "Écluse", Value: "Écluse"}}
	wantEN := []domain.LabelValue{{Label: "Lock", Value: "Écluse"}, {Label: "Streets", Value: "Ruelles"}}
	if !reflect.DeepEqual(mapsFR, wantFR) {
		t.Errorf("options fr = %v, attendu %v", mapsFR, wantFR)
	}
	if !reflect.DeepEqual(mapsEN, wantEN) {
		t.Errorf("options en = %v, attendu %v", mapsEN, wantEN)
	}
	for name, opts := range map[string][]domain.LabelValue{"fr": mapsFR, "en": mapsEN} {
		for _, o := range opts {
			if o.Value == "Ruelles" {
				if got := filterByExplorerMapNames(rows, []string{o.Value}); len(got) != 2 {
					t.Errorf("%s : la valeur %q doit ramener la carte et sa variante, reçu %d lignes", name, o.Value, len(got))
				}
			}
		}
	}
}
