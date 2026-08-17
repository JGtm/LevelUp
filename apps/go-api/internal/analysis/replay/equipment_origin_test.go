package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// equipment_origin_test.go — LA CLASSIFICATION D'ORIGINE, sur donnees synthetiques.
//
// LA MESURE DU CORPUS VIT AILLEURS (`origine_poses_research_test.go`, garde `ORIGINE_FILM`) :
// c'est elle qui a etabli QUE l'origine se lit a la FIN de la vie du poseur et non a son
// debut, sur 11 films et 4 250 poses. Ici on verrouille la MACHINE — la fenetre, la distance,
// le choix de la vie, et l'invariant de couverture — pour qu'un refactor ne la retourne pas
// en silence.
//
// AUCUN OCTET DE FILM N'EST LU.

// origPos fabrique un echantillon de bipede en coordonnees monde.
func origPos(slot uint32, frame int, x, y, z float32) filmdec.BipedPosition {
	p := filmdec.BipedPosition{Slot: slot, TimestampUS: eqTS(frame), X: x, Y: y, Z: z}
	p.HasWorld = true
	return p
}

// origPose fabrique une pose brute a l'instant de la frame donnee.
func origPose(frame int, x, y, z float32) filmdec.EquipmentPlacement {
	return filmdec.EquipmentPlacement{
		Life: filmdec.EquipmentLifeKey{}, T0US: eqTS(frame), T1US: eqTS(frame + 10),
		X: x, Y: y, Z: z, GlobalID: 0x2974c233, Points: 4,
	}
}

// TestEquipmentLivesDecoupeSurLeTrouDeCinqSecondes — le decoupage est celui de lives.go, et un
// slot qui revient apres plus de lifeGapUS est une NOUVELLE vie. Sans ce decoupage, la « fin de
// vie » d'un slot reutilise serait celle de sa DERNIERE vie du match : tous les lachers des vies
// precedentes seraient classes `deployed`.
func TestEquipmentLivesDecoupeSurLeTrouDeCinqSecondes(t *testing.T) {
	// Frames de 100 ms : 60 frames = 6 s > lifeGapUS.
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0),
		origPos(512, 10, 1, 0, 0),
		origPos(512, 70, 50, 0, 0), // 6 s plus tard : autre vie
		origPos(512, 80, 51, 0, 0),
	}
	lives := equipmentLives(pos)
	if got := len(lives[512]); got != 2 {
		t.Fatalf("%d vie(s) pour le slot 512, attendu 2 (trou de 6 s > lifeGapUS)", got)
	}
	if lives[512][0].x != 1 || lives[512][1].x != 51 {
		t.Errorf("la derniere position de chaque vie est perdue : %+v", lives[512])
	}
}

// TestEquipmentLivesIgnoreLesQuantaSansBornes — sans bornes de carte, un echantillon ne porte
// pas de coordonnee : le compter fixerait la fin de vie a une position qui n'existe pas.
func TestEquipmentLivesIgnoreLesQuantaSansBornes(t *testing.T) {
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0),
		{Slot: 512, TimestampUS: eqTS(5)}, // HasWorld faux
		origPos(512, 10, 1, 0, 0),
	}
	lives := equipmentLives(pos)
	if got := len(lives[512]); got != 1 {
		t.Fatalf("%d vie(s), attendu 1", got)
	}
	if lives[512][0].x != 1 {
		t.Errorf("un quantum sans bornes a fixe la fin de vie : %+v", lives[512][0])
	}
}

// TestEquipmentOriginSeparelLacherDuDeploiement — les deux cas que la mesure du corpus separe
// par trois ordres de grandeur (lachers a 20-40 ms, deploiements a 14-42 s).
func TestEquipmentOriginSepareLacherDuDeploiement(t *testing.T) {
	// Une vie de 0 a la frame 100, qui s'acheve en (10, 10, 0).
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0),
		origPos(512, 50, 5, 5, 0),
		origPos(512, 100, 10, 10, 0),
	}
	lives := equipmentLives(pos)[512]
	cas := []struct {
		nom  string
		pose filmdec.EquipmentPlacement
		want string
	}{
		{"lache a la mort, au meme endroit", origPose(100, 10, 10, 0), OriginDropped},
		{"lache 1 frame apres le dernier point", origPose(101, 10.2, 10, 0), OriginDropped},
		{"deploye au milieu de la vie", origPose(50, 5, 5, 0), OriginDeployed},
		// LA FENETRE : 3 frames apres la fin de vie, c'est au-dela des 2 frames du seuil.
		{"trop tard apres la fin de vie", origPose(103, 10, 10, 0), OriginDeployed},
		// LA DISTANCE : au bon instant mais a 5 m — un objet lache tombe aux pieds.
		{"au bon instant mais trop loin", origPose(100, 15, 10, 0), OriginDeployed},
	}
	for _, c := range cas {
		if got := equipmentOrigin(lives, c.pose); got != c.want {
			t.Errorf("%s : origine %q, attendu %q", c.nom, got, c.want)
		}
	}
}

