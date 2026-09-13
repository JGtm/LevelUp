// Package handlers_test — timeseries_weapon_range_test.go : LA SECTION « PORTÉE DES
// ENGAGEMENTS » AU TRAVERS DU HANDLER de la page Séries temporelles.
//
// CES TESTS VENAIENT DE synthesis_handler_test.go : la section a quitté la Synthèse pour
// l'onglet Résumé des Séries temporelles le 2026-09-13. Ni la forme JSON ni le contrat de
// dégradation n'ont changé — seule la page qui la porte.
//
// CE QU'ILS VÉRIFIENT ET QU'AUCUN TEST DE SERVICE NE PEUT VOIR : la forme JSON servie au
// front. Un côté d'arme est un POINTEUR — nil doit disparaître de la charge utile (« aucune
// mesure »), jamais s'y écrire en zéro (« mesuré, à zéro mètre »).
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// postTimeseriesPage joue un POST /pages/timeseries servi par une réponse figée.
func postTimeseriesPage(t *testing.T, page domain.TimeseriesPageResponse) *httptest.ResponseRecorder {
	t.Helper()
	mock := &mockTimeseriesService{page: page}
	r := newTimeseriesRouter(func(_ context.Context, _ string) (port.TimeseriesService, error) {
		return mock, nil
	})
	body, _ := json.Marshal(domain.TimeseriesQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/timeseries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestTimeseriesHandler_WeaponRangeInResponse — la section « Portée par arme » traverse le
// handler et se sérialise telle quelle (lot 4 du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md).
//
// CE QUE CE TEST VÉRIFIE ET QU'AUCUN TEST DE SERVICE NE PEUT VOIR : la forme JSON servie au
// front. Un côté d'arme est un POINTEUR — nil doit disparaître de la charge utile (« aucune
// mesure »), jamais s'y écrire en zéro (« mesuré, à zéro mètre »), et le bloc d'entame obéit à
// la même règle (D5).
func TestTimeseriesHandler_WeaponRangeInResponse(t *testing.T) {
	page := domain.TimeseriesPageResponse{
		TotalMatches: 3,
		WeaponRange: &domain.SynthesisWeaponRange{
			Weapons: []domain.WeaponRangeRow{
				{
					WeaponKey: "hinf_shotgun", Label: "Fusil à pompe",
					Deaths: &domain.WeaponRangeSide{
						Measured: 9, P10: 1, Median: 3, P90: 6,
						AbovePct: 10, LevelPct: 60, BelowPct: 30,
					},
				},
				{
					WeaponKey: "hinf_br75", Label: "Fusil de combat BR75",
					Kills: &domain.WeaponRangeSide{
						Measured: 12, P10: 7, Median: 13, P90: 25,
						AbovePct: 37, LevelPct: 49, BelowPct: 14,
					},
				},
			},
			MedianKillsM: 13, MedianDeathsM: 3,
			MeasuredKills: 12, TotalKills: 20,
			MeasuredDeaths: 9, TotalDeaths: 15,
			BelowThresholdKills: []domain.WeaponBelowThreshold{
				{WeaponKey: "hinf_hydra", Label: "Hydra", Measured: 6},
			},
			Opening: &domain.SynthesisOpening{
				MedianM: 19, MeasuredKills: 8,
				Delta: &domain.SynthesisOpeningDelta{MedianM: -6, ClosingSharePct: 75, N: 8},
			},
		},
	}
	w := postTimeseriesPage(t, page)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var got domain.TimeseriesPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.WeaponRange == nil {
		t.Fatal("weapon_range absent de la réponse")
	}
	if len(got.WeaponRange.Weapons) != 2 {
		t.Fatalf("%d arme(s), attendu 2", len(got.WeaponRange.Weapons))
	}
	if got.WeaponRange.Weapons[0].Kills != nil {
		t.Errorf("le fusil à pompe ne porte aucun frag : kills = %+v, attendu absent",
			got.WeaponRange.Weapons[0].Kills)
	}
	if got.WeaponRange.Weapons[0].Deaths == nil || got.WeaponRange.Weapons[0].Deaths.Measured != 9 {
		t.Errorf("côté morts perdu : %+v", got.WeaponRange.Weapons[0].Deaths)
	}
	if got.WeaponRange.MeasuredKills != 12 || got.WeaponRange.TotalKills != 20 {
		t.Errorf("couverture = %d/%d, attendu 12/20",
			got.WeaponRange.MeasuredKills, got.WeaponRange.TotalKills)
	}
	if got.WeaponRange.Opening == nil || got.WeaponRange.Opening.Delta == nil ||
		got.WeaponRange.Opening.Delta.N != 8 {
		t.Errorf("bloc d'entame = %+v, attendu 8 frags appariés", got.WeaponRange.Opening)
	}
	// La forme brute, parce que c'est elle que le front lit : un côté absent ne doit PAS
	// apparaître, et un côté présent porte bien ses trois parts de dénivelé.
	brut := w.Body.String()
	if strings.Contains(brut, `"kills":null`) || strings.Contains(brut, `"opening":null`) {
		t.Errorf("un champ optionnel est sérialisé à null au lieu d'être omis :\n%s", brut)
	}
	if !strings.Contains(brut, `"above_pct":37`) {
		t.Errorf("la ventilation du dénivelé n'est pas dans la charge utile :\n%s", brut)
	}
}

// TestTimeseriesHandler_SansWeaponRange_ReponseValide — la dégradation. Un titre sans positions
// par kill (ou un scope non décodé) rend une réponse SANS la clé `weapon_range` : le front la
// lit comme « pas de section », jamais comme une section vide.
func TestTimeseriesHandler_SansWeaponRange_ReponseValide(t *testing.T) {
	page := domain.TimeseriesPageResponse{
		TotalMatches: 3,
	}
	w := postTimeseriesPage(t, page)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "weapon_range") {
		t.Errorf("la clé weapon_range est présente alors que la section est absente :\n%s", w.Body.String())
	}
	var got domain.TimeseriesPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.WeaponRange != nil {
		t.Errorf("weapon_range = %+v, attendu nil", got.WeaponRange)
	}
}

// TestTimeseriesHandler_WeaponRangeToutSousLeSeuil — la section reste servie quand AUCUNE arme
// ne passe le seuil de publication (constat F3, revue adversariale du lot 4 ; décision du
// pilote, 2026-09-06).
//
// CE QUE SEUL UN TEST HTTP PEUT VOIR : la liste vide se sérialise `"weapons":[]` et pas
// `"weapons":null`. Le front itère sans garde ; un `null` casserait la carte alors que la
// donnée existe. Les deux médianes et les listes nommées restent dans la charge utile — c'est
// tout ce que la section a à dire dans ce cas, et ce n'est pas rien.
func TestTimeseriesHandler_WeaponRangeToutSousLeSeuil(t *testing.T) {
	page := domain.TimeseriesPageResponse{
		TotalMatches: 3,
		WeaponRange: &domain.SynthesisWeaponRange{
			Weapons:      []domain.WeaponRangeRow{},
			MedianKillsM: 11.5, MedianDeathsM: 8,
			MeasuredKills: 7, TotalKills: 20,
			MeasuredDeaths: 2, TotalDeaths: 9,
			BelowThresholdKills: []domain.WeaponBelowThreshold{
				{WeaponKey: "hinf_hydra", Label: "Hydra", Measured: 4},
			},
		},
	}
	w := postTimeseriesPage(t, page)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	brut := w.Body.String()
	if !strings.Contains(brut, `"weapons":[]`) {
		t.Errorf("la liste d'armes vide doit se sérialiser `\"weapons\":[]` (jamais null) :\n%s", brut)
	}
	if !strings.Contains(brut, `"median_kills_m":11.5`) {
		t.Errorf("la médiane globale doit rester servie — c'est elle qui porte l'information "+
			"quand aucune arme n'est publiable :\n%s", brut)
	}
	var got domain.TimeseriesPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.WeaponRange == nil {
		t.Fatal("weapon_range absent alors que la section est publiée sans arme")
	}
	if len(got.WeaponRange.BelowThresholdKills) != 1 {
		t.Errorf("les armes écartées doivent être nommées : %+v", got.WeaponRange.BelowThresholdKills)
	}
}

// TestTimeseriesHandler_OpeningSansDelta — le sous-bloc `delta` est OMIS de la charge utile
// quand aucun frag ne porte les deux mesures (constat F9, revue adversariale du lot 4,
// 2026-09-06).
//
// C'est la même doctrine que D5, un cran plus bas : trois champs requis publiés à zéro se
// liraient « l'engagement ne se ferme jamais », alors que la vérité est « on ne sait pas ».
func TestTimeseriesHandler_OpeningSansDelta(t *testing.T) {
	page := domain.TimeseriesPageResponse{
		TotalMatches: 3,
		WeaponRange: &domain.SynthesisWeaponRange{
			Weapons:      []domain.WeaponRangeRow{},
			MedianKillsM: 13, MedianDeathsM: 9,
			Opening: &domain.SynthesisOpening{MedianM: 19, MeasuredKills: 8},
		},
	}
	w := postTimeseriesPage(t, page)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	brut := w.Body.String()
	if strings.Contains(brut, `"delta"`) {
		t.Errorf("le sous-bloc delta doit être ABSENT quand rien n'est apparié :\n%s", brut)
	}
	if strings.Contains(brut, `"closing_share_pct"`) {
		t.Errorf("aucun champ de delta ne doit atteindre le front : un zéro s'y lirait comme "+
			"une mesure :\n%s", brut)
	}
	if !strings.Contains(brut, `"measured_kills":8`) {
		t.Errorf("la couverture de l'entame reste publiée, c'est un fait mesuré :\n%s", brut)
	}
}
