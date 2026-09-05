package replay

// usage_summary_test.go — la projection du résumé d'usage : les trois pièges du
// chantier (socle d'arme contre socle de bonus, grenades hors objets lâchés,
// panneaux du mur), les règles d'attribution (slot dernier-gagnant, index de film,
// xuid publié des socles), et les invariants sur le document RÉEL de la fixture
// figée (buildGolden — aucune fixture fabriquée, cf. §5.6 du handoff).

import "testing"

func strPtr(s string) *string { return &s }

// docSocleTest — un document minimal : deux joueurs, un socle d'ARME et un socle
// de BONUS, des occupations nommées et anonymes des deux côtés.
func docSocleTest() *ReplayDocument {
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
		WeaponPads: []WeaponPad{
			{Weapon: "0x11223344"},   // socle d'ARME (famille hexadécimale)
			{Weapon: "powerup_camo"}, // socle de BONUS (nom canonique)
		},
		PadPickups: []PadPickup{
			{Pad: 0, TLow: 10, THigh: 20, XUID: strPtr("111")}, // arme, nommée
			{Pad: 0, TLow: 30, THigh: 40, XUID: nil},           // arme, anonyme
			{Pad: 1, TLow: 50, THigh: 60, XUID: nil},           // bonus, anonyme (le cas réel)
			// LE PIÈGE, NOMMÉ : même si un xuid était un jour publié sur un socle
			// de bonus, il ne doit JAMAIS compter dans pad_pickups — la frontière
			// est PadWeaponFamilyKey, pas la présence d'un nom.
			{Pad: 1, TLow: 70, THigh: 80, XUID: strPtr("111")},
			{Pad: 9, TLow: 90, THigh: 95, XUID: strPtr("111")}, // index hors bornes
		},
	}
}

// TestUsageSummary_SocleArmeContreSocleBonus — l'assertion NOMINATIVE du piège
// central (§4 du handoff) : un socle powerup_camo ne compte JAMAIS dans
// pad_pickups ; ses occupations vont dans powerup_pad_pickups.
func TestUsageSummary_SocleArmeContreSocleBonus(t *testing.T) {
	s := BuildUsageSummary(docSocleTest())

	if s.Match.PadOccupancies != 5 {
		t.Errorf("PadOccupancies = %d, attendu 5 (toutes occupations, hors-bornes comprise)", s.Match.PadOccupancies)
	}
	if s.Match.PadNamed != 1 || s.Match.PadUnnamed != 1 {
		t.Errorf("PadNamed/PadUnnamed = %d/%d, attendu 1/1 (socles d'ARME seuls)",
			s.Match.PadNamed, s.Match.PadUnnamed)
	}
	if got := s.Match.PowerupPadPickups["powerup_camo"]; got != 2 {
		t.Errorf("PowerupPadPickups[powerup_camo] = %d, attendu 2 (les DEUX occupations du socle de bonus, xuid publié ou non)", got)
	}
	// Invariant de complétude : nommées + anonymes + bonus + hors-bornes = total.
	powerups := 0
	for _, n := range s.Match.PowerupPadPickups {
		powerups += n
	}
	if s.Match.PadNamed+s.Match.PadUnnamed+powerups+1 != s.Match.PadOccupancies {
		t.Errorf("ventilation des occupations incomplète : %d+%d+%d+1 != %d",
			s.Match.PadNamed, s.Match.PadUnnamed, powerups, s.Match.PadOccupancies)
	}

	// Le joueur 111 : UNE prise de socle d'arme — jamais celle du socle de bonus.
	var p111 *UsagePlayerSummary
	for i := range s.Players {
		if s.Players[i].XUID == "111" {
			p111 = &s.Players[i]
		}
	}
	if p111 == nil {
		t.Fatal("le joueur 111 n'a pas de ligne")
	}
	if p111.PadPickups != 1 {
		t.Errorf("pad_pickups(111) = %d, attendu 1 — une prise sur socle powerup_camo a fui dans les socles d'arme", p111.PadPickups)
	}
	// La clé est NORMALISÉE par PadWeaponFamilyKey (huit hexa minuscules, sans 0x) —
	// jamais la forme verbatim du document.
	if got := p111.PadPickupsByWeapon["11223344"]; got != 1 {
		t.Errorf("pad_pickups_by_weapon[11223344] = %d, attendu 1", got)
	}
	if _, ok := p111.PadPickupsByWeapon["powerup_camo"]; ok {
		t.Error("powerup_camo ne doit jamais apparaître dans pad_pickups_by_weapon")
	}

	// La liste des socles d'ARME ne porte pas le socle de bonus, et porte la clé
	// normalisée (même langue que pad_pickups_by_weapon — jointure directe).
	if len(s.Match.WeaponPads) != 1 || s.Match.WeaponPads[0].Weapon != "11223344" {
		t.Errorf("WeaponPads = %+v, attendu le seul socle d'arme 11223344 (clé normalisée)", s.Match.WeaponPads)
	}
	if s.Match.WeaponPads[0].Occupations != 2 || s.Match.WeaponPads[0].Named != 1 {
		t.Errorf("socle d'arme : occupations/named = %d/%d, attendu 2/1",
			s.Match.WeaponPads[0].Occupations, s.Match.WeaponPads[0].Named)
	}
}

