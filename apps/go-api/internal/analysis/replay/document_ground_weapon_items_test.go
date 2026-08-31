package replay

// document_ground_weapon_items_test.go — les règles du calque des armes au sol observées.
//
// POURQUOI CES TESTS EXISTENT. Le golden d'assemblage ne couvre pas ce calque (fixture
// antérieur), et ses règles décident de ce que l'utilisateur VOIT : une fin de table ici, un
// objet fantôme là. Chaque test correspond à une décision écrite dans l'en-tête du fichier de
// production.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// gwiClock : origine 1 s, pas 100 ms, 100 frames — les frames attendues se lisent de tête.
func gwiClock() replayClock {
	return replayClock{origin: 1_000_000, step: 100_000, frames: 100}
}

// gwiObj fabrique un objet de chaîne minimal : né à t0, au repos en (x, y), avec une vie delta.
func gwiObj(t0US uint64, x, y float32, fam uint32, class string) gwPickupObject {
	o := gwPickupObject{FamilyID: fam, DropperSlot: -1}
	o.Appar.TUS = t0US
	o.Appar.X, o.Appar.Y = x, y
	o.Appar.Class = class
	o.Appar.HasDelta = true
	o.Pos = [3]float32{x, y, 0}
	o.Bounds.LowUS = t0US
	o.Bounds.HighUS = t0US + 40_000_000
	return o
}

// gwiPos fabrique un échantillon de position monde.
func gwiPos(slot uint32, tUS uint64, x, y float32) filmdec.BipedPosition {
	return filmdec.BipedPosition{Slot: slot, TimestampUS: tUS, X: x, Y: y, HasWorld: true}
}

func TestGroundWeaponItemsFinParPickupLie(t *testing.T) {
	objs := []gwPickupObject{gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)}
	changes := []filmdec.HeldWeaponChange{
		// La prise tombe 3 s après la naissance, le ramasseur est SUR l'objet.
		{TimestampUS: 5_000_000, Slot: 42, Family: 0xAABBCCDD, Kind: filmdec.HeldWeaponTaken},
	}
	pos := []filmdec.BipedPosition{gwiPos(42, 5_000_000, 10.3, 10)}
	got, cov := buildGroundWeaponItems(objs, changes, pos, gwiClock())
	if len(got) != 1 {
		t.Fatalf("publiées = %d, attendu 1", len(got))
	}
	g := got[0]
	if g.End != GroundWeaponEndPickup || g.Picker != 42 {
		t.Fatalf("fin = %q picker = %d : une prise delta à moins de 1,5 m dans la fenêtre de "+
			"vie DOIT dater la fin à la milliseconde", g.End, g.Picker)
	}
	if g.T0 != 10 || g.T1 != 40 {
		t.Errorf("bornes = [%d, %d], attendu [10, 40] (naissance à 1 s de l'origine, prise à 4 s)",
			g.T0, g.T1)
	}
	if cov.PickupLinked != 1 || cov.TakesTotal != 1 {
		t.Errorf("couverture = %+v : le lien et le denominateur des prises doivent se compter", cov)
	}
}

func TestGroundWeaponItemsFinVueAuRecensement(t *testing.T) {
	o := gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)
	// Dernière image-clé qui le voit : 21 s après la naissance. Personne ne le prend.
	o.Bounds.LowUS = 23_000_000
	got, _ := buildGroundWeaponItems([]gwPickupObject{o}, nil, nil, gwiClock())
	if len(got) != 1 || got[0].End != GroundWeaponEndSeen {
		t.Fatalf("fin = %v, attendu %q : sans prise liée, la fin est la DERNIÈRE PREUVE de "+
			"présence — jamais une durée de table", got, GroundWeaponEndSeen)
	}
	if got[0].T1 != 99 {
		// (23 s - 1 s) / 100 ms = 220, borné à frames-1 = 99 par clampFrame.
		t.Errorf("T1 = %d, attendu 99 : la borne ne sort jamais du document", got[0].T1)
	}
}

func TestGroundWeaponItemsFinOuverteSansPreuve(t *testing.T) {
	o := gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)
	o.Bounds.NeverPicked = true
	got, cov := buildGroundWeaponItems([]gwPickupObject{o}, nil, nil, gwiClock())
	if len(got) != 1 || got[0].End != GroundWeaponEndOpen || got[0].T1 != 99 {
		t.Fatalf("fin = %v, attendu %q jusqu'à la dernière frame : sans preuve de disparition, "+
			"l'objet reste affiché — l'effacer affirmerait une disparition que rien ne mesure",
			got, GroundWeaponEndOpen)
	}
	if cov.EndOpen != 1 {
		t.Errorf("couverture endOpen = %d, attendu 1", cov.EndOpen)
	}
}

