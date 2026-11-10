package replay

// document_equipement_porte_identite_test.go — LA FAMILLE D UN OBJET PORTE SE LIT SUR L OBJET (identite ecrite a sa
// creation), quelle que soit la facon dont le joueur l a obtenu : plus aucune jointure par rang d annonce.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/equipenergy"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// familleDeLaGrenadePique : la famille de la palette que les tests de l equipement porte attendent pour la grenade pique.
const familleDeLaGrenadePique = "grenade_spike"

// TestObjetPorteResoluParSonIdentite : la famille et le cout se lisent sur l identifiant de l objet porte, par le
// catalogue (le mur a deux ramassables, dont l un est hors manifeste) ou par le manifeste seul (le grappin).
func TestObjetPorteResoluParSonIdentite(t *testing.T) {
	r := referenceDuTest()
	for id, want := range map[uint32]string{idRepulseur: "repulsor", idTranslocateur: "translocator_beacon",
		idMurRamasse: "wall", idMurReapparu: "wall", idChampReparable: "repair_field", idEcran: familleDeLEcranOccultant,
		idGrappin: "grapple"} {
		if f, issue := r.famille(id); issue != identiteResolue || f != want {
			t.Errorf("famille de 0x%08x : %q (%d), attendu %q", id, f, issue, want)
		}
	}
	if c, st := r.cout(idMurRamasse); st != coutEtabli || c.variation != -0.33 {
		t.Errorf("cout du mur ramasse : %+v (%d) ; deux fiches de meme cout valent un cout", c, st)
	}
	if c, st := r.cout(idMurReapparu); st != coutEtabli || c.variation != -1 {
		t.Errorf("cout du mur de reapparition : %+v (%d)", c, st)
	}
	if _, st := r.cout(idGrappin); st != coutSansFiche {
		t.Errorf("le grappin, nomme au manifeste seul, n a pas de fiche : statut %d", st)
	}
}

// TestIdentiteJamaisDevinee : une grenade ne se publie pas ; un identifiant inconnu, ou dont les sources se
// contredisent, n a pas de famille.
func TestIdentiteJamaisDevinee(t *testing.T) {
	r := referenceDuTest()
	if f, issue := r.famille(idGrenade); issue != identiteGrenade || f != familleDeLaGrenadePique {
		t.Errorf("grenade : %q (%d)", f, issue)
	}
	if _, issue := r.famille(0xa53cd143); issue != identiteInconnue {
		t.Errorf("identifiant hors catalogue et manifeste : issue %d", issue)
	}
	cat := &equipenergy.Catalog{Entries: []equipenergy.Entry{
		{Pickup: "0x7ca85adc", Carried: "0x00000aa1", CarriedEnergy: 1, Threshold: 0.2, Delta: -0.2},
		{Pickup: "0x730dc70f", Carried: "0x00000aa1", CarriedEnergy: 1, Threshold: 1, Delta: -1},
	}}
	contradictoire := nouvelleReferenceDesObjetsPortes(cat, famillesDuTest)
	if _, issue := contradictoire.famille(0x00000aa1); issue != identiteEnDesaccord {
		t.Errorf("deux ramassables de familles differentes : issue %d", issue)
	}
	if _, st := contradictoire.cout(0x00000aa1); st != coutAmbigu {
		t.Errorf("deux fiches de couts differents : statut %d", st)
	}
	direct := nouvelleReferenceDesObjetsPortes(catalogueDuTest(), map[uint32]string{0x7ca85adc: "repulsor",
		0x1e79ebda: "thruster"})
	if _, issue := direct.famille(idRepulseur); issue != identiteEnDesaccord {
		t.Errorf("le manifeste nomme l objet autrement que le catalogue : issue %d", issue)
	}
}

// TestObjetsNonPublies : sans identifiant, grenade ou identifiant inconnu, l objet n est pas publie, et chaque
// refus se compte ; aucun repli sur une annonce du porteur.
func TestObjetsNonPublies(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		lecture(78332.1, 1566, 513, types.CarriedNew),
		avecIdentite(lecture(78400.0, 1567, 513, types.CarriedNew), idGrenade),
		avecIdentite(lecture(78500.0, 1568, 513, types.CarriedNew), 0xa53cd143),
	}
	out, _, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		refs: referenceDuTest(), tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 0 || cov.NoGlobalID != 1 || cov.Grenades != 1 || cov.Unresolved != 1 || cov.Published != 0 {
		t.Fatalf("publies %d, couverture %+v", len(out), cov)
	}
}

