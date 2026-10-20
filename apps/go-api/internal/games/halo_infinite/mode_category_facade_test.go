package halo_infinite

import "testing"

// La façade du titre rend exactement la règle canonique de modelabel (une seule règle).
func TestInferModeCategoryFromPairName_DelegueALaRegleCanonique(t *testing.T) {
	t.Parallel()
	for pair, want := range map[string]string{
		"Ranked:Strongholds on Live Fire": ModeCategoryRanked,
		"Super Fiesta:Slayer on Behemoth": ModeCategorySuperFiesta,
		"CTF:Arena":                       ModeCategoryAssassin,
		"":                                ModeCategoryOther,
	} {
		if got := InferModeCategoryFromPairName(pair); got != want {
			t.Errorf("InferModeCategoryFromPairName(%q) = %q, want %q", pair, got, want)
		}
	}
	if len(AllKnownPairNamePrefixes()) == 0 || PairNamePrefixesForCategory(ModeCategoryBTB) == nil {
		t.Error("les helpers de préfixes doivent déléguer à modelabel")
	}
}
