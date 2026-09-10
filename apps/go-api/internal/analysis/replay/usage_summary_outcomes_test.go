package replay

// usage_summary_outcomes_test.go — LES TROIS ISSUES D'UN OBJET PRIS, au grain du
// résumé (étape E3 du PLAN_EQUIPEMENT_GACHIS_2026-09-09). La règle testée ici est
// celle du web (equipmentUsageLogic.ts, livrée en E2), reproduite à l'identique :
// `kept = max(0, taken - utilisé - lâché)`, « utilisé » étant les ÉPISODES pour les
// deux bonus et les POSES DÉPLOYÉES pour tout le reste (décision P2).

import "testing"

// docIssuesTest — un document minimal qui porte les trois issues sur trois
// familles à la fois :
//
//	111 (slot 1) : 2 murs pris, 1 posé, 0 lâché      -> gardé 1
//	111          : 2 camouflages pris, 1 épisode, 1 lâché -> gardé 0
//	222 (slot 2) : 3 capteurs pris, 0 posé, 1 lâché  -> gardé 2
//	222          : 1 prise de rang NON NOMMÉ         -> aucune famille, réserve
func docIssuesTest() *ReplayDocument {
	return &ReplayDocument{
		SchemaVersion:   SchemaVersion,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		DurationMS:      100000,
		Roster: []RosterEntry{
			{XUID: "111", FilmIndex: 0},
			{XUID: "222", FilmIndex: 1},
		},
		Tracks: []Track{
			{Slot: 1, XUID: "111", StartFrame: 0, EndFrame: 500},
			{Slot: 2, XUID: "222", StartFrame: 0, EndFrame: 500},
		},
		// La palette du film nomme quatre rangs ET publie leur famille (schéma 51) ; le
		// rang 77 n'y est pas. La jointure du bilan lit `Family`, jamais le texte.
		AbilityLabels: map[string]Label{
			"2": {En: "Drop Wall", Fr: "mur de protection", Family: usageFamilyWall},
			"1": {En: "Threat Sensor", Fr: "detecteur de menaces", Family: usageFamilySensor},
			"8": {En: "Active Camouflage", Fr: "camouflage actif",
				Family: usageFamilyPowerupCamo},
			"4": {En: "Grappleshot", Fr: "grappin", Family: usageFamilyGrapple},
		},
		EquipmentEpisodes: []EquipmentEpisode{
			{Slot: 1, T0: 100, T1: 150, Fam: EquipFamilyCamo},
		},
		EquipmentPlacements: []EquipmentPlacement{
			// Mur DÉPLOYÉ par 111 : seul un PANNEAU compte (usageWallPanelIDs).
			{Owner: 1, T0: 120, Family: usageFamilyWall, Origin: OriginDeployed, ID: "0x528fce46"},
			// Camouflage LÂCHÉ par 111 (vocabulaire de POSE : powerup_camo).
			{Owner: 1, T0: 300, Family: usageFamilyPowerupCamo, Origin: OriginDropped},
			// Capteur LÂCHÉ par 222.
			{Owner: 2, T0: 320, Family: usageFamilySensor, Origin: OriginDropped},
		},
		EquipmentChanges: []EquipmentChange{
			{Slot: 1, T: 50, Kind: EquipmentTaken, R: 2},  // mur
			{Slot: 1, T: 60, Kind: EquipmentTaken, R: 2},  // mur
			{Slot: 1, T: 70, Kind: EquipmentTaken, R: 8},  // camouflage
			{Slot: 1, T: 80, Kind: EquipmentTaken, R: 8},  // camouflage
			{Slot: 2, T: 90, Kind: EquipmentTaken, R: 1},  // capteur
			{Slot: 2, T: 91, Kind: EquipmentTaken, R: 1},  // capteur
			{Slot: 2, T: 92, Kind: EquipmentTaken, R: 1},  // capteur
			{Slot: 2, T: 93, Kind: EquipmentTaken, R: 77}, // rang SANS libellé : réserve
			// Une consommation : son rang est sur `From`, jamais sur `R`.
			{Slot: 1, T: 200, Kind: EquipmentSpent, R: NoAbilityRank, From: 2},
			// Consommation dont la chaîne est trouée (`gap`) : `from` n'est pas
			// une identité fiable — elle ne doit ventiler AUCUNE famille.
			{Slot: 2, T: 210, Kind: EquipmentSpent, R: NoAbilityRank, From: 1, Gap: 2},
			// Le grappin est NOMMÉ mais HORS BILAN (P4, comme le répulseur et le
			// propulseur) : ni ligne, ni réserve.
			{Slot: 1, T: 220, Kind: EquipmentTaken, R: 4},
		},
	}
}

