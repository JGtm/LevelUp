package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// zoneAt fabrique une zone cylindrique de rayon 3 m, haute de 2 m au-dessus de son centre
// et de 0,5 m en dessous — l'ordre de grandeur mesure sur les zones de Bastion reelles
// (rayon 1,7 a 5,0 m, UpZ 1,7 a 5,0, DownZ 0,0 a 1,4).
func zoneAt(rank int, instance int32, x, y, z float64) Zone {
	r := 3.0
	shape := &mapvar.Shape{
		Family: mapvar.ShapeCylinder, Radius: &r,
		UpZ: 2, DownZ: 0.5,
		Forward: mapvar.Vec3{X: 1}, Up: mapvar.Vec3{Z: 1},
	}
	center := mapvar.Vec3{X: x, Y: y, Z: z}
	vol, err := mapvar.NewVolume(center, shape)
	if err != nil {
		panic(err)
	}
	return Zone{
		Role: mapvar.RoleStrongholdZone, InstanceID: instance,
		SpatialRank: rank, Center: center, Volume: vol,
	}
}

func action(xuid string, frame int) ObjectiveAction {
	return ObjectiveAction{T: frame, XUID: xuid, Stat: "zone_secures", TimeMS: frame * 100}
}

func track(xuid string, pts ...Point) Track { return Track{XUID: xuid, Points: pts} }

func pointAt(frame int, x, y, z float32) Point { return Point{T: frame, X: x, Y: y, Z: z} }

// checkInvariant garde la promesse ecrite sur ZoneCoverage : les compteurs somment au
// total. Un motif de rejet ajoute sans compteur ferait disparaitre des actions en
// silence.
func checkInvariant(t *testing.T, cov ZoneCoverage) {
	t.Helper()
	somme := cov.Attributed + cov.NoPosition + cov.Outside + cov.Ambiguous
	if somme != cov.Actions {
		t.Errorf("couverture non close : %d attribuees + %d sans position + %d dehors + "+
			"%d ambigues = %d, attendu %d actions",
			cov.Attributed, cov.NoPosition, cov.Outside, cov.Ambiguous, somme, cov.Actions)
	}
}

// TestActionDansLaZoneEstAttribueeALaBonneInstance est le cas nominal : un joueur qui
// prend une zone est dedans, et c'est CETTE zone-la qui lui est attribuee, pas la
// premiere de la liste.
func TestActionDansLaZoneEstAttribueeALaBonneInstance(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, -20, 0, 0), zoneAt(1, 102, 0, 0, 0), zoneAt(2, 103, 20, 0, 0)}
	tracks := []Track{track("2533", pointAt(50, 20.5, 0.5, 0.2))}

	res, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if cov.Attributed != 1 {
		t.Fatalf("attribuees = %d, attendu 1 (couverture %+v)", cov.Attributed, cov)
	}
	if res[0].InstanceID != 103 || res[0].SpatialRank != 2 {
		t.Errorf("zone retenue = instance %d rang %d, attendu 103 / 2",
			res[0].InstanceID, res[0].SpatialRank)
	}
	if res[0].SampleGapFrames != 0 {
		t.Errorf("ecart d'echantillon = %d, attendu 0", res[0].SampleGapFrames)
	}
}

// TestPositionHorsDeTouteZoneNEstPasAttribuee : le compteur Outside est le seul qui parle
// du croisement lui-meme. Le confondre avec NoPosition rendrait le temoin negatif
// illisible.
func TestPositionHorsDeTouteZoneNEstPasAttribuee(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	tracks := []Track{track("2533", pointAt(50, 50, 50, 0))}

	res, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if cov.Outside != 1 || cov.Attributed != 0 || cov.NoPosition != 0 {
		t.Errorf("couverture %+v, attendu 1 dehors et rien d'autre", cov)
	}
	if res[0].Attributed {
		t.Error("action hors zone marquee attribuee")
	}
}

