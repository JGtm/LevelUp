// Package halo — challenges_details_helpers_test.go : tests de comportement pour
// les helpers PURS de data-shaping des challenges (challenges_details.go).
//
// Ces fonctions n'ont aucune dépendance DB/réseau : on teste les branches métier
// (coercion d'entiers Halo imbriqués, résolution de langue/i18n, heuristiques de
// badge, tri, fallbacks). Aucun padding : chaque cas couvre une branche ou un
// invariant réel.
package halo

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// coerceChallengeInt — coercion d'entiers depuis JSON Halo polymorphe
// ---------------------------------------------------------------------------

func TestCoerceChallengeInt(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		want   int
		wantOK bool
	}{
		{name: "int direct", input: 42, want: 42, wantOK: true},
		{name: "int zéro reste valide", input: 0, want: 0, wantOK: true},
		{name: "float64 tronqué vers le bas", input: float64(5.9), want: 5, wantOK: true},
		{name: "float64 négatif tronqué", input: float64(-2.7), want: -2, wantOK: true},
		{name: "map clé value minuscule", input: map[string]any{"value": 7}, want: 7, wantOK: true},
		{name: "map clé Value PascalCase", input: map[string]any{"Value": 8}, want: 8, wantOK: true},
		{name: "map clé Threshold", input: map[string]any{"Threshold": float64(11)}, want: 11, wantOK: true},
		{name: "map clé Count", input: map[string]any{"Count": 3}, want: 3, wantOK: true},
		{
			name:   "map récursion imbriquée",
			input:  map[string]any{"value": map[string]any{"Count": float64(99)}},
			want:   99,
			wantOK: true,
		},
		{
			name:   "ordre des clés value avant Count",
			input:  map[string]any{"value": 1, "Count": 2},
			want:   1,
			wantOK: true,
		},
		{name: "map sans clé connue", input: map[string]any{"foo": 5}, want: 0, wantOK: false},
		{name: "map vide", input: map[string]any{}, want: 0, wantOK: false},
		{name: "type non géré string", input: "12", want: 0, wantOK: false},
		{name: "type non géré nil", input: nil, want: 0, wantOK: false},
		{name: "type non géré bool", input: true, want: 0, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := coerceChallengeInt(tt.input)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("coerceChallengeInt(%v) = (%d, %v), want (%d, %v)",
					tt.input, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveChallengeTarget — priorité ch.Threshold > def.ThresholdForSuccess
// ---------------------------------------------------------------------------

func TestResolveChallengeTarget(t *testing.T) {
	tests := []struct {
		name string
		ch   challengeDeckItemRaw
		def  *challengeDefinitionRaw
		want *int
	}{
		{
			name: "threshold du deck prioritaire sur la def",
			ch:   challengeDeckItemRaw{Threshold: intPtr(5)},
			def:  &challengeDefinitionRaw{ThresholdForSuccess: float64(99)},
			want: intPtr(5),
		},
		{
			name: "fallback def quand deck nil",
			ch:   challengeDeckItemRaw{},
			def:  &challengeDefinitionRaw{ThresholdForSuccess: float64(10)},
			want: intPtr(10),
		},
		{
			name: "def via map imbriquée",
			ch:   challengeDeckItemRaw{},
			def:  &challengeDefinitionRaw{ThresholdForSuccess: map[string]any{"Threshold": float64(7)}},
			want: intPtr(7),
		},
		{
			name: "def nil et deck nil -> nil",
			ch:   challengeDeckItemRaw{},
			def:  nil,
			want: nil,
		},
		{
			name: "def non coercible -> nil",
			ch:   challengeDeckItemRaw{},
			def:  &challengeDefinitionRaw{ThresholdForSuccess: "abc"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveChallengeTarget(tt.ch, tt.def)
			assertIntPtrEqual(t, got, tt.want)
		})
	}
}

// ---------------------------------------------------------------------------
// resolveChallengeXP — priorité ch.XPReward > Reward > SecondaryReward
// ---------------------------------------------------------------------------

func TestResolveChallengeXP(t *testing.T) {
	withReward := func(soft, secondary int) *challengeDefinitionRaw {
		def := &challengeDefinitionRaw{}
		def.Reward.SoftExperience = soft
		def.SecondaryReward.SoftExperience = secondary
		return def
	}
	tests := []struct {
		name string
		ch   challengeDeckItemRaw
		def  *challengeDefinitionRaw
		want *int
	}{
		{
			name: "XPReward du deck prioritaire",
			ch:   challengeDeckItemRaw{XPReward: intPtr(50)},
			def:  withReward(999, 999),
			want: intPtr(50),
		},
		{
			name: "Reward.SoftExperience quand deck nil",
			ch:   challengeDeckItemRaw{},
			def:  withReward(200, 0),
			want: intPtr(200),
		},
		{
			name: "SecondaryReward quand Reward == 0",
			ch:   challengeDeckItemRaw{},
			def:  withReward(0, 75),
			want: intPtr(75),
		},
		{
			name: "def nil -> nil",
			ch:   challengeDeckItemRaw{},
			def:  nil,
			want: nil,
		},
		{
			name: "tous rewards à 0 -> nil",
			ch:   challengeDeckItemRaw{},
			def:  withReward(0, 0),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveChallengeXP(tt.ch, tt.def)
			assertIntPtrEqual(t, got, tt.want)
		})
	}
}

// ---------------------------------------------------------------------------
// resolveChallengeCurrentProgress — Progress > CurrentProgress > nil
// ---------------------------------------------------------------------------

func TestResolveChallengeCurrentProgress(t *testing.T) {
	tests := []struct {
		name string
		ch   challengeDeckItemRaw
		want *int
	}{
		{
			name: "Progress prioritaire",
			ch:   challengeDeckItemRaw{Progress: intPtr(3), CurrentProgress: intPtr(9)},
			want: intPtr(3),
		},
		{
			name: "Progress zéro reste prioritaire (non nil)",
			ch:   challengeDeckItemRaw{Progress: intPtr(0), CurrentProgress: intPtr(9)},
			want: intPtr(0),
		},
		{
			name: "fallback CurrentProgress",
			ch:   challengeDeckItemRaw{CurrentProgress: intPtr(4)},
			want: intPtr(4),
		},
		{
			name: "aucun -> nil",
			ch:   challengeDeckItemRaw{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveChallengeCurrentProgress(tt.ch)
			assertIntPtrEqual(t, got, tt.want)
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeChallengeLang + challengeLanguageCandidates
// ---------------------------------------------------------------------------

func TestNormalizeChallengeLang(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "fr", want: langFR},
		{in: "fr-FR", want: langFR},
		{in: "FR-fr", want: langFR},
		{in: "  fr  ", want: langFR},
		{in: "en", want: langEN},
		{in: "en-US", want: langEN},
		{in: "EN-us", want: langEN},
		{in: "de", want: langFR},    // défaut = FR
		{in: "", want: langFR},      // défaut = FR
		{in: "es-ES", want: langFR}, // défaut = FR
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeChallengeLang(tt.in); got != tt.want {
				t.Fatalf("normalizeChallengeLang(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestChallengeLanguageCandidates(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want []string
	}{
		{name: "FR", lang: langFR, want: []string{langFR, "fr"}},
		{name: "EN", lang: langEN, want: []string{langEN, "en-GB", "en"}},
		{name: "autre avec région", lang: "es-ES", want: []string{"es-ES", "es"}},
		{name: "autre sans région", lang: "pt", want: []string{"pt", "pt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := challengeLanguageCandidates(tt.lang)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("challengeLanguageCandidates(%q) = %v, want %v", tt.lang, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveChallengeLocalizedValue — string direct, translations, fallback value
// ---------------------------------------------------------------------------

func TestResolveChallengeLocalizedValue(t *testing.T) {
	tests := []struct {
		name string
		data any
		lang string
		want string
	}{
		{
			name: "string direct trimé",
			data: "  Tuer 10 ennemis  ",
			lang: "fr",
			want: "Tuer 10 ennemis",
		},
		{
			name: "translation FR exacte",
			data: map[string]any{"translations": map[string]any{langFR: "Défi FR", "en-US": "EN challenge"}},
			lang: "fr",
			want: "Défi FR",
		},
		{
			name: "translation EN via candidat en-US",
			data: map[string]any{"translations": map[string]any{"en-US": "EN challenge"}},
			lang: "en",
			want: "EN challenge",
		},
		{
			name: "translation EN via fallback en-GB",
			data: map[string]any{"translations": map[string]any{"en-GB": "GB challenge"}},
			lang: "en",
			want: "GB challenge",
		},
		{
			name: "translation vide ignorée -> fallback value",
			data: map[string]any{"translations": map[string]any{langFR: "   "}, "value": "brut"},
			lang: "fr",
			want: "brut",
		},
		{
			name: "aucune translation -> fallback obj value",
			data: map[string]any{"value": "valeur par défaut"},
			lang: "fr",
			want: "valeur par défaut",
		},
		{
			name: "map sans translations ni value -> vide",
			data: map[string]any{"foo": "bar"},
			lang: "fr",
			want: "",
		},
		{
			name: "type non-map non-string -> vide",
			data: 123,
			lang: "fr",
			want: "",
		},
		{
			name: "candidat FR absent mais value présent",
			data: map[string]any{"translations": map[string]any{"de-DE": "Deutsch"}, "value": "fb"},
			lang: "fr",
			want: "fb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveChallengeLocalizedValue(tt.data, tt.lang); got != tt.want {
				t.Fatalf("resolveChallengeLocalizedValue(%v, %q) = %q, want %q",
					tt.data, tt.lang, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isSeasonalChallengePath — tokens explicites + heuristique /s + challenges/
// ---------------------------------------------------------------------------

func TestIsSeasonalChallengePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "vide", path: "", want: false},
		{name: "winter token", path: "progression/winterchallenges/foo", want: true},
		{name: "seasonal token", path: "x/seasonalchallenges/y", want: true},
		{name: "event token", path: "eventchallenges/abc", want: true},
		{name: "operation token", path: "operationchallenges/abc", want: true},
		{name: "fracture token", path: "fracturechallenges/abc", want: true},
		{
			name: "heuristique /s + challenges/",
			path: "progression/seasons/challenges/foo",
			want: true,
		},
		{
			name: "daily non saisonnier",
			path: "progression/dailychallenges/normal",
			want: false,
		},
		{
			name: "challenges/ sans /s -> false",
			path: "progression/dailychallenges/normal-challenges/foo",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSeasonalChallengePath(tt.path); got != tt.want {
				t.Fatalf("isSeasonalChallengePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// challengeSortScore — pct prioritaire, sinon micro-score via current>0
// ---------------------------------------------------------------------------

func TestChallengeSortScore(t *testing.T) {
	pct := func(v float64) *float64 { return &v }
	tests := []struct {
		name string
		item domain.ChallengeItem
		want float64
	}{
		{
			name: "ProgressPercent renvoyé tel quel",
			item: domain.ChallengeItem{ProgressPercent: pct(42.5)},
			want: 42.5,
		},
		{
			name: "ProgressPercent zéro prioritaire sur current",
			item: domain.ChallengeItem{ProgressPercent: pct(0), ProgressCurrent: intPtr(100)},
			want: 0,
		},
		{
			name: "current>0 -> micro-score",
			item: domain.ChallengeItem{ProgressCurrent: intPtr(5)},
			want: 0.001 + 5.0/10000.0,
		},
		{
			name: "current==0 -> 0",
			item: domain.ChallengeItem{ProgressCurrent: intPtr(0)},
			want: 0,
		},
		{
			name: "tout nil -> 0",
			item: domain.ChallengeItem{},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := challengeSortScore(tt.item); got != tt.want {
				t.Fatalf("challengeSortScore = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// fallbackChallengeTitle — base path, trim ext, _/- -> espaces, vide -> défaut
// ---------------------------------------------------------------------------

func TestFallbackChallengeTitle(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "base avec ext et underscores", path: "Progression/Challenges/kill_ten_foes.json", want: "kill ten foes"},
		{name: "tirets convertis", path: "x/y/win-three-games", want: "win three games"},
		{name: "vide -> défaut", path: "", want: "Défi actif"},
		{name: "point seul -> défaut", path: ".", want: "Défi actif"},
		// "." est garde-fou explicite : path.Base("foo/.") == "." -> défaut.
		{name: "dossier terminé par point -> défaut", path: "foo/.", want: "Défi actif"},
		{name: "sans dossier", path: "simple", want: "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fallbackChallengeTitle(tt.path); got != tt.want {
				t.Fatalf("fallbackChallengeTitle(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// challengePathOrFallback — path, sinon tracking, sinon Unknown
// ---------------------------------------------------------------------------

func TestChallengePathOrFallback(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		trackingID string
		want       string
	}{
		{name: "path trimé", path: "  Challenges/Foo  ", trackingID: "abc", want: "Challenges/Foo"},
		{name: "path vide -> tracking", path: "   ", trackingID: "trk-1", want: "Challenges/Tracking/trk-1"},
		{name: "path et tracking vides -> Unknown", path: "", trackingID: "", want: "Challenges/Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := challengePathOrFallback(tt.path, tt.trackingID); got != tt.want {
				t.Fatalf("challengePathOrFallback(%q,%q) = %q, want %q", tt.path, tt.trackingID, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// derefInt
// ---------------------------------------------------------------------------

func TestDerefInt(t *testing.T) {
	if got := derefInt(nil); got != 0 {
		t.Fatalf("derefInt(nil) = %d, want 0", got)
	}
	if got := derefInt(intPtr(7)); got != 7 {
		t.Fatalf("derefInt(7) = %d, want 7", got)
	}
}

// ---------------------------------------------------------------------------
// buildChallengeBadgeCandidates — matrice path/category/difficulty + dédup
// ---------------------------------------------------------------------------

func TestBuildChallengeBadgeCandidates(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		category   string
		difficulty string
		want       []string
	}{
		{
			name:       "daily par path",
			path:       "progression/dailychallenges/foo",
			difficulty: "normal",
			want:       []string{"daily-normal"},
		},
		{
			name:       "weekly par path avec famille",
			path:       "progression/weeklychallenges/weapon/foo",
			difficulty: "hard",
			want:       []string{"weekly-weapon-hard"},
		},
		{
			name:       "ultimate par path force mythic et capstone-mythic",
			path:       "progression/ultimate/foo",
			difficulty: "",
			want:       []string{"capstone-mythic"},
		},
		{
			name:       "capstone par catégorie",
			path:       "x/y",
			category:   "capstone",
			difficulty: "",
			want:       []string{"capstone-mythic"},
		},
		{
			name:       "catégorie daily seule",
			path:       "x/y",
			category:   "daily",
			difficulty: "heroic",
			want:       []string{"daily-heroic", "daily-heroic"}, // sera dédupé
		},
		{
			name:       "aucune branche -> vide",
			path:       "x/y",
			category:   "",
			difficulty: "",
			want:       []string{},
		},
		{
			name:       "category+difficulty fallback générique",
			path:       "x/y",
			category:   "mystery",
			difficulty: "legend",
			want:       []string{"mystery-legend"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildChallengeBadgeCandidates(tt.path, tt.category, tt.difficulty)
			// Invariant fort : la sortie est toujours dédupée.
			assertNoDuplicates(t, got)
			// On vérifie que les candidats attendus sont présents et dans l'ordre
			// (la dédup préserve l'ordre de première apparition).
			deduped := dedupeChallengeCandidates(tt.want)
			if !reflect.DeepEqual(got, deduped) {
				t.Fatalf("buildChallengeBadgeCandidates(%q,%q,%q) = %v, want %v",
					tt.path, tt.category, tt.difficulty, got, deduped)
			}
		})
	}
}

// Invariant : buildChallengeBadgeCandidates ne renvoie jamais de doublon, même
// quand path ET category produisent le même stem (ex. daily-normal x2).
func TestBuildChallengeBadgeCandidates_DedupAcrossSources(t *testing.T) {
	got := buildChallengeBadgeCandidates("progression/dailychallenges/foo", "daily", "normal")
	want := []string{"daily-normal"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attendu dédup cross-source %v, got %v", want, got)
	}
}

// ---------------------------------------------------------------------------
// helpers de test
// ---------------------------------------------------------------------------

func assertIntPtrEqual(t *testing.T, got, want *int) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("pointeurs divergents : got=%v want=%v", fmtIntPtr(got), fmtIntPtr(want))
	case *got != *want:
		t.Fatalf("valeurs divergentes : got=%d want=%d", *got, *want)
	}
}

func fmtIntPtr(p *int) string {
	if p == nil {
		return "nil"
	}
	return "ptr"
}

func assertNoDuplicates(t *testing.T, values []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			t.Fatalf("doublon détecté %q dans %v", v, values)
		}
		seen[v] = struct{}{}
	}
}
