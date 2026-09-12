package replay

// grenades_auteur_test.go — LE LANCER REVIENT A SON LANCEUR.
//
// LE DEFAUT QUE CES TESTS VERROUILLENT. `locateThrow` choisissait la naissance de projectile la
// plus proche DANS LE TEMPS, et le departage des naissances simultanees venait du tri — donc de
// X. Deux joueurs qui lancent dans la meme fenetre de 200 ms, et le lancer recevait la position
// du projectile de l'AUTRE. Mesure du banc `grenade_ecart_research_test.go` le 2026-09-11, avant
// correctif : 2 lancers sur 64 a plus de 4 m de leur lanceur sur `000d5950` (pire cas 14,46 m),
// et 100 % des lancers sur les deux films Live Fire (mediane 25 a 27 m).
//
// LA REGLE POSEE. L'auteur est resolu D'ABORD ; parmi TOUTES les naissances de la fenetre, on
// garde celle qui est a portee de sa main, et aucune si elle n'y est pas. Sans auteur, une seule
// candidate reste une lecture — plusieurs sont un tirage au sort, et on s'abstient.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// grenPos fabrique un echantillon de position de biped exploitable (HasWorld).
func grenPos(slot uint32, tsUS uint64, x, y float32) filmdec.BipedPosition {
	return filmdec.BipedPosition{Slot: slot, TimestampUS: tsUS, X: x, Y: y, HasWorld: true}
}

// grenNaissance fabrique une piste de projectile d'UN point : seule sa naissance compte ici.
func grenNaissance(slot uint32, tsUS uint64, x, y float32) filmdec.ProjectileTrack {
	return filmdec.ProjectileTrack{Slot: slot, Gen: 1, Pts: []filmdec.ProjectileSample{
		{TimestampUS: tsUS, X: x, Y: y},
	}}
}

func TestLancerUneSeuleNaissanceEtUnAuteurConnu(t *testing.T) {
	// UNE candidate, un auteur a portee : lecture par projectile, et le SLOT est publie.
	// Il ne l'etait pas — la branche projectile rendait un Grenade sans Slot, donc a zero,
	// et zero RESSEMBLE a un slot.
	pos := []filmdec.BipedPosition{grenPos(1024, 2_000_000, 10, 10)}
	throws := []filmdec.GrenadeThrow{
		{TimestampUS: 2_000_000, FilmIndex: 3, TypeID: filmdec.GrenadeFragmentation},
	}
	proj := []filmdec.ProjectileTrack{grenNaissance(2048, 2_050_000, 10.3, 10.2)}
	gren, cov := buildGrenades(pos, throws, 1_000_000, 100_000,
		map[uint32]int{1024: 3}, proj, nil)
	if len(gren) != 1 {
		t.Fatalf("un lancer attendu, obtenu %d : %+v", len(gren), gren)
	}
	if gren[0].Src != GrenadeSrcProjectile {
		t.Errorf("la position doit venir du projectile, obtenu %q", gren[0].Src)
	}
	if gren[0].Slot != 1024 {
		t.Errorf("le slot du lanceur doit etre publie (1024), obtenu %d", gren[0].Slot)
	}
	if gren[0].X != 10.3 || gren[0].Y != 10.2 {
		t.Errorf("position attendue (10.3,10.2), obtenue (%v,%v)", gren[0].X, gren[0].Y)
	}
	if !cov.Balanced() || cov.Attached != 1 {
		t.Errorf("couverture incoherente : %+v", cov)
	}
}

