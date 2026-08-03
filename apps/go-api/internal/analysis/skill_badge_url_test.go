// Package analysis — skill_badge_url_test.go : chokepoint SkillBadgeURL.
//
// SkillBadgeURL est la SOURCE UNIQUE de la normalisation (tier EN brut,
// sous-palier) → (tier capitalisé, sous-palier 0=Onyx/1..6) attendue par les
// résolveurs d'URL title-aware. Elle sert la home (badge du match récent) ET les
// lignes Explorer/historique ; ces tests verrouillent le contrat que les deux
// consommateurs supposent, en particulier la DÉGRADATION en nil (le front
// retombe alors sur le libellé texte du palier, jamais sur une image cassée).
package analysis

import "testing"

// capturingResolver mémorise les arguments reçus pour prouver la normalisation.
func capturingResolver(gotTier *string, gotSub *int, url string) func(string, int) string {
	return func(tierEN string, subTier int) string {
		*gotTier = tierEN
		*gotSub = subTier
		return url
	}
}

func TestSkillBadgeURL_normalizesTierAndSubTier(t *testing.T) {
	sub := 3
	var gotTier string
	var gotSub int
	got := SkillBadgeURL("gold", &sub, capturingResolver(&gotTier, &gotSub, "/static/ranks/gold3.png"))

	if got == nil || *got != "/static/ranks/gold3.png" {
		t.Fatalf("URL = %v, want /static/ranks/gold3.png", got)
	}
	// Capitalisation OBLIGATOIRE : le format HINF est "120px-HINF-CSR_Gold3".
	if gotTier != "Gold" {
		t.Errorf("tier transmis au résolveur = %q, want \"Gold\"", gotTier)
	}
	if gotSub != 3 {
		t.Errorf("sous-palier transmis = %d, want 3", gotSub)
	}
}

func TestSkillBadgeURL_onyxUsesSubTierZero(t *testing.T) {
	var gotTier string
	gotSub := -1
	// Onyx n'a pas de sous-palier : le résolveur doit recevoir 0, même si la DB
	// en fournit un (contrat partagé avec buildCanonicalSkillBadge).
	sub := 4
	got := SkillBadgeURL("ONYX", &sub, capturingResolver(&gotTier, &gotSub, "/static/ranks/onyx.png"))

	if got == nil || *got != "/static/ranks/onyx.png" {
		t.Fatalf("URL = %v, want /static/ranks/onyx.png", got)
	}
	if gotTier != "Onyx" || gotSub != 0 {
		t.Errorf("résolveur appelé avec (%q, %d), want (\"Onyx\", 0)", gotTier, gotSub)
	}
}

func TestSkillBadgeURL_degradesToNil(t *testing.T) {
	alwaysURL := func(string, int) string { return "/static/ranks/x.png" }
	sub3 := 3
	sub9 := 9
	sub0 := 0

	cases := []struct {
		name     string
		tier     string
		subTier  *int
		resolver func(string, int) string
	}{
		{"tier vide (non classé)", "", &sub3, alwaysURL},
		{"tier blanc", "   ", &sub3, alwaysURL},
		{"sous-palier hors bornes haut", "Gold", &sub9, alwaysURL},
		{"sous-palier hors bornes bas", "Gold", &sub0, alwaysURL},
		{"sous-palier absent hors Onyx", "Gold", nil, alwaysURL},
		{"résolveur absent (titre sans badge)", "Gold", &sub3, nil},
		{"résolveur sans URL constructible", "Gold", &sub3, func(string, int) string { return "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SkillBadgeURL(tc.tier, tc.subTier, tc.resolver); got != nil {
				t.Errorf("URL = %q, want nil (dégradation sur le libellé texte)", *got)
			}
		})
	}
}

// TestBuildCanonicalSkillBadge_unchangedAfterExtraction verrouille le comportement
// du badge de la home APRÈS l'extraction du chokepoint : label et URL doivent
// rester ceux d'avant (l'extraction est une factorisation, pas un changement).
func TestBuildCanonicalSkillBadge_unchangedAfterExtraction(t *testing.T) {
	resolver := func(tierEN string, subTier int) string {
		if subTier == 0 {
			return "/u/" + tierEN
		}
		return "/u/" + tierEN + string(rune('0'+subTier))
	}
	sub := 4

	label, url := buildCanonicalSkillBadge("or", "Gold", &sub, resolver)
	if label == nil || *label != "Or IV" {
		t.Errorf("label = %v, want \"Or IV\"", label)
	}
	if url == nil || *url != "/u/Gold4" {
		t.Errorf("url = %v, want /u/Gold4", url)
	}

	// Onyx : libellé sans chiffre romain, URL sans sous-palier.
	if label, url := buildCanonicalSkillBadge("onyx", "Onyx", nil, resolver); label == nil ||
		*label != "Onyx" || url == nil || *url != "/u/Onyx" {
		t.Errorf("Onyx = (%v, %v), want (\"Onyx\", \"/u/Onyx\")", label, url)
	}

	// Sous-palier hors bornes : label ET url nuls (comportement historique).
	bad := 7
	if label, url := buildCanonicalSkillBadge("or", "Gold", &bad, resolver); label != nil || url != nil {
		t.Errorf("sous-palier 7 = (%v, %v), want (nil, nil)", label, url)
	}

	// Sans résolveur : label seul (la page qui n'affiche pas d'image reste servie).
	if label, url := buildCanonicalSkillBadge("or", "Gold", &sub, nil); label == nil ||
		*label != "Or IV" || url != nil {
		t.Errorf("sans résolveur = (%v, %v), want (\"Or IV\", nil)", label, url)
	}
}
