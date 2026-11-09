package service

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// Preuve de l'étape 5 du plan i18n (lot B) : les champs de données internes passent à i18n.Text,
// mais le JSON de l'API reste identique OCTET POUR OCTET. Chaque test fige les noms de clés et les
// valeurs `_fr` / `_en` d'une réponse touchée. Le point délicat est le repli : un libellé sans
// traduction française doit rester une chaîne VIDE sous `_fr` (l'index direct du Text), jamais le
// texte anglais que Pick rendrait.

func jsonObject(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal : %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal : %v", err)
	}
	return out
}

func wantJSON(t *testing.T, obj map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		if got, ok := obj[k]; !ok || got != v {
			t.Errorf("clé %q : %v (présente=%v), attendu %v", k, got, ok, v)
		}
	}
}

// Succès d'Xbox : `name` / `description` / `locked_desc` dans la langue demandée (ADR 0040 D-3), plus de
// champ suffixé. Un nom sans traduction se replie sur l'anglais (D-4).
func TestContratLangue_Succes(t *testing.T) {
	def := domain.AchievementDefinitionRow{
		AchievementID: "a1",
		Name:          i18n.FromColumns("First Blood", "Premier sang"),
		Description:   i18n.FromColumns("Get a kill", "Faire un frag"),
		LockedDesc:    i18n.FromColumns("Keep going", "Continue"),
	}
	wantJSON(t, jsonObject(t, buildAchievementEntry(def, domain.PlayerAchievementRow{}, i18n.LocaleFR)), map[string]any{
		"achievement_id": "a1", "name": "Premier sang", "description": "Faire un frag", "locked_desc": "Continue",
	})
	enObj := jsonObject(t, buildAchievementEntry(def, domain.PlayerAchievementRow{}, i18n.LocaleEN))
	wantJSON(t, enObj, map[string]any{"name": "First Blood", "description": "Get a kill", "locked_desc": "Keep going"})
	for _, k := range []string{"name_en", "name_fr", "description_en", "description_fr", "locked_desc_en", "locked_desc_fr"} {
		if _, ok := enObj[k]; ok {
			t.Errorf("%s ne fait plus partie du contrat : %v", k, enObj)
		}
	}

	// Sans traduction française : `name` se replie sur l'anglais ; `locked_desc` (omitempty) est omis quand vide.
	enOnly := domain.AchievementDefinitionRow{AchievementID: "a2", Name: i18n.FromColumns("Only EN", "")}
	obj := jsonObject(t, buildAchievementEntry(enOnly, domain.PlayerAchievementRow{}, i18n.LocaleFR))
	wantJSON(t, obj, map[string]any{"name": "Only EN"})
	if _, ok := obj["locked_desc"]; ok {
		t.Errorf("locked_desc est omitempty : absent quand vide, présent ici")
	}
}

// Page Carrière : `max_rank_name` (omitempty), le nom du rang sommet dans la langue demandée ; la
// résolution passe par un Text depuis la source (Halo 5) ou le catalogue de rangs du titre.
func TestJSONContrat_RangMaximalDeLaCarriere(t *testing.T) {
	s := &CareerService{}
	rank := &domain.CareerRankData{MaxRankName: i18n.FromColumns("SR152", "SR 152")}
	names := s.resolveMaxRankName(rank)
	wantJSON(t, jsonObject(t, domain.HeroProgress{MaxRankName: names.PickOptional(i18n.LocaleFR)}),
		map[string]any{"max_rank_name": "SR 152"})
	wantJSON(t, jsonObject(t, domain.HeroProgress{MaxRankName: names.PickOptional(i18n.LocaleEN)}),
		map[string]any{"max_rank_name": "SR152"})

	// Source sans nom anglais : l'anglais reprend le français (règle historique de la résolution).
	frOnly := s.resolveMaxRankName(&domain.CareerRankData{MaxRankName: i18n.FromColumns("", "Héros")})
	wantJSON(t, jsonObject(t, domain.HeroProgress{MaxRankName: frOnly.PickOptional(i18n.LocaleEN)}),
		map[string]any{"max_rank_name": "Héros"})

	// Aucun nom : la clé disparaît (omitempty).
	none := s.resolveMaxRankName(&domain.CareerRankData{})
	obj := jsonObject(t, domain.HeroProgress{MaxRankName: none.PickOptional(i18n.LocaleFR)})
	if _, ok := obj["max_rank_name"]; ok {
		t.Errorf("max_rank_name doit être absent sans nom : %v", obj)
	}
}

// Vue du match : les badges d'impact (clé `label`) viennent désormais d'un Text.
func TestJSONInchange_BadgesDImpactDeLaVueDuMatch(t *testing.T) {
	badges := convertImpactBadgesToDomain([]analysis.ImpactBadge{
		{BadgeKey: "first_blood", Badge: i18n.Text{i18n.LocaleFR: "Premier sang"}, PlayerXUID: "xuid(1)", TimeMS: 1500},
	}, i18n.LocaleFR)
	if len(badges) != 1 {
		t.Fatalf("1 badge attendu, %d", len(badges))
	}
	wantJSON(t, jsonObject(t, badges[0]), map[string]any{
		"key": "first_blood", "label": "Premier sang", "player_xuid": "xuid(1)", "time_ms": float64(1500),
	})
}

// Vue du match : le bandeau résout carte, mode et playlist depuis les Text de MatchMetaRaw ;
// la chaîne servie est celle de la langue demandée, sans repli inattendu.
func TestJSONInchange_EnteteDeLaVueDuMatch(t *testing.T) {
	raw, pair, playlist := "map-raw", "Arena:Slayer", "Ranked"
	meta := &domain.MatchMetaRaw{
		MapName: &raw, PairName: &pair, PlaylistName: &playlist,
		MapNames:      i18n.FromColumns("Aquarius", "Verseau"),
		ModeNames:     i18n.FromColumns("", "Assassin"),
		PlaylistNames: i18n.FromColumns("", "Classée"),
		PairNames:     i18n.FromColumns("", "Assassin"),
	}
	var fr, en domain.MatchViewHeader
	applyMatchHeaderMetaLabels(&fr, meta, "fr")
	applyMatchHeaderMetaLabels(&en, meta, "en")
	wantJSON(t, jsonObject(t, map[string]string{"map": fr.MapUI, "mode": fr.ModeUI, "playlist": fr.PlaylistLabel}),
		map[string]any{"map": "Verseau", "mode": "Assassin", "playlist": "Classée"})
	wantJSON(t, jsonObject(t, map[string]string{"map": en.MapUI, "mode": en.ModeUI, "playlist": en.PlaylistLabel}),
		map[string]any{"map": "Aquarius", "mode": "Slayer", "playlist": "Ranked"})
}
