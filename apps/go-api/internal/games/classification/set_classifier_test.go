package classification

import "testing"

// boolPtrVal déréférence un *bool de test (nil → faux 3e état exprimé via le 2e retour).
func boolPtrVal(b *bool) (val, present bool) {
	if b == nil {
		return false, false
	}
	return *b, true
}

func TestSetClassifier_EmptySet_AlwaysIndeterminate(t *testing.T) {
	// Set vide = pas de donnée autoritative → nil pour tout id (préserve le
	// comportement conservateur « avant data » : jamais de faux classé).
	c := NewSetClassifier(nil, nil)
	for _, id := range []string{"", "anything", "f0c9ef9a-48bd-4b24-9db3-2c76b4e23450"} {
		if _, present := boolPtrVal(c.IsRanked(id)); present {
			t.Errorf("IsRanked(%q) sur set vide = non-nil, want nil (indéterminé)", id)
		}
		if _, present := boolPtrVal(c.IsPvE(id)); present {
			t.Errorf("IsPvE(%q) sur set vide = non-nil, want nil", id)
		}
	}
}

func TestSetClassifier_PopulatedSet_Membership(t *testing.T) {
	ranked := []string{"ranked-a", "ranked-b"}
	pve := []string{"warzone-ff"}
	c := NewSetClassifier(ranked, pve)

	cases := []struct {
		name        string
		got         *bool
		wantVal     bool
		wantPresent bool
	}{
		{"ranked présent → &true", c.IsRanked("ranked-a"), true, true},
		{"ranked absent (set exhaustif) → &false", c.IsRanked("ranked-zzz"), false, true},
		{"ranked id vide → nil", c.IsRanked(""), false, false},
		{"pve présent → &true", c.IsPvE("warzone-ff"), true, true},
		{"pve absent → &false", c.IsPvE("ranked-a"), false, true},
	}
	for _, tc := range cases {
		val, present := boolPtrVal(tc.got)
		if present != tc.wantPresent || val != tc.wantVal {
			t.Errorf("%s: got (val=%v, present=%v), want (val=%v, present=%v)",
				tc.name, val, present, tc.wantVal, tc.wantPresent)
		}
	}
}

func TestSetClassifier_TrimAndDedup(t *testing.T) {
	// Trim sur entrées ET sur requêtes ; entrées vides ignorées ; dédup transparent.
	c := NewSetClassifier([]string{"  ranked-a  ", "ranked-a", "", "   "}, nil)
	if v, present := boolPtrVal(c.IsRanked("ranked-a")); !present || !v {
		t.Errorf("IsRanked(\"ranked-a\") = (%v,%v), want (true,true) après trim", v, present)
	}
	if v, present := boolPtrVal(c.IsRanked("  ranked-a ")); !present || !v {
		t.Errorf("IsRanked avec espaces = (%v,%v), want (true,true) après trim requête", v, present)
	}
	// "   " seul ne crée pas un set fantôme : NewSetClassifier(["   "]) reste vide.
	empty := NewSetClassifier([]string{"   ", ""}, nil)
	if _, present := boolPtrVal(empty.IsRanked("anything")); present {
		t.Errorf("set d'entrées blanches → doit rester vide (verdicts nil)")
	}
}

func TestSetClassifier_NilReceiver(t *testing.T) {
	var c *SetClassifier
	if _, present := boolPtrVal(c.IsRanked("x")); present {
		t.Errorf("récepteur nil IsRanked → want nil")
	}
	if _, present := boolPtrVal(c.IsPvE("x")); present {
		t.Errorf("récepteur nil IsPvE → want nil")
	}
}

// Vérifie que SetClassifier satisfait bien le contrat d'interface partagé.
func TestSetClassifier_ImplementsInterface(t *testing.T) {
	var _ RankedClassifier = NewSetClassifier([]string{"a"}, nil)
}
