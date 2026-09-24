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
//
// DEPUIS LE LOT D-fix (2026-09-24), « au depart » se lit DEUX fois, et les deux doivent valoir
// `premiers` : les entites LUES a la premiere image-cle porteuse (la marche ne perd plus le joueur
// gere que la fausse ancre 192 effacait dans l'image-cle d'avant-match de `bcb6d393` et de
// `fb1a1a72` — ROUGE avant le lot : 7 sur 8), et celles que [PlayerEntityScan.AtStart] dit la au
// coup d'envoi. Aucun doute ne reste sur les sept bobines.
func verifierEntitesContreLaTable(t *testing.T, b bobineEquipes, teams map[int]int, ents PlayerEntityScan) {
	t.Helper()
	auDepart, lusAuDepart := 0, 0
	if len(ents.Doutes) != 0 {
		t.Errorf("%s : %d absence(s) non prouvee(s) %v — la marche du film ne doit plus en laisser",
			b.film, len(ents.Doutes), ents.Doutes)
	}
	for _, e := range ents.Entities {
		if e.FirstKF == 0 {
			lusAuDepart++
		}
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
	if auDepart != b.premiers || lusAuDepart != b.premiers {
		t.Errorf("%s : %d entites au coup d'envoi, %d lues a la premiere image-cle porteuse, attendu %d",
			b.film, auDepart, lusAuDepart, b.premiers)
	}
}

// TestSansPreuveLeDouteRattrapeLaPerte : LE PRINCIPE, sur de vrais octets (lot D-fix). La marche
// SANS preuve perd encore le joueur gere de l'index 0 dans l'image-cle d'avant-match de `bcb6d393`
// et de `fb1a1a72` ; son record etait un candidat que le repli a ECARTE, donc son absence n'y est
// pas prouvee : l'entite reste « au coup d'envoi », et ce doute se compte.
func TestSansPreuveLeDouteRattrapeLaPerte(t *testing.T) {
	for _, film := range []string{"bcb6d393", "fb1a1a72"} {
		_, _, ents := scanPlayerTeamsAvec(NewFilmContext(bobineFilm(t, film)), MarcheDImageCle{})
		var e PlayerEntity
		for _, x := range ents.Entities {
			if x.Slot == 1297 {
				e = x
			}
		}
		if e.Slot != 1297 || e.FirstKF == 0 {
			t.Fatalf("%s : l'entite 1297 est lue a la premiere image-cle sans preuve (%+v) — le "+
				"temoin ne mord plus", film, e)
		}
		if ents.AbsenceProuvee(1297, 0) || !ents.AtStart(e) {
			t.Errorf("%s : absence de 1297 a l'image-cle 0 prouvee=%v, au depart=%v — le doute ne "+
				"rattrape pas la perte", film, ents.AbsenceProuvee(1297, 0), ents.AtStart(e))
		}
		if ents.ImagesClesDouteuses() == 0 || ents.BornesDifferees() == 0 {
			t.Errorf("%s : images douteuses %d, bornes differees %d — le doute n'est pas compte", film,
				ents.ImagesClesDouteuses(), ents.BornesDifferees())
		}
	}
}

// TestDoutesDAbsence : la sante des images-cles, sur un accumulateur fabrique. Quatre images-cles
// porteuses ; E1 lue aux rangs 1-2 avec un doute au rang 0 (arrivee NON prouvee : au depart) ; E2
// lue aux rangs 1-3 sans doute (arrivee prouvee au rang 0) ; E3 lue aux rangs 0-1, un doute au
// rang 2, absence prouvee au rang 3 ; E4 lue aux rangs 0-1, doutes aux rangs 2 et 3 (depart non
// prouve : jusqu'au bout). Un candidat ecarte d'un slot qu'aucune image-cle ne lit (99) n'est
// l'occupant de personne ; un slot lu dans la meme image-cle n'est pas un doute.
func TestDoutesDAbsence(t *testing.T) {
	a := nouvelAccumulateurDEntites()
	for rang := 0; rang < 4; rang++ {
		a.ouvrirImageCle(uint64(100 * (rang + 1)))
	}
	lire := func(slot, de, a2 int) {
		for r := de; r <= a2; r++ {
			a.noter(r, slot, slot, 0)
		}
	}
	lire(10, 1, 2)
	lire(11, 1, 3)
	lire(12, 0, 1)
	lire(13, 0, 1)
	ecarte := func(slot int) []KeyframeRec { return []KeyframeRec{{Slot: slot, TI: managedPlayerTypeIndex}} }
	a.douterDe(0, map[int]bool{12: true, 13: true}, nil, append(ecarte(10), ecarte(12)...))
	a.douterDe(2, map[int]bool{10: true, 11: true}, []int{12}, ecarte(99))
	a.douterDe(3, map[int]bool{11: true}, nil, append(ecarte(13), KeyframeRec{Slot: 12, TI: 38}))
	a.douterDe(2, map[int]bool{10: true, 11: true}, nil, ecarte(13))
	s := a.publier()
	want := []DouteDAbsence{{0, 10}, {2, 12}, {2, 13}, {3, 13}}
	if len(s.Doutes) != len(want) {
		t.Fatalf("doutes %+v, attendu %+v", s.Doutes, want)
	}
	for i := range want {
		if s.Doutes[i] != want[i] {
			t.Fatalf("doutes %+v, attendu %+v", s.Doutes, want)
		}
	}
	par := map[int]PlayerEntity{}
	for _, e := range s.Entities {
		par[e.Slot] = e
	}
	verifierBornes(t, s, par)
}

// verifierBornes tient les bornes des quatre entites de [TestDoutesDAbsence].
func verifierBornes(t *testing.T, s PlayerEntityScan, par map[int]PlayerEntity) {
	t.Helper()
	if !s.AtStart(par[10]) || s.AtStart(par[11]) {
		t.Errorf("au depart : E1 %v (attendu vrai, arrivee non prouvee), E2 %v (attendu faux)",
			s.AtStart(par[10]), s.AtStart(par[11]))
	}
	if r, ok := s.AbsenceProuveeApres(par[12]); !ok || r != 3 {
		t.Errorf("E3 : absence prouvee apres au rang %d (%v), attendu 3", r, ok)
	}
	if !s.AtEnd(par[13]) || s.AtEnd(par[12]) {
		t.Errorf("a la fin : E4 %v (attendu vrai), E3 %v (attendu faux)", s.AtEnd(par[13]), s.AtEnd(par[12]))
	}
	if n := s.ImagesClesDouteuses(); n != 3 {
		t.Errorf("images-cles douteuses : %d, attendu 3 (rangs 0, 2, 3)", n)
	}
	if n := s.BornesDifferees(); n != 3 {
		t.Errorf("bornes differees : %d, attendu 3 (E1 avant, E3 et E4 apres)", n)
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
