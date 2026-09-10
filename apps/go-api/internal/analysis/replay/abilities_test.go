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

// TestRejectAbilityScanNoise — RAPPORT_E0_2026-09-10 §3 : un rang hors du domaine plausible
// d'une palette du titre est du BRUIT DE BALAYAGE, pas une capacité inconnue. Les deux cas
// mesurés sur le parc (`9ffce8ef` rang 32 slot 530, `a03a5e65` rang 34 slot 574) sont
// reproduits ici par leur RANG SEUL — c'est le critère retenu (cf. abilityRankDomainMax),
// pas la distance à une vie : mesuré sur les 64 films, aucun seuil de fenêtre ne sépare ces
// deux lectures des 14 autres lectures i48 légitimement hors-fenêtre du corpus (gaps de 64 à
// 465 images, valeurs qui ENCADRENT les 176 images de `a03a5e65` des deux côtés), alors que
// le domaine du rang les sépare EXACTEMENT (zéro collatéral sur 4 952 lectures du corpus).
func TestRejectAbilityScanNoise(t *testing.T) {
	reads := []AbilityRead{
		{Slot: 512, R: 5, Src: AbilitySrcI48},       // valide : conservee
		{Slot: 530, R: 32, Src: AbilitySrcI48},      // 9ffce8ef, image 6140
		{Slot: 574, R: 34, Src: AbilitySrcI48},      // a03a5e65, image 2746
		{Slot: 517, R: 22, Src: AbilitySrcKeyframe}, // valide : conservee
	}
	got, noise := rejectAbilityScanNoise(reads)
	if noise != 2 {
		t.Fatalf("bruit compte = %d, attendu 2 (rangs 32 et 34)", noise)
	}
	if len(got) != 2 {
		t.Fatalf("lectures conservees = %d, attendu 2 : %+v", len(got), got)
	}
	for _, r := range got {
		if r.R > abilityRankDomainMax {
			t.Errorf("une lecture hors domaine a survecu au filtre : %+v", r)
		}
	}
	if got2, noise2 := rejectAbilityScanNoise(nil); got2 != nil || noise2 != 0 {
		t.Error("sans lecture, rien n'est invente ni compte")
	}
	// LE RANG PILE AU PLAFOND EST CONSERVE : le seuil ecarte ce qui le DEPASSE, pas ce qui
	// l'atteint (cf. justification d'abilityRankDomainMax).
	pileAuPlafond := []AbilityRead{{Slot: 1, R: abilityRankDomainMax, Src: AbilitySrcI48}}
	if got3, noise3 := rejectAbilityScanNoise(pileAuPlafond); len(got3) != 1 || noise3 != 0 {
		t.Errorf("un rang egal au plafond doit etre conserve : got=%+v noise=%d", got3, noise3)
	}
}

// famA / famB : les deux palettes du titre, réduites à ce que le classement consomme.
var (
	famA = AbilityPalette{
		ID:      "famille_a",
		Markers: []int{1, 2, 4, 5, 6, 8, 9, 10, 11, 12, 23},
		Ranks:   map[int]Label{8: {En: "Active Camouflage", Fr: "camouflage actif"}},
	}
	famB = AbilityPalette{
		ID:      "famille_b",
		Markers: []int{19, 20, 21, 22},
		Ranks:   map[int]Label{20: {En: "Grappleshot", Fr: "grappin"}},
	}
)

// readsOf fabrique des lectures à partir d'un histogramme rang -> compte.
func readsOf(hist map[int]int) []AbilityRead {
	var out []AbilityRead
	for rank, n := range hist {
		for i := 0; i < n; i++ {
			out = append(out, AbilityRead{R: rank, Src: AbilitySrcI48})
		}
	}
	return out
}

// TestClassifyAbilityPaletteSurLesSignaturesMESUREES rejoue les SEPT signatures du corpus.
//
// Ce ne sont pas des cas inventés : ce sont les histogrammes que l'instrument i48 a rendus
// film par film (748 lectures, zéro illisible). Si la règle changeait au point de reclasser
// l'un d'eux, elle nommerait des capacités différentes sur des films déjà servis.
func TestClassifyAbilityPaletteSurLesSignaturesMESUREES(t *testing.T) {
	palettes := []AbilityPalette{famA, famB}
	cas := []struct {
		film string
		hist map[int]int
		want string
	}{
		{"00ba2e1c", map[int]int{1: 31, 2: 25, 4: 34, 5: 20, 6: 36, 10: 21, 23: 35}, "famille_a"},
		{"06dfe6d9", map[int]int{1: 19, 2: 25, 4: 35, 5: 22, 6: 38, 8: 2, 9: 2, 10: 34, 11: 3, 12: 4, 23: 35}, "famille_a"},
		{"00162144", map[int]int{2: 14, 4: 9, 9: 2, 10: 10}, "famille_a"},
		// 96,2 % de pureté : quatre lectures `19` et une `44` hors bande, bruit du balayage.
		{"084a804d", map[int]int{1: 13, 4: 48, 5: 11, 6: 18, 8: 10, 9: 8, 10: 2, 19: 4, 23: 15, 44: 1}, "famille_a"},
		{"000d5950", map[int]int{19: 18, 20: 22, 21: 26, 22: 16}, "famille_b"},
		{"00502e52", map[int]int{19: 22, 20: 17, 21: 8, 22: 18}, "famille_b"},
		{"07aa428d", map[int]int{19: 11, 20: 10, 21: 13, 22: 8}, "famille_b"},
	}
	for _, c := range cas {
		got := classifyAbilityPalette(readsOf(c.hist), palettes)
		if got == nil || got.ID != c.want {
			t.Errorf("film %s classé %q, attendu %q", c.film, paletteIDOrNone(got), c.want)
		}
	}
}