// TestLaHauteurTrancheEntreDeuxEtages garde le test 3D. Streets et Cliffhanger placent des
// couloirs a l'aplomb d'une zone : en 2D, le joueur de l'etage du dessus serait declare
// dedans, et l'attribution serait fausse sans que rien ne le signale.
func TestLaHauteurTrancheEntreDeuxEtages(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	tracks := []Track{
		track("auRezDeChaussee", pointAt(50, 1, 1, 0.5)),
		track("aLEtage", pointAt(50, 1, 1, 6)),
	}
	actions := []ObjectiveAction{action("auRezDeChaussee", 50), action("aLEtage", 50)}

	res, cov := AttributeZones(actions, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if !res[0].Attributed {
		t.Error("joueur au rez-de-chaussee : attendu dans la zone")
	}
	if res[1].Attributed {
		t.Error("joueur a 6 m au-dessus (plafond a 2 m) : attendu hors de la zone")
	}
	if cov.Outside != 1 {
		t.Errorf("Outside = %d, attendu 1", cov.Outside)
	}
}

// TestLesViesDUnMemeJoueurSontFusionnees garde la fusion par xuid. Une Track est une VIE,
// pas un joueur : chercher la position dans une seule vie raterait toutes les actions des
// autres — et un joueur de Bastion meurt souvent.
func TestLesViesDUnMemeJoueurSontFusionnees(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	tracks := []Track{
		track("2533", pointAt(10, 1, 0, 0)),  // premiere vie
		track("2533", pointAt(200, 1, 0, 0)), // seconde vie, apres respawn
	}
	actions := []ObjectiveAction{action("2533", 10), action("2533", 200)}

	_, cov := AttributeZones(actions, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if cov.Attributed != 2 {
		t.Errorf("attribuees = %d, attendu 2 (une par vie) — couverture %+v", cov.Attributed, cov)
	}
}

// TestUneVieAnonymeNeSertAPersonne : une piste que RIEN ne nomme — ni le fil des morts, ni le
// pont canonique — ne doit servir a personne : ni a un joueur nomme, ni a une action elle-meme
// sans identite.
//
// LE NOM DU TEST EST CONSERVE TEL QUEL, et c'est deliberé : il figure dans la baseline de
// non-regression (`.ai/baselines/tests_pre_migration.jsonl`), un artefact GELE dont la
// disparition d'une entree fait rougir le gate CI. Depuis la decision du 2026-09-07 une vie
// « anonyme » n'est plus une categorie de donnee mais un DEFAUT de nommage — la doctrine est
// dans le corps du test et dans `unnamed_lives.go`, pas dans un renommage qui couterait un
// faux rouge a tout le monde.
//
// Le second cas est celui qui mord : une action sans xuid et une piste sans xuid partagent
// la meme clé vide. Sans le rejet explicite, elles s'apparieraient, et l'action serait posee
// sur une trajectoire dont on ignore le porteur.
//
// C'EST LA CONTRE-EPREUVE DU CORRECTIF P0-1, et elle doit rester verte : le pont RETRECIT le
// rejet, il ne le supprime pas (`slotXUID` est nil ici — aucun pont).
func TestUneVieAnonymeNeSertAPersonne(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	tracks := []Track{track("", pointAt(50, 1, 0, 0))} // vie non nommee, pile dans la zone

	_, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if cov.NoPosition != 1 || cov.Attributed != 0 {
		t.Errorf("joueur nomme : couverture %+v, attendu 1 sans position", cov)
	}

	_, covAnon := AttributeZones([]ObjectiveAction{action("", 50)}, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, covAnon)
	if covAnon.Attributed != 0 {
		t.Errorf("action sans identite : %d attribuee(s) sur une vie anonyme — "+
			"la cle vide a servi de pont", covAnon.Attributed)
	}
	if covAnon.NoPosition != 1 {
		t.Errorf("action sans identite : couverture %+v, attendu 1 sans position", covAnon)
	}
}

// TestLaToleranceDEchantillonnageEstBornee garde le seuil. Sans borne, on irait chercher
// la position d'un joueur dix secondes plus tot et on l'attribuerait a une zone qu'il a
// quittee depuis longtemps.
func TestLaToleranceDEchantillonnageEstBornee(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	cas := []struct {
		nom          string
		frameEchant  int
		attribuee    bool
		ecartAttendu int
	}{
		{"echantillon a 2 frames (dans la tolerance)", 48, true, 2},
		{"echantillon a 3 frames (au-dela)", 47, false, 0},
	}
	for _, c := range cas {
		tracks := []Track{track("2533", pointAt(c.frameEchant, 1, 0, 0))}
		res, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil, AttributeOptions{MaxGapFrames: DefaultMaxGapFrames})
		checkInvariant(t, cov)
		if res[0].Attributed != c.attribuee {
			t.Errorf("%s : attribuee = %v, attendu %v", c.nom, res[0].Attributed, c.attribuee)
		}
		if res[0].SampleGapFrames != c.ecartAttendu {
			t.Errorf("%s : ecart = %d, attendu %d", c.nom, res[0].SampleGapFrames, c.ecartAttendu)
		}
	}
}

// TestAEgaliteDEcartLEchantillonAnterieurLEmporte fige la regle documentee : une prise de
// zone est la consequence d'une presence, la position d'AVANT la decrit mieux.
func TestAEgaliteDEcartLEchantillonAnterieurLEmporte(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0), zoneAt(1, 102, 20, 0, 0)}
	tracks := []Track{track("2533",
		pointAt(49, 1, 0, 0),  // avant : dans la zone 101
		pointAt(51, 21, 0, 0), // apres : dans la zone 102
	)}

	res, _ := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil, AttributeOptions{})
	if !res[0].Attributed || res[0].InstanceID != 101 {
		t.Errorf("instance retenue = %d (attribuee=%v), attendu 101 (echantillon anterieur)",
			res[0].InstanceID, res[0].Attributed)
	}
}

