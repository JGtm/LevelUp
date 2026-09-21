package replay

// document_stances_test.go — LE PLIAGE DES ETATS DE MOUVEMENT, SANS FILM (schema 65, lot 5.3.6).
//
// CE QUE CES CAS EPINGLENT : la regle de pliage (une lecture POSEE ouvre, une lecture LEVEE
// ferme, la fin de vie ferme), le bornage a la fenetre publiee, l INDEPENDANCE des genres sur un
// meme slot, et la couverture — y compris ses denominateurs, sans lesquels « N intervalles » ne
// se juge pas.
//
// AUCUN FILM N EST LU : les lectures sont fabriquees. C est ce qui permet de prouver la regle
// plutot que de constater un chiffre.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// stTrack rend une piste publiee d UNE vie, bornee en frames.
func stTrack(slot uint32, from, to int) Track {
	return Track{Slot: slot, StartFrame: from, EndFrame: to}
}

// stLecture rend une lecture d etat, datee en microsecondes.
func stLecture(slot uint32, kind string, tsUS uint64, on bool) types.MovementStateRead {
	return types.MovementStateRead{Slot: slot, Kind: kind, TimestampUS: tsUS, On: on}
}

// stEntrees compose les entrees du pliage : origine 0, pas de 1 000 000 us (une frame par
// seconde), une piste par vie.
func stEntrees(reads []types.MovementStateRead, tracks []Track) stanceInputs {
	return stanceInputs{
		reads:  reads,
		stats:  types.MovementStateStats{Scanned: true, Records: 10, Read: len(reads)},
		origin: 0, step: 1_000_000,
		tracks: tracks,
	}
}

func TestBuildStancesPlieUnePaireEnIntervalle(t *testing.T) {
	out, cov := buildStances(stEntrees(
		[]types.MovementStateRead{
			stLecture(7, types.MovementCrouch, 2_000_000, true),
			stLecture(7, types.MovementCrouch, 5_000_000, false),
		},
		[]Track{stTrack(7, 0, 20)},
	))
	if len(out) != 1 {
		t.Fatalf("intervalles = %d, attendu 1 : %+v", len(out), out)
	}
	if got := out[0]; got.Slot != 7 || got.Kind != types.MovementCrouch || got.T0 != 2 || got.T1 != 5 {
		t.Errorf("intervalle = %+v, attendu {slot 7, crouch, 2, 5}", got)
	}
	if cov.Intervals != 1 || cov.Reads != 2 || cov.Lives != 1 || cov.TracksTotal != 1 {
		t.Errorf("couverture = %+v : attendu 1 intervalle, 2 lectures, 1 vie sur 1", cov)
	}
	if cov.ByKind[types.MovementCrouch] != 1 {
		t.Errorf("byKind = %v, attendu crouch:1", cov.ByKind)
	}
}

// TestBuildStancesFermeALaFinDeLaVie : une lecture POSEE que rien ne leve se ferme a la fin de
// la vie publiee, pas a la fin du match. Un etat qui durerait au-dela de la mort serait faux.
func TestBuildStancesFermeALaFinDeLaVie(t *testing.T) {
	out, _ := buildStances(stEntrees(
		[]types.MovementStateRead{stLecture(3, types.MovementSlide, 1_000_000, true)},
		[]Track{stTrack(3, 0, 9)},
	))
	if len(out) != 1 || out[0].T1 != 9 {
		t.Fatalf("intervalle = %+v, attendu une fin a la frame 9 (fin de vie)", out)
	}
}

