package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
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
func origPos(slot uint32, frame int, x, y, z float32) grammar.BipedPosition {
	p := grammar.BipedPosition{Slot: slot, TimestampUS: eqTS(frame), X: x, Y: y, Z: z}
	p.HasWorld = true
	return p
}

// origPose fabrique une pose brute a l'instant de la frame donnee.
func origPose(frame int, x, y, z float32) grammar.EquipmentPlacement {
	return grammar.EquipmentPlacement{
		Life: grammar.EquipmentLifeKey{}, T0US: eqTS(frame), T1US: eqTS(frame + 10),
		X: x, Y: y, Z: z, GlobalID: 0x2974c233, Points: 4,
	}
}

// TestEquipmentLivesDecoupeSurLeTrouDeCinqSecondes — le decoupage est celui de lives.go, et un
// slot qui revient apres plus de lifeGapUS est une NOUVELLE vie. Sans ce decoupage, la « fin de
// vie » d'un slot reutilise serait celle de sa DERNIERE vie du match : tous les lachers des vies
// precedentes seraient classes `deployed`.
func TestEquipmentLivesDecoupeSurLeTrouDeCinqSecondes(t *testing.T) {
	// Frames de 100 ms : 60 frames = 6 s > lifeGapUS.
	pos := []grammar.BipedPosition{
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
	pos := []grammar.BipedPosition{
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

// LES TROIS TESTS DE LA FENETRE TEMPORELLE ONT ETE RETIRES LE 2026-09-15 (lot 1.9.1, decision
// utilisateur) AVEC LA REGLE QU'ILS VERROUILLAIENT : `TestEquipmentOriginSepareLacherDuDeploiement`,
// `TestEquipmentOriginSansVieEstInconnue` et `TestEquipmentOriginChoisitLaVieQuiContientLInstant`.
// La fenetre ne classe plus aucune pose d'equipement — une pose dont le film ne dit rien sort
// `unknown`. Ce que la LECTURE decide est verrouille par `equipment_origin_lecture_test.go`, et
// la regle retiree survit comme TEMOIN de mesure (`f1OrigineParFenetre`, fichier de recherche).
// `.ai/baselines/tests_pre_migration.jsonl` est mis a jour dans le meme commit.

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
		ByFamily: map[string]int{}, ByFamilyOrigin: map[string]int{}, ByCause: map[string]int{},
	}
	tallyEquipmentPlacements(out, []string{
		CausePoseEvenementEngendre, CausePoseMortEcrite, CausePoseMortEcrite, CausePoseSansPoseur,
	}, cov)
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
	pos := []grammar.BipedPosition{
		origPos(512, 0, 0, 0, 0),
		origPos(512, 40, 4, 0, 0),
	}
	raw := []grammar.EquipmentPlacement{
		origPose(40, 4, 0, 0),   // lache : poseur a 0 m, fin de vie
		origPose(20, 300, 0, 0), // aucun bipede a moins de 3 m -> sans poseur
	}
	st := grammar.EquipmentPlacementStats{Lives: 2, Anchors: 9, Confirmed: 2}
	st.Calibration.Widths = grammar.ProfilDeBalayageParDefaut().MPP
	clock := replayClock{origin: eqOrigin, step: eqStep, frames: 200,
		families: map[uint32]string{0x2974c233: "wall"}}
	out, cov := buildEquipmentPlacements(
		equipmentInputs{Raw: raw, Stats: st, Positions: pos}, clock)
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

// origPoseOf — la meme pose brute, pour un GlobalID choisi (le `eqip` de l'objet).
func origPoseOf(frame int, globalID uint32, x, y, z float32) grammar.EquipmentPlacement {
	p := origPose(frame, x, y, z)
	p.GlobalID = globalID
	return p
}

// wallPanelGlobalID — le `eqip` d'un PANNEAU du mur (`kind = "deployed"` au manifeste), tel
// que `usageWallPanelIDs` le transcrit. Ecrit ici en uint32 : c'est la forme que le film rend.
const wallPanelGlobalID uint32 = 0x528fce46

// wallDeviceGlobalID — l'appareil de mur PORTE (`kind = "carried"`), le temoin negatif : sur
// lui, la question temporelle garde tout son sens et la regle ne doit RIEN changer.
const wallDeviceGlobalID uint32 = 0x8e2dc574

// TestPieceEngendreeEstToujoursDeployee — H.2, D-F1 : une PIECE ENGENDREE ne peut etre ni lachee
// ni d'origine inconnue, meme quand aucun evenement 103 ne la designe.
//
// LES DEUX CAS QUE LE PARC PORTE (rapport F.0 §2.3, 7 panneaux sur 216) :
//   - un panneau ne A L'INSTANT EXACT de la fin d'une vie — le mur deploye au dernier souffle.
//     La fenetre de 200 ms le classait `dropped` (3 cas du parc) ;
//   - un panneau SANS POSEUR mesure — aucun bipede a moins de 3 m. L'origine restait `unknown`
//     (4 cas du parc).
//
// Un panneau n'existe qu'une fois deploye : il n'entre jamais dans un inventaire, donc il ne
// tombe jamais. Le manifeste du titre le DIT (`kind = "deployed"`), et c'est une donnee ecrite.
//
// CE TEST EXERCE LE REPLI, PAS LA LECTURE (lot 1.9.1) : aucun evenement 103 n'est fourni ici,
// donc la promotion vient de `repli_piece_engendree_sans_evenement` — le cas MESURE des deux
// films de build les plus anciens du corpus. La LECTURE est verrouillee par
// `equipment_origin_lecture_test.go`.
//
// LES DEUX TEMOINS NEGATIFS SUIVENT LA DECISION UTILISATEUR DU 2026-09-15 : l'appareil PORTE que
// le film dit mort sort `dropped` ; celui dont le film ne dit RIEN sort `unknown`. Aucun des deux
// ne sort `deployed` — ce mot est reserve a ce qu'un 103 designe, ou a une piece engendree.
func TestPieceEngendreeEstToujoursDeployee(t *testing.T) {
	// Une vie de 0 a la frame 40, qui s'acheve en (4, 0, 0). L'echantillon de la frame 20 est ce
	// qui donne un POSEUR aux poses de mi-vie : sans lui elles sortiraient toutes sans poseur, et
	// le temoin negatif ne temoignerait de rien.
	pos := []grammar.BipedPosition{
		origPos(512, 0, 0, 0, 0), origPos(512, 20, 2, 0, 0), origPos(512, 40, 4, 0, 0),
	}
	raw := []grammar.EquipmentPlacement{
		// 1. PANNEAU ne a la fin de la vie de son poseur, a ses pieds : `dropped` avant H.2.
		origPoseOf(40, wallPanelGlobalID, 4, 0, 0),
		// 2. PANNEAU sans poseur mesure (300 m de tout bipede) : `unknown` avant H.2.
		origPoseOf(20, wallPanelGlobalID, 300, 0, 0),
		// 3. TEMOIN NEGATIF : l'appareil PORTE, a l'instant de la mort ECRITE de son porteur.
		//    `dropped` — c'est bien un objet que le mort laisse tomber.
		origPoseOf(40, wallDeviceGlobalID, 4, 0, 0),
		// 4. TEMOIN NEGATIF : l'appareil porte, a mi-vie, et le film ne dit RIEN de cette pose.
		//    `unknown` depuis le 2026-09-15 : la fenetre qui la classait ne classe plus.
		origPoseOf(20, wallDeviceGlobalID, 2, 0, 0),
	}
	st := grammar.EquipmentPlacementStats{Lives: 4, Anchors: 12, Confirmed: 4}
	st.Calibration.Widths = grammar.ProfilDeBalayageParDefaut().MPP
	clock := replayClock{origin: eqOrigin, step: eqStep, frames: 200, families: map[uint32]string{
		wallPanelGlobalID: usageFamilyWall, wallDeviceGlobalID: usageFamilyWall,
	}}
	out, cov := buildEquipmentPlacements(equipmentInputs{
		Raw: raw, Stats: st, Positions: pos,
		Lives: []lifeSpan{{
			slot: 512, from: int64(eqTS(0)), to: int64(eqTS(40)), cause: CauseVieMort,
		}},
	}, clock)
	if len(out) != 4 {
		t.Fatalf("%d pose(s) publiee(s), attendu 4", len(out))
	}

	origines := map[string]map[string]int{}
	for _, pl := range out {
		if origines[pl.ID] == nil {
			origines[pl.ID] = map[string]int{}
		}
		origines[pl.ID][pl.Origin]++
	}
	panneau := origines["0x528fce46"]
	if panneau[OriginDeployed] != 2 || len(panneau) != 1 {
		t.Errorf("origines des PANNEAUX : %v, attendu 2 %q et rien d'autre",
			panneau, OriginDeployed)
	}
	appareil := origines["0x8e2dc574"]
	if appareil[OriginDropped] != 1 || appareil[OriginUnknown] != 1 {
		t.Errorf("origines de l'APPAREIL PORTE : %v, attendu 1 %q (mort ecrite) et 1 %q "+
			"(le film ne dit rien) — la regle des pieces engendrees a deborde sur un objet porte",
			appareil, OriginDropped, OriginUnknown)
	}
	if appareil[OriginDeployed] != 0 {
		t.Errorf("un appareil PORTE sort %q : ce mot est reserve a ce qu'un 103 designe "+
			"(decision utilisateur du 2026-09-15)", OriginDeployed)
	}
	// L'invariant de couverture tient : la promotion passe par le meme comptage.
	if cov.Deployed+cov.Dropped+cov.Unknown != cov.Placements {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
	if cov.Unknown != 1 {
		t.Errorf("%d pose(s) d'origine inconnue, attendu 1 — le panneau sans poseur sort %q, "+
			"l'appareil porte dont le film se tait sort %q", cov.Unknown, OriginDeployed, OriginUnknown)
	}
}

// TestEquipmentIsSpawnedPiece — le predicat, et sa frontiere : les DEUX panneaux du manifeste
// et personne d'autre. L'appareil porte, le capteur, une famille inconnue : faux.
func TestEquipmentIsSpawnedPiece(t *testing.T) {
	for id, want := range map[string]bool{
		"0x528fce46": true,  // panneau (palette rang 19)
		"0x686b40c9": true,  // panneau (ability_deployable_wall)
		"0x8e2dc574": false, // appareil de mur PORTE
		"0x72199cba": false, // capteur de menaces
		"":           false,
	} {
		if got := equipmentIsSpawnedPiece(id); got != want {
			t.Errorf("equipmentIsSpawnedPiece(%q) = %v, attendu %v", id, got, want)
		}
	}
}
