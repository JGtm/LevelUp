package replay

// filmfacts_stats_mouvement_test.go — LES STATS DE LA MARCHE DES ETATS DE MOUVEMENT FONT
// L ALLER-RETOUR, TOUTES (jalon J11.0, 2026-09-28).
//
// # LE DEFAUT QU IL FERME
//
// `TestGoldenInputsFidelite` etait ROUGE sur les huit builds depuis le lot 5.9.2-5.9.5
// (`a9fa54784`, 2026-09-21, schema 66) : ce lot a ajoute `JumpEpisodes` et `JumpsDerived` a
// `types.MovementStateStats`, le document les publie (`coverage.stances.jumpEpisodes` /
// `.jumpsDerived`), et le codec des faits ne les ecrivait pas. Un artefact REJOUE DEPUIS LES
// FAITS perdait donc ces deux compteurs (39 octets sur `000d5950` : 1 498 montees examinees, 295
// sauts derives), la ou le decodage du film les publie.
//
// `TestCodecCouvreFilmInputs` ne pouvait pas le voir : il jugeait alors un champ de [FilmInputs]
// transporte des qu UNE de ses feuilles revenait non nulle, et une structure de vingt compteurs
// passait sur le premier. Le meme trou laissait `VelocityReads`, `DatumBindings` et
// `DatumAmbiguous` (J10.5) hors du blob. Il est ferme au jalon J11.0-bis (chaque feuille comparee a
// l octet) ; ce test reste le temoin NOMME du defaut.
//
// CE TEST EXIGE L EGALITE PROFONDE, champ par champ, et SANS LISTE : chaque feuille recoit une
// valeur non nulle DISTINCTE (un compteur recopie dans son voisin ne passe pas), et un compteur
// ajoute demain a la structure y entre tout seul. Il ne lit aucun film : il tourne en CI.

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestCodecTransporteToutesLesStatsDeMouvement(t *testing.T) {
	entry := goldenEntryPourTest(t)
	var st types.MovementStateStats
	prochain := uint64(0)
	remplirFeuillesDistinctes(t, reflect.ValueOf(&st).Elem(), &prochain)
	if prochain < 20 {
		t.Fatalf("%d feuille(s) remplie(s) sur MovementStateStats : la reflexion ne mesure plus rien",
			prochain)
	}
	g := &FilmFacts{Film: goldenFilm, MapModule: entry.Module, AxisW: entry.AxisWidths}
	g.MovementStateStats = st
	relu, err := DecodeFilmFacts(EncodeFilmFacts(g), entry)
	if err != nil {
		t.Fatalf("aller-retour : %v", err)
	}
	got := relu.MovementStateStats
	if reflect.DeepEqual(got, st) {
		return
	}
	a, b := reflect.ValueOf(st), reflect.ValueOf(got)
	for i := 0; i < a.NumField(); i++ {
		if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
			t.Errorf("MovementStateStats.%s N EST PAS TRANSPORTE : ecrit %v, relu %v — un artefact "+
				"rejoue depuis les faits le perd. L ecrire a la QUEUE de `encodeEtatsDeMouvement`, "+
				"le relire au meme rang, et monter `filmFactsMagic` DANS LE MEME COMMIT.",
				a.Type().Field(i).Name, a.Field(i).Interface(), b.Field(i).Interface())
		}
	}
}

// remplirFeuillesDistinctes pose dans chaque feuille une valeur non nulle, differente de toutes les
// autres. Une forme qu elle ne sait pas remplir fait ECHOUER le test : un champ neuf d une forme
// inconnue ne doit pas sortir en silence de la mesure.
func remplirFeuillesDistinctes(t *testing.T, v reflect.Value, n *uint64) {
	t.Helper()
	switch v.Kind() {
	case reflect.Struct:
		for _, field := range v.Fields() {
			remplirFeuillesDistinctes(t, field, n)
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			remplirFeuillesDistinctes(t, v.Index(i), n)
		}
	case reflect.Bool:
		*n++
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		*n++
		v.SetInt(int64(*n)) //nolint:gosec // petit compteur de test
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		*n++
		v.SetUint(*n)
	default:
		t.Fatalf("forme %s non prise en charge : etendre remplirFeuillesDistinctes", v.Kind())
	}
}
