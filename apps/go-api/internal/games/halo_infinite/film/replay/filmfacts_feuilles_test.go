package replay

// filmfacts_feuilles_test.go — L INSTRUMENT FEUILLE A FEUILLE DU CODEC DES FAITS (jalon J11.0-bis
// de la suite d audit du decodeur, 2026-09-28).
//
// # LE TROU QU IL FERME
//
// `TestCodecCouvreFilmInputs` jugeait un champ de [FilmInputs] transporte des qu UNE de ses
// feuilles revenait non nulle. Une structure de vingt compteurs passait donc sur le premier, et
// c est par ce trou que `MovementStateStats.JumpEpisodes/JumpsDerived` sont sortis du fichier de
// faits (un artefact rejoue depuis les faits perdait `coverage.stances`, corrige en `56edb6c8b`).
//
// # CE QU IL MESURE
//
// Chaque FEUILLE (champ scalaire, non exportes compris, a toute profondeur, sous toute tranche et
// toute table) est comparee avant et apres l aller-retour par le FICHIER de faits
// (`EncodeFilmFactsFile` / `DecodeFilmFactsFile` : le blob ET le complement de la section 1), a
// l octet pres (un flottant par ses bits). Un ecart est range sous le CHAMP DECLARE qui le porte
// (`grammar.FireEvent.Short`), pas sous un chemin : un type repete (les positions de bipede et
// celles des vehicules) n a qu une entree.
//
// Deux mesures l emploient : le TEMOIN SYNTHETIQUE (`TestCodecCouvreFilmInputs`, en CI : chaque
// feuille recoit une valeur non nulle distincte) et les FILMS REELS des huit builds
// (`TestCodecFeuilleAFeuilleSurLesFilms`, gate local : les entrees DECODEES DU FILM, jamais un
// fixture deja passe par le codec).
//
// # LA SEULE ISSUE POUR UNE FEUILLE PERDUE : UNE EXCEPTION NOMMEE ET PROUVEE
//
// Cf. [feuillesNonTransportees] (`filmfacts_feuilles_exceptions_test.go`).

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// ecartDeFeuille compte les valeurs d un champ declare qui n ont pas fait l aller-retour.
type ecartDeFeuille struct {
	n       int
	chemin  string
	exemple string
}

// ecartsDeFeuilles : champ declare (suffixe `#len`, `#cle`, `#nil` pour une forme) -> ecart.
type ecartsDeFeuilles map[string]*ecartDeFeuille

func (e ecartsDeFeuilles) noter(cle, chemin, exemple string) {
	x := e[cle]
	if x == nil {
		x = &ecartDeFeuille{chemin: chemin, exemple: exemple}
		e[cle] = x
	}
	x.n++
}

// cleDeChamp nomme un champ par son TYPE PROPRIETAIRE : `grammar.FireEvent.Short`.
func cleDeChamp(proprietaire reflect.Type, i int) string {
	return nomDeTypeQualifie(proprietaire) + "." + proprietaire.Field(i).Name
}

// nomDeTypeQualifie : `grammar.FireEvent` (dernier segment du chemin de paquet, puis le nom).
func nomDeTypeQualifie(t reflect.Type) string {
	pkg := t.PkgPath()
	if k := strings.LastIndex(pkg, "/"); k >= 0 {
		pkg = pkg[k+1:]
	}
	return pkg + "." + t.Name()
}

// champNomme rend l exception qui couvre une cle d ecart (suffixe de forme retire), ou "".
func champNomme(cle string) string {
	if k := strings.IndexByte(cle, '#'); k >= 0 {
		cle = cle[:k]
	}
	if _, ok := feuillesNonTransportees[cle]; ok {
		return cle
	}
	return ""
}

