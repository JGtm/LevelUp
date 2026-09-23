package replay

// occupants_temoin_test.go — LE TEMOIN AU GABARIT DE `b1ad85eb` (lot M2.3, 2026-09-23).
//
// Les faits sont ceux que la sonde P4 a MESURES sur ce film (note `SONDE_P4_equipes_places.md`),
// recopies en frames du document (100 ms) : huit sieges dont un jamais joue (WNBA Fan A5, siege
// 5), trois bots d'index 8 de designateurs 0, 0, 1 (343 Hundy, 343 PardonMy, 343 Brew Dog) dates
// par BOT_METADATA, deux arrivants humains d'index 9 et 10 (Hanover Cat, Claudors), et les 84 tirs
// de Claudors sous l'index de tireur 5. CE N'EST PAS UNE VALEUR DE PRODUCTION : aucune ligne de
// code ne connait ce match ; le temoin est la forme d'un film a remplacements.
//
// CE QUI DOIT EN SORTIR (regle des places de l'utilisateur, 2026-09-23) :
//
//	place 5 (Eagle) : Hundy -> Hanover Cat -> PardonMy -> Claudors, a presences disjointes ;
//	place 1 (Cobra) : FairyNectar -> Brew Dog ;
//	a CHAQUE frame, au plus 4 occupants affiches par equipe ;
//	aux trois instants signales (0:00, 2:14, 6:24) : exactement 4 contre 4, les bons noms.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : re-agreger l'equipe par INDEX (l'index 8 diverge — deux
// equipes —, les trois bots perdent leur equipe, aucun ne trouve de place).