// TestUsageSummary_GrenadesHorsObjetsLaches — une grenade lâchée à la mort n'est
// pas un « objet lâché au sol » (décision utilisateur 2026-09-04) ; un appareil de
// grappin lâché, si. Et deux pièges de la revue adversariale du 2026-09-04 :
// un LÂCHER de famille déployable (capteur) compte en lâcher, JAMAIS en
// déploiement ; une pose SANS origine (artefact de schéma < 10, backfill) ne
// compte nulle part — non mesuré n'est ni deployed ni dropped.
func TestUsageSummary_GrenadesHorsObjetsLaches(t *testing.T) {
	doc := &ReplayDocument{
		SchemaVersion: SchemaVersion, FrameIntervalMS: 100, FrameCount: 1000,
		Roster: []RosterEntry{{XUID: "111", FilmIndex: 0}},
		Tracks: []Track{{Slot: 1, XUID: "111", StartFrame: 0, EndFrame: 900}},
		EquipmentPlacements: []EquipmentPlacement{
			{Family: "grenade_frag", Origin: OriginDropped, Owner: 1, ID: "0x72b63d70"},
			{Family: "grenade_spike", Origin: OriginDropped, Owner: 1, ID: "0x72b63d71"},
			{Family: "wall", Origin: OriginDropped, Owner: 1, ID: "0x8e2dc574"},
			{Family: "grapple", Origin: OriginDropped, Owner: 1, ID: "0x273fe0eb"},
			{Family: "powerup_overshield", Origin: OriginDropped, Owner: 1, ID: "0xc4b1aebd"},
			{Family: "sensor", Origin: OriginDropped, Owner: 1, ID: "0x11aa22bb"},
			{Family: "sensor", Origin: "", Owner: 1, ID: "0x11aa22bc"},
		},
	}
	s := BuildUsageSummary(doc)
	if len(s.Players) != 1 {
		t.Fatalf("attendu 1 ligne joueur, obtenu %d", len(s.Players))
	}
	p := s.Players[0]
	if p.DroppedObjects != 4 {
		t.Errorf("dropped_objects = %d, attendu 4 (mur + grappin + surbouclier + capteur ; les DEUX grenades exclues, l'origine absente aussi)", p.DroppedObjects)
	}
	if len(p.DeployedByFamily) != 0 {
		t.Errorf("deployed_by_family = %v, attendu vide : un capteur LÂCHÉ n'est pas un déploiement, et une pose sans origine ne compte nulle part", p.DeployedByFamily)
	}
	for _, fam := range []string{"grenade_frag", "grenade_spike"} {
		if _, ok := p.DroppedByFamily[fam]; ok {
			t.Errorf("la famille %q ne doit pas entrer dans dropped_by_family", fam)
		}
	}
	somme := 0
	for _, n := range p.DroppedByFamily {
		somme += n
	}
	if somme != p.DroppedObjects {
		t.Errorf("dropped_by_family somme à %d, dropped_objects = %d — l'invariant est rompu", somme, p.DroppedObjects)
	}
}

