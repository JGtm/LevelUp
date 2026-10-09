package objectives

// replis_des_objectifs_test.go — LES COMPTES DE REPLIS DU LECTEUR D OBJECTIFS (lot J8.7, 2026-09-27).
//
// Maillons tenus ici : chaque site compte, et le resultat que le paquet rend deja (lecture du
// statborg, resolveur d identite par manche) porte le compte. Le balayage du statborg compte dans la
// grammaire (`grammar/signaux/statborg_domaine_branche_test.go`) ; le dernier maillon — du resultat au
// compteur de la cuisson — est tenu par `replay/versement_des_replis_test.go`.
//
// MUTATION JOUEE (2026-09-27) : retirer `replis:  ri.replis,` de [RoundIdentity.copieProfonde] fait
// rougir `TestLesComptesSuiventLaCompletionDuPont` (les comptes de la construction se perdent a la
// completion par elimination).

import (
	"reflect"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/signaux"
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

// TestLaLectureDuStatborgPorteSesComptesEtSesConstats : les deux replis de la lecture de la
// grammaire arrivent chacun sous SON champ (le balayage les compte, `grammar` le prouve ; ici, le
// portage), et les deux constats sortent quand la lecture les rencontre, et seulement alors.
//
// MUTATION — croiser les deux champs dans [comptesDuStatborg] : ROUGE.
func TestLaLectureDuStatborgPorteSesComptesEtSesConstats(t *testing.T) {
	l := signaux.LectureDuStatborg{ChunksDatables: 2, EnregistrementsAbandonnes: 3, ComposantsArretes: 5}
	if got, want := comptesDuStatborg(l), (ComptesDesReplis{EnregistrementsAbandonnes: 3, ComposantsArretes: 5}); got != want {
		t.Errorf("comptes portes %+v, attendu %+v", got, want)
	}
	if diags := diagnosticsDuStatborg(l, nil, "m"); len(diags) != 0 {
		t.Errorf("lecture complete et datee : %d constat(s), attendu aucun", len(diags))
	}
	sansManifeste := diagnosticsDuStatborg(signaux.LectureDuStatborg{}, nil, "m")
	if len(sansManifeste) != 1 || sansManifeste[0].Code != DiagFilmSansManifeste {
		t.Errorf("film sans manifeste : %+v, attendu le seul constat %s", sansManifeste, DiagFilmSansManifeste)
	}
	tronque := signaux.LectureDuStatborg{ChunksDatables: 2, Tronque: true, ChunkTronque: 7,
		Records: make([]types.StatRecord, 4)}
	d := diagnosticsDuStatborg(tronque, nil, "m")
	if len(d) != 1 || d[0].Code != DiagStatborgTronque || !slices.Contains(d[0].Attrs, any(7)) ||
		!slices.Contains(d[0].Attrs, any(4)) {
		t.Errorf("lecture tronquee : %+v, attendu le seul constat %s avec le chunk 7 et 4 enregistrements",
			d, DiagStatborgTronque)
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
