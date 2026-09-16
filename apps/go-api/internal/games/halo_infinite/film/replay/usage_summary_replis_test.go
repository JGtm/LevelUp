package replay

// usage_summary_replis_test.go — LES TROIS REPLIS DU RESUME D'USAGE, ET LEUR COMPTE.
//
// # POURQUOI CE FICHIER EXISTE (revue de jalon M1, lentille L3, 2026-09-16)
//
// Les trois entrees du registre que ce fichier couvre —
// `repli_geste_dernier_occupant_du_match`, `repli_geste_premiere_vie_du_slot`,
// `repli_garde_equipement_negatif_a_zero` — etaient au registre avec
// `CompteurBranche: false` et une cible de comptage qui nommait le lot 1.9.13, DEJA FUSIONNE
// sans les avoir cablees. Leur compte etait donc INCONNU pendant que les trois decidaient un
// fait publie (a qui crediter une pose, une traction, un episode ; combien d'objets gardes).
// D14 (d) fait supprimer un repli dont le compte est a zero : sans compteur, un zero se lit
// comme « jamais declenche » alors qu'il ne dit que « jamais instrumente ».
//
// # CE QUE CHAQUE TEST TIENT
//
// Un test par site, sur une fixture qui le DECLENCHE et sur une fixture voisine qui ne le
// declenche pas — un compteur qui monte toujours ne mesure rien. Puis la MESURE sur les huit
// fixtures d'assemblage par build, qui rend le chiffre que la cible de retrait demande.
//
// # MUTATION QUI DOIT LES FAIRE ROUGIR
//
// Retirer l'appel `o.fb.Declenche(fallback.NomGesteDernierOccupantDuMatch)` de
// `usageOwners.repliDernierOccupant` : `TestRepliGesteDernierOccupantSeCompte` rougit
// (« = 0, attendu 1 »). Jouee et restauree par NOM le 2026-09-16.

import (
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
)

// compteDeRepli lit le compte d'un repli dans le rapport d'une projection.
func compteDeRepli(s UsageSummary, nom fallback.Nom) int {
	for _, d := range s.Match.Fallbacks {
		if d.Nom == nom {
			return d.Declenchements
		}
	}
	return 0
}

// docGesteHorsDeTouteVie — un geste date APRES la derniere vie publiee du slot : aucune vie ne
// couvre l'image, le dernier occupant du slot est credite.
func docGesteHorsDeTouteVie(t0 int) *ReplayDocument {
	return &ReplayDocument{
		SchemaVersion:   SchemaVersion,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		Roster:          []RosterEntry{{XUID: "111", FilmIndex: 0}},
		Tracks:          []Track{{Slot: 1, XUID: "111", StartFrame: 0, EndFrame: 100}},
		GrappleLines:    []GrappleLine{{Slot: 1, T0: t0, T1: t0 + 5}},
	}
}

func TestRepliGesteDernierOccupantSeCompte(t *testing.T) {
	// DECLENCHE : la traction est a la frame 500, la seule vie du slot s'arrete a 100.
	s := BuildUsageSummary(docGesteHorsDeTouteVie(500))
	if got := compteDeRepli(s, fallback.NomGesteDernierOccupantDuMatch); got != 1 {
		t.Errorf("repli_geste_dernier_occupant_du_match = %d, attendu 1 — le site de "+
			"`usageOwners.at` ne compte pas", got)
	}
	// La traction est tout de meme creditee : le repli DECIDE, il ne jette rien.
	if len(s.Players) != 1 || s.Players[0].GrapplePulls != 1 {
		t.Errorf("la traction n'est pas creditee au dernier occupant : %+v", s.Players)
	}

	// NE DECLENCHE PAS : la meme traction DANS la vie se lit, elle ne se replie pas.
	s = BuildUsageSummary(docGesteHorsDeTouteVie(50))
	if got := compteDeRepli(s, fallback.NomGesteDernierOccupantDuMatch); got != 0 {
		t.Errorf("repli_geste_dernier_occupant_du_match = %d sur un geste COUVERT par sa vie, "+
			"attendu 0 — un compteur qui monte toujours ne mesure rien", got)
	}
}

func TestRepliGesteDernierOccupantNeComptePasSansOccupantConnu(t *testing.T) {
	// UNE VIE ANONYME N'OUVRE AUCUNE LIGNE ET NE NOURRIT PAS `dernier` : le repli est
	// TRAVERSE sans rien decider. Le compter ferait croire a un fait publie par un repli.
	doc := docGesteHorsDeTouteVie(500)
	doc.Roster = nil
	doc.Tracks = []Track{{Slot: 1, StartFrame: 0, EndFrame: 100}}
	s := BuildUsageSummary(doc)
	if got := compteDeRepli(s, fallback.NomGesteDernierOccupantDuMatch); got != 0 {
		t.Errorf("repli_geste_dernier_occupant_du_match = %d alors qu'aucun occupant n'est "+
			"connu, attendu 0 : le repli n'a decide aucun fait", got)
	}
}

// docPoseAvantLaPremiereVie — une pose datee AVANT la premiere vie publiee du slot.
func docPoseAvantLaPremiereVie(t0 int) *ReplayDocument {
	return &ReplayDocument{
		SchemaVersion:   SchemaVersion,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		Roster:          []RosterEntry{{XUID: "111", FilmIndex: 0}},
		Tracks:          []Track{{Slot: 1, XUID: "111", StartFrame: 200, EndFrame: 300}},
		EquipmentPlacements: []EquipmentPlacement{
			{Owner: 1, T0: t0, T1: t0 + 5, Family: usageFamilySensor, Origin: OriginDropped},
		},
	}
}

