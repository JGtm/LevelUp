package replay

// occupants_test.go — LA LIAISON ENTREE -> ENTITE, ET CE QU'ELLE REFUSE (lot M2.3, 2026-09-23).
//
//	O-CONTESTEE   deux humains du meme index dont les vies designent la meme entite : elle n'est
//	              liee a PERSONNE (jamais « au premier »), et le compte le dit ;
//	O-INSTABLE    une entite dont l'index ou l'equipe change n'est pas une lecture sure : non liee ;
//	O-SANSVIE     un humain sans vie se lie a l'unique entite libre de son index, s'il est seul ;
//	O-DIVERGENTE  deux entites d'une meme entree d'equipes differentes : pas d'equipe, comptee ;
//	O-SANSBALAYAGE sans entite lue, l'equipe est celle de l'index et la presence l'enveloppe des
//	              vies — ce qui se publiait avant le lot M2.3 ;
//	O-SIMULTANEES le nombre d'entites qu'une equipe tient ensemble a une image-cle (la borne LUE de
//	              sa capacite) compte les trous et pas les entites instables.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// scanDeTest fabrique un balayage aux images-cles donnees (frames d'un document a 100 ms).
func scanDeTest(kf []int, ents ...grammar.PlayerEntity) grammar.PlayerEntityScan {
	s := grammar.PlayerEntityScan{Scanned: true, Entities: ents}
	for _, f := range kf {
		s.KeyframesUS = append(s.KeyframesUS, uint64(f)*100_000)
	}
	return s
}

func entreesDeTest(scan grammar.PlayerEntityScan) entreesDesOccupants {
	return entreesDesOccupants{scan: scan, horloge: replayClock{step: 100_000, frames: siegeFrames}}
}

func TestOccupantsEntiteContesteeNEstLieeAPersonne(t *testing.T) {
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 3, Team: 1, LastKF: 2, Seen: 3})
	roster := []RosterEntry{entree(3, "100"), entree(3, "200")}
	tracks := []Track{vieDe("100", 5, 20), vieDe("200", 30, 60)}
	occ := lierLesOccupants(roster, tracks, entreesDeTest(scan))
	if occ.entitesContestees != 1 || len(occ.parEntree[0].entites) != 0 || len(occ.parEntree[1].entites) != 0 {
		t.Fatalf("contestees %d, liees %v / %v : une entite revendiquee deux fois n'est a personne",
			occ.entitesContestees, occ.parEntree[0].entites, occ.parEntree[1].entites)
	}
}

func TestOccupantsEntiteInstableNEstPasLiee(t *testing.T) {
	scan := scanDeTest([]int{10, 50}, grammar.PlayerEntity{Slot: 7, Index: 3, Team: 1, LastKF: 1, Seen: 2,
		Unstable: true})
	occ := lierLesOccupants([]RosterEntry{entree(3, "100")}, []Track{vieDe("100", 5, 40)}, entreesDeTest(scan))
	if occ.entitesNonLiees != 1 || len(occ.parEntree[0].entites) != 0 || occ.parEntree[0].lue {
		t.Fatalf("non liees %d, entites %v : une entite instable n'est pas une lecture sure",
			occ.entitesNonLiees, occ.parEntree[0].entites)
	}
}

func TestOccupantsHumainSansVieSeLieALUniqueEntite(t *testing.T) {
	// Present sans corps (il n'est jamais apparu) : l'entite dit qu'il etait la, de la frame 50 a
	// la veille de l'image-cle suivante.
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 4, Team: 0, FirstKF: 1,
		LastKF: 1, Seen: 1})
	occ := lierLesOccupants([]RosterEntry{entree(4, "100")}, nil, entreesDeTest(scan))
	p := occ.parEntree[0].presence
	if len(p) != 1 || p[0] != (intervalleDePresence{de: 50, a: 50, aMax: 89}) || !occ.parEntree[0].lue {
		t.Fatalf("presence %+v : attendu [50, 50] affichee jusqu'a 89 (veille de l'image-cle suivante)", p)
	}
}

