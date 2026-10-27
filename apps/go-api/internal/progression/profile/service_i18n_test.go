package profile

import (
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/prestige"
)

// Le nom du tier suit la langue demandée (ADR 0040 D-3) : un seul champ `tier_name`, plus de
// `tier_name_fr`. Sans nom par langue, c'est le nom canonique du tier.
func TestBuildSkillRatingSnapshot_TierNameParLangue(t *testing.T) {
	lusr := LUSRState{Mu: 1500, Sigma: 100}
	tier := TierState{Name: "Diamond", Names: i18n.FromColumns("Diamond", "Diamant"), SubTier: 3, Label: "Diamond III", LowerMu: 1400, UpperMu: 1600}

	if got := buildSkillRatingSnapshot(lusr, tier, TierState{}, i18n.LocaleFR).TierName; got != "Diamant" {
		t.Errorf("TierName(fr) = %q, want Diamant", got)
	}
	if got := buildSkillRatingSnapshot(lusr, tier, TierState{}, i18n.LocaleEN).TierName; got != "Diamond" {
		t.Errorf("TierName(en) = %q, want Diamond", got)
	}

	sansNoms := TierState{Name: "Onyx", Label: "Onyx", LowerMu: 2000, UpperMu: 9999}
	if got := buildSkillRatingSnapshot(lusr, sansNoms, TierState{}, i18n.LocaleFR).TierName; got != "Onyx" {
		t.Errorf("TierName sans noms par langue = %q, want le nom canonique Onyx", got)
	}
}

// Le défi suggéré porte UN libellé et UNE description, dans la langue demandée (plus de
// `label_fr` / `label_en`) ; une description absente reste vide, un libellé sans traduction
// se replie sur l'anglais.
func TestSuggestedChallengeFromTemplate_ParLangue(t *testing.T) {
	tpl := prestige.Template{
		ID:           "tpl_acc",
		Labels:       i18n.FromColumns("Accuracy 50%", "Précision 50 %"),
		Descriptions: i18n.FromColumns("Hold accuracy", "Tiens la précision"),
	}
	fr := suggestedChallengeFromTemplate(tpl, i18n.LocaleFR)
	en := suggestedChallengeFromTemplate(tpl, i18n.LocaleEN)
	if fr.Label != "Précision 50 %" || fr.Description != "Tiens la précision" {
		t.Errorf("fr = %+v", fr)
	}
	if en.Label != "Accuracy 50%" || en.Description != "Hold accuracy" {
		t.Errorf("en = %+v", en)
	}

	sansDescription := suggestedChallengeFromTemplate(prestige.Template{ID: "t2", Labels: i18n.FromColumns("Only EN", "")}, i18n.LocaleFR)
	if sansDescription.Label != "Only EN" || sansDescription.Description != "" {
		t.Errorf("repli en et description vide attendus : %+v", sansDescription)
	}
}
