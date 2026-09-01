package replay

// pickup_origin_test.go — LE GATE de l'origine des ramassages.
//
// Il tient quatre promesses du schema 32, et chacune a une facon connue de se casser en
// silence :
//
//	1. les trois seaux BOUCLENT sur les non-armes publiees — une branche de classement
//	   oubliee ne se voit pas autrement ;
//	2. `spawner` l'emporte sur `ground` quand les deux s'appliquent — un ordre inverse
//	   passerait tous les tests d'existence ;
//	3. une ARME ne recoit JAMAIS d'origine, meme posee sur un point ;
//	4. « carte absente du catalogue » se distingue de « carte connue sans point » — c'est la
//	   decision produit du lot, et sans test elle se perd a la premiere refactorisation.
//
// LES FIXTURES SONT LITTERALES, JAMAIS LES CONSTANTES TESTEES. Le chantier precedent a paye un
// bogue pour avoir ecrit `typ: bipedPickupType` dans une fixture : permuter la constante
// laissait le test vert. Ici les classes sont ecrites 0, 2, 3 en clair.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// poJuge fabrique un juge a partir de points et de poses litteraux.
func poJuge(points []MapSpawnPoint, etat string, pos []filmdec.BipedPosition,
	dropped []droppedSpot,
) *pickupOriginJudge {
	j := &pickupOriginJudge{
		points: points, state: etat, posBySlot: map[uint32][]bipedPos{},
		dropped: dropped,
	}
	for _, p := range pos {
		j.posBySlot[p.Slot] = append(j.posBySlot[p.Slot],
			bipedPos{tsUS: p.TimestampUS, x: p.X, y: p.Y, z: p.Z})
	}
	return j
}

// poPos fabrique une position de bipede.
func poPos(slot uint32, ts uint64, x, y, z float32) filmdec.BipedPosition {
	return filmdec.BipedPosition{Slot: slot, TimestampUS: ts, X: x, Y: y, Z: z, HasWorld: true}
}

// poRamassage fabrique un ramassage. `class` est ecrit en clair par l'appelant.
func poRamassage(slot uint32, ts uint64, class uint8) filmdec.BipedPickup {
	return filmdec.BipedPickup{Slot: slot, TimestampUS: ts, Class: class, CatalogID: 0xbcabbe43}
}

func poClock() replayClock {
	return replayClock{origin: 0, step: 100_000, frames: 1000,
		families: map[uint32]string{0xbcabbe43: "grenade_frag"}}
}

// TestPickupOriginSeauxEtInvariant — promesses 1 et 3.
func TestPickupOriginSeauxEtInvariant(t *testing.T) {
	points := []MapSpawnPoint{{X: 10, Y: 10, Z: 0, Kind: "grenade"}}
	dropped := []droppedSpot{{t: 0, jusqua: -1, x: 50, y: 50, z: 0}}
	pos := []filmdec.BipedPosition{
		poPos(1, 1_000_000, 10.2, 10.1, 0), // sur le point -> spawner
		poPos(2, 2_000_000, 50.3, 50.2, 0), // sur la pose  -> ground
		poPos(3, 3_000_000, 200, 200, 0),   // nulle part   -> abstention
		poPos(4, 4_000_000, 10.1, 10.0, 0), // sur le point, mais c'est une ARME
	}
	in := []filmdec.BipedPickup{
		poRamassage(1, 1_000_000, 2), // grenade
		poRamassage(2, 2_000_000, 3), // equipement
		poRamassage(3, 3_000_000, 2), // grenade
		poRamassage(4, 4_000_000, 0), // ARME (classe 0, ecrite en clair)
	}
	got, cov := buildPickups(in, poClock(), pickupInputs{slotXUID: nil, st: filmdec.BipedPickupStats{}, weaponKeys: nil, judge: poJuge(points, SpawnPointsEstablished, pos, dropped)})
	if len(got) != 4 {
		t.Fatalf("4 ramassages publies attendus, obtenu %d", len(got))
	}
	origines := map[uint32]string{}
	for _, p := range got {
		origines[p.Slot] = p.Origin
	}
	if origines[1] != PickupOriginSpawner {
		t.Errorf("slot 1 sur un point : origine %q, attendu %q", origines[1], PickupOriginSpawner)
	}
	if origines[2] != PickupOriginGround {
		t.Errorf("slot 2 sur une pose lachee : origine %q, attendu %q",
			origines[2], PickupOriginGround)
	}
	if origines[3] != "" {
		t.Errorf("slot 3 loin de tout : origine %q, attendu l'ABSTENTION (chaine vide)",
			origines[3])
	}
	// Promesse 3 : une arme n'a jamais d'origine, meme posee sur un point.
	if origines[4] != "" {
		t.Errorf("slot 4 est une ARME sur un point : origine %q, attendu vide — les armes ont "+
			"deja GroundWeapon, publier une seconde origine donnerait deux reponses",
			origines[4])
	}
	// Promesse 1 : les seaux bouclent sur les non-armes.
	somme := cov.OriginSpawner + cov.OriginGround + cov.OriginUnknown
	if somme != cov.Items {
		t.Errorf("les seaux ne bouclent pas : spawner %d + ground %d + unknown %d = %d, "+
			"items publies %d", cov.OriginSpawner, cov.OriginGround, cov.OriginUnknown,
			somme, cov.Items)
	}
	if cov.OriginSpawner != 1 || cov.OriginGround != 1 || cov.OriginUnknown != 1 {
		t.Errorf("repartition attendue 1/1/1, obtenue %d/%d/%d",
			cov.OriginSpawner, cov.OriginGround, cov.OriginUnknown)
	}
}