// TestUsageSummary_DeploiementsEtPanneauxDuMur — un mur déployé publie DEUX poses
// (appareil + panneaux) : seule celle des panneaux compte ; une grenade « déployée »
// (un lancer) et un appareil de capacité ne comptent jamais.
func TestUsageSummary_DeploiementsEtPanneauxDuMur(t *testing.T) {
	doc := &ReplayDocument{
		SchemaVersion: SchemaVersion, FrameIntervalMS: 100, FrameCount: 1000,
		Roster: []RosterEntry{{XUID: "111", FilmIndex: 0}},
		Tracks: []Track{{Slot: 1, XUID: "111", StartFrame: 0, EndFrame: 900}},
		EquipmentPlacements: []EquipmentPlacement{
			{Family: "wall", Origin: OriginDeployed, Owner: 1, ID: "0x8e2dc574"}, // l'appareil qui vole
			{Family: "wall", Origin: OriginDeployed, Owner: 1, ID: "0x528fce46"}, // ses panneaux
			{Family: "sensor", Origin: OriginDeployed, Owner: 1, ID: "0x72b63d69"},
			{Family: "grenade_frag", Origin: OriginDeployed, Owner: 1, ID: "0x72b63d70"}, // un lancer
			{Family: "thruster", Origin: OriginDeployed, Owner: 1, ID: "0x99953db8"},     // appareil porté
			{Family: "other", Origin: OriginDeployed, Owner: 1, ID: "0xdeadbeef"},
			{Family: "wall", Origin: OriginDeployed, Owner: -1, ID: "0x528fce46"}, // sans poseur : personne
		},
	}
	s := BuildUsageSummary(doc)
	p := s.Players[0]
	if got := p.DeployedByFamily["wall"]; got != 1 {
		t.Errorf("deployed[wall] = %d, attendu 1 (les panneaux seuls, jamais l'appareil)", got)
	}
	if got := p.DeployedByFamily["sensor"]; got != 1 {
		t.Errorf("deployed[sensor] = %d, attendu 1", got)
	}
	if got := p.DeployedByFamily["other"]; got != 1 {
		t.Errorf("deployed[other] = %d, attendu 1 (pose réelle, nature non établie)", got)
	}
	for _, fam := range []string{"grenade_frag", "thruster"} {
		if _, ok := p.DeployedByFamily[fam]; ok {
			t.Errorf("la famille %q ne doit pas entrer dans deployed_by_family", fam)
		}
	}
}

// TestUsageSummary_AttributionSlotEtIndexDeFilm — le grappin et les épisodes se
// joignent par SLOT (dernier gagnant sur un slot recyclé, un bot bloque
// l'attribution), les grenades par INDEX DE FILM.
func TestUsageSummary_AttributionSlotEtIndexDeFilm(t *testing.T) {
	doc := &ReplayDocument{
		SchemaVersion: SchemaVersion, FrameIntervalMS: 100, FrameCount: 1000,
		Roster: []RosterEntry{
			{XUID: "111", FilmIndex: 0},
			{XUID: "222", FilmIndex: 1},
			{Bot: true, Name: "Recrue [bot]", FilmIndex: 2},
		},
		Tracks: []Track{
			// Le slot 5 est RECYCLÉ : 111 puis 222 — l'agrégat de match crédite le
			// DERNIER propriétaire (dette web assumée, reproduite pour comparer).
			{Slot: 5, XUID: "111", StartFrame: 0, EndFrame: 100},
			{Slot: 5, XUID: "222", StartFrame: 200, EndFrame: 400},
			// Le slot 6 appartient à un BOT : ses gestes n'entrent dans aucune ligne.
			{Slot: 6, Bot: "Recrue [bot]", StartFrame: 0, EndFrame: 400},
			{Slot: 7, XUID: "111", StartFrame: 0, EndFrame: 400},
		},
		GrappleLines: []GrappleLine{
			{Slot: 5, T0: 250, T1: 260}, // -> 222 (dernier gagnant)
			{Slot: 6, T0: 10, T1: 20},   // -> bot : personne
			{Slot: 7, T0: 10, T1: 20},   // -> 111
		},
		EquipmentEpisodes: []EquipmentEpisode{
			{Slot: 7, Fam: EquipFamilyCamo, T0: 100, T1: 160, K: 2}, // 60 frames x 100 ms
			{Slot: 7, Fam: EquipFamilyOvershield, T0: 300, T1: 310},
			{Slot: 7, Fam: "inconnue", T0: 0, T1: 10}, // famille hors mesure : ignorée
		},
		Grenades: []Grenade{
			{Idx: 0, Rank: 0}, {Idx: 0, Rank: 2}, // -> 111
			{Idx: 1, Rank: 1}, // -> 222
			{Idx: 2, Rank: 0}, // -> bot : personne
			{Idx: 9, Rank: 0}, // index inconnu du roster : personne
		},
	}
	s := BuildUsageSummary(doc)
	byXUID := map[string]UsagePlayerSummary{}
	for _, p := range s.Players {
		byXUID[p.XUID] = p
	}
	if got := byXUID["222"].GrapplePulls; got != 1 {
		t.Errorf("grapple_pulls(222) = %d, attendu 1 (slot recyclé, dernier gagnant)", got)
	}
	if got := byXUID["111"].GrapplePulls; got != 1 {
		t.Errorf("grapple_pulls(111) = %d, attendu 1", got)
	}
	p111 := byXUID["111"]
	if p111.CamoEpisodes != 1 || p111.CamoMS != 6000 || p111.CamoKills != 2 {
		t.Errorf("camo(111) = %d ep / %d ms / %d kills, attendu 1/6000/2",
			p111.CamoEpisodes, p111.CamoMS, p111.CamoKills)
	}
	if p111.OvershieldEpisodes != 1 || p111.OvershieldMS != 1000 {
		t.Errorf("overshield(111) = %d ep / %d ms, attendu 1/1000",
			p111.OvershieldEpisodes, p111.OvershieldMS)
	}
	if p111.GrenadesThrown != 2 || byXUID["222"].GrenadesThrown != 1 {
		t.Errorf("grenades_thrown = %d/%d, attendu 2/1 (jointure par INDEX DE FILM)",
			p111.GrenadesThrown, byXUID["222"].GrenadesThrown)
	}
	if _, ok := byXUID["bot:Recrue [bot]"]; ok {
		t.Error("un bot ne doit jamais recevoir de ligne persistée")
	}
}