func TestDeuxLanceursDansLaMemeFenetreRecoiventChacunLaLeur(t *testing.T) {
	// LE CAS QUI A COUTE LE DEFAUT. Deux joueurs lancent a 20 ms d'intervalle, aux deux bouts
	// de la carte. Sur le temps seul, les deux naissances tombent dans les deux fenetres et le
	// departage venait du tri par X — donc le lanceur de gauche recevait le projectile de
	// droite une fois sur deux.
	pos := []filmdec.BipedPosition{
		grenPos(1024, 2_000_000, -20, 0),
		grenPos(2048, 2_000_000, 40, 0),
	}
	throws := []filmdec.GrenadeThrow{
		{TimestampUS: 2_000_000, FilmIndex: 3, TypeID: filmdec.GrenadeFragmentation},
		{TimestampUS: 2_020_000, FilmIndex: 7, TypeID: filmdec.GrenadeFragmentation},
	}
	proj := []filmdec.ProjectileTrack{
		grenNaissance(11, 2_030_000, 40.4, 0.2),  // celle du joueur 7 (slot 2048)
		grenNaissance(12, 2_040_000, -19.7, 0.1), // celle du joueur 3 (slot 1024)
	}
	gren, _ := buildGrenades(pos, throws, 1_000_000, 100_000,
		map[uint32]int{1024: 3, 2048: 7}, proj, nil)
	if len(gren) != 2 {
		t.Fatalf("deux lancers attendus, obtenu %d : %+v", len(gren), gren)
	}
	parIdx := map[int]Grenade{}
	for _, g := range gren {
		parIdx[g.Idx] = g
	}
	if g := parIdx[3]; g.Slot != 1024 || g.X != -19.7 {
		t.Errorf("le joueur 3 (slot 1024) doit recevoir SA naissance (-19.7), obtenu %+v", g)
	}
	if g := parIdx[7]; g.Slot != 2048 || g.X != 40.4 {
		t.Errorf("le joueur 7 (slot 2048) doit recevoir SA naissance (40.4), obtenu %+v", g)
	}
}

func TestNaissanceTropLoinDeSonAuteurReplieSurLeBiped(t *testing.T) {
	// UNE naissance a 30 m du lanceur n'est pas la sienne : c'est la signature du repli de
	// quantum mesure sur Live Fire (la moitie de l'etendue Y de la carte). On refuse, et on
	// lit la position du biped — la meme grandeur, lue ailleurs.
	pos := []filmdec.BipedPosition{grenPos(1024, 2_000_000, 10, 10)}
	throws := []filmdec.GrenadeThrow{
		{TimestampUS: 2_000_000, FilmIndex: 3, TypeID: filmdec.GrenadeFragmentation},
	}
	proj := []filmdec.ProjectileTrack{grenNaissance(2048, 2_050_000, 10.2, 41.9)}
	gren, _ := buildGrenades(pos, throws, 1_000_000, 100_000,
		map[uint32]int{1024: 3}, proj, nil)
	if len(gren) != 1 {
		t.Fatalf("un lancer attendu (repli biped), obtenu %d : %+v", len(gren), gren)
	}
	if gren[0].Src != GrenadeSrcBiped {
		t.Errorf("la naissance est hors du rayon d'auteur : repli biped attendu, obtenu %q", gren[0].Src)
	}
	if gren[0].X != 10 || gren[0].Y != 10 {
		t.Errorf("position du biped attendue (10,10), obtenue (%v,%v)", gren[0].X, gren[0].Y)
	}
	if gren[0].Proj != nil {
		t.Errorf("aucune naissance retenue : le lien doit rester nil, obtenu %d", *gren[0].Proj)
	}
}

func TestSansAuteurDeuxCandidatesSAbstient(t *testing.T) {
	// SANS le biped de l'auteur, rien ne departage deux naissances simultanees. Une seule
	// candidate reste une lecture (cf. TestGrenadePlacedFromProjectileWithoutBridge) ;
	// plusieurs sont un tirage au sort, et le lancer n'est pas publie.
	throws := []filmdec.GrenadeThrow{
		{TimestampUS: 2_000_000, FilmIndex: 3, TypeID: filmdec.GrenadeFragmentation},
	}
	proj := []filmdec.ProjectileTrack{
		grenNaissance(11, 2_030_000, 40.4, 0.2),
		grenNaissance(12, 2_040_000, -19.7, 0.1),
	}
	gren, cov := buildGrenades(nil, throws, 1_000_000, 100_000, nil, proj, nil)
	if len(gren) != 0 {
		t.Fatalf("sans auteur et avec deux candidates, le lancer ne doit PAS etre publie : %+v", gren)
	}
	if !cov.Balanced() || cov.NoSlot != 1 {
		t.Errorf("le refus doit etre COMPTE (noSlot=1), obtenu %+v", cov)
	}
}
