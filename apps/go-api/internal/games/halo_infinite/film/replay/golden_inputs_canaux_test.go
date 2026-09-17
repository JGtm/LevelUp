package replay

// golden_inputs_canaux_test.go — L INVENTAIRE DU TYPE, TENU PAR LE COMPILATEUR ET LA REFLEXION.
//
// Le CODEC lui-meme est passe en production au lot 4.1.1-a (`filmfacts_canaux.go` et ses trois
// freres) : il n est plus un instrument de fixture mais la porte d ecriture et de lecture des
// faits persistes par film. Ce qui reste ici est ce qui doit rester un test — la garde qui exige
// que TOUT champ de [FilmInputs] soit soit serialise, soit NOMME comme deliberement absent.

import (
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// L INVENTAIRE, TENU PAR LE COMPILATEUR
// ---------------------------------------------------------------------------

// champsNonTransportes : les champs de [FilmInputs] que le codec ne porte PAS, avec la raison.
//
// LES TROIS SONT DES CALQUES GARDES PAR L APPELANT : le marqueur de portage du drapeau, l etat
// des zones et l anneau de la bombe ne se balaient que si `Options.Flag` / `.Zone` / `.Bomb`
// portent la garde de mode correspondante. Le fixture n en fournit AUCUNE (il ne connait ni la
// variante du match ni le catalogue de zones de la carte), donc ces champs sont vides des deux
// cotes — frais comme relus — et les serialiser figerait des zeros. Le jour ou un fixture
// porterait un catalogue de zones, ils devraient entrer au codec : c est ce que ce test force a
// decider.
var champsNonTransportes = map[string]string{
	"FlagMarks":   "calque garde par Options.Flag — vide sans garde de mode CTF",
	"ZoneReads":   "calque garde par Options.Zone.Zones — vide sans catalogue de zones",
	"ZoneScanned": "temoin du precedent",
	"BombReads":   "calque garde par Options.Bomb.Scanned — vide sans garde de mode Assaut",
}

// TestCodecCouvreFilmInputs : TOUT champ de [FilmInputs] est soit serialise, soit NOMME comme
// deliberement absent.
//
// # CE QU IL FERME, ET IL A DEJA COUTE
//
// Le fixture porte le type de la PRODUCTION depuis le lot 1.0 : un canal ajoute a l etage de
// balayage apparait donc tout seul dans `FilmFacts`, et l assemblage le lira — mais le CODEC,
// lui, ne l apprend pas. Le fixture relu rendrait alors un calque vide la ou la production en
// publie un plein, exactement le defaut que la decouverte D9 avait mesure sur sept builds.
// `TestGoldenInputsFidelite` l attrape, mais il EXIGE LE CACHE DE FILMS : il saute en CI. Ce
// test-ci, lui, ne lit aucun octet de film et tourne partout.
//
// IL SE FONDE SUR L ALLER-RETOUR, PAS SUR UNE LISTE ECRITE A LA MAIN : une valeur non nulle est
// posee dans chaque champ, le blob est encode puis relu, et un champ qui revient VIDE n a pas ete
// transporte. Une liste de noms aurait derive du codec au premier oubli.
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
			relu, err := DecodeFilmFacts(EncodeFilmFacts(g), entry)
			if err != nil {
				t.Fatalf("aller-retour sur %s : %v", f.Name, err)
			}
			transporte := !reflect.ValueOf(relu.FilmInputs).FieldByName(f.Name).IsZero()
			raison, nomme := champsNonTransportes[f.Name]
			switch {
			case transporte && nomme:
				t.Fatalf("%s est transporte par le codec ET declare absent (%q) : retirer l entree "+
					"de champsNonTransportes", f.Name, raison)
			case !transporte && !nomme:
				t.Fatalf("FilmInputs.%s N EST PAS TRANSPORTE par le codec du fixture.\n"+
					"L assemblage le consomme (cf. FilmInputs.applyTo) : relu vide, le golden "+
					"figerait un calque que la production publie plein.\n"+
					"Soit l ajouter au codec (golden_inputs_canaux_test.go) et monter "+
					"filmFactsMagic, soit l inscrire dans champsNonTransportes avec sa raison.",
					f.Name)
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
