package replay

// equipment_origin_lecture_test.go — LA GRAMMAIRE DECIDE, LE REPLI COMPTE (lot 1.9.1, D13/D14).
//
// # CE QUE CE FICHIER VERROUILLE
//
// `equipment_origin_test.go` verrouille la MACHINE de la fenetre temporelle — le decoupage des
// vies, le choix de la vie, l'invariant de couverture. Ce fichier-ci verrouille la CASCADE qui
// est passee devant elle : les trois lectures du film, leur ordre, leur contradiction comptee,
// et les deux replis nommes.
//
// # CHAQUE TEST PORTE SA MUTATION
//
// Un test qui verifie qu'une lecture DECIDE ne prouve rien tant qu'on n'a pas montre que sans
// elle le verdict change. Chaque cas ci-dessous se joue donc DEUX FOIS : avec le signal ecrit,
// puis sans — et la seconde passe doit basculer la PROVENANCE (et, quand c'est le cas, le
// verdict). C'est ce qui interdit qu'un refactor debranche une lecture en laissant les tests
// verts parce que la fenetre rendait par hasard la meme chose.
//
// AUCUN OCTET DE FILM N'EST LU.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// lecPose fabrique une pose brute avec une CLE DE VIE choisie — c'est elle que l'evenement 103
// designe, et sans elle la lecture n'a rien a apparier.
func lecPose(frame int, globalID uint32, life filmdec.EquipmentLifeKey, x float32,
) filmdec.EquipmentPlacement {
	p := origPoseOf(frame, globalID, x, 0, 0)
	p.Life = life
	return p
}

// lecSpawn fabrique un evenement 103 qui designe `life` a la frame donnee.
func lecSpawn(frame int, life filmdec.EquipmentLifeKey) filmdec.EquipmentSpawnEvent {
	return filmdec.EquipmentSpawnEvent{
		TimestampUS: eqTS(frame), Spawned: life, SpawnedValid: true,
		Source: filmdec.EquipmentLifeKey{Slot: 900}, SourceValid: true,
	}
}

// lecVieMorte fabrique une vie de siege qui se TERMINE PAR UNE MORT a la frame donnee — la
// forme exacte que le registre d'identite rend (`lifeSpan.cause == CauseVieMort`).
func lecVieMorte(slot uint32, deFrame, aFrame int) lifeSpan {
	return lifeSpan{
		slot: slot, from: int64(eqTS(deFrame)), to: int64(eqTS(aFrame)), cause: CauseVieMort,
	}
}

// lecPrise fabrique un ramassage d'equipement (`taken`) sur un siege, a la frame donnee.
func lecPrise(slot uint32, frame int) filmdec.EquipmentChange {
	return filmdec.EquipmentChange{
		TimestampUS: eqTS(frame), Slot: slot, Kind: filmdec.EquipmentTaken,
	}
}

// lecClock rend l'horloge de ces tests, avec son compteur de replis.
func lecClock(fb *fallback.Compteur) replayClock {
	return replayClock{origin: eqOrigin, step: eqStep, frames: 200, fb: fb, families: map[uint32]string{
		wallPanelGlobalID: usageFamilyWall, wallDeviceGlobalID: usageFamilyWall,
	}}
}

// lecCuisson assemble une cuisson et rend ses poses, sa couverture et le compte des replis.
func lecCuisson(in equipmentInputs) ([]EquipmentPlacement, *EquipmentPlacementCoverage, *fallback.Compteur) {
	fb := fallback.NouveauCompteur()
	in.Stats = filmdec.EquipmentPlacementStats{Lives: len(in.Raw), Anchors: 12, Confirmed: len(in.Raw)}
	in.Stats.Calibration.Widths = filmdec.ProfilDeBalayageParDefaut().MPP
	// LES DENOMINATEURS DU BALAYAGE, comme la production les fournit : sans eux `spawnLists`
	// resterait a zero et le test ne dirait rien du cas « le film porte des listes mais aucun
	// evenement », qui est precisement la question ouverte D2 (1.9.1).
	in.SpawnStats = filmdec.EquipmentSpawnStats{
		Chunks: 1, Packets: 100, Lists: 40, Events: len(in.Spawns), WithSpawned: len(in.Spawns),
	}
	out, cov := buildEquipmentPlacements(in, lecClock(fb))
	return out, cov, fb
}

