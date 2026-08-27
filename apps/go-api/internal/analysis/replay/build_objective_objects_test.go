package replay

// build_objective_objects_test.go — LE CALQUE DES OBJETS D'OBJECTIF LIBRES, teste SANS FILM.
//
// Le producteur est PUR : il ne consomme que le balayage `ti=42` deja fait et les deux tables du
// manifeste. Ces tests montent donc des balayages a la main et verifient les quatre proprietes
// qui font la valeur du calque — publier la bonne famille, borner sur l'axe, compter ce qu'il
// ecarte, et se taire proprement.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	ooTestCrane   = uint32(0x0017592c)
	ooTestDrapeau = uint32(0x2a392328)
)

// ooClock : 100 frames de 1 s, origine a zero.
func ooClock() replayClock {
	return replayClock{origin: 0, step: 1_000_000, frames: 100}
}

// ooScan monte un balayage portant UNE creation par identifiant donne, a des instants distincts.
func ooScan(ids ...uint32) WorldObjectScan {
	scan := WorldObjectScan{Scanned: true}
	for i, id := range ids {
		scan.Creations = append(scan.Creations,
			gwTestCreation(uint32(10+i), 0, uint64(10+i)*1_000_000, id, float32(i), float32(2*i)))
	}
	return scan
}

// ooTables rend les deux tables du manifeste pour les identifiants donnes.
func ooTables(fams map[uint32]string) (map[uint32]Label, map[uint32]string) {
	labels := map[uint32]Label{}
	for id, f := range fams {
		labels[id] = Label{En: f, Fr: f}
	}
	return labels, fams
}

// TestObjectiveObjects_PublieLeCraneEtPasLeDrapeau — LA PROPRIETE CENTRALE, et elle est un
// RESULTAT DE MESURE, pas une preference : le controle 3 du lot du drapeau a echoue sur ses
// vies libres (75,6 % contre un seuil de 90 %), celui du crane a tenu.
func TestObjectiveObjects_PublieLeCraneEtPasLeDrapeau(t *testing.T) {
	labels, fams := ooTables(map[uint32]string{ooTestCrane: "ball", ooTestDrapeau: "flag"})
	lives, cov := buildObjectiveObjects(ooScan(ooTestCrane, ooTestDrapeau), labels, fams, ooClock())
	if cov.Declared != 1 {
		t.Fatalf("declares = %d, attendu 1 (le crane seul est publiable)", cov.Declared)
	}
	if len(lives) != 1 {
		t.Fatalf("%d vie(s) publiee(s), attendu 1", len(lives))
	}
	if lives[0].Family != "ball" {
		t.Errorf("famille publiee = %q, attendu \"ball\" — le drapeau ne doit PAS sortir "+
			"(controle 3 de son lot : 75,6 %% contre un seuil de 90 %%)", lives[0].Family)
	}
}

// TestObjectiveObjects_NeNommeJamaisLObjetEnDur : le libelle vient du manifeste, jamais du Go.
func TestObjectiveObjects_NeNommeJamaisLObjetEnDur(t *testing.T) {
	labels := map[uint32]Label{ooTestCrane: {En: "Oddball", Fr: "Crane"}}
	fams := map[uint32]string{ooTestCrane: "ball"}
	lives, _ := buildObjectiveObjects(ooScan(ooTestCrane), labels, fams, ooClock())
	if len(lives) != 1 || lives[0].En != "Oddball" || lives[0].Fr != "Crane" {
		t.Fatalf("libelles publies = %+v, attendus ceux du manifeste (Oddball / Crane)", lives)
	}
}

// TestObjectiveObjects_UneVieImmobileEstUneVieReelle : nee au socle et jamais bougee, elle a UN
// point — et elle se COMPTE, parce que c'est le cas qui ressemble le plus a un defaut.
func TestObjectiveObjects_UneVieImmobileEstUneVieReelle(t *testing.T) {
	labels, fams := ooTables(map[uint32]string{ooTestCrane: "ball"})
	lives, cov := buildObjectiveObjects(ooScan(ooTestCrane), labels, fams, ooClock())
	if len(lives) != 1 || len(lives[0].Pts) != 1 {
		t.Fatalf("vies = %+v, attendu une vie a un point", lives)
	}
	if lives[0].T0 != lives[0].T1 {
		t.Errorf("T0=%d T1=%d, attendus egaux pour une vie immobile", lives[0].T0, lives[0].T1)
	}
	if cov.Motionless != 1 || cov.Lives != 1 || cov.Points != 1 {
		t.Errorf("couverture = %+v, attendu 1 vie / 1 point / 1 immobile", *cov)
	}
}

// TestObjectiveObjects_HorsAxeEstECARTE_ET_COMPTE : une vie hors de l'axe n'est pas dessinable.
// La taire SANS la compter ferait passer un decalage d'horloge pour une absence d'objet.
func TestObjectiveObjects_HorsAxeEstECARTE_ET_COMPTE(t *testing.T) {
	labels, fams := ooTables(map[uint32]string{ooTestCrane: "ball"})
	scan := WorldObjectScan{Scanned: true, Creations: []filmdec.EquipmentCreation{
		gwTestCreation(10, 0, 5_000_000_000, ooTestCrane, 1, 1), // frame 5000 : hors des 100
	}}
	lives, cov := buildObjectiveObjects(scan, labels, fams, ooClock())
	if len(lives) != 0 {
		t.Fatalf("%d vie(s) publiee(s) hors axe, attendu 0", len(lives))
	}
	if cov.OutOfAxis != 1 {
		t.Errorf("horsAxe = %d, attendu 1 — l'ecart doit se COMPTER, pas disparaitre", cov.OutOfAxis)
	}
}

// TestObjectiveObjects_TroisSilencesDistincts : la couverture doit distinguer « pas balaye »,
// « rien de declare » et « declare mais absent du film ». Sans elle, les trois se confondent.
func TestObjectiveObjects_TroisSilencesDistincts(t *testing.T) {
	labels, fams := ooTables(map[uint32]string{ooTestCrane: "ball"})

	_, pasBalaye := buildObjectiveObjects(WorldObjectScan{}, labels, fams, ooClock())
	if pasBalaye.Scanned || pasBalaye.Lives != 0 {
		t.Errorf("pas balaye : %+v, attendu Scanned=false et 0 vie", *pasBalaye)
	}
	_, rienDeclare := buildObjectiveObjects(ooScan(ooTestCrane), nil, nil, ooClock())
	if !rienDeclare.Scanned || rienDeclare.Declared != 0 {
		t.Errorf("rien declare : %+v, attendu Scanned=true et 0 declare", *rienDeclare)
	}
	_, absentDuFilm := buildObjectiveObjects(ooScan(ooTestDrapeau), labels, fams, ooClock())
	if !absentDuFilm.Scanned || absentDuFilm.Declared != 1 || absentDuFilm.Lives != 0 {
		t.Errorf("declare mais absent : %+v, attendu Scanned=true, 1 declare, 0 vie", *absentDuFilm)
	}
}