func joueur(t *testing.T, s UsageSummary, xuid string) *UsagePlayerSummary {
	t.Helper()
	for i := range s.Players {
		if s.Players[i].XUID == xuid {
			return &s.Players[i]
		}
	}
	t.Fatalf("le joueur %s n'a pas de ligne", xuid)
	return nil
}

// TestUsageSummary_IssuesParFamille — la règle E2 reproduite : les prises ventilées
// par famille, et le gardé DÉRIVÉ de taken - utilisé - lâché.
func TestUsageSummary_IssuesParFamille(t *testing.T) {
	s := BuildUsageSummary(docIssuesTest())

	p111 := joueur(t, s, "111")
	if got := p111.TakenByFamily[usageFamilyWall]; got != 2 {
		t.Errorf("TakenByFamily[wall](111) = %d, attendu 2", got)
	}
	if got := p111.TakenByFamily[usageFamilyPowerupCamo]; got != 2 {
		t.Errorf("TakenByFamily[powerup_camo](111) = %d, attendu 2", got)
	}
	// Le grappin est hors bilan : aucune famille ne le nomme.
	if got, ok := p111.TakenByFamily[usageFamilyGrapple]; ok {
		t.Errorf("TakenByFamily[grapple](111) = %d, attendu ABSENT (hors bilan, P4)", got)
	}
	// Mur : 2 pris, 1 posé (le panneau), 0 lâché -> 1 gardé.
	if got := p111.KeptByFamily[usageFamilyWall]; got != 1 {
		t.Errorf("KeptByFamily[wall](111) = %d, attendu 1 (2 pris - 1 posé - 0 lâché)", got)
	}
	// Camouflage : 2 pris, 1 épisode (le côté « utilisé » des bonus, P2), 1 lâché -> 0 gardé.
	if got, ok := p111.KeptByFamily[usageFamilyPowerupCamo]; ok && got != 0 {
		t.Errorf("KeptByFamily[powerup_camo](111) = %d, attendu 0 (2 - 1 épisode - 1 lâché)", got)
	}
	// La consommation ventile la famille de `From`.
	if got := p111.SpentByFamily[usageFamilyWall]; got != 1 {
		t.Errorf("SpentByFamily[wall](111) = %d, attendu 1", got)
	}

	p222 := joueur(t, s, "222")
	if got := p222.TakenByFamily[usageFamilySensor]; got != 3 {
		t.Errorf("TakenByFamily[sensor](222) = %d, attendu 3", got)
	}
	if got := p222.KeptByFamily[usageFamilySensor]; got != 2 {
		t.Errorf("KeptByFamily[sensor](222) = %d, attendu 2 (3 pris - 0 posé - 1 lâché)", got)
	}
	// La consommation trouée (`gap`) ne ventile rien.
	if got, ok := p222.SpentByFamily[usageFamilySensor]; ok {
		t.Errorf("SpentByFamily[sensor](222) = %d, attendu ABSENT (chaîne trouée : `from` non fiable)", got)
	}
}