// TestEquipmentOriginSansVieEstInconnue — pas de vie, pas d'origine. La deviner serait
// exactement ce que ce lot a supprime.
func TestEquipmentOriginSansVieEstInconnue(t *testing.T) {
	if got := equipmentOrigin(nil, origPose(10, 0, 0, 0)); got != OriginUnknown {
		t.Errorf("origine %q sans vie de poseur, attendu %q", got, OriginUnknown)
	}
}

// TestEquipmentOriginChoisitLaVieQuiContientLInstant — un slot a plusieurs vies ; la pose
// appartient a celle qui couvre son instant, jamais a la plus recente.
func TestEquipmentOriginChoisitLaVieQuiContientLInstant(t *testing.T) {
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0),
		origPos(512, 30, 3, 0, 0), // fin de la 1re vie, en (3,0,0)
		origPos(512, 90, 40, 0, 0),
		origPos(512, 120, 44, 0, 0), // fin de la 2e vie
	}
	lives := equipmentLives(pos)[512]
	if len(lives) != 2 {
		t.Fatalf("%d vie(s), attendu 2", len(lives))
	}
	// Lache a la fin de la PREMIERE vie : si la machine prenait la derniere vie, elle
	// classerait `deployed` (90 frames d'ecart et 41 m).
	if got := equipmentOrigin(lives, origPose(30, 3, 0, 0)); got != OriginDropped {
		t.Errorf("lacher de la 1re vie classe %q, attendu %q", got, OriginDropped)
	}
}

// TestPlacementCoverageEquilibreLesOrigines — L'INVARIANT : tout ce qui est publie porte une
// origine, et les trois comptes somment au total. Un ecart signale une origine non comptee,
// c'est-a-dire un chemin de classification qui a fui (meme regle que LayerCoverage.Balanced).
func TestPlacementCoverageEquilibreLesOrigines(t *testing.T) {
	out := []EquipmentPlacement{
		{Family: "wall", Origin: OriginDeployed, Owner: 512},
		{Family: "wall", Origin: OriginDropped, Owner: 513},
		{Family: "grenade_frag", Origin: OriginDropped, Owner: 514},
		{Family: equipmentFamilyOther, Origin: OriginUnknown, Owner: -1},
	}
	cov := &EquipmentPlacementCoverage{
		ByFamily: map[string]int{}, ByFamilyOrigin: map[string]int{},
	}
	tallyEquipmentPlacements(out, cov)
	if cov.Deployed+cov.Dropped+cov.Unknown != cov.Placements {
		t.Errorf("%d deployees + %d lachees + %d inconnues != %d poses",
			cov.Deployed, cov.Dropped, cov.Unknown, cov.Placements)
	}
	if cov.Deployed != 1 || cov.Dropped != 2 || cov.Unknown != 1 {
		t.Errorf("comptes par origine faux : %+v", cov)
	}
	if got := cov.ByFamilyOrigin["wall/"+OriginDeployed]; got != 1 {
		t.Errorf("croisement famille x origine `wall/deployed` = %d, attendu 1", got)
	}
	if got := cov.ByFamilyOrigin["grenade_frag/"+OriginDropped]; got != 1 {
		t.Errorf("croisement `grenade_frag/dropped` = %d, attendu 1", got)
	}
}

// TestBuildEquipmentPlacementsPubliUneOrigineToujours — aucune pose publiee sans origine. Une
// chaine vide serait lue comme « pas de mesure » par un client, alors que `unknown` le DIT.
func TestBuildEquipmentPlacementsPublieUneOrigineToujours(t *testing.T) {
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0),
		origPos(512, 40, 4, 0, 0),
	}
	raw := []filmdec.EquipmentPlacement{
		origPose(40, 4, 0, 0),   // lache : poseur a 0 m, fin de vie
		origPose(20, 300, 0, 0), // aucun bipede a moins de 3 m -> sans poseur
	}
	st := filmdec.EquipmentPlacementStats{Lives: 2, Anchors: 9, Confirmed: 2}
	st.Calibration.Widths = filmdec.CurrentMPPWidths()
	clock := replayClock{origin: eqOrigin, step: eqStep, frames: 200,
		families: map[uint32]string{0x2974c233: "wall"}}
	out, cov := buildEquipmentPlacements(raw, st, pos, clock)
	if len(out) != 2 {
		t.Fatalf("%d pose(s) publiee(s), attendu 2", len(out))
	}
	for i, pl := range out {
		if pl.Origin == "" {
			t.Errorf("pose %d publiee sans origine : %+v", i, pl)
		}
	}
	if cov.Deployed+cov.Dropped+cov.Unknown != cov.Placements {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}
