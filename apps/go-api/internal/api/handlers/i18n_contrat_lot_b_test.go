package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/coach_advisor"
	"levelup/go-api/internal/service"
)

// Tests du contrat i18n du lot B (ADR 0040 D-3, plan i18n étape 6, 6.6) : le MÊME endpoint, appelé avec
// `X-LevelUp-Locale: fr` puis `en`, rend des libellés différents dans UN champ (`name`, `label`...) ;
// une locale inconnue ou absente rend la langue par défaut (`fr`) ; aucun champ `*_fr` / `*_en` ne
// subsiste. Le routeur passe par le vrai middleware de titre et de locale (`middleware.TitleExtractor`),
// donc l'en-tête est lu comme en production.

// routeurLangue monte un handler derrière le middleware qui lit X-LevelUp-Locale.
func routeurLangue(mount func(r chi.Router)) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.TitleExtractor(titlePkg.NewRegistry()))
	mount(r)
	return r
}

// getLangue émet un GET avec la locale donnée (vide : aucun en-tête) et rend le corps JSON décodé.
func getLangue(t *testing.T, h http.Handler, path, locale string) any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if locale != "" {
		req.Header.Set("X-LevelUp-Locale", locale)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s (%q) : status %d : %s", path, locale, rec.Code, rec.Body.String())
	}
	var out any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s : %v", path, err)
	}
	return out
}

// champ lit une clé d'un objet JSON décodé.
func champ(t *testing.T, obj any, cle string) any {
	t.Helper()
	m, ok := obj.(map[string]any)
	if !ok {
		t.Fatalf("objet JSON attendu, reçu %T", obj)
	}
	return m[cle]
}

// aucunChampSuffixe échoue si l'objet porte l'un des champs de langue listés.
func aucunChampSuffixe(t *testing.T, obj any, interdits ...string) {
	t.Helper()
	m, ok := obj.(map[string]any)
	if !ok {
		t.Fatalf("objet JSON attendu, reçu %T", obj)
	}
	for _, k := range interdits {
		if _, present := m[k]; present {
			t.Errorf("le champ %q ne fait plus partie du contrat : %v", k, m)
		}
	}
}

// ─── Succès Xbox ─────────────────────────────────────────────────────────────

type fauxSuccesJoueur struct{}

func (fauxSuccesJoueur) GetPlayerAchievements(context.Context) ([]domain.PlayerAchievementRow, error) {
	return nil, nil
}

type fauxSuccesDefinitions struct{}

func (fauxSuccesDefinitions) GetAchievementDefinitions(context.Context, string) ([]domain.AchievementDefinitionRow, error) {
	return []domain.AchievementDefinitionRow{{
		AchievementID: "a1",
		Name:          i18n.FromColumns("First Blood", "Premier sang"),
		Description:   i18n.FromColumns("Get a kill", "Faire un frag"),
		LockedDesc:    i18n.FromColumns("Keep going", "Continue"),
		Gamerscore:    10,
	}}, nil
}

func TestContratLangue_PageSucces(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.AchievementsService, error) {
		return service.NewAchievementsService(fauxSuccesJoueur{}, fauxSuccesDefinitions{}), nil
	}
	h := routeurLangue(func(r chi.Router) {
		r.Route("/players/{player_slug}", func(r chi.Router) { handlers.NewAchievementsHandler(factory).Mount(r) })
	})

	cas := []struct{ locale, name, desc, locked string }{
		{"fr", "Premier sang", "Faire un frag", "Continue"},
		{"en", "First Blood", "Get a kill", "Keep going"},
		{"zz", "Premier sang", "Faire un frag", "Continue"}, // locale inconnue : défaut fr
		{"", "Premier sang", "Faire un frag", "Continue"},   // en-tête absent : défaut fr
	}
	for _, c := range cas {
		body := getLangue(t, h, "/players/p/pages/achievements", c.locale)
		list := champ(t, body, "achievements").([]any)
		if len(list) != 1 {
			t.Fatalf("locale %q : %d succès", c.locale, len(list))
		}
		a := list[0]
		if champ(t, a, "name") != c.name || champ(t, a, "description") != c.desc || champ(t, a, "locked_desc") != c.locked {
			t.Errorf("locale %q : %v", c.locale, a)
		}
		aucunChampSuffixe(t, a, "name_en", "name_fr", "description_en", "description_fr", "locked_desc_en", "locked_desc_fr")
	}
}

// ─── Tiroir d'actifs (cartes, armes, médailles) ──────────────────────────────

