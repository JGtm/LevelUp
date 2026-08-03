// Package service — match_history_skill_badge_test.go : propagation des deux
// champs ajoutés au contrat des lignes Explorer/historique (V73-L2 items 2.4a/2.5).
//
// personal_score était DÉJÀ chargé depuis la DB (Q5SharedHistory) puis jeté par
// enrichRow ; skill_rank_image_url est nouveau et vient du résolveur d'assets du
// titre. Ces tests verrouillent les deux chemins ET la dégradation (titre sans
// badge → nil, jamais d'URL bidon).
package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// strPtr : helper partagé du package (testhelpers_test.go).

func TestEnrichRow_personalScoreSurvivesEnrichment(t *testing.T) {
	score := 1450
	row := enrichRow(domain.MatchHistoryRawRow{MatchID: "m1", PersonalScore: &score}, nil, rowFormatters{})
	if row.PersonalScore == nil || *row.PersonalScore != 1450 {
		t.Fatalf("PersonalScore = %v, want 1450 (chargé en DB, ne doit plus être perdu)", row.PersonalScore)
	}

	// Donnée absente → nil (le front affiche "-", pas un 0 trompeur).
	if got := enrichRow(domain.MatchHistoryRawRow{MatchID: "m2"}, nil, rowFormatters{}); got.PersonalScore != nil {
		t.Errorf("PersonalScore sans donnée = %v, want nil", *got.PersonalScore)
	}
}

func TestEnrichRow_skillRankImageURLFromTitleResolver(t *testing.T) {
	sub := 4
	raw := domain.MatchHistoryRawRow{
		MatchID:        "m1",
		SkillTier:      strPtr("Diamond"),
		SubTier:        &sub,
		SkillTierLabel: strPtr("Diamant IV"),
	}

	var gotTier string
	var gotSub int
	fmts := rowFormatters{skillBadgeURL: func(tierEN string, subTier int) string {
		gotTier, gotSub = tierEN, subTier
		return "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond4.png"
	}}

	row := enrichRow(raw, nil, fmts)
	if row.SkillRankImageURL == nil {
		t.Fatal("SkillRankImageURL = nil, want l'URL rendue par le résolveur du titre")
	}
	if *row.SkillRankImageURL != "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond4.png" {
		t.Errorf("SkillRankImageURL = %q, inattendu", *row.SkillRankImageURL)
	}
	if gotTier != "Diamond" || gotSub != 4 {
		t.Errorf("résolveur appelé avec (%q, %d), want (\"Diamond\", 4)", gotTier, gotSub)
	}
	// Le libellé texte reste servi À CÔTÉ de l'image : c'est lui qui sert d'alt
	// côté front et de repli quand l'image manque.
	if row.SkillTierLabel == nil || *row.SkillTierLabel != "Diamant IV" {
		t.Errorf("SkillTierLabel = %v, want \"Diamant IV\" (conservé)", row.SkillTierLabel)
	}
}

// TestEnrichRow_skillRankImageURLDegrades : sans résolveur (titre sans badge, ex.
// H5 selon son adaptateur) ou sans palier (match non classé / en placement), le
// champ reste nil et la colonne « Rang » retombe sur le texte localisé.
func TestEnrichRow_skillRankImageURLDegrades(t *testing.T) {
	sub := 4
	withResolver := rowFormatters{skillBadgeURL: func(string, int) string { return "/x.png" }}

	cases := []struct {
		name string
		raw  domain.MatchHistoryRawRow
		fmts rowFormatters
	}{
		{
			name: "titre sans résolveur de badge",
			raw:  domain.MatchHistoryRawRow{MatchID: "m1", SkillTier: strPtr("Diamond"), SubTier: &sub},
			fmts: rowFormatters{},
		},
		{
			name: "match non classé (aucun palier)",
			raw:  domain.MatchHistoryRawRow{MatchID: "m2"},
			fmts: withResolver,
		},
		{
			name: "palier sans sous-palier (placement)",
			raw:  domain.MatchHistoryRawRow{MatchID: "m3", SkillTier: strPtr("Diamond")},
			fmts: withResolver,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := enrichRow(tc.raw, nil, tc.fmts); got.SkillRankImageURL != nil {
				t.Errorf("SkillRankImageURL = %q, want nil (dégradation sur le texte)", *got.SkillRankImageURL)
			}
		})
	}
}
