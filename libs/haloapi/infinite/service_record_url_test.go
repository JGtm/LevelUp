package infinite

// service_record_url_test.go — l'adresse du service record filtré par saison (test du fournisseur
// d'avant la brique, porté sous son nom sur la fonction de la brique).

import (
	"strings"
	"testing"
)

func TestBuildSeasonServiceRecordURL(t *testing.T) {
	tru, fls := true, false
	e := DefaultEndpoints().AvecPrefixeJeu(PrefixeJeuDefaut)
	expectedPrefix := e.Hote(CleStatsSansPort) + "/hi/players/JGtm/Matchmade/servicerecord?"

	// Sans filtre classé : seasonId encodé (barres -> %2F), pas d'isRanked ; segment de jeu « hi ».
	u := BuildSeasonServiceRecordURL(e, "JGtm", "Seasons/Season7.json", nil)
	if !strings.HasPrefix(u, expectedPrefix) {
		t.Fatalf("préfixe inattendu: %s", u)
	}
	if !strings.Contains(u, "seasonId=Seasons%2FSeason7.json") {
		t.Errorf("seasonId mal encodé: %s", u)
	}
	if strings.Contains(u, "isRanked") {
		t.Errorf("isRanked ne doit pas apparaître quand nil: %s", u)
	}
	if u := BuildSeasonServiceRecordURL(e, "JGtm", "Seasons/Season7.json", &tru); !strings.Contains(u, "isRanked=true") {
		t.Errorf("isRanked=true attendu: %s", u)
	}
	if u := BuildSeasonServiceRecordURL(e, "JGtm", "Seasons/Season7.json", &fls); !strings.Contains(u, "isRanked=false") {
		t.Errorf("isRanked=false attendu: %s", u)
	}
}