func TestContratLangue_TiroirDActifs(t *testing.T) {
	repo := service.NewStaticAssetMetaRepo(
		[]canonical.AssetMeta{
			{ID: "m1", Names: i18n.FromColumns("Starboard", "Tribord"), ImageURL: "/m1.jpg"},
			{ID: "m2", Names: i18n.FromColumns("Aquarius", ""), ImageURL: "/m2.jpg"}, // sans traduction fr : repli en
		},
		[]canonical.AssetMeta{{ID: "100", Names: i18n.FromColumns("BR75 Battle Rifle", "Fusil BR75")}},
	).WithFallbackMedals([]canonical.AssetMeta{{
		ID:           "9",
		Names:        i18n.FromColumns("Double Kill", "Double élimination"),
		Descriptions: i18n.FromColumns("Kill 2 enemies.", "Éliminez 2 ennemis."),
	}})
	svc := service.NewAssetService(repo)
	h := routeurLangue(func(r chi.Router) {
		handlers.NewAssetMetadataHandler(svc, func(string, titlePkg.Capability) bool { return true }).Mount(r)
	})

	nom := func(body any, i int) any { return champ(t, body.([]any)[i], "name") }

	// Cartes : l'ordre est celui du nom canonique (Aquarius avant Starboard), identique dans toutes les langues.
	fr := getLangue(t, h, "/assets/halo_infinite/maps", "fr")
	en := getLangue(t, h, "/assets/halo_infinite/maps", "en")
	if nom(fr, 0) != "Aquarius" || nom(fr, 1) != "Tribord" {
		t.Errorf("cartes fr : %v", fr)
	}
	if nom(en, 0) != "Aquarius" || nom(en, 1) != "Starboard" {
		t.Errorf("cartes en : %v", en)
	}
	if def := getLangue(t, h, "/assets/halo_infinite/maps", "zz"); nom(def, 1) != "Tribord" {
		t.Errorf("locale inconnue : défaut fr attendu : %v", def)
	}
	aucunChampSuffixe(t, fr.([]any)[0], "name_en", "name_fr", "description_fr")

	if w := getLangue(t, h, "/assets/halo_infinite/weapons", "fr"); nom(w, 0) != "Fusil BR75" {
		t.Errorf("armes fr : %v", w)
	}
	if w := getLangue(t, h, "/assets/halo_infinite/weapons", "en"); nom(w, 0) != "BR75 Battle Rifle" {
		t.Errorf("armes en : %v", w)
	}

	medFR := getLangue(t, h, "/assets/halo_infinite/medals", "fr")
	medEN := getLangue(t, h, "/assets/halo_infinite/medals", "en")
	if nom(medFR, 0) != "Double élimination" || champ(t, medFR.([]any)[0], "description") != "Éliminez 2 ennemis." {
		t.Errorf("médailles fr : %v", medFR)
	}
	if nom(medEN, 0) != "Double Kill" || champ(t, medEN.([]any)[0], "description") != "Kill 2 enemies." {
		t.Errorf("médailles en : %v", medEN)
	}
	// La recherche trouve un actif par son nom dans n'importe quelle langue, quelle que soit celle de la requête.
	if r := getLangue(t, h, "/assets/halo_infinite/maps?q=tribord", "en"); len(r.([]any)) != 1 || nom(r, 0) != "Starboard" {
		t.Errorf("recherche « tribord » en locale en : %v", r)
	}
}

// ─── Propositions du coach ───────────────────────────────────────────────────

func TestContratLangue_PropositionsDuCoach(t *testing.T) {
	svc := newFakeAdvisor()
	p := mkPendingProp("p1")
	p.ReasonKey = i18n.FromColumns("coach.proposal.accuracy.en", "coach.proposal.accuracy.fr")
	p.CreatedAt = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	svc.proposals["p1"] = p
	resolver := func(context.Context, string) (coach_advisor.Service, string, error) { return svc, "u1", nil }
	h := routeurLangue(func(r chi.Router) {
		r.Route("/players/{player_slug}", func(r chi.Router) {
			handlers.NewCoachProposalsHandler(resolver, "halo_infinite").Mount(r)
		})
	})

	for locale, attendu := range map[string]string{
		"fr": "coach.proposal.accuracy.fr",
		"en": "coach.proposal.accuracy.en",
		"zz": "coach.proposal.accuracy.fr",
		"":   "coach.proposal.accuracy.fr",
	} {
		body := getLangue(t, h, "/players/p/coach/proposals", locale)
		item := champ(t, body, "items").([]any)[0]
		if champ(t, item, "reason_key") != attendu {
			t.Errorf("locale %q : reason_key = %v, attendu %s", locale, champ(t, item, "reason_key"), attendu)
		}
		aucunChampSuffixe(t, item, "reason_key_en", "reason_key_fr")
	}
}