// TestPickupOriginSocleLEmporteSurLeSol — promesse 2.
func TestPickupOriginSocleLEmporteSurLeSol(t *testing.T) {
	// Le point ET la pose sont au MEME endroit : les deux regles s'appliquent.
	points := []MapSpawnPoint{{X: 10, Y: 10, Z: 0, Kind: "equipment"}}
	dropped := []droppedSpot{{t: 0, jusqua: -1, x: 10, y: 10, z: 0}}
	pos := []filmdec.BipedPosition{poPos(1, 1_000_000, 10, 10, 0)}
	in := []filmdec.BipedPickup{poRamassage(1, 1_000_000, 3)}
	got, _ := buildPickups(in, poClock(), pickupInputs{slotXUID: nil, st: filmdec.BipedPickupStats{}, weaponKeys: nil, judge: poJuge(points, SpawnPointsEstablished, pos, dropped)})
	if len(got) != 1 || got[0].Origin != PickupOriginSpawner {
		t.Fatalf("point et pose au meme endroit : origine %q, attendu %q — un fait de carte "+
			"au centimetre l'emporte sur une inference de film", got[0].Origin,
			PickupOriginSpawner)
	}
	// LE TEMOIN DE L'ORDRE : sans le point, le meme ramassage doit devenir `ground`. Sans ce
	// second appel, le test ne distinguerait pas « spawner gagne » de « ground ne marche pas ».
	gotSansPoint, _ := buildPickups(in, poClock(), pickupInputs{slotXUID: nil, st: filmdec.BipedPickupStats{}, weaponKeys: nil, judge: poJuge(nil, SpawnPointsEstablished, pos, dropped)})
	if gotSansPoint[0].Origin != PickupOriginGround {
		t.Fatalf("sans point catalogue, la meme pose doit rendre %q, obtenu %q",
			PickupOriginGround, gotSansPoint[0].Origin)
	}
}

// TestPickupOriginTroisEtatsDuCatalogue — promesse 4, la decision produit.
//
// DEUX ETATS NE SUFFISAIENT PAS, ET LE DEFAUT ETAIT EXACTEMENT INVERSE DE L'INTENTION : les
// SEIZE cartes SAUTEES pour derive de source (Deadlock, Fragmentation, Highpower, Oasis...)
// sortaient FAUX a l'ancien booleen — retire au schema 32 — avec zero point, ce qui se lit
// « carte connue, aucun point ». Le drapeau cense faire VOIR le trou affirmait que tout allait
// bien, et precisement la ou l'origine est le moins fiable.
func TestPickupOriginTroisEtatsDuCatalogue(t *testing.T) {
	pos := []filmdec.BipedPosition{poPos(1, 1_000_000, 200, 200, 0)}
	in := []filmdec.BipedPickup{poRamassage(1, 1_000_000, 2)}
	// LES ATTENDUS SONT DES LITTERAUX, JAMAIS LES CONSTANTES TESTEES. Ecrits avec les
	// constantes des deux cotes, ces cas etaient TAUTOLOGIQUES : renommer la valeur de
	// `SpawnPointsNotEstablished` en `"established"` — c'est-a-dire replier l'etat sur celui
	// qui portait le defaut d'origine — laissait le test VERT. L'inversion l'a montre.
	cas := []struct {
		nom     string
		etat    string
		attendu string
	}{
		{"carte absente du catalogue", SpawnPointsMapAbsent, "map_absent"},
		{"carte connue, points NON ETABLIS", SpawnPointsNotEstablished, "not_established"},
		{"carte connue, points etablis a zero", SpawnPointsEstablished, "established"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, cov := buildPickups(in, poClock(), pickupInputs{
				judge: poJuge(nil, c.etat, pos, nil)})
			if cov.SpawnPointsState != c.attendu {
				t.Errorf("spawnPointsState = %q, attendu %q", cov.SpawnPointsState, c.attendu)
			}
			// LE COMPTE SE FIGE, et il ne pouvait pas se figer avec deux etats : `0` ne
			// distinguait rien. Ici il est zero dans les TROIS cas, et c'est l'etat qui dit
			// ce que ce zero veut dire.
			if cov.MapCatalogPoints != 0 {
				t.Errorf("mapCatalogPoints = %d, attendu 0", cov.MapCatalogPoints)
			}
		})
	}
	// Sans juge du tout : la carte est absente, jamais « etablie a vide ».
	_, covSansJuge := buildPickups(in, poClock(), pickupInputs{})
	if covSansJuge.SpawnPointsState != "map_absent" {
		t.Errorf("sans juge : spawnPointsState = %q, attendu \"map_absent\"",
			covSansJuge.SpawnPointsState)
	}
	// LES TROIS VALEURS SONT DISTINCTES — sans quoi les distinguer ne veut rien dire.
	vues := map[string]bool{
		SpawnPointsMapAbsent: true, SpawnPointsNotEstablished: true, SpawnPointsEstablished: true,
	}
	if len(vues) != 3 {
		t.Errorf("les trois etats doivent porter trois valeurs DIFFERENTES, obtenu %v", vues)
	}
}

