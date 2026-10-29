package grammar

import (
	"strings"
	"testing"
)

// composants_vehicule_ti40_test.go — les vecteurs des composants `ti=40` (i30 a i47), ecrits
// d apres les deserialiseurs du jeu (`composants_vehicule_ti40.go`, T6 §4 et §6.1), et la loi du
// masque : lus dans un record a masque, jamais dans un etat complet d image-cle.

// tamponDeBits rend les bits d une chaine de `0` et `1` (espaces ignores), MSB d abord, suivis
// de 32 bits a 1 : un lecteur qui lirait trop loin s arreterait a une autre position.
func tamponDeBits(s string) []byte {
	var w bitWriter
	for _, c := range strings.ReplaceAll(s, " ", "") {
		w.bit(uint64(c - '0'))
	}
	w.bits(^uint64(0), 32)
	return w.buf
}

// lireTi40 lit `nom` comme la chaine de dispatch, dans un record a masque (ou un etat complet),
// et rend la position atteinte et le drapeau `ported`.
func lireTi40(t *testing.T, bits, nom string, etatComplet bool) (int, bool) {
	t.Helper()
	br := LecteurSur(tamponDeBits(bits))
	enEtatComplet(br, etatComplet)
	_, _, porte := consumeByName(br, nom, 40, 1)
	return br.BitPos(), porte
}

// TestComposantsTi40LargeurDuFlux : chaque vecteur s arrete exactement ou le deserialiseur du jeu
// s arrete, dans un record a masque.
func TestComposantsTi40LargeurDuFlux(t *testing.T) {
	treize := strings.Repeat("1", 13)
	cas := []struct {
		nom, bits string
		fin       int
	}{
		{compVehicleAutoTurretTriggers, "101", 3},
		{compVehicleAutoTurretAimingVector, strings.Repeat("10", 9) + "1", 19},
		{compVehicleTransformedOpenState, "1 10000000", 9},
		{compVehicleAutoTurretTarget, "0", 1},
		{compVehicleAutoTurretTarget, "1 1 101010101 01", 13},
		{compVehicleAutoTurretTarget, "1 0 " + treize + " 01", 17},
		{compVehicleSentryState, "110 1", 4},
		{compVehicleWeaponSet, "011 1 010", 7},
		{compVehicleWeaponSet, "011 001 011", 9},
		{compVehicleAutoTurret, "10", 2},
		{compVehicleEquipmentTurretParent, "0", 1},
		{compVehicleEquipmentTurretParent, "1 " + treize + " 10", 16},
		{compVehicleSeatsOverridePitch, strings.Repeat("0", 16), 16},
		{compVehicleSeatsOverrideYaw, strings.Repeat("1", 16), 16},
		{compAirDropFlight, "10 " + strings.Repeat("0", 14) + " " + strings.Repeat("1", 8), 24},
		{compWarp, "10 11110000", 10},
		{compWarp, "00", 2},
		{compWarp, "11 " + strings.Repeat("0", 16), 18},
		{compVehicleLowFrequency, "1", 1},
		{compVehicleLowFrequency, "0 10101", 6},
		{compVehicleEmpTimer, "10101010", 8},
	}
	for _, c := range cas {
		fin, porte := lireTi40(t, c.bits, c.nom, false)
		if !porte || fin != c.fin {
			t.Errorf("%s sur %q : porte=%v fin=%d, attendu porte fin=%d", c.nom, c.bits, porte, fin, c.fin)
		}
	}
}

