// Package wire — post_sync_deltas_bespoke_test.go : tests du nom de rang carrière des notifications
// career_rank (V72-27a, item 11.R). Fichier séparé de post_sync_deltas_test.go (verrouillé côté
// chantier parallèle) — réutilise ses helpers `recordingEmitter`/`hasCategory` (même package `wire`).
//
// Le nom du rang est écrit une fois par langue (rank_name_fr, rank_name_en...) : la lecture le résout dans la
// langue de la requête (notifications.LocalizeParams).
package wire

import (
	"context"
	"encoding/json"
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/notifications"
)

// withRankLabelResolver câble un résolveur pour la durée du test et le
// désarme via t.Cleanup — évite qu'un résolveur de test fuite vers un autre
// test du package (var globale boot-only).
func withRankLabelResolver(t *testing.T, fn RankLabelResolver) {
	t.Helper()
	SetRankLabelResolver(fn)
	t.Cleanup(func() { SetRankLabelResolver(nil) })
}

func emitRank(t *testing.T, slug string, rank int, name string) map[string]any {
	t.Helper()
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: rank - 1, CurrentRankName: name}
	after := &PlayerSnapshot{CurrentRank: rank, CurrentRankName: name}
	emitCareerRankDelta(context.Background(), em, slug, "p1", before, after)
	if !hasCategory(em.emitted, notifications.CategoryCareerRank) {
		t.Fatal("career_rank aurait dû être émis")
	}
	return em.emitted[0].Params
}

// TestEmitCareerRankDelta_NomDuRangUneCleParLangue : quand le résolveur est câblé et connaît (titleSlug,
// rankID), chaque langue de l'application porte son libellé (pas CurrentRankName, baké EN depuis
// career_ranks.title_en, cf. career.go), le chiffre final en romain. Aucune clé `rank_name` seule.
func TestEmitCareerRankDelta_NomDuRangUneCleParLangue(t *testing.T) {
	withRankLabelResolver(t, func(titleSlug string, rankID int, loc i18n.Locale) (string, bool) {
		if titleSlug != "halo_infinite" || rankID != 192 {
			return "", false
		}
		if loc == i18n.LocaleFR {
			return "Colonel Platine 2", true
		}
		return "Colonel Platinum 2", true
	})
	params := emitRank(t, "halo_infinite", 192, "Colonel Platinum 2")

	for _, l := range i18n.Locales() {
		want := "Colonel Platinum II"
		if l == i18n.LocaleFR {
			want = "Colonel Platine II"
		}
		if got := params[i18n.ParamKey(notifications.ParamRankName, l)]; got != want {
			t.Errorf("rank_name_%s = %v, want %q", l, got, want)
		}
	}
	if _, ok := params["rank_name"]; ok {
		t.Errorf("la clé rank_name seule ne doit plus être émise : %v", params)
	}
	if params["rank"] != 192 || params["previous"] != 191 {
		t.Errorf("rank / previous inattendus : %v", params)
	}
}

// TestEmitCareerRankDelta_NomDuRangLuDansLaLangueDeLaRequete : de l'émission à la lecture, le nom du rang sort
// dans la langue de la requête, sans clé par langue.
func TestEmitCareerRankDelta_NomDuRangLuDansLaLangueDeLaRequete(t *testing.T) {
	withRankLabelResolver(t, func(_ string, _ int, loc i18n.Locale) (string, bool) {
		if loc == i18n.LocaleFR {
			return "Colonel Platine 2", true
		}
		return "Colonel Platinum 2", true
	})
	raw, err := json.Marshal(emitRank(t, "halo_infinite", 192, "Colonel Platinum 2"))
	if err != nil {
		t.Fatal(err)
	}
	for l, want := range map[i18n.Locale]string{i18n.LocaleFR: "Colonel Platine II", i18n.LocaleEN: "Colonel Platinum II"} {
		items := []notifications.Notification{{Params: append(json.RawMessage(nil), raw...)}}
		notifications.LocalizeParams(context.Background(), items, l)
		var got map[string]any
		if err := json.Unmarshal(items[0].Params, &got); err != nil {
			t.Fatal(err)
		}
		if got["rank_name"] != want {
			t.Errorf("langue %s : rank_name = %v, want %q", l, got["rank_name"], want)
		}
		if _, ok := got["rank_name_fr"]; ok {
			t.Errorf("langue %s : la clé par langue ne doit pas sortir : %v", l, got)
		}
	}
}

// TestEmitCareerRankDelta_RepliSansResolveur : résolveur nil (dégradation gracieuse) → toutes les langues
// retombent sur CurrentRankName (EN existant).
func TestEmitCareerRankDelta_RepliSansResolveur(t *testing.T) {
	withRankLabelResolver(t, nil)
	params := emitRank(t, "halo_infinite", 192, "Colonel Platinum 2")
	for _, l := range i18n.Locales() {
		if got := params[i18n.ParamKey(notifications.ParamRankName, l)]; got != "Colonel Platinum II" {
			t.Errorf("rank_name_%s = %v, want %q (repli EN + romain)", l, got, "Colonel Platinum II")
		}
	}
}

// TestEmitCareerRankDelta_RepliSurRangInconnu : résolveur câblé mais qui ne connaît PAS ce (titleSlug, rankID)
// (autre titre, rank_id absent du catalogue) → repli EN, jamais de chaîne vide.
func TestEmitCareerRankDelta_RepliSurRangInconnu(t *testing.T) {
	withRankLabelResolver(t, func(string, int, i18n.Locale) (string, bool) { return "", false })
	params := emitRank(t, "halo_5", 2, "Private")
	for _, l := range i18n.Locales() {
		if got := params[i18n.ParamKey(notifications.ParamRankName, l)]; got != "Private" {
			t.Errorf("rank_name_%s = %v, want %q (repli EN, résolveur sans correspondance)", l, got, "Private")
		}
	}
}