// TestClassifyAbilityPaletteRefuseCeQuElleNeSaitPas — le refus EST la fonctionnalité.
func TestClassifyAbilityPaletteRefuseCeQuElleNeSaitPas(t *testing.T) {
	palettes := []AbilityPalette{famA, famB}
	cas := []struct {
		nom  string
		hist map[int]int
	}{
		// Moitié-moitié : nommer reviendrait à tirer à pile ou face sur le sens d'un rang.
		{"signature mélangée", map[int]int{4: 10, 20: 10}},
		// 80 % : au-dessus de la moitié, mais sous le seuil — la marge est là pour ça.
		{"majorité trop faible", map[int]int{4: 8, 20: 2}},
		// Moins de 10 lectures et PAS unanime (une d'une autre famille) : le regime sous le
		// plancher EXIGE l'unanimite, il ne relache rien (RAPPORT_E0_2026-09-10 §2).
		{"corpus maigre et non unanime", map[int]int{4: 8, 20: 1}},
		// Aucun marqueur connu : un film d'une palette qu'on n'a jamais vue.
		{"rangs inconnus", map[int]int{30: 20, 31: 15}},
	}
	for _, c := range cas {
		if got := classifyAbilityPalette(readsOf(c.hist), palettes); got != nil {
			t.Errorf("%s : classé %q, attendu AUCUN classement", c.nom, got.ID)
		}
	}
	if classifyAbilityPalette(readsOf(map[int]int{4: 20}), nil) != nil {
		t.Error("sans palette declaree, rien ne se classe")
	}
}

// TestClassifyAbilityPaletteUnanimiteSousLePlancher — RAPPORT_E0_2026-09-10 §2 : sous le
// plancher historique de 10 lectures, on n'assouplit pas le critère, on l'EXIGE ENTIER. La
// cause prouvée du rapport est structurelle (le canal image-clé ne voit QUE les rangs
// 16..23, donc un film de famille A ne récolte jamais de lecture `kf` et reste rare) — sept
// films du parc à n <= 7, tous à 100 % de pureté famille A, doivent désormais se classer.
// Vérifié sur les 64 films du corpus : 0 reclassement fautif, 0 perte (le film réellement
// mélangé, `4f77afc1`, reste refusé — cf. TestClassifyAbilityPaletteSurLesSignaturesMESUREES
// et TestClassifyAbilityPaletteRefuseCeQuElleNeSaitPas pour les cas qui doivent RESTER non
// classés).
func TestClassifyAbilityPaletteUnanimiteSousLePlancher(t *testing.T) {
	palettes := []AbilityPalette{famA, famB}
	// 3 lectures UNANIMES (toutes famille A) : classée.
	if got := classifyAbilityPalette(readsOf(map[int]int{4: 3}), palettes); got == nil || got.ID != "famille_a" {
		t.Errorf("3 lectures unanimes doivent classer en famille_a, obtenu %q", paletteIDOrNone(got))
	}
	// 3 lectures dont UNE d'une autre famille : n'est PAS unanime, refusée.
	if got := classifyAbilityPalette(readsOf(map[int]int{4: 2, 20: 1}), palettes); got != nil {
		t.Errorf("3 lectures non unanimes (une autre famille) ne doivent PAS se classer, obtenu %q", got.ID)
	}
	// n=1, le plancher le plus bas du parc (`b0fe12b1`, une seule lecture i48, 100 % famille
	// A) : la généralisation vaut aussi là — la découverte de la lecture unique en elle-même
	// reste une anomalie distincte (hors périmètre), mais la RÈGLE de classement, elle,
	// s'applique uniformément.
	if got := classifyAbilityPalette(readsOf(map[int]int{4: 1}), palettes); got == nil || got.ID != "famille_a" {
		t.Errorf("1 lecture unanime doit classer, obtenu %q", paletteIDOrNone(got))
	}
}

