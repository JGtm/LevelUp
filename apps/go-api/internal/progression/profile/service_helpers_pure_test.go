package profile

import (
	"database/sql"
	"reflect"
	"testing"

	"levelup/go-api/internal/prestige"
	syncpkg "levelup/go-api/internal/sync"
)

// service_helpers_pure_test.go — helpers purs non couverts par
// service_helpers_test.go (requiredCompositeFor, snapshot branches,
// compositeWeightsExternal, nullStringToString, IsPositive, decodeStringListLocal).

func TestRequiredCompositeFor_TargetNotAboveCurrent(t *testing.T) {
	// targetMu <= currentMu → pas d'objectif → 0 (sortie avant délégation).
	if got := requiredCompositeFor(1500, 1500, 100); got != 0 {
		t.Errorf("target==current: got %v, want 0", got)
	}
	if got := requiredCompositeFor(1500, 1400, 100); got != 0 {
		t.Errorf("target<current: got %v, want 0", got)
	}
}

func TestRequiredCompositeFor_DelegatesInRange(t *testing.T) {
	// targetMu > currentMu, sigma valide → résultat clampé dans [0,1].
	got := requiredCompositeFor(1500, 1520, 150)
	if got < 0 || got > 1 {
		t.Errorf("composite = %v, hors [0,1]", got)
	}
	// Une cible très lointaine en 1 match → borne haute 1.0.
	if far := requiredCompositeFor(1500, 1900, 150); far != 1.0 {
		t.Errorf("cible lointaine: got %v, want 1.0 (cap)", far)
	}
}

func TestRequiredCompositeFor_SigmaNonPositiveDefaultsToOne(t *testing.T) {
	// sigma<=0 → remplacé par 1.0 en interne (pas de panic/division dégénérée).
	// On vérifie surtout que ça ne panique pas et reste dans [0,1].
	for _, sigma := range []float64{0, -50} {
		got := requiredCompositeFor(1500, 1520, sigma)
		if got < 0 || got > 1 {
			t.Errorf("sigma=%v → composite %v hors [0,1]", sigma, got)
		}
	}
}

func TestBuildSkillRatingSnapshot_NextTierEmpty_NoGap(t *testing.T) {
	// NextTier vide (joueur au max) → pas de NextTierLabel/Gap.
	lusr := LUSRState{Mu: 2500, Sigma: 100}
	tier := TierState{Name: "Onyx", NameFR: "Onyx", SubTier: 0, Label: "Onyx", LowerMu: 2000, UpperMu: 9999}
	got := buildSkillRatingSnapshot(lusr, tier, TierState{})
	if got.NextTierLabel != "" {
		t.Errorf("NextTierLabel = %q, want empty (pas de tier suivant)", got.NextTierLabel)
	}
	if got.GapToNext != 0 {
		t.Errorf("GapToNext = %v, want 0", got.GapToNext)
	}
	if got.NextTierMu != 0 {
		t.Errorf("NextTierMu = %v, want 0", got.NextTierMu)
	}
}

func TestBuildSkillRatingSnapshot_ProgressClampLow(t *testing.T) {
	// μ sous LowerMu → ratio négatif → clamp à 0.
	lusr := LUSRState{Mu: 1380} // sous LowerMu=1400
	tier := TierState{Name: "Gold", LowerMu: 1400, UpperMu: 1600}
	got := buildSkillRatingSnapshot(lusr, tier, TierState{})
	if got.ProgressRatio != 0 {
		t.Errorf("ProgressRatio = %v, want 0 (clamp bas)", got.ProgressRatio)
	}
}

func TestBuildSkillRatingSnapshot_ProgressClampHigh(t *testing.T) {
	// μ au-dessus de UpperMu → ratio > 1 → clamp à 1.
	lusr := LUSRState{Mu: 1700} // au-dessus de UpperMu=1600
	tier := TierState{Name: "Gold", LowerMu: 1400, UpperMu: 1600}
	got := buildSkillRatingSnapshot(lusr, tier, TierState{})
	if got.ProgressRatio != 1 {
		t.Errorf("ProgressRatio = %v, want 1 (clamp haut)", got.ProgressRatio)
	}
}

func TestBuildSkillRatingSnapshot_NoProgressWhenDegenerateBounds(t *testing.T) {
	// UpperMu == LowerMu (pas d'intervalle) → ProgressRatio reste 0 (pas de div/0).
	lusr := LUSRState{Mu: 1500}
	tier := TierState{Name: "X", LowerMu: 1500, UpperMu: 1500}
	got := buildSkillRatingSnapshot(lusr, tier, TierState{})
	if got.ProgressRatio != 0 {
		t.Errorf("ProgressRatio = %v, want 0 (bornes dégénérées)", got.ProgressRatio)
	}
}

func TestCompositeWeightsExternal_DefensiveCopy(t *testing.T) {
	got := compositeWeightsExternal()
	if len(got) != len(syncpkg.CompositeWeights) {
		t.Fatalf("len = %d, want %d", len(got), len(syncpkg.CompositeWeights))
	}
	for k, v := range syncpkg.CompositeWeights {
		if got[k] != v {
			t.Errorf("weight[%s] = %v, want %v", k, got[k], v)
		}
	}
	// Muter la copie ne doit PAS toucher la source (copie défensive).
	for k := range got {
		got[k] = -999
		break
	}
	for k, v := range got {
		if v == -999 {
			if syncpkg.CompositeWeights[k] == -999 {
				t.Errorf("mutation de la copie a corrompu la source CompositeWeights[%s]", k)
			}
			break
		}
	}
}

func TestNullStringToString(t *testing.T) {
	if got := nullStringToString(sql.NullString{String: "hello", Valid: true}); got != "hello" {
		t.Errorf("valid: got %q, want 'hello'", got)
	}
	if got := nullStringToString(sql.NullString{String: "ignored", Valid: false}); got != "" {
		t.Errorf("NULL: got %q, want ''", got)
	}
}

func TestLOWESSTrend_IsPositive(t *testing.T) {
	tests := []struct {
		name      string
		slope     float64
		window    int
		minWindow int
		want      bool
	}{
		{"slope>0 et window suffisant", 0.5, 10, 5, true},
		{"slope nul → faux", 0, 10, 5, false},
		{"slope négatif → faux", -0.3, 10, 5, false},
		{"window insuffisant → faux", 0.5, 3, 5, false},
		{"window pile au seuil → vrai", 0.5, 5, 5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := LOWESSTrend{Slope: tt.slope, Window: tt.window}
			if got := tr.IsPositive(tt.minWindow); got != tt.want {
				t.Errorf("IsPositive = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeStringListLocal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"vide → nil", "", nil},
		{"un élément", "kills", []string{"kills"}},
		{"plusieurs", "kills,deaths,wins", []string{"kills", "deaths", "wins"}},
		{"virgule finale → élément vide conservé", "a,", []string{"a", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeStringListLocal(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDefaultTargetTierForTemplate_AlwaysNormal(t *testing.T) {
	// Heuristique V1 : toujours "normal" quel que soit le template.
	if got := defaultTargetTierForTemplate(prestige.Template{}); got != "normal" {
		t.Errorf("got %q, want 'normal'", got)
	}
}
