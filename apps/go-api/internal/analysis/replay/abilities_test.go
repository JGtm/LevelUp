package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// TestInvAbilityRankHighEstLaQueueDuMotif verrouille la DERIVATION, pas la valeur.
//
// Le facteur 16 n'est pas une constante posee a cote du motif d'ancrage : ce sont ses trois
// derniers bits. L'ecrire deux fois aurait permis a l'un des deux de deriver ; ce test dit
// que la reconstruction suit le motif, et que la fenetre visible du canal d'image-cle est
// exactement 16..23 — ce qui est la cause, enfin nommee, des 21 films sur 40 muets.
func TestInvAbilityRankHighEstLaQueueDuMotif(t *testing.T) {
	if invAbilityRankHigh != 2 {
		t.Fatalf("les 3 bits de poids fort du rang valent %d, attendu 2 (la queue `010` du motif)",
			invAbilityRankHigh)
	}
	if got := invAbilityRankOf(0); got != 16 {
		t.Errorf("le bas de la fenetre vaut %d, attendu 16", got)
	}
	if got := invAbilityRankOf(7); got != 23 {
		t.Errorf("le haut de la fenetre vaut %d, attendu 23", got)
	}
	// CONTROLE SUR PIECES (film 000d5950) : le canal i48, independant, rend 20 sur les slots
	// que le releve Theater nomme grappin (bas 4), 21 sur le propulseur (5), 19 sur le mur (3).
	for low, want := range map[uint32]int{3: 19, 4: 20, 5: 21, 6: 22} {
		if got := invAbilityRankOf(low); got != want {
			t.Errorf("bits bas %d -> rang %d, attendu %d", low, got, want)
		}
	}
}

func TestBuildAbilityReadsFusionneLesDeuxCanaux(t *testing.T) {
	const origin, step = 1_000_000, 100_000
	ranks := []filmdec.AbilityRank{
		{TimestampUS: 500_000, Slot: 512, Rank: 8},   // avant l'origine : ecarte
		{TimestampUS: 1_300_000, Slot: 512, Rank: 8}, // camouflage — invisible du canal kf
	}
	inv := []KeyframeInventory{
		{TimestampUS: 1_300_000, Slot: 513, AbilityRank: 20},
		{TimestampUS: 1_200_000, Slot: 512, AbilityRank: -1}, // non lu : ecarte
		{TimestampUS: 400_000, Slot: 513, AbilityRank: 23},   // avant l'origine : ecarte
	}
	got := buildAbilityReads(ranks, inv, origin, step)
	if len(got) != 2 {
		t.Fatalf("attendu 2 lectures retenues, obtenu %d : %+v", len(got), got)
	}
	// Tri par image, puis slot, puis canal : un artefact reproductible d'un build a l'autre.
	if got[0].T != 3 || got[0].Slot != 512 || got[0].R != 8 || got[0].Src != AbilitySrcI48 {
		t.Errorf("lecture i48 mal portee : %+v", got[0])
	}
	if got[1].Slot != 513 || got[1].R != 20 || got[1].Src != AbilitySrcKeyframe {
		t.Errorf("lecture d'image-cle mal portee : %+v", got[1])
	}
	if buildAbilityReads(nil, nil, origin, step) != nil {
		t.Error("sans lecture, rien n'est invente")
	}
}

func TestKeepAbilitiesOfPublishedTracks(t *testing.T) {
	reads := []AbilityRead{{Slot: 512, R: 8}, {Slot: 999, R: 9}}
	got := keepAbilitiesOfPublishedTracks(reads, []Track{{Slot: 512}})
	if len(got) != 1 || got[0].Slot != 512 {
		t.Errorf("une capacite sans trace ou se poser doit etre ecartee : %+v", got)
	}
}

func TestAbilityLabelsUsedNamesOnlyWhatItKnows(t *testing.T) {
	// LA TABLE EST PARTIELLE, et propre a la PALETTE du film. Un rang hors table garde son
	// numero a l'ecran — le combler par le nom d'une capacite voisine se lirait comme une
	// certitude. Le catalogue vient du TITRE (replay_labels.toml) et non d'une table Go : le
	// test l'injecte, comme le fait cmd/replay-build.
	catalogue := map[int]Label{20: {En: "Grappleshot", Fr: "grappin"}}
	reads := []AbilityRead{{R: 20}, {R: 9}}
	got := abilityLabelsUsed(reads, catalogue)
	if got["20"].Fr != "grappin" || got["20"].En != "Grappleshot" {
		t.Errorf("rang connu non nomme dans les deux langues : %+v", got)
	}
	if _, named := got["9"]; named {
		t.Error("un rang hors table ne doit PAS etre nomme")
	}
	if abilityLabelsUsed(nil, catalogue) != nil {
		t.Error("sans lecture, pas de table inventee")
	}
	// Catalogue absent (titre sans libelles, ou palette non classee) : aucun nom, aucun panic.
	if abilityLabelsUsed(reads, nil) != nil {
		t.Error("sans catalogue, aucune capacite ne doit etre nommee")
	}
}
