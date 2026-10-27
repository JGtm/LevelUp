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
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/progression/coach_advisor"
)

// Tests du contrat i18n de l'étape 7 (ADR 0040, plan i18n 7.9) : les textes stockés une fois par langue
// (`title_fr`, `label_en`...) sortent en UN champ dans la langue de la requête, y compris pour les lignes
// déjà stockées ; aucune paire par langue n'atteint le web.

func objetJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("JSON illisible : %v : %s", err, raw)
	}
	return m
}

func TestContratLangue_ParamsDeNotification(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.items = []notifications.Notification{
		// Ligne stockée avant l'étape 7 : le format n'a pas changé, la lecture le résout.
		{ID: 2, Category: notifications.CategoryMilestoneUnlocked, TitleKey: "notif.milestone_unlocked.title",
			Params: json.RawMessage(`{"milestone_id":"m1","title_en":"Sharpshooter","title_fr":"Tireur d'élite","threshold":50}`),
			Source: "test", CreatedAt: time.Now()},
		{ID: 1, Category: notifications.CategoryMedalFirstEarned, TitleKey: "notif.medal_first_earned.title",
			Params: json.RawMessage(`{"medal_name_fr":"Tir parfait","medal_name_en":"Perfect"}`),
			Source: "test", CreatedAt: time.Now()},
	}
	svc := notifications.NewService(repo)
	h := routeurLangue(func(r chi.Router) {
		r.Route("/players/{player_slug}", func(sub chi.Router) {
			handlers.NewNotificationsHandler(makeFactory(svc, "p1")).Mount(sub)
		})
	})

	cas := []struct{ locale, titre, medaille string }{
		{"fr", "Tireur d'élite", "Tir parfait"},
		{"en", "Sharpshooter", "Perfect"},
		{"zz", "Tireur d'élite", "Tir parfait"}, // locale inconnue : défaut fr
		{"", "Tireur d'élite", "Tir parfait"},   // en-tête absent : défaut fr
	}
	for _, c := range cas {
		body := getLangue(t, h, "/players/p1/notifications", c.locale).(map[string]any)
		items := body["items"].([]any)
		jalon := items[0].(map[string]any)["params"].(map[string]any)
		medaille := items[1].(map[string]any)["params"].(map[string]any)
		if jalon["title"] != c.titre || medaille["medal_name"] != c.medaille {
			t.Errorf("locale %q : title=%v medal_name=%v, attendu %q / %q", c.locale, jalon["title"], medaille["medal_name"], c.titre, c.medaille)
		}
		for _, params := range []map[string]any{jalon, medaille} {
			for k := range params {
				if strings.HasSuffix(k, "_fr") || strings.HasSuffix(k, "_en") {
					t.Errorf("locale %q : la clé par langue %q ne doit pas sortir : %v", c.locale, k, params)
				}
			}
		}
		if jalon["milestone_id"] != "m1" || jalon["threshold"] != 50.0 {
			t.Errorf("locale %q : paramètres non localisés altérés : %v", c.locale, jalon)
		}
	}
}

func TestContratLangue_ReasonParamsDesPropositionsDuCoach(t *testing.T) {
	svc := newFakeAdvisor()
	defi := mkPendingProp("p1")
	defi.ReasonParams = `{"metric":"kda","label_en":"Improve KDA over a session","label_fr":"Améliore le KDA sur une session"}`
	arc := mkPendingProp("p2")
	arc.Kind = coach_advisor.ProposalKindArc
	arc.ReasonParams = `{"radar_axis":"combat","title_en":"Combat Mastery","title_fr":"Domination en combat",` +
		`"description_en":"A 3-step arc","description_fr":"Un arc en 3 étapes"}`
	svc.proposals["p1"], svc.proposals["p2"] = defi, arc
	resolver := func(context.Context, string) (coach_advisor.Service, string, error) { return svc, "u1", nil }
	h := routeurLangue(func(r chi.Router) {
		r.Route("/players/{player_slug}", func(sub chi.Router) {
			handlers.NewCoachProposalsHandler(resolver, "halo_infinite").Mount(sub)
		})
	})

	lire := func(locale string) map[string]map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/players/jgtm/coach/proposals", nil)
		if locale != "" {
			req.Header.Set("X-LevelUp-Locale", locale)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET (%q) : status %d : %s", locale, rec.Code, rec.Body.String())
		}
		var body struct {
			Items []struct {
				ID           string `json:"id"`
				ReasonParams string `json:"reason_params"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode : %v", err)
		}
		out := map[string]map[string]any{}
		for _, it := range body.Items {
			out[it.ID] = objetJSON(t, it.ReasonParams)
		}
		return out
	}

	fr, en, inconnue := lire("fr"), lire("en"), lire("zz")
	if fr["p1"]["label"] != "Améliore le KDA sur une session" || en["p1"]["label"] != "Improve KDA over a session" {
		t.Errorf("label du défi : fr=%v en=%v", fr["p1"]["label"], en["p1"]["label"])
	}
	if fr["p2"]["title"] != "Domination en combat" || en["p2"]["title"] != "Combat Mastery" ||
		en["p2"]["description"] != "A 3-step arc" {
		t.Errorf("titre et description de l'arc : fr=%v en=%v", fr["p2"], en["p2"])
	}
	if inconnue["p1"]["label"] != fr["p1"]["label"] {
		t.Errorf("locale inconnue : défaut fr attendu, obtenu %v", inconnue["p1"]["label"])
	}
	for _, m := range []map[string]any{fr["p1"], en["p1"], fr["p2"], en["p2"]} {
		for k := range m {
			if strings.HasSuffix(k, "_fr") || strings.HasSuffix(k, "_en") {
				t.Errorf("la clé par langue %q ne doit pas sortir : %v", k, m)
			}
		}
	}
	if fr["p1"]["metric"] != "kda" {
		t.Errorf("les paramètres non localisés doivent rester : %v", fr["p1"])
	}
}
