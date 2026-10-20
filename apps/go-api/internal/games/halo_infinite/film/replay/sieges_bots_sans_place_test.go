package replay

// sieges_bots_sans_place_test.go — LE BOT DECLARE SANS VIE NI PLACE N'ENTRE PAS AU ROSTER PUBLIE
// (sieges_bots_sans_place.go).
//
//	B-ECARTE    gabarit de `d1dfbc02` : l'equipe du bot tient toutes ses places jusqu'a la fin, le bot n'a
//	            aucune vie : il sort du roster publie, aucune place en trop, aucun depassement,
//	            AVERTISSEMENT ;
//	B-AVEC-VIE  le meme bot avec une vie reste au roster, sans place (`index`), compte dans `sansPlace`
//	            et journalise en ERREUR — jamais masque ;
//	B-HUMAIN    un humain sans place n'est jamais ecarte.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// rosterDuBotSansPlace : deux equipes de deux joueurs presents tout le film (places 0 a 3), et un
// arrivant d'equipe 0 (index 8) present de 90 a 99 — un bot quand `bot` est vrai.
func rosterDuBotSansPlace(bot bool) ([]RosterEntry, occupants) {
	roster := []RosterEntry{entree(0, "100"), entree(1, "110"), entree(2, "120"), entree(3, "130"),
		{FilmIndex: 8, Name: "343 Ham Sammich [bot]", XUID: ""}}
	if !bot {
		roster[4] = entree(8, "180")
	}
	occ := occupantsFabriques([]int{0, 0, 1, 1, 0}, iv(0, siegeFrames-1), iv(0, siegeFrames-1),
		iv(0, siegeFrames-1), iv(0, siegeFrames-1), iv(90, siegeFrames-1))
	if bot {
		poserDesBots(roster, &occ, 4)
	}
	return roster, occ
}

// journalDesPlaces joue la pose et son journal sous un journal capture ; rend la couverture et les
// enregistrements (niveau, message).
func journalDesPlaces(t *testing.T, roster []RosterEntry, occ occupants) (SeatCoverage, [][2]string) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(prev)
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2, 3),
		horloge: horlogeDeSieges(nil)})
	journaliserLesPlaces(context.Background(), "temoin", cov)
	var out [][2]string
	for _, ligne := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var r struct{ Level, Msg string }
		if json.Unmarshal([]byte(ligne), &r) == nil {
			out = append(out, [2]string{r.Level, r.Msg})
		}
	}
	return cov, out
}

// aUnEnregistrement dit qu'un enregistrement du niveau donne contient `extrait`.
func aUnEnregistrement(journal [][2]string, niveau, extrait string) bool {
	for _, r := range journal {
		if r[0] == niveau && strings.Contains(r[1], extrait) {
			return true
		}
	}
	return false
}

func TestBotSansVieNiPlaceHorsDuRosterPublie(t *testing.T) { // B-ECARTE
	roster, occ := rosterDuBotSansPlace(true)
	cov, journal := journalDesPlaces(t, roster, occ)
	publie := sansLesBotsEcartes(roster, cov)
	if len(publie) != 4 || cov.Entrees != 4 {
		t.Fatalf("roster publie %d entrees (couverture %d), attendu 4 : le bot sans vie ni place n'y entre pas",
			len(publie), cov.Entrees)
	}
	for _, e := range publie {
		if e.Bot {
			t.Errorf("le bot %q est publie", e.Name)
		}
	}
	if cov.SansPlace != 0 || cov.PlacesEnTrop != 0 || cov.Depassements != 0 || cov.SansPresence != 0 {
		t.Errorf("couverture %+v : ni place en trop, ni depassement, ni entree sans place ou sans presence", cov)
	}
	if !aUnEnregistrement(journal, "WARN", "bot declare sans aucune vie ni place libre") {
		t.Errorf("journal %v : l'ecart se dit en AVERTISSEMENT", journal)
	}
	if aUnEnregistrement(journal, "ERROR", "bot sans place qui a une vie") {
		t.Errorf("journal %v : aucune ERREUR pour un bot sans vie", journal)
	}
}

func TestBotSansPlaceAvecUneVieResteUnDefaut(t *testing.T) { // B-AVEC-VIE
	roster, occ := rosterDuBotSansPlace(true)
	occ.parEntree[4].vies = [][2]int{{92, 97}}
	cov, journal := journalDesPlaces(t, roster, occ)
	publie := sansLesBotsEcartes(roster, cov)
	if len(publie) != 5 || publie[4].SeatSource != SeatSourceIndex || cov.SansPlace != 1 {
		t.Fatalf("roster publie %d entrees, source du bot %q, sansPlace %d : attendu 5, %q, 1 — le bot qui a une "+
			"vie reste, sans place", len(publie), publie[len(publie)-1].SeatSource, cov.SansPlace, SeatSourceIndex)
	}
	if !aUnEnregistrement(journal, "ERROR", "bot sans place qui a une vie") {
		t.Errorf("journal %v : un bot sans place qui a une vie se dit en ERREUR", journal)
	}
}

func TestHumainSansPlaceNEstJamaisEcarte(t *testing.T) { // B-HUMAIN
	roster, occ := rosterDuBotSansPlace(false)
	cov, _ := journalDesPlaces(t, roster, occ)
	if publie := sansLesBotsEcartes(roster, cov); len(publie) != 5 || cov.SansPlace != 1 {
		t.Fatalf("roster publie %d entrees, sansPlace %d : un humain sans place reste au roster", len(publie),
			cov.SansPlace)
	}
}
