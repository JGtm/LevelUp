package grammar

// player_entities_test.go — CE QUE LE BALAYAGE PAR ENTITE DOIT TENIR (lot M2.1, 2026-09-23).
//
//	ENT-ACC     L'accumulateur rend une entree par slot, ses rangs de premiere et de derniere
//	            image-cle, ses TROUS (jamais un depart), une presence par image-cle meme quand le
//	            slot y porte deux records, et l'instabilite comptee.
//	ENT-BOBINE  Sur les sept bobines par build : une entite par slot lu (le compte historique du
//	            rapport), images-cles porteuses strictement croissantes, aucune entite instable,
//	            et l'equipe de chaque entite egale a celle de la table de CONTROLE quand son index
//	            y est publie.
//	ENT-REPRISE `11de8353` porte 27 entites pour 26 index : l'index repris l'est par DEUX entites
//	            a fenetres DISJOINTES — deux occupants successifs, jamais une entite reutilisee.

import (
	"reflect"
	"testing"
)

// TestAccumulateurDEntites execute ENT-ACC sur une suite d'images-cles fabriquee.
//
// LE TEMOIN EST AU GABARIT DE LA SONDE P4 (`b1ad85eb`) : un occupant qui manque a UNE image-cle
// puis revient (le trou de MONEY a f613), deux entites qui se relaient sur le meme index avec
// deux designateurs differents (les bots d'index 8), et un record double dans une image-cle.
func TestAccumulateurDEntites(t *testing.T) {
	a := nouvelAccumulateurDEntites()
	suite := []struct {
		ts      uint64
		records [][3]int // slot, index, designateur
	}{
		{100, [][3]int{{1297, 0, 0}, {1299, 1, 1}, {1530, 8, 0}}},
		{200, [][3]int{{1297, 0, 0}, {1299, 1, 1}, {1530, 8, 0}, {1299, 1, 1}}},
		{300, [][3]int{{1299, 1, 1}}}, // 1297 manque : un TROU, pas un depart
		{400, [][3]int{{1297, 0, 0}, {1299, 1, 1}, {2145, 8, 1}}},
		{500, [][3]int{{1297, 0, 0}, {2145, 8, 1}, {9, 2, 0}, {9, 2, 1}}}, // 9 : instable
	}
	for _, kf := range suite {
		rang := a.ouvrirImageCle(kf.ts)
		for _, r := range kf.records {
			a.noter(rang, r[0], r[1], r[2])
		}
	}
	got := a.publier()
	want := PlayerEntityScan{
		Scanned:     true,
		KeyframesUS: []uint64{100, 200, 300, 400, 500},
		Entities: []PlayerEntity{
			{Slot: 1297, Index: 0, Team: 0, FirstKF: 0, LastKF: 4, Seen: 4},
			{Slot: 1299, Index: 1, Team: 1, FirstKF: 0, LastKF: 3, Seen: 4},
			{Slot: 1530, Index: 8, Team: 0, FirstKF: 0, LastKF: 1, Seen: 2},
			{Slot: 2145, Index: 8, Team: 1, FirstKF: 3, LastKF: 4, Seen: 2},
			{Slot: 9, Index: 2, Team: 0, FirstKF: 4, LastKF: 4, Seen: 1, Unstable: true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("balayage par entite :\n obtenu %+v\n attendu %+v", got, want)
	}
	if h := got.Holes(); h != 1 {
		t.Errorf("trous : %d, attendu 1 (1297 absent de la 3e image-cle)", h)
	}
	if u := got.UnstableEntities(); u != 1 {
		t.Errorf("entites instables : %d, attendu 1", u)
	}
	if d := a.divergences(); d != 1 {
		t.Errorf("divergences de designateur : %d, attendu 1 (le compteur historique du rapport)", d)
	}
	if !got.AtStart(got.Entities[0]) || !got.AtEnd(got.Entities[0]) || got.AtEnd(got.Entities[1]) {
		t.Errorf("AtStart / AtEnd mal lus sur %+v", got.Entities[:2])
	}
}

// TestEntitesTi9SurLesBobines execute ENT-BOBINE et ENT-REPRISE.
func TestEntitesTi9SurLesBobines(t *testing.T) {
	for _, b := range bobinesEquipes() {
		teams, rep, ents := ScanPlayerTeams(NewFilmContext(bobineFilm(t, b.film)))
		if !ents.Scanned {
			t.Fatalf("%s : entites non balayees alors que la table est lue", b.film)
		}
		if len(ents.Entities) != rep.Entities || len(ents.Entities) != b.entites {
			t.Errorf("%s : %d entites, rapport %d, gel %d", b.film, len(ents.Entities),
				rep.Entities, b.entites)
		}
		if len(ents.KeyframesUS) != rep.Packets {
			t.Errorf("%s : %d images-cles porteuses, le rapport compte %d paquets porteurs",
				b.film, len(ents.KeyframesUS), rep.Packets)
		}
		for i := 1; i < len(ents.KeyframesUS); i++ {
			if ents.KeyframesUS[i] <= ents.KeyframesUS[i-1] {
				t.Errorf("%s : images-cles porteuses non croissantes au rang %d", b.film, i)
			}
		}
		verifierEntitesContreLaTable(t, b, teams, ents)
		if b.film == "11de8353" {
			verifierReprise(t, ents)
		}
	}
}

// verifierEntitesContreLaTable tient la coherence des deux vues d'une meme passe.
func verifierEntitesContreLaTable(t *testing.T, b bobineEquipes, teams map[int]int, ents PlayerEntityScan) {
	t.Helper()
	auDepart := 0
	for _, e := range ents.Entities {
		if e.FirstKF > e.LastKF || e.Seen < 1 || e.Seen > e.LastKF-e.FirstKF+1 {
			t.Errorf("%s slot %d : rangs [%d..%d] vus %d incoherents", b.film, e.Slot, e.FirstKF,
				e.LastKF, e.Seen)
		}
		if e.Unstable {
			t.Errorf("%s slot %d : entite instable — mesure zero sur les bobines", b.film, e.Slot)
		}
		if v, publie := teams[e.Index]; publie && v != e.Team {
			t.Errorf("%s slot %d index %d : equipe %d, la table de controle dit %d", b.film,
				e.Slot, e.Index, e.Team, v)
		}
		if ents.AtStart(e) {
			auDepart++
		}
	}
	if auDepart != b.premiers {
		t.Errorf("%s : %d entites a la premiere image-cle porteuse, attendu %d", b.film, auDepart,
			b.premiers)
	}
}

// verifierReprise : sur `11de8353`, l'index repris l'est par deux entites SUCCESSIVES.
func verifierReprise(t *testing.T, ents PlayerEntityScan) {
	t.Helper()
	parIndex := map[int][]PlayerEntity{}
	for _, e := range ents.Entities {
		parIndex[e.Index] = append(parIndex[e.Index], e)
	}
	reprises := 0
	for idx, es := range parIndex {
		if len(es) < 2 {
			continue
		}
		reprises++
		if len(es) != 2 || es[0].LastKF >= es[1].FirstKF {
			t.Errorf("11de8353 index %d : %d entites, fenetres %+v — une reprise est une SUITE "+
				"d'occupants, jamais deux a la fois", idx, len(es), es)
		}
	}
	if reprises != 1 {
		t.Errorf("11de8353 : %d index repris, attendu 1 (27 entites pour 26 index)", reprises)
	}
}