// TestUsageSummary_LachersVentiles — DroppedByFamily est désormais une grandeur
// PERSISTÉE, et sa somme reste égale à DroppedObjects (invariant historique).
func TestUsageSummary_LachersVentiles(t *testing.T) {
	s := BuildUsageSummary(docIssuesTest())
	total := 0
	for i := range s.Players {
		p := &s.Players[i]
		somme := 0
		for _, n := range p.DroppedByFamily {
			somme += n
		}
		if somme != p.DroppedObjects {
			t.Errorf("%s : somme(DroppedByFamily) = %d != DroppedObjects = %d", p.XUID, somme, p.DroppedObjects)
		}
		total += somme
	}
	if total != 2 {
		t.Errorf("lâchers totaux = %d, attendu 2 (un camouflage, un capteur)", total)
	}
}

// TestUsageSummary_CouvertureDesPrises — les prises que le canal ne sait pas
// rattacher sont COMPTÉES, jamais rangées dans une famille inventée.
func TestUsageSummary_CouvertureDesPrises(t *testing.T) {
	s := BuildUsageSummary(docIssuesTest())
	if s.Match.EquipmentChanges.UnnamedRankTaken != 1 {
		t.Errorf("UnnamedRankTaken = %d, attendu 1 (le rang 77)", s.Match.EquipmentChanges.UnnamedRankTaken)
	}
	// Le grappin est nommé : il n'entre PAS dans la réserve des rangs muets.
	if s.Match.EquipmentChanges.UnnamedRankTaken > 1 {
		t.Errorf("une famille NOMMÉE mais hors bilan a fui dans la réserve des rangs muets")
	}
	if s.Match.EquipmentChanges.UnattributedSlot != 0 {
		t.Errorf("UnattributedSlot = %d, attendu 0 (les deux slots sont publiés)",
			s.Match.EquipmentChanges.UnattributedSlot)
	}
}

// TestUsageSummary_SlotNonRattache — un changement dont le slot n'ouvre aucune
// ligne (vie anonyme, bot) est compté à la couverture, jamais crédité à un voisin.
func TestUsageSummary_SlotNonRattache(t *testing.T) {
	doc := docIssuesTest()
	doc.EquipmentChanges = append(doc.EquipmentChanges,
		EquipmentChange{Slot: 42, T: 400, Kind: EquipmentTaken, R: 2})
	s := BuildUsageSummary(doc)
	if s.Match.EquipmentChanges.UnattributedSlot != 1 {
		t.Errorf("UnattributedSlot = %d, attendu 1 (le slot 42 n'a aucune vie publiée)",
			s.Match.EquipmentChanges.UnattributedSlot)
	}
	// Et surtout : personne n'a hérité de cette prise.
	total := 0
	for i := range s.Players {
		total += s.Players[i].TakenByFamily[usageFamilyWall]
	}
	if total != 2 {
		t.Errorf("prises de mur attribuées = %d, attendu 2 — le slot orphelin a été crédité à quelqu'un", total)
	}
}

// TestUsageSummary_GardeJamaisNegatif — l'écart résiduel mesuré par E0.4 (2,45 %
// toutes familles) est absorbé par le clamp à zéro, jamais affiché en négatif.
func TestUsageSummary_GardeJamaisNegatif(t *testing.T) {
	doc := docIssuesTest()
	// Quatre poses de mur pour deux prises : une pose est une CHARGE, pas un objet
	// (piège d'unité n°1 de la référence des canaux).
	for i := 0; i < 3; i++ {
		doc.EquipmentPlacements = append(doc.EquipmentPlacements, EquipmentPlacement{
			Owner: 1, T0: 130 + i, Family: usageFamilyWall,
			Origin: OriginDeployed, ID: "0x528fce46",
		})
	}
	s := BuildUsageSummary(doc)
	p111 := joueur(t, s, "111")
	if got, ok := p111.KeptByFamily[usageFamilyWall]; ok && got != 0 {
		t.Errorf("KeptByFamily[wall] = %d, attendu 0 (clampé) — un gardé négatif a été publié", got)
	}
}