import (
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// temoinFrames : la longueur du document temoin.
const temoinFrames = 5100

// temoinImagesCles : les images-cles porteuses du temoin, en frames (f12, f212, f412, f613, f813,
// puis une toutes les 200 frames jusqu'a f5013).
func temoinImagesCles() []int {
	out := []int{12, 212, 412, 613, 813}
	for f := 1013; f < temoinFrames; f += 200 {
		out = append(out, f)
	}
	return out
}

// rangDe rend le rang de l'image-cle porteuse d'une frame du temoin.
func rangDe(kf []int, f int) int {
	for i, v := range kf {
		if v == f {
			return i
		}
	}
	panic("image-cle hors du temoin")
}

// entiteTemoin fabrique une entite lue de la premiere a la derniere image-cle donnees.
func entiteTemoin(kf []int, slot, idx, des, de, a int) grammar.PlayerEntity {
	d, f := rangDe(kf, de), rangDe(kf, a)
	return grammar.PlayerEntity{Slot: slot, Index: idx, Team: des, FirstKF: d, LastKF: f, Seen: f - d + 1}
}

// temoinB1ad85eb rend le roster, les vies, les entrees de la liaison et celles de la pose.
func temoinB1ad85eb() ([]RosterEntry, []Track, entreesDesOccupants, entreesDesPlaces) {
	kf := temoinImagesCles()
	fin := kf[len(kf)-1]
	scan := grammar.PlayerEntityScan{Scanned: true}
	for _, f := range kf {
		scan.KeyframesUS = append(scan.KeyframesUS, uint64(f)*100_000)
	}
	money := entiteTemoin(kf, 1297, 0, 0, 12, fin)
	money.Seen-- // le trou de f613 : une perte de marche, pas un depart
	scan.Entities = []grammar.PlayerEntity{
		money, entiteTemoin(kf, 1299, 1, 1, 12, 3013), entiteTemoin(kf, 1301, 2, 1, 12, fin),
		entiteTemoin(kf, 1303, 3, 0, 12, fin), entiteTemoin(kf, 1305, 4, 1, 12, fin),
		entiteTemoin(kf, 1309, 6, 0, 12, fin), entiteTemoin(kf, 1311, 7, 1, 12, fin),
		entiteTemoin(kf, 1530, 8, 0, 12, 212), entiteTemoin(kf, 1610, 9, 0, 412, 613),
		entiteTemoin(kf, 1687, 8, 0, 813, 813), entiteTemoin(kf, 1713, 10, 0, 1013, fin),
		entiteTemoin(kf, 2145, 8, 1, 3213, fin),
	}
	humains := []struct {
		nom string
		idx int
	}{{"MONEY x BUTTER", 0}, {"FairyNectar5788", 1}, {"Madina97294", 2}, {"Namikidori", 3},
		{"Chocoboflor", 4}, {"WNBA Fan A5", 5}, {"DRghie", 6}, {"JGtm", 7}, {"Hanover Cat", 9},
		{"Claudors", 10}}
	var roster []RosterEntry
	for k, h := range humains {
		roster = append(roster, RosterEntry{XUID: string(rune('a' + k)), FilmIndex: h.idx, Name: h.nom})
	}
	bots := []BotIdentity{
		{FilmIndex: 8, Name: "343 Hundy [bot]", BotID: 16, Declarations: [][2]uint64{{1_200_000, 27_300_000}}},
		{FilmIndex: 8, Name: "343 PardonMy [bot]", BotID: 7, Declarations: [][2]uint64{{81_300_000, 83_100_000}}},
		{FilmIndex: 8, Name: "343 Brew Dog [bot]", BotID: 19, Declarations: [][2]uint64{{315_500_000, 0}}},
	}
	for _, b := range bots {
		roster = append(roster, RosterEntry{FilmIndex: 8, Name: b.Name, Bot: true, Bid: b.Bid()})
	}
	vie := func(cle, bot string, de, a int) Track {
		return Track{XUID: cle, Bot: bot, StartFrame: de, EndFrame: a}
	}
	tracks := []Track{
		vie("a", "", 0, 1000), vie("a", "", 1100, temoinFrames-1), vie("b", "", 0, 3117),
		vie("c", "", 0, temoinFrames-1), vie("d", "", 0, temoinFrames-1), vie("e", "", 0, temoinFrames-1),
		vie("g", "", 0, temoinFrames-1), vie("h", "", 0, temoinFrames-1), vie("i", "", 661, 662),
		vie("j", "", 1184, temoinFrames-1), vie("", "343 Hundy [bot]", 0, 270),
		vie("", "343 PardonMy [bot]", 774, 829), vie("", "343 Brew Dog [bot]", 3236, temoinFrames-1),
	}
	var tirs []FireEventRef
	for f := 1284; f <= 3100; f += 200 { // les tirs de Claudors, sous la PLACE 5
		tirs = append(tirs, FireEventRef{FilmIndex: 5, TimestampUS: uint64(f) * 100_000})
	}
	tirs = append(tirs, FireEventRef{FilmIndex: 0, TimestampUS: 50_000_000}) // MONEY, sa place
	table := FilmPlayerTable{}
	for i := 0; i < 8; i++ {
		table.Seats = append(table.Seats, FilmPlayerSeat{FilmIndex: i, XUID: uint64(100 + i)})
	}
	horloge := replayClock{origin: 0, step: 100_000, frames: temoinFrames}
	parIndex := map[int]int{0: 0, 1: 1, 2: 1, 3: 0, 4: 1, 6: 0, 7: 1, 9: 0, 10: 0} // l'index 8 diverge
	return roster, tracks, entreesDesOccupants{scan: scan, bots: bots, horloge: horloge, parIndex: parIndex},
		entreesDesPlaces{table: table, fire: tirs, horloge: horloge}
}

// occupantsDeLaPlace rend les noms des occupants d'une place, dans l'ordre de leur arrivee.
func occupantsDeLaPlace(roster []RosterEntry, place int) []string {
	type o struct {
		nom string
		de  int
	}
	var out []o
	for _, e := range roster {
		if e.Seat == place && len(e.Presence) > 0 {
			out = append(out, o{e.Name, e.Presence[0].From})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].de < out[j].de })
	noms := make([]string, len(out))
	for i, v := range out {
		noms[i] = v.nom
	}
	return noms
}

// affichesA rend, par equipe, les noms des occupants affiches a une frame.
func affichesA(roster []RosterEntry, f int) map[int][]string {
	out := map[int][]string{}
	for _, e := range roster {
		if e.Team == nil {
			continue
		}
		for _, p := range e.Presence {
			fin := p.To
			if p.ToMax != nil {
				fin = *p.ToMax
			}
			if f >= p.From && f <= fin {
				out[*e.Team] = append(out[*e.Team], e.Name)
				break
			}
		}
	}
	for t := range out {
		sort.Strings(out[t])
	}
	return out
}