// TestObjetSansPisteNonPublie : un porteur sans piste publiee n a rien a montrer.
func TestObjetSansPisteNonPublie(t *testing.T) {
	reads := []types.CarriedEquipmentRead{avecIdentite(lecture(78332.1, 1566, 513, types.CarriedNew), idRepulseur)}
	out, _, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		refs: referenceDuTest(), origin: origine0d9a, step: pas100ms})
	if len(out) != 0 || cov.NoIdentity != 1 {
		t.Fatalf("publies %d, couverture %+v", len(out), cov)
	}
}

// TestDeuxObjetsDansLeMemePaquet : a la reapparition le jeu cree DEUX objets attaches dans le meme paquet ; chacun
// est publie sous sa famille, sans annonce du porteur. C est le cas que la jointure par rang perdait.
func TestDeuxObjetsDansLeMemePaquet(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		avecIdentite(lecture(78332.1, 1566, 513, types.CarriedNew), idRepulseur),
		avecIdentite(lecture(78332.1, 1570, 513, types.CarriedNew), idMurReapparu),
	}
	out, _, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		refs: referenceDuTest(), tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 2 || cov.Published != 2 || out[0].Family != "repulsor" || out[0].R != 6 ||
		out[1].Family != "wall" || out[1].R != 2 {
		t.Fatalf("publies %+v, couverture %+v", out, cov)
	}
}

// TestRangInconnuQuandLaPaletteNeNommePas : sans palette classee, l objet est publie, sa famille dit ce qu il est, et
// son rang vaut -1.
func TestRangInconnuQuandLaPaletteNeNommePas(t *testing.T) {
	reads := []types.CarriedEquipmentRead{avecIdentite(lecture(78332.1, 1566, 513, types.CarriedNew), idRepulseur)}
	out, _, _ := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, refs: referenceDuTest(),
		tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 1 || out[0].Family != "repulsor" || out[0].R != rangInconnu {
		t.Fatalf("publies %+v", out)
	}
}

// TestObjetCreeAvantLOrigine : l equipement de depart, cree avant la premiere frame, est publie, sa prise a la
// frame 0 ; ses lectures d avant l origine n ont pas d instant (l energie de la frame 0 est la derniere lue), et rien
// ne se lit a une frame negative. Les lectures d apres l origine gardent leur frame.
func TestObjetCreeAvantLOrigine(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		avecIdentite(lecture(1000, 1566, 513, types.CarriedNew), idRepulseur), // 13,4 s avant l origine
		avecActivation(lecture(5000, 1566, 513, types.CarriedDelta)),
		avecEnergie(lecture(5100, 1566, 513, types.CarriedDelta), 6553),
		avecEnergie(lecture(9000, 1566, 513, types.CarriedDelta), 3277),
		avecSuivis(lecture(9100, 1566, 513, types.CarriedDelta), 1700),
		avecActivation(lecture(20000, 1566, 513, types.CarriedDelta)),
		avecEnergie(lecture(20100, 1566, 513, types.CarriedDelta), 1),
	}
	out, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		refs: referenceDuTest(), tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 1 || len(poses) != 0 || cov.BeforeOrigin != 1 || cov.ReadsBeforeOrigin != 2 {
		t.Fatalf("publies %d, poses %d, couverture %+v", len(out), len(poses), cov)
	}
	o := out[0]
	if o.T0 != 0 || len(o.Uses) != 1 || o.Uses[0] != 55 || len(o.Tracked) != 0 {
		t.Fatalf("T0 %d, activations %v, objets engendres %v : seule l activation d apres l origine se publie",
			o.T0, o.Uses, o.Tracked)
	}
	if len(o.Energy) != 2 || o.Energy[0].T != 0 || o.Energy[0].N == nil || *o.Energy[0].N != 1 || o.Energy[1].T != 56 {
		t.Fatalf("energie %+v : l etat a la frame 0 est la derniere lecture d avant l origine (0,2 = 1 usage)", o.Energy)
	}
}

// TestObjetFiniAvantLOrigineNonPublie : un objet supprime avant la premiere frame n a aucune existence sur l axe.
func TestObjetFiniAvantLOrigineNonPublie(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		avecIdentite(lecture(1000, 1566, 513, types.CarriedNew), idRepulseur),
		lecture(5000, 1566, 513, types.CarriedDel),
	}
	out, _, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		refs: referenceDuTest(), tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 0 || cov.EndedBeforeOrigin != 1 {
		t.Fatalf("publies %d, couverture %+v", len(out), cov)
	}
}

// TestFrameNeDebordePas : un instant d avant l origine vaut la frame 0, jamais un entier non signe deborde.
func TestFrameNeDebordePas(t *testing.T) {
	in := entreeDEquipementPorte{origin: origine0d9a, step: pas100ms}
	if got := in.frame(origine0d9a - 1); got != 0 {
		t.Fatalf("frame d un instant anterieur : %d", got)
	}
	if got := in.frame(origine0d9a + 3*pas100ms); got != 3 {
		t.Fatalf("frame 3 : %d", got)
	}
}