// TestClassifyAbilityPaletteSeuilInchangeAuDessusDuPlancher — le régime n >= 10 (pureté
// >= 90 %) ne bouge pas : 91 % classe, 85 % non, exactement le seuil actuel.
func TestClassifyAbilityPaletteSeuilInchangeAuDessusDuPlancher(t *testing.T) {
	palettes := []AbilityPalette{famA, famB}
	if got := classifyAbilityPalette(readsOf(map[int]int{4: 91, 20: 9}), palettes); got == nil || got.ID != "famille_a" {
		t.Errorf("91%% de purete (n=100 >= 10) doit classer, obtenu %q", paletteIDOrNone(got))
	}
	if got := classifyAbilityPalette(readsOf(map[int]int{4: 85, 20: 15}), palettes); got != nil {
		t.Errorf("85%% de purete (n=100 >= 10) ne doit PAS classer, obtenu %q", got.ID)
	}
}

// TestAbilityPaletteSeuilNonSensible : le classement du corpus ne tient pas au seuil.
//
// C'est ce qui distingue une règle d'un réglage choisi après coup. Six films sont purs à
// 100 % et le septième à 96,2 % : tout seuil de 50 % à 96 % rend le MÊME verdict.
func TestAbilityPaletteSeuilNonSensible(t *testing.T) {
	if abilityPalettePurity < 0.50 || abilityPalettePurity > 0.96 {
		t.Fatalf("seuil %.2f hors de la plage ou le corpus est insensible [0,50 ; 0,96]",
			abilityPalettePurity)
	}
	// Le minimum de lectures est DERIVE du seuil : c'est le plus petit n tel qu'UNE lecture
	// parasite ne disqualifie pas a elle seule un film pur.
	n := float64(abilityPaletteMinReads)
	if (n-1)/n < abilityPalettePurity {
		t.Errorf("minimum de %d lectures incohérent avec un seuil de %.2f : une seule lecture "+
			"parasite y suffirait à refuser un film pur", abilityPaletteMinReads, abilityPalettePurity)
	}
	if m := n - 1; m >= 1 && (m-1)/m >= abilityPalettePurity {
		t.Errorf("minimum de %d lectures trop haut : %d suffirait", abilityPaletteMinReads,
			abilityPaletteMinReads-1)
	}
}

func TestAbilityLabelsUsedNamesOnlyWhatItKnows(t *testing.T) {
	// LA TABLE EST PARTIELLE, et propre a la PALETTE du film. Un rang hors table garde son
	// numero a l'ecran — le combler par le nom d'une capacite voisine se lirait comme une
	// certitude. Le catalogue vient du TITRE (replay_labels.toml) et non d'une table Go : le
	// test l'injecte, comme le fait cmd/replay-build.
	reads := []AbilityRead{{R: 20}, {R: 9}}
	got := abilityLabelsUsed(reads, &famB)
	if got["20"].Fr != "grappin" || got["20"].En != "Grappleshot" {
		t.Errorf("rang connu non nomme dans les deux langues : %+v", got)
	}
	if _, named := got["9"]; named {
		t.Error("un rang hors table ne doit PAS etre nomme")
	}
	if abilityLabelsUsed(nil, &famB) != nil {
		t.Error("sans lecture, pas de table inventee")
	}
	// PALETTE NON CLASSEE : aucun nom, aucun panic. C'est le cas central de l'etape 2 —
	// un film dont la signature est ambigue sort avec ses rangs et sans un seul nom.
	if abilityLabelsUsed(reads, nil) != nil {
		t.Error("sans palette classee, aucune capacite ne doit etre nommee")
	}
}

// TestAbilityLabelsUsedPublieLaFamilleDuManifeste — LA TABLE RANG -> FAMILLE VOYAGE DANS
// LE DOCUMENT (lot 4.3, item 2).
//
// Sans elle, tout lecteur du document cuit devait reconstruire la famille depuis la RACINE
// du libellé — deux copies de la même table (Go et web), plafond de la règle n°6 du dépôt.
// Un rang que le manifeste nomme SANS lui donner de famille garde son libellé et n'en
// invente pas : `Family` reste vide.
//
// MUTATION : ne pas recopier `palette.FamilyOf` -> la famille est vide sur un rang qui en
// a une, rouge.
func TestAbilityLabelsUsedPublieLaFamilleDuManifeste(t *testing.T) {
	palette := AbilityPalette{
		ID:       "test",
		Ranks:    map[int]Label{2: {En: "Drop Wall", Fr: "mur"}, 8: {En: "Camo", Fr: "camo"}},
		Families: map[int]string{2: "wall"},
	}
	got := abilityLabelsUsed([]AbilityRead{{R: 2}, {R: 8}}, &palette)
	if got["2"].Family != "wall" {
		t.Errorf("rang 2 : family = %q, attendu \"wall\" — la table du manifeste doit voyager",
			got["2"].Family)
	}
	if got["8"].Family != "" {
		t.Errorf("rang 8 : family = %q, attendu vide — un rang sans famille au manifeste n'en "+
			"invente pas", got["8"].Family)
	}
}