// TestUsageSummary_FixtureFigee — la projection sur le document RÉEL de la fixture
// figée (000d5950) : les invariants de complétude doivent tenir sur un vrai match,
// pas seulement sur des documents synthétiques.
func TestUsageSummary_FixtureFigee(t *testing.T) {
	doc := buildGolden(t)
	s := BuildUsageSummary(&doc)

	if s.Match.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, attendu %d", s.Match.SchemaVersion, SchemaVersion)
	}
	if s.Match.FrameCount != doc.FrameCount || s.Match.FrameIntervalMS != doc.FrameIntervalMS {
		t.Errorf("axe de temps non recopié verbatim : %d/%d vs %d/%d",
			s.Match.FrameCount, s.Match.FrameIntervalMS, doc.FrameCount, doc.FrameIntervalMS)
	}
	if s.Match.PadOccupancies != len(doc.PadPickups) {
		t.Errorf("PadOccupancies = %d, attendu %d (toutes les occupations du document)",
			s.Match.PadOccupancies, len(doc.PadPickups))
	}
	sommeJoueurs := 0
	for _, p := range s.Players {
		sommeJoueurs += p.PadPickups
		sousSomme := 0
		for _, n := range p.PadPickupsByWeapon {
			sousSomme += n
		}
		if sousSomme != p.PadPickups {
			t.Errorf("joueur %s : pad_pickups_by_weapon somme à %d, pad_pickups = %d",
				p.XUID, sousSomme, p.PadPickups)
		}
		dropSomme := 0
		for _, n := range p.DroppedByFamily {
			dropSomme += n
		}
		if dropSomme != p.DroppedObjects {
			t.Errorf("joueur %s : dropped_by_family somme à %d, dropped_objects = %d",
				p.XUID, dropSomme, p.DroppedObjects)
		}
	}
	if sommeJoueurs != s.Match.PadNamed {
		t.Errorf("somme des pad_pickups joueurs = %d, PadNamed = %d — chaque prise nommée doit avoir une ligne",
			sommeJoueurs, s.Match.PadNamed)
	}
	grapples := 0
	for _, p := range s.Players {
		grapples += p.GrapplePulls
	}
	if grapples > len(doc.GrappleLines) {
		t.Errorf("tractions attribuées (%d) > tractions du document (%d)", grapples, len(doc.GrappleLines))
	}
	grenades := 0
	for _, p := range s.Players {
		grenades += p.GrenadesThrown
	}
	if grenades > len(doc.Grenades) {
		t.Errorf("lancers attribués (%d) > lancers du document (%d)", grenades, len(doc.Grenades))
	}
}