// TestOriginePoseLitLEvenementEngendre — LA PREMIERE LECTURE, ET SA MUTATION.
//
// Un panneau ne A L'INSTANT EXACT de la mort de son poseur : la fenetre temporelle dit `dropped`,
// et le film dit « une PIECE a ete engendree ». C'est la lecture qui decide.
//
// MUTATION : l'evenement retire, la meme pose passe au REPLI du manifeste — compte, et nomme.
// Le VERDICT ne change pas (un panneau n'existe que deploye), mais la PROVENANCE si : c'est
// exactement la difference que `coverage.placements.byCause` existe pour publier.
func TestOriginePoseLitLEvenementEngendre(t *testing.T) {
	vie := filmdec.EquipmentLifeKey{Slot: 1030, Gen: 2}
	pos := []filmdec.BipedPosition{origPos(512, 0, 0, 0, 0), origPos(512, 40, 4, 0, 0)}
	raw := []filmdec.EquipmentPlacement{lecPose(40, wallPanelGlobalID, vie, 4)}

	_, cov, _ := lecCuisson(equipmentInputs{
		Raw: raw, Positions: pos, Spawns: []filmdec.EquipmentSpawnEvent{lecSpawn(40, vie)},
		Lives: []lifeSpan{lecVieMorte(512, 0, 40)},
	})
	if got := cov.ByCause[CausePoseEvenementEngendre]; got != 1 {
		t.Fatalf("byCause[%s] = %d, attendu 1 — l'evenement 103 ne decide pas",
			CausePoseEvenementEngendre, got)
	}
	if cov.SpawnEvents != 1 || cov.SpawnLists != 40 {
		t.Errorf("spawnEvents = %d / spawnLists = %d, attendu 1 et 40 : les denominateurs de la "+
			"lecture ne sont pas publies", cov.SpawnEvents, cov.SpawnLists)
	}

	// MUTATION : plus d'evenement. Le panneau reste `deployed` — il n'existe QUE deploye — mais
	// par le REPLI du manifeste et non plus par une lecture : c'est ce que `byCause` existe pour
	// dire.
	out, cov, fb := lecCuisson(equipmentInputs{
		Raw: raw, Positions: pos, Lives: []lifeSpan{lecVieMorte(512, 0, 40)},
	})
	if cov.ByCause[CausePoseEvenementEngendre] != 0 {
		t.Errorf("sans evenement, byCause[%s] = %d, attendu 0",
			CausePoseEvenementEngendre, cov.ByCause[CausePoseEvenementEngendre])
	}
	if got := cov.ByCause[CausePoseManifeste]; got != 1 {
		t.Errorf("sans evenement, byCause[%s] = %d, attendu 1 — le repli du manifeste doit "+
			"prendre le relais", CausePoseManifeste, got)
	}
	if got := fb.Compte(fallback.NomPieceEngendreeSansEvenement); got != 1 {
		t.Errorf("le repli %q s'est declenche %d fois, attendu 1",
			fallback.NomPieceEngendreeSansEvenement, got)
	}
	if out[0].Origin != OriginDeployed {
		t.Errorf("origine %q, attendu %q : un panneau n'existe que deploye",
			out[0].Origin, OriginDeployed)
	}
}

// TestFenetreFausseeNeChangeRienAuPanneau — LA SECONDE MOITIE DE LA MUTATION DU MUR : une
// fenetre temporelle faussee ne deplace pas un panneau que le film designe.
//
// La pose est posee A MI-VIE (la fenetre dirait `deployed`) puis A LA FIN (elle dirait
// `dropped`) : l'evenement 103 rend `deployed` dans les DEUX cas, par la meme provenance.
func TestFenetreFausseeNeChangeRienAuPanneau(t *testing.T) {
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0), origPos(512, 20, 2, 0, 0), origPos(512, 40, 4, 0, 0),
	}
	for _, frame := range []int{20, 40} {
		vie := filmdec.EquipmentLifeKey{Slot: uint32(1000 + frame), Gen: 1}
		out, cov, _ := lecCuisson(equipmentInputs{
			Raw:       []filmdec.EquipmentPlacement{lecPose(frame, wallPanelGlobalID, vie, 4)},
			Positions: pos,
			Spawns:    []filmdec.EquipmentSpawnEvent{lecSpawn(frame, vie)},
			Lives:     []lifeSpan{lecVieMorte(512, 0, 40)},
		})
		if out[0].Origin != OriginDeployed || cov.ByCause[CausePoseEvenementEngendre] != 1 {
			t.Errorf("pose a la frame %d : origine %q / causes %v, attendu %q par %q",
				frame, out[0].Origin, cov.ByCause, OriginDeployed, CausePoseEvenementEngendre)
		}
	}
}