// TestUsageSummary_FamillesDuBilan — la liste exportée est le vocabulaire de POSE
// (celui de DeployedByFamily / DroppedByFamily), pas celui des épisodes : c'est ce
// qui permet à l'agrégat de session de joindre les quatre ventilations sur UNE clé.
func TestUsageSummary_FamillesDuBilan(t *testing.T) {
	fams := EquipmentOutcomeFamilies()
	attendues := map[string]bool{
		usageFamilyWall: true, usageFamilySensor: true, "translocator_beacon": true,
		"shroud_screen": true, "threat_seeker": true, "repair_field": true,
		usageFamilyPowerupCamo: true, usageFamilyPowerupOvershield: true,
	}
	if len(fams) != len(attendues) {
		t.Fatalf("EquipmentOutcomeFamilies() = %v, attendu %d familles", fams, len(attendues))
	}
	for _, f := range fams {
		if !attendues[f] {
			t.Errorf("famille inattendue au bilan : %q", f)
		}
	}
	// Le répulseur, le grappin et le propulseur n'y sont PAS (P4 et hors bilan).
	for _, f := range fams {
		if f == usageFamilyRepulsor || f == usageFamilyGrapple || f == usageFamilyThruster {
			t.Errorf("%q ne doit pas avoir de ligne d'issue (négatif mesuré / hors bilan)", f)
		}
	}
}

// TestUsageSummary_JointureSurLaFamillePubliee — LE RÉSUMÉ JOINT SUR LA FAMILLE CUITE,
// PLUS SUR LA RACINE DU LIBELLÉ (lot 4.3, item 2).
//
// Les deux moitiés du test se contredisent sous l'ancienne règle, et c'est le but :
//   - un rang dont le LIBELLÉ ne ressemble à rien mais qui PORTE la famille est classé ;
//   - un rang dont le libellé dit « mur de protection » mais qui ne porte AUCUNE famille
//     ne l'est pas — il est nommé, donc hors réserve, et hors bilan.
//
// MUTATION : rétablir la reconnaissance par racine de libellé -> la première moitié
// tombe dans le vide (aucune racine ne dit « xyzzy ») et la seconde classe un mur, rouge.
func TestUsageSummary_JointureSurLaFamillePubliee(t *testing.T) {
	doc := docIssuesTest()
	doc.AbilityLabels = map[string]Label{
		// Libellé opaque, famille publiée : la jointure passe par la FAMILLE.
		"2": {En: "xyzzy", Fr: "xyzzy", Family: usageFamilyWall},
		// Libellé parlant, aucune famille : le manifeste ne le classe pas, donc nous non plus.
		"1": {En: "Drop Wall", Fr: "mur de protection"},
		"8": {En: "Active Camouflage", Fr: "camouflage actif", Family: usageFamilyPowerupCamo},
		"4": {En: "Grappleshot", Fr: "grappin", Family: usageFamilyGrapple},
	}
	s := BuildUsageSummary(doc)
	if got := joueur(t, s, "111").TakenByFamily[usageFamilyWall]; got != 2 {
		t.Errorf("TakenByFamily[wall](111) = %d, attendu 2 — la famille PUBLIÉE fait foi, "+
			"pas la racine du libellé", got)
	}
	p222 := joueur(t, s, "222")
	if got, ok := p222.TakenByFamily[usageFamilyWall]; ok {
		t.Errorf("TakenByFamily[wall](222) = %d, attendu ABSENT — le rang 1 est NOMMÉ mais "+
			"sans famille au manifeste : hors bilan, jamais reclassé par son texte", got)
	}
	// NOMMÉ SANS FAMILLE N'EST PAS MUET : la réserve ne compte que le rang 77.
	if got := s.Match.EquipmentChanges.UnnamedRankTaken; got != 1 {
		t.Errorf("UnnamedRankTaken = %d, attendu 1 (le seul rang 77) — un rang nommé sans "+
			"famille a fui dans la réserve des rangs muets", got)
	}
}

