package service

import (
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/breakdown"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// ligneHistorique : une partie dont la carte, le mode et la playlist portent un nom dans chaque langue.
func ligneHistorique() domain.MatchHistoryRawRow {
	return domain.MatchHistoryRawRow{
		MatchID:      "m1",
		MapName:      i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"},
		PairName:     i18n.Text{i18n.Default: "Assassin", i18n.Fallback: "Slayer"},
		PlaylistName: i18n.Text{i18n.Default: "Arène : Quick Play", i18n.Fallback: "Arena: Quick Play"},
	}
}

// Les noms affichés d'une ligne (map_ui, mode_ui, playlist_label) suivent la langue de la requête ; en
// français rien ne change.
func TestEnrichRow_NomsAffichesSuiventLaLangue(t *testing.T) {
	fmts := rowFormatters{playlistLabel: analysis.PlaylistLabelConfig{StripCategory: true}.DisplayIn}
	fr := enrichRow(ligneHistorique(), nil, fmts, i18n.Default)
	en := enrichRow(ligneHistorique(), nil, fmts, i18n.Fallback)

	if got := derefString(fr.MapUI); got != "Ruelles" {
		t.Errorf("fr map_ui = %q, attendu Ruelles", got)
	}
	if got := derefString(fr.ModeUI); got != "Assassin" {
		t.Errorf("fr mode_ui = %q, attendu Assassin", got)
	}
	if got := derefString(fr.PlaylistLabel); got != "Quick Play" {
		t.Errorf("fr playlist_label = %q, attendu le nom sans préfixe de catégorie (comme avant)", got)
	}
	if got := derefString(en.MapUI); got != "Streets" {
		t.Errorf("en map_ui = %q, attendu Streets", got)
	}
	if got := derefString(en.ModeUI); got != "Slayer" {
		t.Errorf("en mode_ui = %q, attendu Slayer", got)
	}
	// Le retrait du préfixe de catégorie ne vise que le nom de la langue par défaut.
	if got := derefString(en.PlaylistLabel); got != "Arena: Quick Play" {
		t.Errorf("en playlist_label = %q, attendu le nom anglais tel quel", got)
	}
}

// Un nom sans valeur dans la langue demandée retombe sur la langue de repli (D3), jamais sur l'autre.
func TestEnrichRow_NomSansValeurDansLaLangueRetombeSurLAnglais(t *testing.T) {
	r := ligneHistorique()
	r.MapName = i18n.Text{i18n.Fallback: "Streets"}
	if got := derefString(enrichRow(r, nil, rowFormatters{}, i18n.Default).MapUI); got != "Streets" {
		t.Errorf("fr sans nom français : map_ui = %q, attendu le nom anglais", got)
	}
}

// Les options des filtres de l'Explorateur : la VALEUR est le nom de la langue par défaut (clé de
// `playlists`, `maps`, `modes`), le LIBELLÉ celui de la langue de la requête, et la valeur d'une option
// retenue filtre pareil dans les deux langues.
func TestComputeExplorerAvailableOptions_ValeurIndependanteDeLaLangue(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{ligneHistorique()}
	for _, loc := range []i18n.Locale{i18n.Default, i18n.Fallback} {
		_, playlists, maps, modes := computeExplorerAvailableOptions(rows, loc)
		wantLabel := map[i18n.Locale][3]string{
			i18n.Default:  {"Arène : Quick Play", "Ruelles", "Assassin"},
			i18n.Fallback: {"Arena: Quick Play", "Streets", "Slayer"},
		}[loc]
		wantValue := [3]string{"Arène : Quick Play", "Ruelles", "Assassin"}
		for i, opts := range [][]domain.LabelValue{playlists, maps, modes} {
			if len(opts) != 1 || opts[0].Label != wantLabel[i] || opts[0].Value != wantValue[i] {
				t.Errorf("%s : option %d = %+v, attendu libellé %q valeur %q", loc, i, opts, wantLabel[i], wantValue[i])
			}
		}
		for name, got := range map[string]int{
			"playlists": len(filterByExplorerPlaylists(rows, []string{playlists[0].Value})),
			"maps":      len(filterByExplorerMapNames(rows, []string{maps[0].Value})),
			"modes":     len(filterByExplorerModeNames(rows, []string{modes[0].Value})),
		} {
			if got != 1 {
				t.Errorf("%s : la valeur de l'option %s doit retenir la partie, %d retenue(s)", loc, name, got)
			}
		}
	}
}

// Le briefing de l'Explorateur : libellés dans la langue de la requête, regroupement par la clé de la
// langue par défaut.
func TestRawRowsToBreakdownRows_ClesStablesLibellesParLangue(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{ligneHistorique()}
	for loc, want := range map[i18n.Locale][3]string{
		i18n.Default:  {"Ruelles", "Assassin", "Arène : Quick Play"},
		i18n.Fallback: {"Streets", "Assassin", "Arène : Quick Play"},
	} {
		out, names := rawRowsToBreakdownRows(rows, domain.GroupesDeCartes{}, loc)
		if len(out) != 1 {
			t.Fatalf("%s : %d lignes", loc, len(out))
		}
		if out[0].MapLabel != want[0] {
			t.Errorf("%s : map = %q, attendu %q", loc, out[0].MapLabel, want[0])
		}
		if out[0].ModeName != want[1] || out[0].PlaylistName != want[2] {
			t.Errorf("%s : clés mode/playlist = %q / %q, attendu %q / %q", loc, out[0].ModeName, out[0].PlaylistName, want[1], want[2])
		}
		labelMode, labelPlaylist := names.modes.labelOr("Assassin"), names.playlists.labelOr("Arène : Quick Play")
		if loc == i18n.Fallback && (labelMode != "Slayer" || labelPlaylist != "Arena: Quick Play") {
			t.Errorf("en : libellés mode/playlist = %q / %q", labelMode, labelPlaylist)
		}
		if loc == i18n.Default && (labelMode != "Assassin" || labelPlaylist != "Arène : Quick Play") {
			t.Errorf("fr : libellés mode/playlist = %q / %q", labelMode, labelPlaylist)
		}
	}
}

// En plein historique, les ex aequo du briefing se départagent par le nom de la langue par défaut : la
// sélection et l'ordre des entrées ne changent pas avec la langue de la requête (seuls les libellés).
func TestBuildDimension_ExAequoDepartagesParLeNomParDefaut(t *testing.T) {
	counts := breakdown.Counts{Played: 12, Wins: 6, WinRate: 0.5}
	keys := optionLabels{"id-1": "Zeta", "id-2": "Alpha"}
	build := func(label1, label2 string) []string {
		ka := []breakdown.KeyedAggregate{
			{Key: "id-1", Label: label1, Counts: counts},
			{Key: "id-2", Label: label2, Counts: counts},
		}
		d := buildDimension("map", ka, ka, true, keys)
		if d == nil || len(d.Entries) != 2 {
			t.Fatalf("deux entrées attendues : %+v", d)
		}
		return []string{d.Entries[0].Label, d.Entries[1].Label}
	}
	fr := build("Zeta", "Alpha")
	en := build("Alpha-en", "Zeta-en") // libellés anglais dans l'ordre alphabétique inverse
	if fr[0] != "Alpha" || fr[1] != "Zeta" {
		t.Errorf("fr : %v, attendu Alpha puis Zeta (nom par défaut)", fr)
	}
	if en[0] != "Zeta-en" || en[1] != "Alpha-en" {
		t.Errorf("en : %v, attendu le même ordre qu'en français (id-2 puis id-1)", en)
	}
}