// comparerFeuilles parcourt `a` (avant) et `b` (apres) en parallele. `cle` est le champ declare
// courant. Sous un champ NOMME, le sous-arbre entier se range sous son nom : ce qu on declare
// perdu, on le declare en bloc.
func comparerFeuilles(cle, chemin string, a, b reflect.Value, out ecartsDeFeuilles) {
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			c := cleDeChamp(a.Type(), i)
			sous := chemin + "." + a.Type().Field(i).Name
			if _, nomme := feuillesNonTransportees[c]; nomme {
				bloc := ecartsDeFeuilles{}
				comparerFeuilles(c, sous, a.Field(i), b.Field(i), bloc)
				if len(bloc) > 0 {
					out.noter(c, sous, "sous-arbre perdu")
				}
				continue
			}
			comparerFeuilles(c, sous, a.Field(i), b.Field(i), out)
		}
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				out.noter(cle+"#nil", chemin, fmt.Sprintf("nil avant=%v apres=%v", a.IsNil(), b.IsNil()))
			}
			return
		}
		comparerFeuilles(cle, chemin, a.Elem(), b.Elem(), out)
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			out.noter(cle+"#len", chemin, fmt.Sprintf("longueur avant %d, apres %d", a.Len(), b.Len()))
		}
		for i := 0; i < min(a.Len(), b.Len()); i++ {
			comparerFeuilles(cle, chemin+"[]", a.Index(i), b.Index(i), out)
		}
	case reflect.Map:
		comparerTables(cle, chemin, a, b, out)
	default:
		if ex, egal := comparerScalaires(a, b); !egal {
			out.noter(cle, chemin, ex)
		}
	}
}

func comparerTables(cle, chemin string, a, b reflect.Value, out ecartsDeFeuilles) {
	if a.Len() != b.Len() {
		out.noter(cle+"#len", chemin, fmt.Sprintf("entrees avant %d, apres %d", a.Len(), b.Len()))
	}
	for _, k := range a.MapKeys() {
		vb := b.MapIndex(k)
		if !vb.IsValid() {
			out.noter(cle+"#cle", chemin, fmt.Sprintf("cle %v perdue", k))
			continue
		}
		comparerFeuilles(cle, chemin+"{}", a.MapIndex(k), vb, out)
	}
}

// comparerScalaires compare deux feuilles A L OCTET PRES : un flottant se compare par ses bits
// (`-0` et `0` different, et le document publie la difference — cf. `encodeTracks`).
func comparerScalaires(a, b reflect.Value) (string, bool) {
	switch a.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("avant %v, apres %v", a.Float(), b.Float()),
			math.Float64bits(a.Float()) == math.Float64bits(b.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("avant %d, apres %d", a.Int(), b.Int()), a.Int() == b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("avant %d, apres %d", a.Uint(), b.Uint()), a.Uint() == b.Uint()
	case reflect.Bool:
		return fmt.Sprintf("avant %v, apres %v", a.Bool(), b.Bool()), a.Bool() == b.Bool()
	case reflect.String:
		return fmt.Sprintf("avant %q, apres %q", a.String(), b.String()), a.String() == b.String()
	case reflect.Func, reflect.Chan:
		return "fonction ou canal", a.IsNil() == b.IsNil()
	default:
		return "forme " + a.Kind().String() + " non comparee", false
	}
}

// allerRetourParLeFichier encode puis relit des faits par le FICHIER, comme la production.
func allerRetourParLeFichier(t *testing.T, g *FilmFacts, entry profile.MapQuantEntry) *FilmFacts {
	t.Helper()
	blob, err := EncodeFilmFactsFile(&FilmFactsFile{Facts: *g, EmpreinteDeCle: EmpreinteDeCle(entry)})
	if err != nil {
		t.Fatalf("encodage du fichier de faits : %v", err)
	}
	relu, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture du fichier de faits : %v", err)
	}
	return &relu.Facts
}

// ecartsDeLAllerRetour rend les ecarts feuille a feuille de [FilmFacts] entre avant et apres.
func ecartsDeLAllerRetour(avant, apres *FilmFacts) ecartsDeFeuilles {
	out := ecartsDeFeuilles{}
	comparerFeuilles("FilmFacts", "FilmFacts", reflect.ValueOf(*avant), reflect.ValueOf(*apres), out)
	return out
}

// nonNommes rend, tries, les ecarts qu aucune exception ne couvre.
func (e ecartsDeFeuilles) nonNommes() []string {
	var out []string
	for c, x := range e {
		if champNomme(c) == "" {
			out = append(out, fmt.Sprintf("%s (%d valeur(s), %s : %s)", c, x.n, x.chemin, x.exemple))
		}
	}
	sort.Strings(out)
	return out
}