// TestZonesQuiSeRecouvrentSontDeclareesAmbigues : trancher par « la premiere trouvee »
// rendrait le resultat dependant de l'ordre de tri, donc instable a la prochaine
// regeneration du catalogue.
func TestZonesQuiSeRecouvrentSontDeclareesAmbigues(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0), zoneAt(1, 102, 1, 0, 0)} // rayon 3, centres a 1 m
	tracks := []Track{track("2533", pointAt(50, 0.5, 0, 0))}

	res, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if cov.Ambiguous != 1 || cov.Attributed != 0 {
		t.Errorf("couverture %+v, attendu 1 ambigue", cov)
	}
	if res[0].InstanceID != 0 {
		t.Errorf("une action ambigue ne doit porter aucune instance, obtenu %d", res[0].InstanceID)
	}
}

// TestLeTemoinNegatifEffondreLAttribution est le controle anti-tautologie. Si des zones
// deplacees de 12 m attrapaient autant de monde que les vraies, « etre dedans » ne serait
// pas informatif — c'est exactement ce qui a fait rejeter la lecture demi-extents.
func TestLeTemoinNegatifEffondreLAttribution(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0), zoneAt(1, 102, 30, 0, 0)}
	var tracks []Track
	var actions []ObjectiveAction
	for i := 0; i < 10; i++ {
		x := float32(i%2) * 30 // alterne entre les deux zones
		tracks = append(tracks, track("j", pointAt(i, x, 0, 0)))
		actions = append(actions, action("j", i))
	}

	_, vrai := AttributeZones(actions, tracks, zones, nil, AttributeOptions{})
	_, temoin := AttributeZones(actions, tracks, TranslateZones(zones, mapvar.Vec3{X: 12, Y: 12}), nil, AttributeOptions{})
	checkInvariant(t, vrai)
	checkInvariant(t, temoin)

	if vrai.Attributed != 10 {
		t.Fatalf("vraies zones : %d attribuees sur 10 (%+v)", vrai.Attributed, vrai)
	}
	if temoin.Attributed != 0 {
		t.Errorf("temoin a 12 m : %d attribuees, attendu 0 — le croisement attrape tout", temoin.Attributed)
	}
}

// TestTranslateZonesNeTouchePasALOriginal garde l'independance des deux jeux : mesurer le
// temoin ne doit pas abimer la mesure reelle.
func TestTranslateZonesNeTouchePasALOriginal(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 5, 5, 0)}
	_ = TranslateZones(zones, mapvar.Vec3{X: 100})
	if zones[0].Center.X != 5 || !zones[0].Volume.Contains(mapvar.Vec3{X: 5, Y: 5}) {
		t.Errorf("le jeu d'origine a bouge : centre %+v", zones[0].Center)
	}
}

// TestLeSeuilDeDistanceRelacheLAppartenance garde la tolerance spatiale : a zero elle est
// STRICTE (c'est le defaut, et c'est ce que « dedans » veut dire), au-dela elle attribue
// la zone la plus proche tant qu'elle est dans le seuil.
//
// Sans ce test, un seuil par defaut non nul pourrait s'installer sans que personne ne
// remarque que la proposition a change de sens.
func TestLeSeuilDeDistanceRelacheLAppartenance(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)} // cylindre de rayon 3
	tracks := []Track{track("2533", pointAt(50, 8, 0, 0))}
	actions := []ObjectiveAction{action("2533", 50)}

	res, strict := AttributeZones(actions, tracks, zones, nil, AttributeOptions{})
	checkInvariant(t, strict)
	if strict.Attributed != 0 || strict.Outside != 1 {
		t.Errorf("seuil strict : couverture %+v, attendu 1 dehors", strict)
	}
	if got := res[0].DistanceM; got < 4.99 || got > 5.01 {
		t.Errorf("distance publiee = %.3f, attendu 5,0 (8 m du centre, rayon 3)", got)
	}

	_, large := AttributeZones(actions, tracks, zones, nil, AttributeOptions{MaxDistanceM: 6})
	checkInvariant(t, large)
	if large.Attributed != 1 {
		t.Errorf("seuil 6 m : couverture %+v, attendu 1 attribuee", large)
	}
	_, juste := AttributeZones(actions, tracks, zones, nil, AttributeOptions{MaxDistanceM: 4.9})
	if juste.Attributed != 0 {
		t.Errorf("seuil 4,9 m pour une distance de 5,0 : attendu 0 attribuee, obtenu %d",
			juste.Attributed)
	}
}

