package analysis

// sql_fragments_retention_test.go — LA FENETRE DE RETENTION ET L'ELIGIBILITE A LA CUISSON.
//
// Ces fragments repondent a « la file de cuisson reprendra-t-elle ce match ? », et DEUX
// composants s'en servent : la file elle-meme, et la lecture tactique qui l'annonce a
// l'utilisateur. Une divergence entre les deux envoie attendre une cuisson qui n'arrive pas.
// Ils n'etaient couverts par aucun test (revue de 7.10, P1).

import (
	"strings"
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

// TestSQLDansFenetreRetention_PasseParLeFragmentCanonique — l'horodatage n'est JAMAIS
// `start_time` brut (regle n 8) : le fragment canonique replie `start_time_utc` sur
// `start_time AT TIME ZONE 'UTC'`. Un filtre sur la colonne brute lirait faux d'un fuseau.
func TestSQLDansFenetreRetention_PasseParLeFragmentCanonique(t *testing.T) {
	frag := SQLDansFenetreRetention("r")
	if !strings.Contains(frag, SQLStartTimeCanonical("r")) {
		t.Fatalf("fragment = %q, attendu qu'il reprenne SQLStartTimeCanonical", frag)
	}
	if !strings.HasSuffix(frag, " >= ?") {
		t.Fatalf("fragment = %q, attendu une comparaison INCLUSIVE a un parametre lie", frag)
	}
}

// TestSQLEligibleALaCuisson_TroisConditionsEtLeurOrdre — le predicat porte les trois
// conditions de la file, et le `IS NOT NULL` PRECEDE la comparaison de fenetre.
//
// L'ORDRE EST LA CORRECTION DU P0. Les deux horodatages du registre sont nullables ; sans
// `IS NOT NULL` en amont du AND, le predicat vaut NULL pour un match indatable, et le
// scanner dans un `bool` nu rendait `converting NULL to bool` — donc 500 sur TOUTE la
// lecture de la carte. En logique ternaire, `FALSE AND NULL` vaut FALSE : l'ordre suffit.
func TestSQLEligibleALaCuisson_TroisConditionsEtLeurOrdre(t *testing.T) {
	sql, args := SQLEligibleALaCuisson("mr", 3, 1<<22)

	if !strings.Contains(sql, "COALESCE(mr.backfill_completed, 0) & ? = 0") {
		t.Fatalf("sql = %q : le marqueur de film perdu manque", sql)
	}
	iNotNull := strings.Index(sql, "IS NOT NULL")
	iFenetre := strings.Index(sql, ">= ?")
	if iNotNull < 0 || iFenetre < 0 {
		t.Fatalf("sql = %q : il manque le non-NULL ou la fenetre", sql)
	}
	if iNotNull > iFenetre {
		t.Fatalf("sql = %q : `IS NOT NULL` doit PRECEDER la comparaison de fenetre, sinon "+
			"le predicat vaut NULL pour un match indatable", sql)
	}
	if len(args) != 2 {
		t.Fatalf("args = %v, attendu le bit PUIS la borne, dans cet ordre", args)
	}
	if args[0] != int64(1<<22) {
		t.Fatalf("args[0] = %v, attendu le bit de film absent EN PREMIER — un argument au "+
			"mauvais rang decale silencieusement tous les suivants", args[0])
	}
	if _, ok := args[1].(time.Time); !ok {
		t.Fatalf("args[1] = %T, attendu la borne de retention", args[1])
	}
}

// TestSQLEligibleALaCuisson_SansFenetre_UnSeulParametre — fenetre illimitee : la comparaison
// disparait, et son parametre AUSSI.
//
// UN `?` SANS SON ARGUMENT DECALE TOUT LE RESTE : c'est ce qui transformerait un xuid en
// identifiant de carte dans `QTacticalUnivers`. Le non-NULL, lui, RESTE — un match indatable
// n'entre pas dans la file, fenetre ou pas.
func TestSQLEligibleALaCuisson_SansFenetre_UnSeulParametre(t *testing.T) {
	sql, args := SQLEligibleALaCuisson("mr", 0, 1<<22)
	if len(args) != 1 {
		t.Fatalf("args = %v, attendu le seul bit de film absent", args)
	}
	if strings.Contains(sql, ">= ?") {
		t.Fatalf("sql = %q : aucune comparaison de fenetre sans retention bornee", sql)
	}
	if !strings.Contains(sql, "IS NOT NULL") {
		t.Fatalf("sql = %q : un match indatable reste hors file, fenetre ou pas", sql)
	}
}

// TestSQLEligibleALaCuisson_SansAlias — le fragment se compose aussi sans alias de table.
func TestSQLEligibleALaCuisson_SansAlias(t *testing.T) {
	sql, _ := SQLEligibleALaCuisson("", 0, 1<<22)
	if strings.Contains(sql, "COALESCE(.backfill_completed") {
		t.Fatalf("sql = %q : un alias vide ne doit pas laisser de point orphelin", sql)
	}
}