// nommes rend l ensemble des exceptions que ces ecarts exercent.
func (e ecartsDeFeuilles) nommes() map[string]bool {
	out := map[string]bool{}
	for c := range e {
		if n := champNomme(c); n != "" {
			out[n] = true
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// LE TEMOIN SYNTHETIQUE : CHAQUE FEUILLE RECOIT UNE VALEUR NON NULLE DISTINCTE
// ---------------------------------------------------------------------------

// temoinDeFeuilles remplit TOUTES les feuilles d une valeur (champs non exportes compris) d une
// valeur non nulle, distincte de ses voisines ; une tranche recoit un element, une table une
// entree. `nonRemplis` nomme les feuilles qu il n a pas su remplir hors d une exception : une
// forme neuve ne sort pas en silence de la mesure.
type temoinDeFeuilles struct {
	n          uint64
	nonRemplis []string
}

func (w *temoinDeFeuilles) remplir(cle string, v reflect.Value, sousException bool) {
	if !v.CanSet() && v.CanAddr() {
		v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem() //nolint:gosec // temoin de test
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			c := cleDeChamp(v.Type(), i)
			_, nomme := feuillesNonTransportees[c]
			w.remplir(c, v.Field(i), sousException || nomme)
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			w.remplir(cle, v.Index(i), sousException)
		}
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		w.remplir(cle, p.Elem(), sousException)
		v.Set(p)
	case reflect.Slice:
		e := reflect.New(v.Type().Elem()).Elem()
		w.remplir(cle, e, sousException)
		v.Set(reflect.Append(reflect.MakeSlice(v.Type(), 0, 1), e))
	case reflect.Map:
		k, e := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		w.remplir(cle, k, sousException)
		w.remplir(cle, e, sousException)
		m := reflect.MakeMap(v.Type())
		m.SetMapIndex(k, e)
		v.Set(m)
	default:
		w.remplirScalaire(cle, v, sousException)
	}
}

func (w *temoinDeFeuilles) remplirScalaire(cle string, v reflect.Value, sousException bool) {
	w.n++
	x := w.n%120 + 1 // jamais nul, tient dans un octet, et deux voisines different toujours
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(x)) //nolint:gosec // petit compteur de test
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(x)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(x) + 0.25)
	case reflect.String:
		v.SetString(fmt.Sprintf("temoin%d", x))
	default:
		if !sousException {
			w.nonRemplis = append(w.nonRemplis, cle+" ("+v.Kind().String()+")")
		}
	}
}

// accorderLeTemoin rend COHERENTES les feuilles que le codec DERIVE ou BORNE par construction —
// et seulement celles-la, chacune nommee. Ce ne sont pas des pertes : le fichier les porte sous
// une autre forme, un film reel les produit toujours ainsi, et la mesure sur les huit films les
// compare a l octet sans accord.
//
//   - `grammar.BipedPosition.X/Y/Z` valent `DequantBipedAxis(Q[a], a, decoupage, bornes)` : le
//     fichier porte les QUANTA et redequantifie par le meme chemin (cf. `encodePositionSection`).
//   - `grammar.componentDirs.FwdMode` tient sur deux bits (`gdModeMask`) : le balayage ne rend
//     rien d autre.
func accorderLeTemoin(g *FilmFacts, entry profile.MapQuantEntry) {
	lay, world := profile.I0Layout{AxisW: g.AxisW}, entry.Range()
	accorder := func(pos []grammar.BipedPosition) {
		for i := range pos {
			p := &pos[i]
			p.X = grammar.DequantBipedAxis(p.Q[0], 0, lay, world)
			p.Y = grammar.DequantBipedAxis(p.Q[1], 1, lay, world)
			p.Z = grammar.DequantBipedAxis(p.Q[2], 2, lay, world)
			if p.FwdMode &= gdModeMask; p.FwdMode == 0 {
				p.FwdMode = 1
			}
		}
	}
	accorder(g.Positions)
	accorder(g.Vehicles.Positions)
}
