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
func poJuge(points []MapSpawnPoint, connu bool, pos []filmdec.BipedPosition,
	dropped []droppedSpot,
) *pickupOriginJudge {
	j := &pickupOriginJudge{
		points: points, catalogKnown: connu, posBySlot: map[uint32][]bipedPos{},
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
	dropped := []droppedSpot{{t: 0, x: 50, y: 50, z: 0}}
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
	got, cov := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil,
		poJuge(points, true, pos, dropped))
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
	dropped := []droppedSpot{{t: 0, x: 10, y: 10, z: 0}}
	pos := []filmdec.BipedPosition{poPos(1, 1_000_000, 10, 10, 0)}
	in := []filmdec.BipedPickup{poRamassage(1, 1_000_000, 3)}
	got, _ := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil,
		poJuge(points, true, pos, dropped))
	if len(got) != 1 || got[0].Origin != PickupOriginSpawner {
		t.Fatalf("point et pose au meme endroit : origine %q, attendu %q — un fait de carte "+
			"au centimetre l'emporte sur une inference de film", got[0].Origin,
			PickupOriginSpawner)
	}
	// LE TEMOIN DE L'ORDRE : sans le point, le meme ramassage doit devenir `ground`. Sans ce
	// second appel, le test ne distinguerait pas « spawner gagne » de « ground ne marche pas ».
	gotSansPoint, _ := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil,
		poJuge(nil, true, pos, dropped))
	if gotSansPoint[0].Origin != PickupOriginGround {
		t.Fatalf("sans point catalogue, la meme pose doit rendre %q, obtenu %q",
			PickupOriginGround, gotSansPoint[0].Origin)
	}
}

// TestPickupOriginCarteAbsenteSeDistingueDeCarteVide — promesse 4, la decision produit.
func TestPickupOriginCarteAbsenteSeDistingueDeCarteVide(t *testing.T) {
	pos := []filmdec.BipedPosition{poPos(1, 1_000_000, 200, 200, 0)}
	in := []filmdec.BipedPickup{poRamassage(1, 1_000_000, 2)}

	// Carte CONNUE, sans aucun point : le trou est celui de la carte, pas du catalogue.
	_, covConnue := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil,
		poJuge(nil, true, pos, nil))
	if covConnue.MapCatalogMissing {
		t.Error("carte CONNUE sans point : mapCatalogMissing doit etre FAUX — sinon on ne peut " +
			"plus distinguer une carte pauvre d'une carte absente")
	}
	// Carte ABSENTE du catalogue : le trou est celui du catalogue, et il doit se VOIR.
	_, covAbsente := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil,
		poJuge(nil, false, pos, nil))
	if !covAbsente.MapCatalogMissing {
		t.Error("carte ABSENTE du catalogue : mapCatalogMissing doit etre VRAI — c'est la " +
			"decision produit du lot, le trou se compte et ne se telecharge pas en cuisson")
	}
	// Sans juge du tout, c'est le meme trou et il se dit de la meme facon.
	_, covSansJuge := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil, nil)
	if !covSansJuge.MapCatalogMissing {
		t.Error("sans juge : mapCatalogMissing doit etre VRAI")
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
	got, cov := buildPickups(in, poClock(), nil, filmdec.BipedPickupStats{}, nil,
		poJuge(points, true, pos, nil))
	if got[0].Origin != "" {
		t.Fatalf("position trop vieille : origine %q, attendu l'abstention — sinon on invente "+
			"un lieu au ramassage", got[0].Origin)
	}
	if cov.OriginUnknown != 1 {
		t.Errorf("le refus doit compter dans originUnknown, obtenu %d", cov.OriginUnknown)
	}
}
