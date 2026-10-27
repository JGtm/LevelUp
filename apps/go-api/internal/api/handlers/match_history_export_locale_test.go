package handlers_test

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// exportLocaleService rend les noms de carte, de mode, de playlist et le mot de l'issue dans la langue du
// contexte, comme le service réel (match_history_service).
type exportLocaleService struct{ mockMatchHistoryService }

func (s *exportLocaleService) ExportCSV(ctx context.Context, _ domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error) {
	pick := func(fr, en string) *string {
		v := fr
		if i18n.FromContext(ctx) == i18n.Fallback {
			v = en
		}
		return &v
	}
	return []domain.MatchHistoryRow{{
		MatchID: "m1", StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MapUI: pick("Pic de la Foudre", "Lightning Peak"), ModeUI: pick("Capture du drapeau", "Capture the Flag"),
		PlaylistLabel: pick("Quick Play FR", "Quick Play EN"),
	}}, nil
}

func (s *exportLocaleService) OutcomeText(ctx context.Context, _ int) string {
	if i18n.FromContext(ctx) == i18n.Fallback {
		return "Win"
	}
	return "Victoire"
}

// TestMatchHistoryHandler_Export_LangueDeLaPage : le navigateur qui suit le lien de téléchargement n'envoie pas
// X-LevelUp-Locale ; la page passe sa langue en ?lang= et les colonnes carte, mode, playlist et issue sortent
// dans cette langue (rapport de revue 2026-10-27, constat 4).
func TestMatchHistoryHandler_Export_LangueDeLaPage(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return &exportLocaleService{}, testXUID1B, "Player1", nil
	}
	r := chi.NewRouter()
	h := handlers.NewMatchHistoryHandler(factory)
	r.Use(func(next http.Handler) http.Handler { // comme TitleExtractor : la langue de l'en-tête, sinon la défaut
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(ctxkeys.WithLocale(req.Context(), req.Header.Get("X-LevelUp-Locale"))))
		})
	})
	r.Route("/players/{player_slug}", func(r chi.Router) { r.Get("/pages/match-history/export", h.Export) })

	for _, tc := range []struct {
		name, query, header string
		want                []string // issue, carte, mode, playlist
	}{
		{"lang=fr", "&lang=fr", "", []string{"Victoire", "Pic de la Foudre", "Capture du drapeau", "Quick Play FR"}},
		{"lang=en", "&lang=en", "", []string{"Win", "Lightning Peak", "Capture the Flag", "Quick Play EN"}},
		{"lang=en-US normalise", "&lang=en-US", "", []string{"Win", "Lightning Peak", "Capture the Flag", "Quick Play EN"}},
		{"sans lang : langue par defaut", "", "", []string{"Victoire", "Pic de la Foudre", "Capture du drapeau", "Quick Play FR"}},
		{"lang inconnue : l'en-tete s'applique", "&lang=zz", "en", []string{"Win", "Lightning Peak", "Capture the Flag", "Quick Play EN"}},
		{"lang prime sur l'en-tete", "&lang=fr", "en", []string{"Victoire", "Pic de la Foudre", "Capture du drapeau", "Quick Play FR"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/match-history/export?token="+makeExportToken(t)+tc.query, nil)
			if tc.header != "" {
				req.Header.Set("X-LevelUp-Locale", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("attendu 200, obtenu %d : %s", w.Code, w.Body.String())
			}
			rows, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
			if err != nil || len(rows) != 2 {
				t.Fatalf("CSV illisible (%v) : %q", err, w.Body.String())
			}
			col := map[string]int{}
			for i, name := range rows[0] {
				col[name] = i
			}
			got := []string{rows[1][col["outcome"]], rows[1][col["map"]], rows[1][col["mode"]], rows[1][col["playlist"]]}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("colonnes issue/carte/mode/playlist = %v, attendu %v", got, tc.want)
					break
				}
			}
		})
	}
}