// TestOriginePoseLitLaMortEcrite — LA DEUXIEME LECTURE, ET SA MUTATION.
//
// Un appareil PORTE cree a l'instant d'une mort ECRITE de son poseur : `dropped`, par lecture.
// MUTATION : la mort retiree, la meme pose tombe au REPLI de la fenetre — compte.
func TestOriginePoseLitLaMortEcrite(t *testing.T) {
	pos := []filmdec.BipedPosition{origPos(512, 0, 0, 0, 0), origPos(512, 40, 4, 0, 0)}
	raw := []filmdec.EquipmentPlacement{
		lecPose(40, wallDeviceGlobalID, filmdec.EquipmentLifeKey{Slot: 1040, Gen: 0}, 4),
	}

	out, cov, _ := lecCuisson(equipmentInputs{
		Raw: raw, Positions: pos, Lives: []lifeSpan{lecVieMorte(512, 0, 40)},
	})
	if out[0].Origin != OriginDropped || cov.ByCause[CausePoseMortEcrite] != 1 {
		t.Fatalf("origine %q / causes %v, attendu %q par %q",
			out[0].Origin, cov.ByCause, OriginDropped, CausePoseMortEcrite)
	}

	// MUTATION : plus de mort ecrite. AUCUN signal ne couvre la pose, et elle SORT DU VOCABULAIRE
	// PLEIN : `unknown`, cause `none` (decision utilisateur du 2026-09-15). La fenetre temporelle
	// qui la classait `dropped` a ete retiree du paquet avec ses trois tests.
	out, cov, _ = lecCuisson(equipmentInputs{Raw: raw, Positions: pos})
	if cov.ByCause[CausePoseMortEcrite] != 0 {
		t.Errorf("sans mort ecrite, byCause[%s] = %d, attendu 0",
			CausePoseMortEcrite, cov.ByCause[CausePoseMortEcrite])
	}
	if got := cov.ByCause[CausePoseAucunSignal]; got != 1 {
		t.Errorf("sans mort ecrite, byCause[%s] = %d, attendu 1 — le silence du film doit se "+
			"publier tel quel", CausePoseAucunSignal, got)
	}
	if out[0].Origin != OriginUnknown {
		t.Errorf("origine %q, attendu %q : rien ne se devine", out[0].Origin, OriginUnknown)
	}
}

// TestOriginePoseLitLaPriseEcrite — LA TROISIEME LECTURE : un `taken` du poseur a l'instant de
// la pose dit que le porteur a ECHANGE. Il LACHE donc l'objet qu'il tenait, et l'etiquette est
// `dropped` — comme a la mort. Decision utilisateur du 2026-09-15 : l'etiquette ne distingue pas
// les deux lachers, la CAUSE se publie dans la couverture.
//
// LE CAS EST CHOISI POUR QUE LE SILENCE SOIT VISIBLE : la pose est a MI-VIE, aucune mort n'est
// ecrite, une prise l'est. La mutation retire la prise : plus rien ne couvre la pose, elle sort
// `unknown`.
func TestOriginePoseLitLaPriseEcrite(t *testing.T) {
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0), origPos(512, 20, 2, 0, 0), origPos(512, 40, 4, 0, 0),
	}
	raw := []filmdec.EquipmentPlacement{
		lecPose(20, wallDeviceGlobalID, filmdec.EquipmentLifeKey{Slot: 1041, Gen: 0}, 2),
	}

	out, cov, _ := lecCuisson(equipmentInputs{
		Raw: raw, Positions: pos, Changes: []filmdec.EquipmentChange{lecPrise(512, 20)},
	})
	if out[0].Origin != OriginDropped || cov.ByCause[CausePosePriseEcrite] != 1 {
		t.Fatalf("origine %q / causes %v, attendu %q par %q",
			out[0].Origin, cov.ByCause, OriginDropped, CausePosePriseEcrite)
	}

	// MUTATION : plus de prise ecrite.
	out, cov, _ = lecCuisson(equipmentInputs{Raw: raw, Positions: pos})
	if cov.ByCause[CausePoseAucunSignal] != 1 || out[0].Origin != OriginUnknown {
		t.Errorf("sans prise ecrite : origine %q / causes %v, attendu %q par %q",
			out[0].Origin, cov.ByCause, OriginUnknown, CausePoseAucunSignal)
	}
}

// TestOriginePoseCompteLaContradiction — D14 (b) : deux lectures qui se contredisent ne sont PAS
// un repli, elles sont une CONTRADICTION comptee. La mort tranche — elle libere tout ce que le
// joueur tenait —, et la contradiction est publiee sous son propre nom.
func TestOriginePoseCompteLaContradiction(t *testing.T) {
	pos := []filmdec.BipedPosition{origPos(512, 0, 0, 0, 0), origPos(512, 40, 4, 0, 0)}
	out, cov, fb := lecCuisson(equipmentInputs{
		Raw: []filmdec.EquipmentPlacement{
			lecPose(40, wallDeviceGlobalID, filmdec.EquipmentLifeKey{Slot: 1042, Gen: 0}, 4),
		},
		Positions: pos,
		Lives:     []lifeSpan{lecVieMorte(512, 0, 40)},
		Changes:   []filmdec.EquipmentChange{lecPrise(512, 40)},
	})
	if got := cov.ByCause[CausePoseContradiction]; got != 1 {
		t.Fatalf("byCause[%s] = %d, attendu 1 — la contradiction disparait en silence",
			CausePoseContradiction, got)
	}
	if out[0].Origin != OriginDropped {
		t.Errorf("origine %q, attendu %q : les deux causes sont des lachers",
			out[0].Origin, OriginDropped)
	}
	if got := fb.Compte(fallback.NomPieceEngendreeSansEvenement); got != 0 {
		t.Errorf("une contradiction a declenche un repli %d fois : D14 (b) l'interdit — un "+
			"desaccord entre deux lectures n'est pas un film muet", got)
	}
}