// TestPorteTi40LoiDuMasque : `i33` et `i34` se lisent porte posee dans un record a masque (les
// ecrivains `FUN_142f09c74` et `FUN_142f0cca0` n annoncent le composant que porte posee).
func TestPorteTi40LoiDuMasque(t *testing.T) {
	for _, c := range []struct {
		bits string
		fin  int
	}{{"01 000101", 8}, {"10", 2}, {"11 111111", 8}, {"00", 2}} {
		fin, porte := lireTi40(t, c.bits, compVehicleTypeState, false)
		if !porte || fin != c.fin {
			t.Errorf("i33 sur %q : porte=%v fin=%d, attendu porte fin=%d", c.bits, porte, fin, c.fin)
		}
	}
	// i34 mode 2 : R(1)=1, puis la paire avant/haut brute et la vitesse brute.
	if fin, porte := lireTi40(t, "1"+strings.Repeat("0", fwdUpDynPrecMode2Bits+rawVec3Bits), compVehicleTypePhysics,
		false); !porte || fin != 1+fwdUpDynPrecMode2Bits+rawVec3Bits {
		t.Errorf("i34 mode 2 : porte=%v fin=%d, attendu %d", porte, fin, 1+fwdUpDynPrecMode2Bits+rawVec3Bits)
	}
}

// TestComposantsTi40EtatComplet : dans un etat complet d image-cle (aucun masque), aucun
// composant propre au vehicule ne se lit — sauf `i37`, dont la largeur ne depend de rien.
func TestComposantsTi40EtatComplet(t *testing.T) {
	for _, nom := range []string{compVehicleAutoTurretTriggers, compVehicleAutoTurretAimingVector,
		compVehicleTransformedOpenState, compVehicleTypeState, compVehicleTypePhysics, compVehicleAutoTurretTarget,
		compVehicleSentryState, compVehicleWeaponSet, compVehicleAutoTurret, compVehicleEquipmentTurretParent,
		compVehicleSeatsOverridePitch, compVehicleSeatsOverrideYaw, compAirDropFlight, compWarp,
		compVehicleLowFrequency} {
		if fin, porte := lireTi40(t, "1111", nom, true); porte || fin != 0 {
			t.Errorf("%s en etat complet : porte=%v fin=%d, attendu arret a 0", nom, porte, fin)
		}
		if fin, porte := lireTi40(t, "1111", nom, false); !porte || fin == 0 {
			t.Errorf("%s dans un record a masque : porte=%v fin=%d, attendu une lecture", nom, porte, fin)
		}
	}
	if fin, porte := lireTi40(t, "10101010", compVehicleEmpTimer, true); !porte || fin != 8 {
		t.Errorf("i37 en etat complet : porte=%v fin=%d, attendu 8", porte, fin)
	}
}

// TestEtatCompletPoseParLaMarcheDImageCle : la marche d etat complet (`FUN_142e2c690`, sans
// masque) pose [Lecteur.etatComplet] — `i37` se lit, `i30` arrete la boucle — et une traversee a
// masque lit les memes composants jusqu au bout.
func TestEtatCompletPoseParLaMarcheDImageCle(t *testing.T) {
	reg := &Registry{Archetypes: make([]Archetype, 41)}
	reg.Archetypes[40] = Archetype{Index: 40, Components: []string{compVehicleEmpTimer, compVehicleAutoTurretTriggers}}
	var w bitWriter
	w.bits(0, keyframeRecordTIBit)
	w.bits(40, 6) // le typeIndex, aux 6 bits de queue du second mot
	w.bits(0, 44) // fin de l en-tete de 108 bits
	w.bits(0, 32) // n1 = 0 : aucun etat par defaut
	w.bits(1, 32) // n2 = 1 : la boucle de composants
	w.bits(0, 8)  // i37
	w.bits(5, 3)  // i30
	w.bits(0, 32) // marge
	tr := WalkKeyframeFullState(w.buf, 0, reg, ContexteDeLecture{Profil: ProfilDeBalayageParDefaut()})
	if tr.TypeIndex != 40 || tr.DesyncAt != 1 {
		t.Fatalf("etat complet : ti=%d arret a %d, attendu ti=40 arret a i30 (1)", tr.TypeIndex, tr.DesyncAt)
	}
	masque := EntityTrace{DesyncAt: -1, Mask: 0b11}
	br := LecteurSur(tamponDeBits("00000000 101"))
	traverseComponentLoop(br, reg.Archetypes[40], &masque)
	if masque.DesyncAt != -1 || br.BitPos() != 11 {
		t.Errorf("record a masque : arret a %d fin %d, attendu aucun arret, fin 11", masque.DesyncAt, br.BitPos())
	}
}
