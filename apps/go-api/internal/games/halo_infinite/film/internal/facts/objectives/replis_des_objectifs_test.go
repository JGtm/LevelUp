package objectives

// replis_des_objectifs_test.go — LES COMPTES DE REPLIS DU LECTEUR D OBJECTIFS (lot J8.7, 2026-09-27).
//
// Maillons tenus ici : chaque site compte, et le resultat que le paquet rend deja (balayage du
// statborg, resolveur d identite par manche) porte le compte. Le dernier maillon — du resultat au
// compteur de la cuisson — est tenu par `replay/versement_des_replis_test.go`.
//
// MUTATION JOUEE (2026-09-27) : retirer `replis:  ri.replis,` de [RoundIdentity.copieProfonde] fait
// rougir `TestLesComptesSuiventLaCompletionDuPont` (les comptes de la construction se perdent a la
// completion par elimination).

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestPlusSommeChaqueChampDesComptes : [ComptesDesReplis.Plus] nomme ses champs ; un champ ajoute et
// oublie la serait perdu a la premiere somme.
func TestPlusSommeChaqueChampDesComptes(t *testing.T) {
	var c ComptesDesReplis
	v := reflect.ValueOf(&c).Elem()
	for i := 0; i < v.NumField(); i++ {
		v.Field(i).SetInt(int64(i + 1))
	}
	s := reflect.ValueOf(c.Plus(c))
	for i := 0; i < s.NumField(); i++ {
		if got, want := s.Field(i).Int(), int64(2*(i+1)); got != want {
			t.Errorf("Plus perd le champ %s : %d, attendu %d", s.Type().Field(i).Name, got, want)
		}
	}
}

// TestLeBalayageDuStatborgCompteSesAbandons : un enregistrement dont un compteur sort du domaine est
// abandonne ET compte ; un composant non decodable arrete la boucle et le dit.
func TestLeBalayageDuStatborgCompteSesAbandons(t *testing.T) {
	var c ComptesDesReplis
	if recs := scanFrameAvecReplis(statVecteurUnCompo(statMaxCounter+1), 764967, &c); len(recs) != 0 {
		t.Fatalf("enregistrement hors domaine publie : %v", recs)
	}
	if c.EnregistrementsAbandonnes == 0 {
		t.Error("l enregistrement hors domaine est abandonne sans etre compte")
	}
	var sain ComptesDesReplis
	if recs := scanFrameAvecReplis(statVecteurUnCompo(10), 764967, &sain); len(recs) == 0 {
		t.Fatal("l enregistrement sain n est plus lu")
	}
	pay := statVecteurUnCompo(10)
	_, idx, at, ok := matchRecordHeader(pay, 1)
	if !ok {
		t.Fatal("le vecteur sain ne porte plus d en-tete a son bit d ancrage")
	}
	if comps, _, arrete := decodeComponentsAvecArret(pay, at, idx); len(comps) == 0 || arrete {
		t.Errorf("vecteur sain : %d composant(s), arret %v — attendu une lecture complete", len(comps), arrete)
	}
	if _, _, arrete := decodeComponentsAvecArret(make([]byte, 2), 0, []int{0, 1}); !arrete { // tampon trop court : le premier composant deborde
		t.Error("un composant non decodable doit arreter la boucle ET le dire")
	}
}

// TestLeResolveurPorteLesComptesDeSaConstruction : pont sans mort (table vide), debut de manche au
// minimum — chacun au resolveur qui en est ne.
func TestLeResolveurPorteLesComptesDeSaConstruction(t *testing.T) {
	recs, _, _ := filmUneMancheDeuxMortsFixture()
	if got := ResolveRoundIdentity(recs, nil, nil).ComptesDesReplis().TablesIdentiteVides; got != 1 {
		t.Errorf("pont sans mort : %d table(s) vide(s) comptee(s), attendu 1", got)
	}
	var c ComptesDesReplis
	roundStartsOfCompte(recsExAequo(), byRoundExAequo(), &c)
	consensus := RoundStartsMS(recsExAequo())
	if want := len(byRoundExAequo()) - len(consensus); c.DebutsDeMancheAuMinimum != want {
		t.Errorf("debuts au minimum : %d, attendu %d (manches hors consensus)", c.DebutsDeMancheAuMinimum, want)
	}
}

// TestLesComptesSuiventLaCompletionDuPont : la completion par la feuille compte l attribution
// qu elle abandonne, et ni elle ni les completions suivantes ne perdent les comptes deja portes.
func TestLesComptesSuiventLaCompletionDuPont(t *testing.T) {
	recs, _, _ := filmUneMancheDeuxMortsFixture()
	// « D » est deja porte par le slot 20 : le triplet du slot 22 est abandonne (cf.
	// TestCompletedByLinesNAttribueJamaisUnXUIDDejaPris).
	lines := []types.PlayerLine{{XUID: "D", Kills: 7, Deaths: 2, Assists: 1}}
	_, deaths, _ := filmUneMancheDeuxMortsFixture()
	complete := ResolveRoundIdentity(recs, deaths, nil).CompletedByLines(recs, lines)
	if got := complete.ComptesDesReplis().SlotsAbandonnes; got != 1 {
		t.Fatalf("attribution abandonnee au premier arrive : %d comptee(s), attendu 1", got)
	}
	suite := complete.CompletedByElimination(recs, lines).CompletedByRoundResidue(recs, lines)
	if got := suite.ComptesDesReplis(); got != complete.ComptesDesReplis() {
		t.Errorf("les completions suivantes perdent les comptes : %+v, attendu %+v", got, complete.ComptesDesReplis())
	}
}