func TestRepliGestePremiereVieDuSlotSeCompte(t *testing.T) {
	// DECLENCHE : la pose est a la frame 10, la premiere vie du slot commence a 200.
	s := BuildUsageSummary(docPoseAvantLaPremiereVie(10))
	if got := compteDeRepli(s, fallback.NomGestePremiereVieDuSlot); got != 1 {
		t.Errorf("repli_geste_premiere_vie_du_slot = %d, attendu 1 — le site de "+
			"`usageOwners.atOrJustBefore` ne compte pas", got)
	}
	// NE DECLENCHE PAS : une pose DANS la vie se lit.
	s = BuildUsageSummary(docPoseAvantLaPremiereVie(250))
	if got := compteDeRepli(s, fallback.NomGestePremiereVieDuSlot); got != 0 {
		t.Errorf("repli_geste_premiere_vie_du_slot = %d sur une pose COUVERTE par sa vie, "+
			"attendu 0", got)
	}
}

// docGardeNegative — un joueur qui PREND une fois le camouflage et l'ACTIVE deux fois :
// `kept = 1 - 2 - 0 = -1`, la contradiction que le clamp ecrase.
func docGardeNegative(episodes int) *ReplayDocument {
	doc := &ReplayDocument{
		SchemaVersion:   SchemaVersion,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		Roster:          []RosterEntry{{XUID: "111", FilmIndex: 0}},
		Tracks:          []Track{{Slot: 1, XUID: "111", StartFrame: 0, EndFrame: 900}},
		AbilityLabels:   map[string]Label{"8": {En: "Camo", Fr: "Camo", Family: usageFamilyPowerupCamo}},
		EquipmentChanges: []EquipmentChange{
			{Slot: 1, T: 10, Kind: EquipmentTaken, R: 8, From: NoAbilityRank},
		},
	}
	for i := 0; i < episodes; i++ {
		t0 := 100 + i*100
		doc.EquipmentEpisodes = append(doc.EquipmentEpisodes,
			EquipmentEpisode{Slot: 1, Fam: EquipFamilyCamo, T0: t0, T1: t0 + 10})
	}
	return doc
}

func TestRepliGardeEquipementNegatifAZeroSeCompte(t *testing.T) {
	// DECLENCHE : deux activations pour une seule prise.
	s := BuildUsageSummary(docGardeNegative(2))
	if got := compteDeRepli(s, fallback.NomGardeEquipementNegatifAZero); got != 1 {
		t.Fatalf("repli_garde_equipement_negatif_a_zero = %d, attendu 1 — le clamp de "+
			"`deriveUsageKept` ne compte pas", got)
	}
	if len(s.Players) != 1 || s.Players[0].KeptByFamily[usageFamilyPowerupCamo] != 0 {
		t.Errorf("la garde ecrasee doit valoir 0 : %+v", s.Players)
	}
	// NE DECLENCHE PAS : une prise, une activation, garde nulle SANS contradiction.
	s = BuildUsageSummary(docGardeNegative(1))
	if got := compteDeRepli(s, fallback.NomGardeEquipementNegatifAZero); got != 0 {
		t.Errorf("repli_garde_equipement_negatif_a_zero = %d sur une somme COHERENTE, "+
			"attendu 0", got)
	}
}

// TestReplisDuResumeDUsageSurLesHuitBuilds — LA MESURE que la cible de retrait demande.
//
// Elle tourne sur les huit fixtures d'assemblage par build (aucun octet de film lu) : le
// document est assemble comme en production, puis projete. Le test ne FIGE pas les chiffres —
// ils bougeront avec les fixtures — il exige seulement que le canal existe et il IMPRIME la
// table, qui est ce que D14 (d) relit pour decider d'un retrait sec.
func TestReplisDuResumeDUsageSurLesHuitBuilds(t *testing.T) {
	totaux := map[fallback.Nom]int{}
	for _, b := range goldenBuilds() {
		g, entry := chargerGoldenBuild(t, b)
		s := BuildUsageSummary(ptrDocument(assemblerGoldenBuild(t, b, g, entry)))
		ligne := make([]string, 0, 3)
		for _, d := range s.Match.Fallbacks {
			totaux[d.Nom] += d.Declenchements
			ligne = append(ligne, string(d.Nom))
		}
		sort.Strings(ligne)
		t.Logf("%-10s %-12s %d joueur(s) · replis : %v", b.Short8, b.Build, len(s.Players), ligne)
	}
	for _, nom := range []fallback.Nom{
		fallback.NomGesteDernierOccupantDuMatch,
		fallback.NomGestePremiereVieDuSlot,
		fallback.NomGardeEquipementNegatifAZero,
	} {
		if _, ok := fallback.Lire(nom); !ok {
			t.Errorf("%s n'est pas au registre", nom)
		}
		t.Logf("TOTAL 8 builds  %-45s %d", nom, totaux[nom])
	}
}

// ptrDocument rend l'adresse d'un document assemble — `BuildUsageSummary` prend un pointeur.
func ptrDocument(d ReplayDocument) *ReplayDocument { return &d }