// TestSousSeuilLaPlusProcheLEmporte : des que le seuil est relache, deux zones peuvent
// etre dans la fenetre. C'est la PLUS PROCHE qui gagne, pas la premiere du tri.
func TestSousSeuilLaPlusProcheLEmporte(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0), zoneAt(1, 102, 14, 0, 0)}
	tracks := []Track{track("2533", pointAt(50, 10, 0, 0))} // 7 m du bord de 101, 1 m de celui de 102

	res, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, nil,
		AttributeOptions{MaxDistanceM: 10})
	checkInvariant(t, cov)
	if res[0].InstanceID != 102 {
		t.Errorf("instance retenue = %d, attendu 102 (la plus proche)", res[0].InstanceID)
	}
}

// TestSansActionLaCouvertureEstVideEtClose : un match sans action ne doit pas produire de
// division par zero ni de compteur incoherent chez l'appelant.
func TestSansActionLaCouvertureEstVideEtClose(t *testing.T) {
	res, cov := AttributeZones(nil, nil, []Zone{zoneAt(0, 101, 0, 0, 0)}, nil, AttributeOptions{})
	checkInvariant(t, cov)
	if len(res) != 0 || cov.Actions != 0 {
		t.Errorf("res = %d, actions = %d, attendu 0/0", len(res), cov.Actions)
	}
}

// TestCaptureSurVieNonNommeeEstAttribueeParLePont — LE CORRECTIF P0-1 (audit du 2026-09-06).
//
// `samplesByXUID` n'indexait que les pistes dont `XUID != ""`. Une capture tombant pendant une
// vie dont le nommage a echoue ne trouvait alors aucun echantillon a moins de `MaxGapFrames`,
// sortait `NoPosition`, ne votait plus a l'appariement jauge <-> proprietaire — et quand plus
// aucune capture n'etait attribuee, le calque `zoneStates` ENTIER disparaissait de l'artefact
// (`buildZoneStates` rend nil hors mode a colline). Mesure du parc : `696a9d7c` 11 captures
// perdues sur 77, `7344d24f` 12 sur 71, `af13e2b2` 5 sur 19.
//
// MUTATION : revenir a l'index bati sur `tr.XUID != ""` rougit
// (« attribuees = 0, attendu 1 » et « NoPosition = 1 »).
func TestCaptureSurVieNonNommeeEstAttribueeParLePont(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	// La piste est PUBLIEE et pile dans la zone ; son nommage a echoue, mais le pont nomme
	// son slot.
	tracks := []Track{{Slot: 536, Points: []Point{pointAt(50, 1, 0, 0)}}}
	pont := map[uint32]uint64{536: 2533}

	att, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, pont,
		AttributeOptions{})
	checkInvariant(t, cov)
	if cov.Attributed != 1 || cov.NoPosition != 0 {
		t.Fatalf("couverture %+v, attendu 1 attribuee et 0 sans position — "+
			"le pont nomme le slot 536", cov)
	}
	if !att[0].Attributed || !att[0].HasSample {
		t.Errorf("attribution %+v : la position de la vie doit avoir servi", att[0])
	}
}

// TestCaptureSansPontResteSansPosition — LA CONTRE-EPREUVE : le correctif RETRECIT le rejet.
// Trois sous-cas, comme `TestFlagCarriesVieAnonymeSansPontResteEcartee` : pont muet, pont sur
// un autre slot, pont sur un autre nom. On n'invente aucun capteur.
func TestCaptureSansPontResteSansPosition(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	tracks := []Track{{Slot: 536, Points: []Point{pointAt(50, 1, 0, 0)}}}
	for nom, pont := range map[string]map[uint32]uint64{
		"pont muet":           nil,
		"pont sur autre slot": {999: 2533},
		"pont sur autre nom":  {536: 7777},
	} {
		t.Run(nom, func(t *testing.T) {
			_, cov := AttributeZones([]ObjectiveAction{action("2533", 50)}, tracks, zones, pont,
				AttributeOptions{})
			checkInvariant(t, cov)
			if cov.Attributed != 0 || cov.NoPosition != 1 {
				t.Errorf("couverture %+v, attendu 0 attribuee et 1 sans position", cov)
			}
		})
	}
}