func TestGroundWeaponItemsEcarteLesObjetsAuRepos(t *testing.T) {
	o := gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassSpawned)
	o.Appar.HasDelta = false // jamais bougé : une arme de socle
	got, cov := buildGroundWeaponItems([]gwPickupObject{o}, nil, nil, gwiClock())
	if len(got) != 0 || cov.AtRest != 1 {
		t.Fatalf("publiées = %d atRest = %d : une arme de socle appartient au calque des "+
			"socles — deux vérités pour un même objet seraient pires qu'une", len(got), cov.AtRest)
	}
}

func TestGroundWeaponItemsPublieLeLacheurDeLaChaine(t *testing.T) {
	// Le lâcheur n'est PAS un second appariement : c'est la vie de bipède qui a CLASSÉ
	// l'apparition `dropped` (gwPadsClass), et la chaîne le porte déjà (DropperSlot).
	o := gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)
	o.DropperSlot = 17
	got, cov := buildGroundWeaponItems([]gwPickupObject{o}, nil, nil, gwiClock())
	if len(got) != 1 || got[0].Dropper != 17 {
		t.Fatalf("dropper = %v, attendu 17 : le lâcheur nommé est exactement la vie qui a fait "+
			"la classe", got)
	}
	if cov.DropperNamed != 1 {
		t.Errorf("couverture dropperNamed = %d, attendu 1", cov.DropperNamed)
	}
}

func TestGroundWeaponItemsLaFamilleEstUnCritere(t *testing.T) {
	// Un ramasseur SUR l'objet, dans la fenêtre — mais la prise nomme une AUTRE arme (le cas
	// réel : la prise du drapeau à côté d'une arme au sol). Elle ne doit PAS se lier.
	objs := []gwPickupObject{gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)}
	changes := []filmdec.HeldWeaponChange{
		{TimestampUS: 5_000_000, Slot: 42, Family: 0x2A392328, Kind: filmdec.HeldWeaponTaken},
	}
	pos := []filmdec.BipedPosition{gwiPos(42, 5_000_000, 10.3, 10)}
	got, cov := buildGroundWeaponItems(objs, changes, pos, gwiClock())
	if len(got) != 1 || got[0].End == GroundWeaponEndPickup || cov.PickupLinked != 0 {
		t.Fatalf("got = %+v cov = %+v : la famille est un CRITÈRE du lien — sans elle, une "+
			"prise de drapeau volait le lien de l'arme voisine (27 mauvaises familles sur 33 "+
			"liens de la première version)", got, cov)
	}
}

func TestGroundWeaponItemsLoinOuTardNeLiePas(t *testing.T) {
	objs := []gwPickupObject{gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)}
	changes := []filmdec.HeldWeaponChange{
		// Prise dans la fenêtre mais à 5 m : un autre objet, pas celui-ci.
		{TimestampUS: 5_000_000, Slot: 42, Family: 0xAABBCCDD, Kind: filmdec.HeldWeaponTaken},
		// Prise à 30 cm mais HORS de la fenêtre de vie.
		{TimestampUS: 50_000_000, Slot: 43, Family: 0xAABBCCDD, Kind: filmdec.HeldWeaponTaken},
	}
	pos := []filmdec.BipedPosition{
		gwiPos(42, 5_000_000, 15, 10),
		gwiPos(43, 50_000_000, 10.3, 10),
	}
	got, cov := buildGroundWeaponItems(objs, changes, pos, gwiClock())
	if len(got) != 1 || got[0].End == GroundWeaponEndPickup || cov.PickupLinked != 0 {
		t.Fatalf("got = %+v cov = %+v : ni la distance ni le temps ne doivent céder — un lien "+
			"approximatif poserait la mauvaise arme dans les mauvaises mains", got, cov)
	}
}

func TestGroundWeaponItemsIntervalleDeDisparition(t *testing.T) {
	// Objet ne a 2 s, JAMAIS recense (vie plus courte qu un intervalle d image-cle) : la
	// derniere preuve de presence est sa naissance, la premiere preuve d absence l image-cle
	// suivante. T1 seul l effacait a la frame de naissance — affiche zero frame.
	o := gwiObj(2_000_000, 10, 10, 0xAABBCCDD, gwClassDropped)
	o.Bounds.LowUS = 2_000_000
	o.Bounds.HighUS = 9_000_000
	got, _ := buildGroundWeaponItems([]gwPickupObject{o}, nil, nil, gwiClock())
	if len(got) != 1 {
		t.Fatalf("publiees = %d, attendu 1", len(got))
	}
	g := got[0]
	if g.T1 != 10 || g.T1Max != 80 {
		t.Fatalf("bornes de disparition = [%d, %d], attendu [10, 80] : la disparition est un "+
			"INTERVALLE mesure — derniere preuve de presence, premiere preuve d absence — et "+
			"le client choisit son rendu DEDANS", g.T1, g.T1Max)
	}
}
