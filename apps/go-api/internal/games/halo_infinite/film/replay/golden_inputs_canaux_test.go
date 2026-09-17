package replay

// golden_inputs_canaux_test.go — L INVENTAIRE DU TYPE, TENU PAR LE COMPILATEUR ET LA REFLEXION.
//
// Le CODEC lui-meme est passe en production au lot 4.1.1-a (`filmfacts_canaux.go` et ses freres) :
// il n est plus un instrument de fixture mais la porte d ecriture et de lecture des faits
// persistes par film. Ce qui reste ici est ce qui doit rester un test — la garde qui exige que
// TOUT champ de [FilmInputs] soit transporte.

import (
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// L INVENTAIRE, TENU PAR LE COMPILATEUR
// ---------------------------------------------------------------------------

// TestCodecCouvreFilmInputs : TOUT champ de [FilmInputs] est serialise. SANS EXCEPTION.
//
// # CE QU IL FERME, ET IL A DEJA COUTE
//
// Le codec porte le type de la PRODUCTION depuis le lot 1.0 : un canal ajoute a l etage de
// balayage apparait donc tout seul dans [FilmFacts], et l assemblage le lira — mais le CODEC, lui,
// ne l apprend pas. Les faits relus rendraient alors un calque vide la ou la production en publie
// un plein, exactement le defaut que la decouverte D9 avait mesure sur sept builds.
// `TestGoldenInputsFidelite` l attrape, mais il EXIGE LE CACHE DE FILMS : il saute en CI. Ce
// test-ci, lui, ne lit aucun octet de film et tourne partout.
//
// # LA TABLE D EXCEPTIONS A DISPARU (lot 4.1.1-b, 2026-09-17)
//
// `champsNonTransportes` nommait QUATRE champs deliberement absents — `FlagMarks`, `ZoneReads`,
// `ZoneScanned`, `BombReads` — au motif que « le fixture ne fournit AUCUNE garde » de mode. Le
// motif etait vrai POUR UN FIXTURE et faux EN PRODUCTION (tout CTF remplit `FlagMarks`, tout
// KOTH/Strongholds `ZoneReads`, tout Assaut armable `BombReads`), et le codec est en production
// depuis le lot 4.1.1-a. Les quatre sont entres, la table est VIDEE et SON MECANISME EST SUPPRIME
// avec sa derniere entree (meme doctrine que les trois allowlists de `film_layers_deps_test.go`) :
// une table vide qu on garde « au cas ou » invite a la remplir, et une exception neuve doit etre
// une DECISION ecrite, pas une ligne a remplir.
//
// # LA MESURE PORTE SUR LE FICHIER DE FAITS, PAS SUR LE SEUL BLOB DES ENTREES
//
// Les quatre canaux gardes voyagent dans la SECTION 1 du fichier de faits, a la suite du blob
// delta-code (`encodeGardesDeMode`), et pas DANS ce blob. La raison est ecrite et mesurable : le
// blob des entrees est aussi le format des HUIT FIXTURES versionnees
// (`testdata/inputs_<short8>.bin.gz`), qui sont les entrees d un fixture — lequel ne fournit
// JAMAIS de garde de mode, donc n a jamais rien a y mettre. Changer leur format aurait exige de
// re-decoder huit films pour ajouter huit suites de zeros.
//
// CE QUI EMPECHE LA DERIVE est que ce test mesure le FICHIER : un champ ajoute a [FilmInputs] doit
// etre transporte, que ce soit par le blob (sa place naturelle, pres de sa famille) ou par le
// complement de la section 1. Les deux satisfont la garde ; aucun oubli ne la satisfait.
//
// IL SE FONDE SUR L ALLER-RETOUR, PAS SUR UNE LISTE ECRITE A LA MAIN : une valeur non nulle est
// posee dans chaque champ, le fichier est encode puis relu, et un champ qui revient VIDE n a pas
// ete transporte. Une liste de noms aurait derive du codec au premier oubli.
func TestCodecCouvreFilmInputs(t *testing.T) {
	entry := goldenEntryPourTest(t)
	champs := reflect.VisibleFields(reflect.TypeOf(FilmInputs{}))
	if len(champs) < 30 {
		t.Fatalf("%d champ(s) lus sur FilmInputs : la reflexion ne mesure plus rien", len(champs))
	}
	for _, f := range champs {
		if f.Anonymous || !f.IsExported() || strings.Contains(f.Name, ".") {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			g := &FilmFacts{Film: goldenFilm, MapModule: entry.Module, AxisW: entry.AxisWidths}
			v := reflect.ValueOf(&g.FilmInputs).Elem().FieldByName(f.Name)
			if !remplirChampTemoin(v) {
				t.Skipf("aucun temoin fabricable pour %s (%s)", f.Name, f.Type)
			}
			blob, err := EncodeFilmFactsFile(&FilmFactsFile{Facts: *g})
			if err != nil {
				t.Fatalf("encodage du fichier de faits sur %s : %v", f.Name, err)
			}
			relu, err := DecodeFilmFactsFile(blob, entry)
			if err != nil {
				t.Fatalf("aller-retour sur %s : %v", f.Name, err)
			}
			if reflect.ValueOf(relu.Facts.FilmInputs).FieldByName(f.Name).IsZero() {
				t.Fatalf("FilmInputs.%s N EST PAS TRANSPORTE par le fichier de faits.\n"+
					"L assemblage le consomme (cf. FilmInputs.applyTo) : relu vide, un artefact "+
					"rejoue depuis les faits publierait un calque VIDE la ou la production en "+
					"publie un plein.\n"+
					"DEUX PLACES LEGITIMES, et il en faut UNE : le blob des entrees (pres de sa "+
					"famille — monter `filmFactsMagic` DANS LE MEME COMMIT, et les huit fixtures "+
					"de testdata/ se re-decodent alors), ou `encodeGardesDeMode` (le complement "+
					"de la section 1 — monter `SchemaDesFaits`).\n"+
					"IL N Y A PLUS DE TABLE D EXCEPTIONS : elle s est videe au lot 4.1.1-b avec "+
					"sa derniere entree, et rouvrir une exception est une DECISION a ecrire, pas "+
					"une ligne a remplir.", f.Name)
			}
		})
	}
}

// remplirChampTemoin pose une valeur NON NULLE dans un champ, quelle que soit sa forme. Rend faux
// quand la forme n est pas fabricable — aucune ne l est aujourd hui, et le test le dirait.
func remplirChampTemoin(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		v.SetInt(7)
	case reflect.Uint32, reflect.Uint64:
		v.SetUint(7)
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		if !remplirChampTemoin(p.Elem()) {
			return false
		}
		v.Set(p)
	case reflect.Slice:
		e := reflect.New(v.Type().Elem()).Elem()
		remplirStructTemoin(e)
		v.Set(reflect.Append(v, e))
	case reflect.Struct:
		return remplirStructTemoin(v)
	default:
		return false
	}
	return true
}

// remplirStructTemoin pose une valeur non nulle dans le PREMIER champ remplissable d une
// structure (recursivement). Un seul suffit : ce qu on mesure est si le champ REVIENT vide.
func remplirStructTemoin(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return remplirChampTemoin(v)
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		if remplirChampTemoin(f) {
			return true
		}
	}
	return false
}