func TestOccupantsEquipesDivergentesNeSePublientPas(t *testing.T) {
	scan := scanDeTest([]int{10, 50, 90},
		grammar.PlayerEntity{Slot: 7, Index: 4, Team: 0, LastKF: 0, Seen: 1},
		grammar.PlayerEntity{Slot: 8, Index: 4, Team: 1, FirstKF: 2, LastKF: 2, Seen: 1})
	occ := lierLesOccupants([]RosterEntry{entree(4, "100")},
		[]Track{vieDe("100", 5, 12), vieDe("100", 70, 95)}, entreesDeTest(scan))
	if occ.parEntree[0].equipe != nil || occ.equipesDivergentes != 1 {
		t.Fatalf("equipe %v, divergentes %d : deux equipes pour une entree, aucune ne se publie",
			deref(occ.parEntree[0].equipe), occ.equipesDivergentes)
	}
}

func TestOccupantsSansBalayagePublientCeQuiSePubliait(t *testing.T) {
	in := entreesDeTest(grammar.PlayerEntityScan{})
	in.parIndex = map[int]int{3: 1}
	occ := lierLesOccupants([]RosterEntry{entree(3, "100")}, []Track{vieDe("100", 5, 20), vieDe("100", 30, 40)}, in)
	o := occ.parEntree[0]
	if o.equipe == nil || *o.equipe != 1 || o.lue || len(o.presence) != 1 ||
		o.presence[0] != (intervalleDePresence{de: 5, a: 40, aMax: 40}) {
		t.Fatalf("occupant %+v (equipe %v) : sans entite, l'equipe de l'index et l'enveloppe des vies",
			o, deref(o.equipe))
	}
}

func TestOccupantsBotSansDeclarationNEstPasLie(t *testing.T) {
	scan := scanDeTest([]int{10, 50}, grammar.PlayerEntity{Slot: 7, Index: 8, Team: 0, LastKF: 1, Seen: 2})
	roster := []RosterEntry{{FilmIndex: 8, Name: "343 X [bot]", Bot: true}}
	occ := lierLesOccupants(roster, []Track{{Bot: "343 X [bot]", StartFrame: 5, EndFrame: 30}}, entreesDeTest(scan))
	if len(occ.parEntree[0].entites) != 0 || occ.entitesNonLiees != 1 {
		t.Fatalf("un bot se lie par ses declarations BOT_METADATA, jamais par son seul index : %+v",
			occ.parEntree[0])
	}
}

// vieDe fabrique une piste nommee par un xuid sur l'intervalle de frames donne.
func vieDe(xuid string, debut, fin int) Track {
	return Track{Slot: 1, XUID: xuid, StartFrame: debut, EndFrame: fin}
}

// TestEntitesSimultaneesComptentLesTrousEtPasLesInstables (O-SIMULTANEES) : a l'image-cle 1,
// l'equipe 0 tient trois entites — dont une dans un TROU (lue avant et apres, pas un depart) ; une
// entite instable ne compte pas, et une entite partie avant l'arrivee d'une autre ne s'y ajoute pas.
func TestEntitesSimultaneesComptentLesTrousEtPasLesInstables(t *testing.T) {
	scan := scanDeTest([]int{10, 30, 50},
		grammar.PlayerEntity{Slot: 1, Index: 0, Team: 0, FirstKF: 0, LastKF: 2, Seen: 3},
		grammar.PlayerEntity{Slot: 2, Index: 1, Team: 0, FirstKF: 0, LastKF: 2, Seen: 2}, // trou a 1
		grammar.PlayerEntity{Slot: 3, Index: 2, Team: 0, FirstKF: 1, LastKF: 1, Seen: 1},
		grammar.PlayerEntity{Slot: 4, Index: 3, Team: 0, FirstKF: 0, LastKF: 2, Seen: 3, Unstable: true},
		grammar.PlayerEntity{Slot: 5, Index: 4, Team: 1, FirstKF: 0, LastKF: 0, Seen: 1},
		grammar.PlayerEntity{Slot: 6, Index: 5, Team: 1, FirstKF: 1, LastKF: 2, Seen: 2})
	got := entitesSimultanees(scan)
	if got[0] != 3 || got[1] != 1 || len(got) != 2 {
		t.Fatalf("simultanees %v : attendu equipe 0 -> 3 (trou compris, instable exclue), "+
			"equipe 1 -> 1 (un relais n'est pas deux occupants)", got)
	}
}
