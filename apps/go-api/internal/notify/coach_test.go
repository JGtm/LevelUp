package notify

import (
	"strings"
	"testing"
)

func TestBuildCoachEmbed_BasicFR(t *testing.T) {
	embed := BuildCoachEmbed(CoachEmbedInput{
		Category: "milestone_unlocked",
		Severity: "success",
		Player:   "JGtm",
		Params:   map[string]any{"metric": "kills", "value": 100},
		Lang:     "fr",
	}, nil)

	if embed.Title != T("discord_coach_title", "fr") {
		t.Errorf("title = %q", embed.Title)
	}
	if embed.Description != "Palier débloqué" {
		t.Errorf("description = %q ; want humanized FR label", embed.Description)
	}
	if embed.Color != colorSuccess {
		t.Errorf("color = %d ; want colorSuccess", embed.Color)
	}
	// Footer title-aware (labels nil → Halo).
	if embed.Footer == nil || !strings.Contains(embed.Footer.Text, "LevelUp") {
		t.Errorf("footer = %+v", embed.Footer)
	}
	// Champs : joueur + catégorie + détails.
	var hasPlayer, hasCategory, hasDetails bool
	for _, f := range embed.Fields {
		switch f.Name {
		case T("discord_coach_player", "fr"):
			hasPlayer = f.Value == "JGtm"
		case T("discord_coach_category", "fr"):
			hasCategory = f.Value == "milestone_unlocked"
		case T("discord_coach_details", "fr"):
			hasDetails = strings.Contains(f.Value, "metric") && strings.Contains(f.Value, "value")
		}
	}
	if !hasPlayer || !hasCategory || !hasDetails {
		t.Errorf("champs manquants : player=%v category=%v details=%v (%+v)", hasPlayer, hasCategory, hasDetails, embed.Fields)
	}
}

func TestBuildCoachEmbed_EN(t *testing.T) {
	embed := BuildCoachEmbed(CoachEmbedInput{
		Category: "trend_consolidate",
		Severity: "warn",
		Lang:     "en",
	}, nil)
	if embed.Description != "Axis to consolidate" {
		t.Errorf("EN description = %q", embed.Description)
	}
	if embed.Color != colorWarning {
		t.Errorf("color = %d ; want colorWarning", embed.Color)
	}
}

func TestBuildCoachEmbed_UnknownCategoryFallback(t *testing.T) {
	embed := BuildCoachEmbed(CoachEmbedInput{
		Category: "totally_unknown_cat",
		Lang:     "fr",
	}, nil)
	// Fallback : clé brute, pas de panic.
	if embed.Description != "totally_unknown_cat" {
		t.Errorf("fallback description = %q ; want raw key", embed.Description)
	}
	// Severity vide → couleur par défaut (blurple).
	if embed.Color != colorBlurple {
		t.Errorf("default color = %d ; want colorBlurple", embed.Color)
	}
}

func TestBuildCoachEmbed_ParamsSortedAndCapped(t *testing.T) {
	params := map[string]any{}
	for _, k := range []string{"z", "a", "m", "b", "c", "d", "e", "f"} {
		params[k] = 1
	}
	embed := BuildCoachEmbed(CoachEmbedInput{
		Category: "pattern_lever",
		Params:   params,
		Lang:     "fr",
	}, nil)
	var details string
	for _, f := range embed.Fields {
		if f.Name == T("discord_coach_details", "fr") {
			details = f.Value
		}
	}
	if details == "" {
		t.Fatal("details field manquant")
	}
	// Plafonné à maxCoachParamFields lignes, triées → commence par `a`.
	lines := strings.Split(details, "\n")
	if len(lines) != maxCoachParamFields {
		t.Errorf("lignes = %d ; want %d (plafond)", len(lines), maxCoachParamFields)
	}
	if !strings.HasPrefix(lines[0], "`a`") {
		t.Errorf("première ligne = %q ; want tri par clé (a en tête)", lines[0])
	}
}

func TestBuildCoachEmbed_LinkFieldOptional(t *testing.T) {
	withLink := BuildCoachEmbed(CoachEmbedInput{Category: "personal_record", AppURL: "https://x/y", Lang: "fr"}, nil)
	var found bool
	for _, f := range withLink.Fields {
		if f.Name == T("discord_coach_link", "fr") && f.Value == "https://x/y" {
			found = true
		}
	}
	if !found {
		t.Error("champ lien attendu quand AppURL renseigné")
	}

	noLink := BuildCoachEmbed(CoachEmbedInput{Category: "personal_record", Lang: "fr"}, nil)
	for _, f := range noLink.Fields {
		if f.Name == T("discord_coach_link", "fr") {
			t.Error("champ lien inattendu quand AppURL vide")
		}
	}
}