// TestUsageSummary_UtiliseLuSurLesConsommations — LE DÉFAUT PROUVÉ PAR LE RAPPORT E0
// (question 5, 2026-09-10) : pour un déployable qui n'engendre AUCUNE pièce, une pose
// `origin: deployed` mesure un LÂCHER VOLONTAIRE à mi-vie, pas un déploiement — zéro
// de ses 202 consommations du parc n'est couverte par une pose (contre 84 % pour le
// mur). Le côté « utilisé » de ces familles se lit donc sur les CONSOMMATIONS.
//
// MUTATION : rétablir `DeployedByFamily` pour le capteur -> le gardé remonte à 1
// (3 pris - 1 pose - 1 lâché), rouge. C'est exactement l'état d'avant ce lot.
func TestUsageSummary_UtiliseLuSurLesConsommations(t *testing.T) {
	doc := docIssuesTest()
	// La consommation de capteur du fixture porte une chaîne trouée : on la répare, et on
	// en ajoute une seconde — 222 consomme DEUX charges de capteur sur ses trois prises.
	for i := range doc.EquipmentChanges {
		if c := &doc.EquipmentChanges[i]; c.Kind == EquipmentSpent && c.From == 1 {
			c.Gap = 0
		}
	}
	doc.EquipmentChanges = append(doc.EquipmentChanges,
		EquipmentChange{Slot: 2, T: 230, Kind: EquipmentSpent, R: NoAbilityRank, From: 1})
	// Et UNE pose `deployed` de capteur : elle reste comptée là où elle l'était
	// (DeployedByFamily), mais elle ne vaut plus « utilisé ».
	doc.EquipmentPlacements = append(doc.EquipmentPlacements, EquipmentPlacement{
		Owner: 2, T0: 240, Family: usageFamilySensor, Origin: OriginDeployed,
	})

	s := BuildUsageSummary(doc)
	p222 := joueur(t, s, "222")
	if got := p222.SpentByFamily[usageFamilySensor]; got != 2 {
		t.Fatalf("SpentByFamily[sensor](222) = %d, attendu 2 — le fixture du test est faux", got)
	}
	if got := p222.DeployedByFamily[usageFamilySensor]; got != 1 {
		t.Fatalf("DeployedByFamily[sensor](222) = %d, attendu 1 — la pose reste comptée", got)
	}
	if got := p222.KeptByFamily[usageFamilySensor]; got != 0 {
		t.Errorf("KeptByFamily[sensor](222) = %d, attendu 0 (3 pris - 2 consommées - 1 lâché) — "+
			"le côté « utilisé » d'un déployable sans pièce engendrée se lit sur les "+
			"CONSOMMATIONS, jamais sur ses poses (rapport E0 question 5)", got)
	}
}

// TestUsageSummary_MurLuSurSesPanneaux — LA SEULE EXCEPTION, et elle est mesurée : le
// mur est le seul équipement du manifeste qui ENGENDRE UNE PIÈCE DISTINCTE
// (`kind = "deployed"`, les panneaux), et 84 % de ses consommations tombent sur une de
// ses poses. Son « utilisé » reste donc ses poses de panneau.
//
// MUTATION : basculer le mur sur les consommations -> gardé 0 au lieu de 1, rouge.
func TestUsageSummary_MurLuSurSesPanneaux(t *testing.T) {
	doc := docIssuesTest()
	// DEUX consommations de mur pour UN panneau posé : les deux lectures divergent, et
	// c'est ce qui rend ce test discriminant.
	doc.EquipmentChanges = append(doc.EquipmentChanges,
		EquipmentChange{Slot: 1, T: 205, Kind: EquipmentSpent, R: NoAbilityRank, From: 2})

	s := BuildUsageSummary(doc)
	p111 := joueur(t, s, "111")
	if got := p111.SpentByFamily[usageFamilyWall]; got != 2 {
		t.Fatalf("SpentByFamily[wall](111) = %d, attendu 2 — le fixture du test est faux", got)
	}
	if got := p111.KeptByFamily[usageFamilyWall]; got != 1 {
		t.Errorf("KeptByFamily[wall](111) = %d, attendu 1 (2 pris - 1 panneau posé - 0 lâché) — "+
			"le mur engendre une pièce, son « utilisé » reste ses POSES", got)
	}
}

