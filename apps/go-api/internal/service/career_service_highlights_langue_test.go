package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// repoPoolLangue : un pool de matchs marquants et des noms de mode et de playlist par langue.
type repoPoolLangue struct {
	*mockCareerRepo
	pool  []domain.HighlightMatchPoolRow
	modes map[i18n.Locale]map[string]string
	pls   map[i18n.Locale]map[string]string
}

func (r *repoPoolLangue) GetHighlightPool(_ context.Context) ([]domain.HighlightMatchPoolRow, error) {
	return append([]domain.HighlightMatchPoolRow(nil), r.pool...), nil
}

func (r *repoPoolLangue) LoadModeTranslations(_ context.Context, loc i18n.Locale, _ []string) (map[string]string, error) {
	return r.modes[loc], nil
}

func (r *repoPoolLangue) LoadPlaylistAssetTranslations(_ context.Context, loc i18n.Locale, _ []string) (map[string]string, error) {
	return r.pls[loc], nil
}

func repoMarquants() *repoPoolLangue {
	return &repoPoolLangue{
		mockCareerRepo: &mockCareerRepo{},
		pool: []domain.HighlightMatchPoolRow{
			{MatchID: "m1", ModeUI: "Slayer", ModeUISource: "Slayer", PlaylistName: "Quick Play", PlaylistNameRaw: "Quick Play", PlaylistID: "pl1"},
			{MatchID: "m2", ModeUI: "Slayer", ModeUISource: "Slayer", PlaylistName: "Quick Play", PlaylistNameRaw: "Quick Play", PlaylistID: "pl1"},
		},
		modes: map[i18n.Locale]map[string]string{i18n.Default: {"Slayer": "Assassin"}},
		pls: map[i18n.Locale]map[string]string{
			i18n.Default:  {"pl1": "Partie rapide"},
			i18n.Fallback: {"pl1": "Quick Play"},
		},
	}
}

// Les listes de modes et de playlists des matchs marquants : la valeur (clé de `mode_uis` et de
// `playlist_names`) est le nom de la langue par défaut, le libellé celui de la langue de la requête.
func TestGetHighlightMatchIDs_ValeurStableLibelleParLangue(t *testing.T) {
	cases := []struct {
		loc                      i18n.Locale
		labelMode, labelPlaylist string
	}{
		{i18n.Default, "Assassin", "Partie rapide"},
		{i18n.Fallback, "Slayer", "Quick Play"},
	}
	for _, c := range cases {
		svc := NewCareerService(repoMarquants())
		ctx := ctxkeys.WithLocale(context.Background(), string(c.loc))
		data, err := svc.GetHighlightMatchIDs(ctx, domain.HighlightFilterInput{Experience: "all"})
		if err != nil {
			t.Fatalf("%s : %v", c.loc, err)
		}
		if len(data.AvailableModes) != 1 || data.AvailableModes[0].Value != "Assassin" ||
			data.AvailableModes[0].Label != c.labelMode || data.AvailableModes[0].Count != 2 {
			t.Errorf("%s : modes = %+v, attendu valeur Assassin, libellé %q, 2 matchs", c.loc, data.AvailableModes, c.labelMode)
		}
		if len(data.AvailablePlaylists) != 1 || data.AvailablePlaylists[0].Value != "Partie rapide" ||
			data.AvailablePlaylists[0].Label != c.labelPlaylist || data.AvailablePlaylists[0].Count != 2 {
			t.Errorf("%s : playlists = %+v, attendu valeur Partie rapide, libellé %q, 2 matchs", c.loc, data.AvailablePlaylists, c.labelPlaylist)
		}
	}
}

// La sélection du filtre (`mode_uis`, `playlist_names`) porte la valeur de la langue par défaut : elle se
// résout en valeurs brutes de la même façon quelle que soit la langue de la requête.
func TestGetHighlightMatchIDs_SelectionResoutIndependammentDeLaLangue(t *testing.T) {
	for _, loc := range []i18n.Locale{i18n.Default, i18n.Fallback} {
		repo := repoMarquants()
		svc := NewCareerService(repo)
		ctx := ctxkeys.WithLocale(context.Background(), string(loc))
		data, err := svc.GetHighlightMatchIDs(ctx, domain.HighlightFilterInput{
			Experience: "all", ModeUIs: []string{"Assassin"}, PlaylistNames: []string{"Partie rapide"},
		})
		if err != nil {
			t.Fatalf("%s : %v", loc, err)
		}
		// Avec la sélection, chaque liste reste calculée sur les autres filtres : 2 matchs dans les deux langues.
		if len(data.AvailableModes) != 1 || data.AvailableModes[0].Count != 2 {
			t.Errorf("%s : modes sous sélection = %+v", loc, data.AvailableModes)
		}
	}
}
