package service

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

func ligneFiltre(id, mapFR, mapEN, pairFR, pairEN, plFR, plEN string) domain.FilterMatchRow {
	return domain.FilterMatchRow{
		MatchID:      id,
		MapName:      i18n.Text{i18n.Default: mapFR, i18n.Fallback: mapEN},
		PairName:     i18n.Text{i18n.Default: pairFR, i18n.Fallback: pairEN},
		PlaylistName: i18n.Text{i18n.Default: plFR, i18n.Fallback: plEN},
	}
}

// Le filtre global : la valeur d'une option reste le nom de la langue par défaut, le libellé suit la
// langue de la requête, et la valeur retient les mêmes matchs dans les deux langues.
func TestLocalizeCascadeOptions_ValeurStableLibelleParLangue(t *testing.T) {
	rows := []domain.FilterMatchRow{
		ligneFiltre("m1", "Ruelles", "Streets", "Assassin", "Slayer", "Partie rapide", "Quick Play"),
		ligneFiltre("m2", "Écluse", "Lock", "Capture", "CTF", "Partie rapide", "Quick Play"),
	}
	for _, loc := range []i18n.Locale{i18n.Default, i18n.Fallback} {
		avail := buildAvailableOptions(rows, domain.CascadeFilter{})
		localizeCascadeOptions(&avail, rows, loc)

		wantValues := map[string][]string{
			"maps":      {"Écluse", "Ruelles"},
			"modes":     {"Assassin", "Capture"},
			"playlists": {"Partie rapide"},
		}
		got := map[string][]string{
			"maps": labelValues(avail.Maps), "modes": labelValues(avail.Modes), "playlists": labelValues(avail.Playlists),
		}
		// L'ordre suit le libellé affiché : on compare donc en ensemble.
		for name, want := range wantValues {
			if !sameSet(got[name], want) {
				t.Errorf("%s : valeurs %s = %v, attendu %v (indépendantes de la langue)", loc, name, got[name], want)
			}
		}
		labels := map[string]string{}
		for _, o := range append(append(append([]domain.LabelValue{}, avail.Maps...), avail.Modes...), avail.Playlists...) {
			labels[o.Value] = o.Label
		}
		want := map[i18n.Locale]map[string]string{
			i18n.Default:  {"Ruelles": "Ruelles", "Écluse": "Écluse", "Assassin": "Assassin", "Capture": "Capture", "Partie rapide": "Partie rapide"},
			i18n.Fallback: {"Ruelles": "Streets", "Écluse": "Lock", "Assassin": "Slayer", "Capture": "CTF", "Partie rapide": "Quick Play"},
		}[loc]
		if !reflect.DeepEqual(labels, want) {
			t.Errorf("%s : libellés = %v, attendu %v", loc, labels, want)
		}
		// La valeur retient les mêmes matchs quelle que soit la langue.
		if n := len(FilteredMatchIDs(rows, domain.FilterContextInput{Cascade: domain.CascadeFilter{Maps: []string{"Ruelles"}}})); n != 1 {
			t.Errorf("%s : ?maps=Ruelles retient %d match(s), attendu 1", loc, n)
		}
	}
}

// En français les options gardent l'ordre alphabétique des valeurs (libellé = valeur) ; en anglais elles
// suivent l'ordre alphabétique des libellés.
func TestLocalizeCascadeOptions_OrdreSuitLeLibelle(t *testing.T) {
	rows := []domain.FilterMatchRow{
		ligneFiltre("m1", "Ruelles", "Streets", "Assassin", "Slayer", "Partie rapide", "Quick Play"),
		ligneFiltre("m2", "Écluse", "Lock", "Capture", "CTF", "Partie rapide", "Quick Play"),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	localizeCascadeOptions(&avail, rows, i18n.Fallback)
	if got := labelValues(avail.Maps); !reflect.DeepEqual(got, []string{"Écluse", "Ruelles"}) { // Lock, Streets
		t.Errorf("en : cartes = %v, attendu Lock puis Streets", got)
	}
	if got := labelValues(avail.Modes); !reflect.DeepEqual(got, []string{"Capture", "Assassin"}) { // CTF, Slayer
		t.Errorf("en : modes = %v, attendu CTF puis Slayer", got)
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		seen[v]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}
