package modelabel

import "testing"

// StripMapSuffix retire le suffixe de CARTE, et RIEN D'AUTRE — c'est toute sa raison d'être.
// La normalisation complète (analysis.NormalizeModeLabel) jette en plus le préfixe de
// playlist et le sous-mode ; ce découpage-ci les garde, parce que l'appariement des jetons de
// mode en a besoin (« Super Fiesta:Slayer » doit conserver « Slayer »).
func TestStripMapSuffix(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		// Le suffixe, EN et FR, insensible à la casse.
		"Arena:CTF on Aquarius":                    "Arena:CTF",
		"Arena:CTF sur Aquarius":                   "Arena:CTF",
		"Community:Team Slayer ON Starboard":       "Community:Team Slayer",
		"Super Fiesta:Slayer on Forbidden - Forge": "Super Fiesta:Slayer",
		// Rien à retirer : le libellé sort intact (trimé).
		"Arena:Strongholds":   "Arena:Strongholds",
		"  Team Slayer:Arena": "Team Slayer:Arena",
		"":                    "",
		"   ":                 "",
		// « on » DANS un mot n'est pas le suffixe : la regex exige des espaces autour.
		"Arena:Total Control": "Arena:Total Control",
	}
	for in, want := range cases {
		if got := StripMapSuffix(in); got != want {
			t.Errorf("StripMapSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

// Le retrait est ce qui protège l'appariement d'une collision entre un jeton de mode et un
// NOM DE CARTE : sans lui, « Arena:Strongholds on Oddball Bay » attraperait « Oddball ».
func TestStripMapSuffix_ProtegeDesCollisions(t *testing.T) {
	t.Parallel()
	label := StripMapSuffix("Arena:Strongholds on Oddball Bay")
	if label != "Arena:Strongholds" {
		t.Fatalf("StripMapSuffix = %q, want %q", label, "Arena:Strongholds")
	}
	if got := ExtractKnownMode(label, []string{"Oddball", "Strongholds"}); got != "Strongholds" {
		t.Errorf("ExtractKnownMode après retrait = %q, want %q", got, "Strongholds")
	}
	// Sans le retrait, le nom de carte gagnerait — c'est le piège que la fonction ferme.
	if got := ExtractKnownMode("Arena:Strongholds on Oddball Bay", []string{"Oddball"}); got != "Oddball" {
		t.Errorf("contrôle négatif : sans retrait, on attend la collision (got %q)", got)
	}
}