// TestByCauseSommeAuxPlacements — L'INVARIANT DE LA TABLE DES PROVENANCES : toute pose publiee
// porte une cause, et une seule. Un ecart signale un chemin de classification qui a fui — la
// meme regle que l'equilibre des origines, un etage plus bas.
func TestByCauseSommeAuxPlacements(t *testing.T) {
	vie := filmdec.EquipmentLifeKey{Slot: 1050, Gen: 3}
	pos := []filmdec.BipedPosition{
		origPos(512, 0, 0, 0, 0), origPos(512, 20, 2, 0, 0), origPos(512, 40, 4, 0, 0),
	}
	_, cov, _ := lecCuisson(equipmentInputs{
		Raw: []filmdec.EquipmentPlacement{
			lecPose(40, wallPanelGlobalID, vie, 4),                                     // spawn_event
			lecPose(40, wallDeviceGlobalID, filmdec.EquipmentLifeKey{Slot: 1051}, 4),   // death_written
			lecPose(20, wallDeviceGlobalID, filmdec.EquipmentLifeKey{Slot: 1052}, 2),   // none
			lecPose(20, wallDeviceGlobalID, filmdec.EquipmentLifeKey{Slot: 1053}, 300), // no_owner
			lecPose(20, wallPanelGlobalID, filmdec.EquipmentLifeKey{Slot: 1054}, 300),  // manifest_piece
		},
		Positions: pos,
		Spawns:    []filmdec.EquipmentSpawnEvent{lecSpawn(40, vie)},
		Lives:     []lifeSpan{lecVieMorte(512, 0, 40)},
	})
	somme := 0
	for _, n := range cov.ByCause {
		somme += n
	}
	if somme != cov.Placements {
		t.Fatalf("somme des causes %d != %d poses publiees : %v", somme, cov.Placements, cov.ByCause)
	}
	for _, c := range []string{
		CausePoseEvenementEngendre, CausePoseMortEcrite, CausePoseAucunSignal,
		CausePoseSansPoseur, CausePoseManifeste,
	} {
		if cov.ByCause[c] != 1 {
			t.Errorf("byCause[%s] = %d, attendu 1 — table complete : %v", c, cov.ByCause[c], cov.ByCause)
		}
	}
}

// TestDesignationExigeLeTempsEtPasSeulementLaCle — LA GENERATION NE FAIT QUE DEUX BITS, donc un
// siege repasse par la MEME cle plusieurs fois dans un match. Un appariement par cle SEULE fait
// « designer » des poses qu'aucun evenement ne concerne : mesure du 2026-09-15, 3 evenements
// « designaient » 83 poses de `d9781168`. Le temps est la seconde moitie de la cle.
func TestDesignationExigeLeTempsEtPasSeulementLaCle(t *testing.T) {
	vie := filmdec.EquipmentLifeKey{Slot: 1060, Gen: 1}
	src := nouvelleSourceOrigine([]filmdec.EquipmentSpawnEvent{lecSpawn(40, vie)}, nil, nil)
	cas := []struct {
		nom   string
		frame int
		want  bool
	}{
		// Une frame vaut 100 ms (`eqStep`) : la fenetre de 200 ms en couvre deux, APRES la
		// creation seulement.
		{"l'evenement suit la creation de deux images (+200 ms, la borne)", 38, true},
		{"l'evenement suit la creation d'une image (+100 ms)", 39, true},
		{"l'evenement est simultane", 40, true},
		{"l'evenement suit de TROIS images (+300 ms) : hors fenetre", 37, false},
		{"l'evenement PRECEDE la creation d'une image", 41, false},
		{"la cle a reboucle dix secondes plus tot", 140, false},
	}
	for _, c := range cas {
		p := lecPose(c.frame, wallPanelGlobalID, vie, 0)
		if got := src.designeParUnEvenement(p); got != c.want {
			t.Errorf("%s : designe = %v, attendu %v", c.nom, got, c.want)
		}
	}
	autre := lecPose(40, wallPanelGlobalID, filmdec.EquipmentLifeKey{Slot: 1060, Gen: 2}, 0)
	if src.designeParUnEvenement(autre) {
		t.Error("une GENERATION differente est designee : la cle de vie est la paire complete")
	}
}
