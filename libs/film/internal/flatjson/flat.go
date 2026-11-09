// Package flatjson porte la forme « à plat » d'un texte par langue dans un objet JSON qui porte
// d'autres champs : COPIE du codec `internal/i18n/flat.go` de LevelUp, réécrite pour une brique qui
// ne connaît AUCUNE liste de langues (plan D2, décision D2-D ; règle 6 de CLAUDE.md).
//
// LE FIL EST CELUI DE LEVELUP, OCTET POUR OCTET, et c'est le contrat : le document de rejeu garde un
// libellé par langue AU MÊME NIVEAU que les autres champs, et des artefacts cuits se relisent sans
// recuisson (ADR 0040 D-9). Le garde-rail qui refuse toute divergence vit côté LevelUp, là où les
// deux codecs sont visibles : `internal/games/halo_infinite/replaylabels/fil_jumeau_test.go` encode
// les mêmes libellés par `i18n.MarshalFlat` et par les types du rejeu (qui passent par ce paquet)
// pour les cinq dispositions du document, exige les mêmes octets, puis relit chaque fil par l'autre
// codec.
//
// # CE QUI DIFFÈRE DE L'ORIGINAL, ET POURQUOI
//
// L'original tire ses langues de la liste de l'application. La brique TRANSPORTE des libellés, elle
// ne les choisit pas (D2-D) :
//
//   - ÉCRITURE : les langues émises sont les CLÉS DU TEXTE, dans l'ordre alphabétique (celui de
//     l'original). Sans `OmitEmpty`, une clé à valeur vide est émise ; une langue ABSENTE du texte
//     ne l'est pas. La complétude est la charge du PRODUCTEUR : la conversion de LevelUp vers la
//     brique (`replaylabels.TexteVersFilm`) pose chaque langue de la liste, vide comprise, et
//     aucune autre — ce qui rend le fil de l'original.
//   - LECTURE : un membre est une langue quand son nom n'est celui d'aucun champ du parent (même
//     règle de casse qu'`encoding/json`), qu'il a la forme d'un code de langue de l'application
//     (sous-étiquette primaire BCP-47 : deux ou trois lettres ASCII minuscules, ADR 0040) et que sa
//     valeur est une chaîne. Un membre à `null` ou d'un autre type n'est pas une langue (l'original
//     refuse un membre de SA liste qui ne serait pas une chaîne ; la brique, sans liste, ne peut
//     que l'écarter). L'original ne retient que les langues de sa liste : la conversion inverse
//     (`replaylabels.TexteDepuisFilm`) fait ce filtre, de sorte que LevelUp lit exactement ce qu'il
//     lisait.
//
// COMMENT UN STRUCT S'Y BRANCHE (modèle : `replay.Label`) :
//
//	type Label struct {
//		Text Text   `json:"i18n_text,omitempty" required:"true"` // jamais écrit sur le fil
//		Img  string `json:"img,omitempty"`
//	}
//	type labelWire Label // mêmes champs, sans méthodes (évite la récursion)
//	func (l Label) MarshalJSON() ([]byte, error)  { return flatjson.MarshalFlat[Text](labelWire(l), layout) }
//	func (l *Label) UnmarshalJSON(b []byte) error { return flatjson.UnmarshalFlat[Text](b, (*labelWire)(l)) }
package flatjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// FlatEnd, en valeur de FlatLayout.After, place les langues après tous les autres membres.
const FlatEnd = "\x00end"

// FlatLayout dit où et comment les langues s'écrivent dans l'objet du parent.
type FlatLayout struct {
	// After est le nom JSON du membre après lequel les langues s'insèrent ; "" les place avant
	// tous les autres, FlatEnd après. Un membre nommé mais absent du corps (omitempty) est une
	// erreur de programmation : MarshalFlat la rend.
	After string
	// OmitEmpty : une langue à valeur vide n'est pas émise.
	OmitEmpty bool
}

// textField retrouve l'unique champ de type T d'un struct (ou d'un pointeur vers struct).
func textField[T ~map[string]string](v reflect.Value) (reflect.Value, error) {
	textType := reflect.TypeFor[T]()
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("flatjson: %s n'est pas un struct", v.Type())
	}
	var found reflect.Value
	for i := range v.NumField() {
		if v.Type().Field(i).Type == textType {
			if found.IsValid() {
				return reflect.Value{}, fmt.Errorf("flatjson: %s porte plusieurs champs %s", v.Type(), textType)
			}
			found = v.Field(i)
		}
	}
	if !found.IsValid() {
		return reflect.Value{}, fmt.Errorf("flatjson: %s ne porte aucun champ %s", v.Type(), textType)
	}
	return found, nil
}

// MarshalFlat sérialise `wire` (le parent, converti en type sans méthode) en y insérant les
// langues de son champ de type T selon `layout`. Le résultat est l'objet JSON complet ; `wire`
// n'est pas modifié (le champ texte est vidé sur une copie, pour que son membre ne soit jamais
// écrit).
func MarshalFlat[T ~map[string]string](wire any, layout FlatLayout) ([]byte, error) {
	cp := reflect.New(reflect.TypeOf(wire)).Elem()
	cp.Set(reflect.ValueOf(wire))
	field, err := textField[T](cp)
	if err != nil {
		return nil, err
	}
	text, _ := field.Interface().(T)
	field.SetZero()

	rest, err := json.Marshal(cp.Interface())
	if err != nil {
		return nil, err
	}
	members, err := flatMembers(text, layout.OmitEmpty)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return rest, nil
	}
	at, err := insertionOffset(rest, layout.After)
	if err != nil {
		return nil, err
	}
	hasBefore := at > 1
	// Insérée au début, la suite des membres commence juste derrière : elle demande sa virgule.
	// Insérée après un membre, la virgule qui sépare ce membre du suivant est déjà dans `rest[at:]`.
	hasAfter := at == 1 && len(rest) > 2
	out := make([]byte, 0, len(rest)+len(members)+2)
	out = append(out, rest[:at]...)
	if hasBefore {
		out = append(out, ',')
	}
	out = append(out, members...)
	if hasAfter {
		out = append(out, ',')
	}
	return append(out, rest[at:]...), nil
}

