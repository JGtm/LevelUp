package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/legacymatch"
)

// Séries temporelles : le regroupement par carte (map_id) et l'ordre ne dépendent pas de la langue ; le
// nom affiché (map_ui) suit la langue de la requête.
func TestBuildSoloMapBreakdown_CleStableLibelleParLangue(t *testing.T) {
	base := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	win := analysis.OutcomeWin
	mapNames := i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"}
	current := []legacymatch.StatsMatchRow{
		{MatchID: "a", StartTime: base, MapName: mapNames, Outcome: &win},
		{MatchID: "b", StartTime: base.Add(time.Hour), MapName: mapNames, Outcome: &win},
	}
	fr := buildSoloMapBreakdown(current, current, domain.GroupesDeCartes{}, i18n.Default)
	en := buildSoloMapBreakdown(current, current, domain.GroupesDeCartes{}, i18n.Fallback)
	if len(fr) != 1 || len(en) != 1 {
		t.Fatalf("une carte attendue, fr=%d en=%d", len(fr), len(en))
	}
	if fr[0].MapID != "Ruelles" || en[0].MapID != "Ruelles" {
		t.Errorf("map_id = %q (fr) / %q (en), attendu la clé de la langue par défaut", fr[0].MapID, en[0].MapID)
	}
	if fr[0].MapUI != "Ruelles" || en[0].MapUI != "Streets" {
		t.Errorf("map_ui = %q (fr) / %q (en), attendu Ruelles / Streets", fr[0].MapUI, en[0].MapUI)
	}
	if fr[0].MatchCount != en[0].MatchCount || fr[0].HistoricalWinRate == nil || en[0].HistoricalWinRate == nil {
		t.Errorf("les agrégats et la référence historique doivent être identiques dans les deux langues")
	}
}

// Séries temporelles : le premier sang porte la carte et le mode de la langue de la requête.
func TestStatsMatchRowFirstBloodMeta_SuitLaLangue(t *testing.T) {
	m := legacymatch.StatsMatchRow{
		MatchID: "m", MapName: i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"},
		PairName: i18n.Text{i18n.Default: "Assassin", i18n.Fallback: "Slayer"},
	}
	fr := statsMatchRowFirstBloodMeta(m, i18n.Default)
	en := statsMatchRowFirstBloodMeta(m, i18n.Fallback)
	if fr.MapUI != "Ruelles" || fr.ModeUI != "Assassin" || en.MapUI != "Streets" || en.ModeUI != "Slayer" {
		t.Errorf("fr = %q / %q, en = %q / %q", fr.MapUI, fr.ModeUI, en.MapUI, en.ModeUI)
	}
}

// Synthèse : classements par carte et par mode regroupés et départagés par la clé de la langue par
// défaut ; nom affiché dans la langue de la requête.
func TestBuildBreakdownsFromCanonical_CleStableLibelleParLangue(t *testing.T) {
	ref := func(kind, def string, labels map[string]string) *canonical.AssetReference {
		return &canonical.AssetReference{Kind: kind, DefaultLabel: def, Labels: labels}
	}
	row := func(id string, mapRef, modeRef *canonical.AssetReference) canonical.PlayerMatchRow {
		var r canonical.PlayerMatchRow
		r.Summary.MatchID = id
		r.Summary.Map = mapRef
		r.Summary.PairMode = modeRef
		r.Self.Outcome = canonical.OutcomeWin
		return r
	}
	streets := ref("map", "Streets", map[string]string{"fr": "Ruelles", "en": "Streets"})
	lock := ref("map", "Lock", map[string]string{"fr": "Écluse", "en": "Lock"})
	slayer := ref("pair", "Slayer", map[string]string{"fr": "Assassin", "en": "Slayer"})
	rows := []canonical.PlayerMatchRow{row("1", streets, slayer), row("2", lock, slayer)}

	fr := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{}, i18n.Default)
	en := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{}, i18n.Fallback)
	// Mêmes comptes, même ordre : départage par la clé, octets de Ruelles < octets de Écluse, dans les deux langues.
	if len(fr.TopMaps) != 2 || len(en.TopMaps) != 2 {
		t.Fatalf("deux cartes attendues, fr=%d en=%d", len(fr.TopMaps), len(en.TopMaps))
	}
	if fr.TopMaps[0].MapName != "Ruelles" || fr.TopMaps[1].MapName != "Écluse" {
		t.Errorf("fr : %q, %q", fr.TopMaps[0].MapName, fr.TopMaps[1].MapName)
	}
	if en.TopMaps[0].MapName != "Streets" || en.TopMaps[1].MapName != "Lock" {
		t.Errorf("en : l'ordre est celui de la clé (Ruelles, Écluse), les noms sont anglais : %q, %q",
			en.TopMaps[0].MapName, en.TopMaps[1].MapName)
	}
	if fr.TopModes[0].ModeName != "Assassin" || en.TopModes[0].ModeName != "Slayer" || fr.TopModes[0].MatchCount != 2 || en.TopModes[0].MatchCount != 2 {
		t.Errorf("modes : fr %+v, en %+v", fr.TopModes, en.TopModes)
	}
}

// Session : l'étiquette de la portée porte le nom de la carte dans la langue de la requête.
func TestSessionRangeScope_NomDeCarteSuitLaLangue(t *testing.T) {
	m := []legacymatch.StatsMatchRow{{MatchID: "m", MapName: i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"}}}
	if got := sessionRangeScope(m, i18n.Default)[0].MapName; got != "Ruelles" {
		t.Errorf("fr : %q", got)
	}
	if got := sessionRangeScope(m, i18n.Fallback)[0].MapName; got != "Streets" {
		t.Errorf("en : %q", got)
	}
}

// Séries temporelles : le nom de carte (map_name) et la sélection (playlist_name) d'une ligne suivent la
// langue de la requête ; en anglais ils restent ceux du registre, en français ils prennent le nom français.
func TestBuildMatchRows_NomsSuiventLaLangue(t *testing.T) {
	m := []legacymatch.StatsMatchRow{{
		MatchID:      "m",
		MapName:      i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"},
		PlaylistName: i18n.Text{i18n.Default: "Partie rapide", i18n.Fallback: "Quick Play"},
	}}
	fr := buildMatchRows(i18n.Default, m, false, nil, nil)[0]
	en := buildMatchRows(i18n.Fallback, m, false, nil, nil)[0]
	if fr.MapName != "Ruelles" || fr.PlaylistName != "Partie rapide" {
		t.Errorf("fr : carte %q, sélection %q", fr.MapName, fr.PlaylistName)
	}
	if en.MapName != "Streets" || en.PlaylistName != "Quick Play" {
		t.Errorf("en : carte %q, sélection %q", en.MapName, en.PlaylistName)
	}
	// Une langue sans nom retombe sur la langue de repli (D3).
	only := []legacymatch.StatsMatchRow{{MatchID: "m", MapName: i18n.Text{i18n.Fallback: "Streets"}}}
	if got := buildMatchRows(i18n.Default, only, false, nil, nil)[0].MapName; got != "Streets" {
		t.Errorf("repli : %q", got)
	}
}