// TestPickupOriginComptePointsEtablis fige `mapCatalogPoints` sur une carte qui EN PORTE —
// sans quoi le compteur pourrait rester a zero pour toujours sans rien casser.
func TestPickupOriginComptePointsEtablis(t *testing.T) {
	points := []MapSpawnPoint{
		{X: 10, Y: 10, Z: 0, Kind: "grenade"},
		{X: 50, Y: 50, Z: 0, Kind: "equipment"},
		{X: 90, Y: 90, Z: 0, Kind: "unknown"},
	}
	pos := []filmdec.BipedPosition{
		poPos(1, 1_000_000, 10, 10, 0),
		poPos(2, 2_000_000, 50, 50, 0),
	}
	in := []filmdec.BipedPickup{poRamassage(1, 1_000_000, 2), poRamassage(2, 2_000_000, 3)}
	_, cov := buildPickups(in, poClock(), pickupInputs{
		judge: poJuge(points, SpawnPointsEstablished, pos, nil)})
	if cov.MapCatalogPoints != 3 {
		t.Errorf("mapCatalogPoints = %d, attendu 3", cov.MapCatalogPoints)
	}
	// La VENTILATION par nature du point : c'est le controle en production du typage.
	if cov.SpawnerByPointKind["grenade"] != 1 || cov.SpawnerByPointKind["equipment"] != 1 {
		t.Errorf("ventilation par nature = %v, attendu grenade:1 equipment:1",
			cov.SpawnerByPointKind)
	}
}

// TestPickupOriginRefuseUnePositionTropLointaineDansLeTemps — la garde temporelle.
//
// Sans elle, un ramassage serait place a la derniere position connue du joueur, si vieille
// soit-elle : on lui inventerait un lieu.
func TestPickupOriginRefuseUnePositionTropLointaineDansLeTemps(t *testing.T) {
	points := []MapSpawnPoint{{X: 10, Y: 10, Z: 0, Kind: "grenade"}}
	// La position est sur le point, mais une SECONDE avant le ramassage — dix fois la garde.
	pos := []filmdec.BipedPosition{poPos(1, 1_000_000, 10, 10, 0)}
	in := []filmdec.BipedPickup{poRamassage(1, 1_000_000+10*PickupOriginPosMaxUS, 2)}
	got, cov := buildPickups(in, poClock(), pickupInputs{slotXUID: nil, st: filmdec.BipedPickupStats{}, weaponKeys: nil, judge: poJuge(points, SpawnPointsEstablished, pos, nil)})
	if got[0].Origin != "" {
		t.Fatalf("position trop vieille : origine %q, attendu l'abstention — sinon on invente "+
			"un lieu au ramassage", got[0].Origin)
	}
	if cov.OriginUnknown != 1 {
		t.Errorf("le refus doit compter dans originUnknown, obtenu %d", cov.OriginUnknown)
	}
}

