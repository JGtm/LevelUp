package analysis

// sql_fragments_retention_test.go — LA FENETRE DE RETENTION ET L'ELIGIBILITE A LA CUISSON.
//
// Ces fragments repondent a « la file de cuisson reprendra-t-elle ce match ? », et DEUX
// composants s'en servent : la file elle-meme, et la lecture tactique qui l'annonce a
// l'utilisateur. Une divergence entre les deux envoie attendre une cuisson qui n'arrive pas.
// Ils n'etaient couverts par aucun test (revue de 7.10, P1).

import (
	"testing"
	"time"
)

// TestBorneRetention_IllimiteeSousZero — `mois <= 0` VEUT DIRE ILLIMITEE, jamais « zero
// mois ». C'est le reglage par defaut (`ReplayRetentionMonths`), et la convention est
// partagee par la purge, la file et la lecture : la lire a l'envers ferait purger tout le
// parc au premier demarrage sans reglage.
func TestBorneRetention_IllimiteeSousZero(t *testing.T) {
	for _, mois := range []int{0, -1, -12} {
		if _, bornee := BorneRetention(mois); bornee {
			t.Fatalf("mois=%d : bornee=true, attendu une fenetre ILLIMITEE", mois)
		}
	}
}

// TestBorneRetentionDepuis_EstNowMoinsMois — la borne est exactement « l'instant de
// reference moins N mois », et elle se calcule sur l'horloge FOURNIE.
//
// L'HORLOGE INJECTEE EST LE POINT DU TEST : le cron de purge s'en sert pour se placer a une
// date choisie, et c'est la seule facon de prouver la borne sans dependre de la date du
// jour.
func TestBorneRetentionDepuis_EstNowMoinsMois(t *testing.T) {
	ref := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	borne, bornee := BorneRetentionDepuis(ref, 3)
	if !bornee {
		t.Fatal("bornee=false pour 3 mois")
	}
	attendu := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	if !borne.Equal(attendu) {
		t.Fatalf("borne = %s, attendu %s", borne, attendu)
	}
	if _, bornee := BorneRetentionDepuis(ref, 0); bornee {
		t.Fatal("mois=0 : bornee=true, attendu illimitee meme avec une horloge fournie")
	}
}
