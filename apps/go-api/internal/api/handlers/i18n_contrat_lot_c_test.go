package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// Tests du contrat i18n du lot C (ADR 0040 D-3, plan i18n étape 6, 6.6) : armes, cartes, lieux et rangs
// sont servis dans la langue de la requête, dans UN champ (`label`, `map_label`, `nom`, `max_rank_name`...) ;
// une locale inconnue ou absente rend la langue par défaut (`fr`) ; aucun champ `*_fr` / `*_en` ne
// subsiste. Les services sont les vrais (seuls leurs dépôts sont des doublures), derrière le vrai
// middleware de locale (`routeurLangue`, lot B).

// postLangue émet un POST JSON avec la locale donnée et rend le corps décodé.
func postLangue(t *testing.T, h http.Handler, path, corps, locale string) any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(corps))
	req.Header.Set("Content-Type", "application/json")
	if locale != "" {
		req.Header.Set("X-LevelUp-Locale", locale)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST %s (%q) : status %d : %s", path, locale, rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec.Body.Bytes())
}

// ─── Page Carrière : prochain rang et rang sommet ───────────────────────────

// fauxDepotCarriere ne porte que les trois lectures de la page ; les autres méthodes du port ne sont
// jamais appelées (interface nil embarquée : un appel imprévu échoue bruyamment).
type fauxDepotCarriere struct{ port.CareerRepository }

func (fauxDepotCarriere) GetLatestRank(context.Context) (*domain.CareerRankData, error) {
	return &domain.CareerRankData{RankNumber: 100, RecordedAt: time.Now()}, nil
}
func (fauxDepotCarriere) GetXPHistory(context.Context) ([]domain.XPHistoryPoint, error) {
	return nil, nil
}
func (fauxDepotCarriere) GetLUSRHistory(context.Context) ([]domain.LUSRCheckpointDTO, error) {
	return nil, nil
}

func TestContratLangue_PageCarriere(t *testing.T) {
	catalogue := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 100, Title: map[string]string{"fr": "Colonel", "en": "Colonel"}},
		{ID: 101, Title: map[string]string{"fr": "Général", "en": "General"}},
		{ID: 272, Title: map[string]string{"fr": "Héros", "en": "Hero"}},
	})
	factory := func(context.Context, string) (port.CareerService, error) {
		return service.NewCareerService(fauxDepotCarriere{}).WithTitleSlug("halo_infinite").WithRankCatalog(catalogue), nil
	}
	h := routeurLangue(func(r chi.Router) {
		r.Route("/players/{player_slug}", func(sub chi.Router) { handlers.NewCareerHandler(factory, nil).Mount(sub) })
	})

	cas := []struct{ locale, suivant, sommet string }{
		{"fr", "Général", "Héros"},
		{"en", "General", "Hero"},
		{"zz", "Général", "Héros"}, // locale inconnue : défaut fr
		{"", "Général", "Héros"},   // en-tête absent : défaut fr
	}
	for _, c := range cas {
		body := getLangue(t, h, "/players/p/pages/career", c.locale)
		summary := champ(t, body, "summary")
		if got := champ(t, summary, "next_rank_name"); got != c.suivant {
			t.Errorf("locale %q : next_rank_name = %v, want %q", c.locale, got, c.suivant)
		}
		hero := champ(t, body, "hero_progress")
		if got := champ(t, hero, "max_rank_name"); got != c.sommet {
			t.Errorf("locale %q : max_rank_name = %v, want %q", c.locale, got, c.sommet)
		}
		aucunChampSuffixe(t, summary, "next_rank_name_fr", "next_rank_name_en")
		aucunChampSuffixe(t, hero, "max_rank_name_fr", "max_rank_name_en")
	}
}

// ─── Onglet Tactique : cartes jouées ─────────────────────────────────────────

// fauxDepotTactique ne porte que la liste des cartes jouées.
type fauxDepotTactique struct{ port.TacticalRepository }

func (fauxDepotTactique) MapsPlayed(context.Context, domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	return []domain.TacticalMapRow{{
		MapID: "streets", MapName: i18n.Text{i18n.LocaleEN: "Streets", i18n.LocaleFR: "Ruelles"},
		Matchs: 12, Victoires: 7, Defaites: 5,
	}}, nil
}

func TestContratLangue_CartesDeLOngletTactique(t *testing.T) {
	factory := func(context.Context, string) (port.TacticalService, error) {
		return service.NewTacticalService(fauxDepotTactique{}, games.CapabilityMap{}, "xuid(1)"), nil
	}
	h := routeurLangue(func(r chi.Router) {
		r.Route("/players/{player_slug}", func(sub chi.Router) {
			handlers.NewTacticalHandler(factory, tacticalReplayFactory(&mockReplayService{}, nil)).Mount(sub)
		})
	})

	// Le libellé et le nom suivent la langue ; la valeur de filtre de l'Explorateur NE LA SUIT PAS (c'est une clé).
	cas := []struct{ locale, label string }{
		{"fr", "Ruelles"},
		{"en", "Streets"},
		{"zz", "Ruelles"}, // locale inconnue : défaut fr
		{"", "Ruelles"},   // en-tête absent : défaut fr
	}
	for _, c := range cas {
		body := postLangue(t, h, "/players/p/tactical/maps", `{}`, c.locale)
		cartes, ok := champ(t, body, "cartes").([]any)
		if !ok || len(cartes) != 1 {
			t.Fatalf("locale %q : cartes = %v", c.locale, champ(t, body, "cartes"))
		}
		carte := cartes[0]
		if got := champ(t, carte, "map_label"); got != c.label {
			t.Errorf("locale %q : map_label = %v, want %q", c.locale, got, c.label)
		}
		if got := champ(t, carte, "map_filter_key"); got != "Ruelles" {
			t.Errorf("locale %q : map_filter_key = %v, want la valeur canonique Ruelles", c.locale, got)
		}
		if got := champ(t, carte, "map_name"); got != c.label {
			t.Errorf("locale %q : map_name = %v, want %q (le nom suit la langue, 8.8.f)", c.locale, got, c.label)
		}
		aucunChampSuffixe(t, carte, "map_name_fr")
	}
}

// decodeJSON décode un corps JSON en valeur générique.
func decodeJSON(t *testing.T, raw []byte) any {
	t.Helper()
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode : %v : %s", err, raw)
	}
	return out
}