// TestUsageSummary_BonusRestentSurLeursEpisodes — les deux bonus ne bougent pas : leur
// « utilisé » est l'ÉPISODE d'état actif (décision P2), et le film annonce pourtant des
// consommations pour eux (54 camouflages et 34 surboucliers au parc, mesure E0).
//
// MUTATION : basculer les bonus sur les consommations -> gardé 0 au lieu de 1, rouge.
func TestUsageSummary_BonusRestentSurLeursEpisodes(t *testing.T) {
	doc := docIssuesTest()
	// Une troisième prise de camouflage, et DEUX consommations annoncées par le film.
	doc.EquipmentChanges = append(doc.EquipmentChanges,
		EquipmentChange{Slot: 1, T: 85, Kind: EquipmentTaken, R: 8},
		EquipmentChange{Slot: 1, T: 240, Kind: EquipmentSpent, R: NoAbilityRank, From: 8},
		EquipmentChange{Slot: 1, T: 250, Kind: EquipmentSpent, R: NoAbilityRank, From: 8})

	s := BuildUsageSummary(doc)
	p111 := joueur(t, s, "111")
	if got := p111.SpentByFamily[usageFamilyPowerupCamo]; got != 2 {
		t.Fatalf("SpentByFamily[powerup_camo](111) = %d, attendu 2 — le fixture du test est faux", got)
	}
	if got := p111.KeptByFamily[usageFamilyPowerupCamo]; got != 1 {
		t.Errorf("KeptByFamily[powerup_camo](111) = %d, attendu 1 (3 pris - 1 épisode - 1 lâché) — "+
			"un bonus s'utilise en s'ACTIVANT, ses consommations ne comptent pas deux fois", got)
	}
}

// TestUsageSummary_ConsommationsAuDelaDesPrises — l'identité `taken = utilisé + lâché +
// gardé` reste BORNÉE quand le nouveau canal déborde : une charge se consomme plusieurs
// fois par objet (piège d'unité n°1 de la référence des canaux) et l'équipement de
// RÉAPPARITION se consomme sans jamais être `taken`. Le clamp tient, le gardé ne devient
// jamais négatif.
func TestUsageSummary_ConsommationsAuDelaDesPrises(t *testing.T) {
	doc := docIssuesTest()
	for i := range doc.EquipmentChanges {
		if c := &doc.EquipmentChanges[i]; c.Kind == EquipmentSpent && c.From == 1 {
			c.Gap = 0
		}
	}
	// Cinq consommations de capteur pour trois prises et un lâcher.
	for i := 0; i < 4; i++ {
		doc.EquipmentChanges = append(doc.EquipmentChanges, EquipmentChange{
			Slot: 2, T: 230 + i, Kind: EquipmentSpent, R: NoAbilityRank, From: 1,
		})
	}

	s := BuildUsageSummary(doc)
	p222 := joueur(t, s, "222")
	if got := p222.SpentByFamily[usageFamilySensor]; got != 5 {
		t.Fatalf("SpentByFamily[sensor](222) = %d, attendu 5 — le fixture du test est faux", got)
	}
	if got, ok := p222.KeptByFamily[usageFamilySensor]; !ok || got != 0 {
		t.Errorf("KeptByFamily[sensor](222) = %d (présent=%v), attendu 0 — 3 - 5 - 1 doit se "+
			"clamper, jamais publier un gardé négatif", got, ok)
	}
}