// flatMembers rend les membres JSON des langues du texte (sans accolades), dans l'ordre
// alphabétique des codes, séparés par des virgules.
func flatMembers[T ~map[string]string](t T, omitEmpty bool) ([]byte, error) {
	codes := make([]string, 0, len(t))
	for code := range t {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	var b bytes.Buffer
	for _, code := range codes {
		v := t[code]
		if v == "" && omitEmpty {
			continue
		}
		key, err := json.Marshal(code)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	return b.Bytes(), nil
}

// insertionOffset rend l'offset, dans l'objet `obj`, où les membres de langue s'insèrent : 1
// (après l'accolade ouvrante) pour After == "", len-1 (avant la fermante) pour FlatEnd, sinon
// juste après la valeur du membre `after`.
func insertionOffset(obj []byte, after string) (int, error) {
	switch after {
	case "":
		return 1, nil
	case FlatEnd:
		return len(obj) - 1, nil
	}
	dec := json.NewDecoder(bytes.NewReader(obj))
	if _, err := dec.Token(); err != nil { // '{'
		return 0, err
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return 0, err
		}
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return 0, err
		}
		if key, _ := keyTok.(string); key == after {
			return int(dec.InputOffset()), nil
		}
	}
	return 0, fmt.Errorf("flatjson: membre %q absent de l'objet, les langues ne peuvent pas s'y insérer", after)
}

// nomsDesChamps : par type de parent, les noms JSON de ses champs (ceux qu'`encoding/json` lit),
// calculés une fois par type.
var nomsDesChamps sync.Map // reflect.Type -> []string

// nomsJSON rend les noms JSON des champs exportés d'un struct, champs anonymes promus compris.
func nomsJSON(t reflect.Type) []string {
	if v, ok := nomsDesChamps.Load(t); ok {
		noms, _ := v.([]string)
		return noms
	}
	var noms []string
	for i := range t.NumField() {
		f := t.Field(i)
		nom, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if nom == "-" {
			continue
		}
		if f.Anonymous && nom == "" && f.Type.Kind() == reflect.Struct {
			noms = append(noms, nomsJSON(f.Type)...)
			continue
		}
		if !f.IsExported() {
			continue
		}
		if nom == "" {
			nom = f.Name
		}
		noms = append(noms, nom)
	}
	nomsDesChamps.Store(t, noms)
	return noms
}

// estUnChamp dit si `membre` est lu par un champ du parent (comparaison sans casse, comme
// `encoding/json`).
func estUnChamp(noms []string, membre string) bool {
	for _, n := range noms {
		if strings.EqualFold(n, membre) {
			return true
		}
	}
	return false
}

// aLaFormeDUnCode dit si `membre` a la forme d'un code de langue de l'application : deux ou trois
// lettres ASCII minuscules (sous-étiquette primaire BCP-47, ADR 0040). C'est une FORME, pas une
// liste : la brique ne sait pas quelles langues existent.
func aLaFormeDUnCode(membre string) bool {
	if len(membre) < 2 || len(membre) > 3 {
		return false
	}
	for i := range len(membre) {
		if membre[i] < 'a' || membre[i] > 'z' {
			return false
		}
	}
	return true
}

// estUneChaine dit si la valeur JSON brute est une chaîne (et non `null`, un nombre, un objet...).
func estUneChaine(brut json.RawMessage) bool {
	brut = bytes.TrimSpace(brut)
	return len(brut) > 0 && brut[0] == '"'
}

// UnmarshalFlat lit `data` dans `wire` (pointeur vers le parent converti en type sans méthode),
// puis range dans son champ de type T les membres qui sont des langues (cf. l'en-tête). Un membre
// à `null` n'entre pas dans le texte ; un membre à "" y entre (« langue connue, valeur vide »). Le
// texte repart de nil à chaque lecture.
func UnmarshalFlat[T ~map[string]string](data []byte, wire any) error {
	field, err := textField[T](reflect.ValueOf(wire))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, wire); err != nil {
		return err
	}
	field.SetZero()
	var membres map[string]json.RawMessage
	if err := json.Unmarshal(data, &membres); err != nil {
		return err
	}
	noms := nomsJSON(reflect.TypeOf(wire).Elem())
	var t T
	for membre, brut := range membres {
		if !aLaFormeDUnCode(membre) || estUnChamp(noms, membre) || !estUneChaine(brut) {
			continue
		}
		var v string
		if err := json.Unmarshal(brut, &v); err != nil {
			return fmt.Errorf("flatjson: membre %q : %w", membre, err)
		}
		if t == nil {
			t = T{}
		}
		t[membre] = v
	}
	field.Set(reflect.ValueOf(t))
	return nil
}