// TestPickupOriginGroundEstBorneAuxDeuxBOUTS — P1-B, et le trou que la revue a releve.
//
// LE DEFAUT CORRIGE : le juge retenait `T0` et JETAIT `Until`/`UntilMax`/`End`, que le document
// mesure depuis le schema 28. Un ramassage des milliers de frames APRES la disparition mesuree
// de la pose sortait quand meme `ground` — le document se contredisait lui-meme.
//
// LES DEUX BORNES SONT TESTEES SEPAREMENT, et c'est le second point de la revue : jusqu'ici
// seule l'INVERSION de la garde d'anachronisme etait attrapee (un ramassage avant la pose), pas
// sa SUPPRESSION. Les deux cas ci-dessous tombent si l'une ou l'autre garde disparait.
func TestPickupOriginGroundEstBorneAuxDeuxBOUTS(t *testing.T) {
	// La pose existe de la frame 100 a la frame 200 (premiere preuve d'absence).
	dropped := []droppedSpot{{t: 100, jusqua: 200, x: 10, y: 10, z: 0}}
	cas := []struct {
		nom     string
		frame   int
		attendu string
	}{
		// SUPPRIMER la garde `d.t > frame` fait passer ce cas a `ground` : il tombe.
		{"avant la pose — anachronisme", 50, ""},
		{"pendant la vie de la pose", 150, PickupOriginGround},
		{"a la premiere preuve d absence", 200, PickupOriginGround},
		// SUPPRIMER la garde `frame > d.jusqua` fait passer ce cas a `ground` : il tombe.
		// C'est le defaut d'origine, a l'echelle ou il se produisait vraiment.
		{"apres la disparition mesuree", 2990, ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			// L'horloge : step 100 ms, donc frame N <=> N * 100_000 us.
			ts := uint64(c.frame) * 100_000
			pos := []filmdec.BipedPosition{poPos(1, ts, 10, 10, 0)}
			in := []filmdec.BipedPickup{poRamassage(1, ts, 2)}
			got, _ := buildPickups(in, poClock(), pickupInputs{
				judge: poJuge(nil, SpawnPointsEstablished, pos, dropped)})
			if len(got) != 1 {
				t.Fatalf("1 ramassage attendu, obtenu %d", len(got))
			}
			if got[0].Origin != c.attendu {
				t.Errorf("frame %d : origine %q, attendu %q — la pose vit de la frame %d a la "+
					"frame %d", c.frame, got[0].Origin, c.attendu, dropped[0].t,
					dropped[0].jusqua)
			}
		})
	}
}

// TestPickupOriginGroundSansPreuveDeDisparitionNEstPasBorne — le pendant du test precedent.
//
// Une pose `End == "open"` n'a AUCUNE preuve de disparition : rien ne justifie de refuser un
// ramassage tardif. Borner quand meme inventerait une abstention, ce qui est aussi faux que de
// ne pas borner du tout — dans l'autre sens.
func TestPickupOriginGroundSansPreuveDeDisparitionNEstPasBorne(t *testing.T) {
	dropped := []droppedSpot{{t: 100, jusqua: -1, x: 10, y: 10, z: 0}}
	ts := uint64(2990) * 100_000
	pos := []filmdec.BipedPosition{poPos(1, ts, 10, 10, 0)}
	in := []filmdec.BipedPickup{poRamassage(1, ts, 2)}
	got, _ := buildPickups(in, poClock(), pickupInputs{
		judge: poJuge(nil, SpawnPointsEstablished, pos, dropped)})
	if got[0].Origin != PickupOriginGround {
		t.Fatalf("pose sans preuve de disparition : origine %q, attendu %q — borner sans preuve "+
			"inventerait une abstention", got[0].Origin, PickupOriginGround)
	}
}

// TestPickupOriginBorneHauteVientDeLaPoseMesuree — le CABLAGE, pas seulement la logique.
//
// Les tests ci-dessus construisent des `droppedSpot` a la main : ils valident le juge, pas le
// fait que la borne vienne bien du document. Celui-ci part d'un `EquipmentPlacement` reel et
// verifie que `UntilMax` atterrit dans la borne — sans quoi le juge serait correct et le
// cablage muet.
func TestPickupOriginBorneHauteVientDeLaPoseMesuree(t *testing.T) {
	poses := []EquipmentPlacement{
		{T0: 100, X: 10, Y: 10, Z: 0, Origin: OriginDropped,
			Until: 180, UntilMax: 200, End: GroundWeaponEndSeen},
		// Une pose `deployed` n'est PAS un objet qui gisait : elle ne doit pas entrer.
		{T0: 100, X: 50, Y: 50, Z: 0, Origin: "deployed",
			Until: 180, UntilMax: 200, End: GroundWeaponEndSeen},
	}
	j := newPickupOriginJudge(Options{SpawnPointsState: SpawnPointsEstablished}, nil, poses)
	if len(j.dropped) != 1 {
		t.Fatalf("1 pose `dropped` retenue attendue, obtenu %d — une pose `deployed` est un "+
			"geste du joueur, pas un objet au sol", len(j.dropped))
	}
	if j.dropped[0].jusqua != 200 {
		t.Errorf("borne haute = %d, attendu 200 (UntilMax de la pose) — si elle vaut -1, la "+
			"borne n'est pas cablee au document", j.dropped[0].jusqua)
	}
	if j.dropped[0].t != 100 {
		t.Errorf("borne basse = %d, attendu 100 (T0)", j.dropped[0].t)
	}
}