// TestBuildStancesGenresIndependants : deux genres du MEME slot sont deux machines a etats.
// S accroupir et glisser ne s excluent pas dans le flux — les confondre perdrait un des deux.
func TestBuildStancesGenresIndependants(t *testing.T) {
	out, cov := buildStances(stEntrees(
		[]types.MovementStateRead{
			stLecture(5, types.MovementCrouch, 1_000_000, true),
			stLecture(5, types.MovementSlide, 2_000_000, true),
			stLecture(5, types.MovementSlide, 3_000_000, false),
			stLecture(5, types.MovementCrouch, 4_000_000, false),
		},
		[]Track{stTrack(5, 0, 20)},
	))
	if len(out) != 2 {
		t.Fatalf("intervalles = %d, attendu 2 (un par genre) : %+v", len(out), out)
	}
	// L ORDRE EST CELUI DU DOCUMENT : (t0, slot, genre). Le crouch ouvre a 1, le slide a 2.
	if out[0].Kind != types.MovementCrouch || out[0].T0 != 1 || out[0].T1 != 4 {
		t.Errorf("premier = %+v, attendu {crouch, 1, 4}", out[0])
	}
	if out[1].Kind != types.MovementSlide || out[1].T0 != 2 || out[1].T1 != 3 {
		t.Errorf("second = %+v, attendu {slide, 2, 3}", out[1])
	}
	if cov.Lives != 1 {
		t.Errorf("vies = %d, attendu 1 : deux genres sur la MEME vie", cov.Lives)
	}
}

// TestBuildStancesEcarteUneVieNonPubliee : une lecture dont le slot n a aucune piste publiee n a
// aucune fiche ou s afficher. Elle est JETEE et COMPTEE — la jeter en silence ferait croire que
// le film n en portait pas.
func TestBuildStancesEcarteUneVieNonPubliee(t *testing.T) {
	out, cov := buildStances(stEntrees(
		[]types.MovementStateRead{
			stLecture(99, types.MovementCrouch, 1_000_000, true),
			stLecture(99, types.MovementCrouch, 2_000_000, false),
		},
		[]Track{stTrack(7, 0, 20)},
	))
	if len(out) != 0 {
		t.Fatalf("intervalles = %+v, attendu aucun : le slot 99 n est pas publie", out)
	}
	if cov.Dropped != 2 {
		t.Errorf("ecartees = %d, attendu 2", cov.Dropped)
	}
	if cov.ByKind != nil {
		t.Errorf("byKind = %v, attendu nil : une table vide dirait « genre mesure a zero »", cov.ByKind)
	}
}

// TestBuildStancesSansLectureRendLaCouvertureQuandMeme : zero lecture n est pas zero
// information. `Scanned`, `Absent` et les denominateurs doivent survivre — sans eux, un film ou
// personne ne s accroupit serait indistinguable d un balayage qui n a pas tourne.
func TestBuildStancesSansLectureRendLaCouvertureQuandMeme(t *testing.T) {
	in := stEntrees(nil, []Track{stTrack(7, 0, 20)})
	in.stats = types.MovementStateStats{Scanned: true, Absent: true, Records: 42, Desyncs: 3,
		EventPacketsUnlocated: 11, MapWidths: [3]uint{15, 15, 17}}
	out, cov := buildStances(in)
	if out != nil {
		t.Fatalf("intervalles = %+v, attendu nil", out)
	}
	if !cov.Scanned || !cov.Absent || cov.Records != 42 || cov.Desyncs != 3 {
		t.Errorf("couverture = %+v : les temoins du balayage doivent survivre a une liste vide", cov)
	}
	if cov.EventPacketsUnlocated != 11 || cov.MapWidths != [3]uint{15, 15, 17} {
		t.Errorf("couverture = %+v : les paquets non localises et les largeurs doivent survivre", cov)
	}
	if cov.TracksTotal != 1 {
		t.Errorf("viesPubliees = %d, attendu 1 : le denominateur ne depend pas des lectures",
			cov.TracksTotal)
	}
}

// TestBuildStancesSansPistePublieeNePlieRien : sans piste publiee, il n y a aucune fenetre ou
// borner un intervalle. Le pliage rend la couverture et rien d autre.
func TestBuildStancesSansPistePublieeNePlieRien(t *testing.T) {
	out, cov := buildStances(stEntrees(
		[]types.MovementStateRead{stLecture(7, types.MovementCrouch, 1_000_000, true)},
		nil,
	))
	if out != nil || cov.Intervals != 0 {
		t.Fatalf("intervalles = %+v (couverture %+v), attendu aucun", out, cov)
	}
	if cov.Reads != 1 {
		t.Errorf("lectures = %d, attendu 1 : elles ont ete LUES, simplement pas plieees", cov.Reads)
	}
}
