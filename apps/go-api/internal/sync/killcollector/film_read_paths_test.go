package killcollector

// film_read_paths_test.go — LE VERROU D EGALITE entre le decodeur et le vocabulaire partage.
//
// Les voies de lecture d un decodage de film ont DEUX domiciles, et c est structurel :
//
//	games/halo_infinite/film/killsource   `Path` — le TYPE, chez le decodeur qui les produit.
//	                                      Paquet title-specific : ni `persist` ni `migration`
//	                                      ne peuvent l importer.
//	domain/killscope                      les CHAINES, dans une feuille sans import, lisibles
//	                                      par les quatre ecrivains et par la migration.
//
// Ce paquet est le SEUL du depot qui importe les deux. Le verrou vit donc ici — c est exactement
// la promesse que le constat J4R-3 avait trouvee non tenue ailleurs (un commentaire renvoyait a un
// « test d egalite » qui n existait pas), et la dette H3 la reclamait pour ces deux valeurs-la.
//
// CE QUE LA DERIVE COUTERAIT : c est sur `read_path` que se detecte la presence d un
// ENRICHISSEMENT de film. Une copie qui diverge d un caractere ne leve aucune erreur — le
// detecteur cesse simplement de voir les passes de film, et un producteur credit reecrit
// par-dessus sans les relire. Le nombre de lignes ne bouge pas, aucun nom ne change : seule la
// source du degat fatal disparait de la lecture.

import (
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/persist"
)

func TestFilmReadPathsEgalesAuDecodeur(t *testing.T) {
	if got, want := killscope.ReadPathFilmWalk, string(killsource.PathWalk); got != want {
		t.Errorf("killscope.ReadPathFilmWalk = %q, decodeur = %q — la detection d un "+
			"enrichissement de film cesserait de voir la voie MARCHE, sans erreur ni compteur",
			got, want)
	}
	if got, want := killscope.ReadPathFilmScan, string(killsource.PathScan); got != want {
		t.Errorf("killscope.ReadPathFilmScan = %q, decodeur = %q — la detection d un "+
			"enrichissement de film cesserait de voir la voie SCAN, sans erreur ni compteur",
			got, want)
	}
}

// TestFilmReadPathsCouvrentToutesLesVoiesDuDecodeur — le verrou d EXHAUSTIVITE.
//
// L egalite deux a deux ne suffit pas : le jour ou le decodeur ajoute une TROISIEME voie, les
// deux valeurs existantes resteraient justes et la liste, elle, serait incomplete — la nouvelle
// voie passerait pour du credit et se ferait ecraser. Ce test compare les CARDINAUX.
func TestFilmReadPathsCouvrentToutesLesVoiesDuDecodeur(t *testing.T) {
	// Les voies connues du decodeur, a tenir a jour AVEC lui : ajouter une valeur ici sans
	// l ajouter a `killscope.FilmReadPaths` fait echouer ce test.
	duDecodeur := []killsource.Path{killsource.PathWalk, killsource.PathScan}

	partagees := map[string]bool{}
	for _, p := range killscope.FilmReadPaths() {
		partagees[p] = true
	}
	if len(partagees) != len(duDecodeur) {
		t.Fatalf("%d voie(s) partagees pour %d voie(s) au decodeur — une voie absente de la liste "+
			"passe pour du credit et se fait ecraser par le producteur suivant",
			len(partagees), len(duDecodeur))
	}
	for _, p := range duDecodeur {
		if !partagees[string(p)] {
			t.Errorf("voie %q du decodeur absente de killscope.FilmReadPaths()", p)
		}
	}
	// `persist.FilmReadPaths` n est plus une copie : il LIT la liste partagee. Le verifier ici
	// interdit qu il redevienne un litteral.
	if len(persist.FilmReadPaths) != len(partagees) {
		t.Errorf("persist.FilmReadPaths porte %d voie(s) pour %d partagees — la copie est revenue",
			len(persist.FilmReadPaths), len(partagees))
	}
}