func TestTemoinB1ad85ebPlacesEtPresences(t *testing.T) {
	roster, tracks, occIn, placeIn := temoinB1ad85eb()
	occ := lierLesOccupants(roster, tracks, occIn)
	var pub teamPublication
	pub.poserEquipesParEntree(roster, occ)
	cov := poserLesSieges(roster, occ, placeIn)

	eagle := []string{"343 Hundy [bot]", "Hanover Cat", "343 PardonMy [bot]", "Claudors"}
	if got := occupantsDeLaPlace(roster, 5); !egaux(got, eagle) {
		t.Errorf("place 5 : %v, attendu %v", got, eagle)
	}
	cobra := []string{"FairyNectar5788", "343 Brew Dog [bot]"}
	if got := occupantsDeLaPlace(roster, 1); !egaux(got, cobra) {
		t.Errorf("place 1 : %v, attendu %v", got, cobra)
	}
	for f := 0; f < temoinFrames; f++ {
		for eq, noms := range affichesA(roster, f) {
			if len(noms) > 4 {
				t.Fatalf("frame %d : l'equipe %d affiche %d occupants %v — jamais plus que ses 4 places",
					f, eq, len(noms), noms)
			}
		}
	}
	instants := map[int]map[int][]string{
		227:  {0: {"343 Hundy [bot]", "DRghie", "MONEY x BUTTER", "Namikidori"}},
		1567: {0: {"Claudors", "DRghie", "MONEY x BUTTER", "Namikidori"}},
		4067: {1: {"343 Brew Dog [bot]", "Chocoboflor", "JGtm", "Madina97294"}},
	}
	for f, attendu := range instants {
		got := affichesA(roster, f)
		for eq, noms := range attendu {
			if !egaux(got[eq], noms) {
				t.Errorf("frame %d, equipe %d : %v, attendu %v", f, eq, got[eq], noms)
			}
		}
	}
	if cov.Depassements != 0 || cov.SansPlace != 0 || cov.PlacesTirs != 1 || cov.Presences != PresencesDuFilm ||
		cov.EntitesNonLiees != 0 || cov.EntitesContestees != 0 || cov.RelaisBornes != 2 {
		t.Errorf("couverture %+v : 0 depassement, 0 sans place, 1 place lue dans les tirs, presences "+
			"lues, 12 entites liees, 2 relais bornes", cov)
	}
	verifierLesPresencesDuTemoin(t, roster)
}

// verifierLesPresencesDuTemoin tient les bornes que la regle des places fixe : le depart pendant la
// mort sort a l'image-cle suivante, borne par l'arrivee du remplacant (FairyNectar -> Brew Dog a
// f3155, paquet de changement BOT_METADATA) ; un bot sort a la frame exacte de son retrait ; un
// arrivant est present des sa premiere image-cle, avant son premier corps (Claudors : f1013).
func verifierLesPresencesDuTemoin(t *testing.T, roster []RosterEntry) {
	t.Helper()
	attendu := map[string]PresenceInterval{
		"FairyNectar5788": {From: 0, To: 3117, ToMax: entierDe(3154)},
		"343 Hundy [bot]": {From: 0, To: 272},
		"Hanover Cat":     {From: 412, To: 662, ToMax: entierDe(773)},
		"Claudors":        {From: 1013, To: temoinFrames - 1},
	}
	for _, e := range roster {
		want, ok := attendu[e.Name]
		if !ok {
			continue
		}
		if len(e.Presence) != 1 || e.Presence[0].From != want.From || e.Presence[0].To != want.To ||
			!memeBorne(e.Presence[0].ToMax, want.ToMax) {
			t.Errorf("%s : presence %+v, attendu %+v (toMax %v)", e.Name, e.Presence, want, deref(want.ToMax))
		}
	}
}

func entierDe(v int) *int { return &v }

func memeBorne(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func egaux(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
