package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

// Forme « à plat » d'un Text dans un objet JSON qui porte d'autres champs.
//
// POURQUOI. Le document de rejeu garde un libellé PAR LANGUE dans le document (gros document,
// mis en cache côté client, artefacts cuits qu'on ne re-cuit pas) : un libellé d'arme y est
// `{"en": "...", "fr": "...", "fx": "...", "key": "..."}`, les langues AU MÊME NIVEAU que les
// autres champs. Une langue de plus ne doit pas demander un champ Go de plus : le struct porte
// un `Text`, et la forme de fil reste plate, octet pour octet celle d'avant pour `en` / `fr`.
//
// COMMENT UN STRUCT S'Y BRANCHE (le modèle est `domain/replaydoc.Label`) :
//
//	type Label struct {
//		Text i18n.Text `json:"i18n_text,omitempty" required:"true"` // déclaré où les langues s'écrivent
//		Img  string    `json:"img,omitempty"`
//	}
//	type labelWire Label // mêmes champs, sans méthodes (évite la récursion)
//	func (l Label) MarshalJSON() ([]byte, error)  { return i18n.MarshalFlat(labelWire(l), layout) }
//	func (l *Label) UnmarshalJSON(b []byte) error { return i18n.UnmarshalFlat(b, (*labelWire)(l)) }
//
// Le membre `i18n_text` n'est JAMAIS écrit sur le fil (MarshalFlat vide le champ sur une copie
// avant de sérialiser) : le tag ne sert qu'à la génération du contrat, où `openapigen` remplace
// ce membre par une propriété par langue de `config/locales.toml`, à la même place dans
// `required`. `required:"true"` dit « toutes les langues sont émises, vides comprises » ; sans
// lui, une langue vide est omise (le cas de `omitempty` d'avant) et FlatLayout.OmitEmpty doit être vrai :
// le tag décrit le contrat, le layout décrit le fil, et les tests de chaque parent les comparent à
// la forme d'avant.
//
// LANGUES ÉMISES : celles de la liste de l'application, dans l'ordre alphabétique de leur code
// (`en` avant `fr`, comme l'ordre de déclaration des deux anciens champs). Un texte qui porte une
// clé hors liste ne l'émet pas ; à la lecture, un membre dont le nom n'est pas une langue de la
// liste n'entre pas dans le Text (un artefact cuit sous une langue retirée de la liste se lit).

// FlatEnd, en valeur de FlatLayout.After, place les langues après tous les autres membres.
const FlatEnd = "\x00end"

// FlatLayout dit où et comment les langues s'écrivent dans l'objet du parent.
type FlatLayout struct {
	// After est le nom JSON du membre après lequel les langues s'insèrent ; "" les place avant
	// tous les autres, FlatEnd après. Un membre nommé mais absent du corps (omitempty) est une
	// erreur de programmation : MarshalFlat la rend.
	After string
	// OmitEmpty : une langue vide ou absente n'est pas émise. Sinon toutes les langues de la
	// liste sont émises, "" pour celles que le texte ne porte pas.
	OmitEmpty bool
}

// FlatCodes retourne les codes des langues de la liste de l'application dans l'ordre où le
// fil les écrit (alphabétique).
func FlatCodes() []Locale {
	out := Locales()
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

var textType = reflect.TypeFor[Text]()

// textField retrouve l'unique champ de type Text d'un struct (ou d'un pointeur vers struct).
func textField(v reflect.Value) (reflect.Value, error) {
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("i18n: %s n'est pas un struct", v.Type())
	}
	var found reflect.Value
	for i := range v.NumField() {
		if v.Type().Field(i).Type == textType {
			if found.IsValid() {
				return reflect.Value{}, fmt.Errorf("i18n: %s porte plusieurs champs Text", v.Type())
			}
			found = v.Field(i)
		}
	}
	if !found.IsValid() {
		return reflect.Value{}, fmt.Errorf("i18n: %s ne porte aucun champ Text", v.Type())
	}
	return found, nil
}

// MarshalFlat sérialise `wire` (le parent, converti en type sans méthode) en y insérant les
// langues de son champ Text selon `layout`. Le résultat est l'objet JSON complet ; `wire` n'est
// pas modifié (le champ Text est vidé sur une copie, pour que le membre `i18n_text` ne soit
// jamais écrit).
func MarshalFlat(wire any, layout FlatLayout) ([]byte, error) {
	cp := reflect.New(reflect.TypeOf(wire)).Elem()
	cp.Set(reflect.ValueOf(wire))
	field, err := textField(cp)
	if err != nil {
		return nil, err
	}
	text, _ := field.Interface().(Text)
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

// flatMembers rend les membres JSON des langues (sans accolades), séparés par des virgules.
func flatMembers(t Text, omitEmpty bool) ([]byte, error) {
	var b bytes.Buffer
	for _, code := range FlatCodes() {
		v := t[code]
		if v == "" && omitEmpty {
			continue
		}
		key, err := json.Marshal(string(code))
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
	return 0, fmt.Errorf("i18n: membre %q absent de l'objet, les langues ne peuvent pas s'y insérer", after)
}

// langFields est le type dynamique `struct{ L0 *string `json:"en"`; L1 *string `json:"fr"` ... }`
// d'une lecture : json.Unmarshal y saute sans copie tous les membres qui ne sont pas une langue.
type langFields struct {
	once  sync.Once
	typ   reflect.Type
	codes []Locale
}

var flatFields langFields

func (f *langFields) init() {
	f.once.Do(func() {
		f.codes = FlatCodes()
		fields := make([]reflect.StructField, len(f.codes))
		for i, c := range f.codes {
			fields[i] = reflect.StructField{
				Name: fmt.Sprintf("L%d", i),
				Type: reflect.TypeFor[*string](),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:%q`, string(c))),
			}
		}
		f.typ = reflect.StructOf(fields)
	})
}

// UnmarshalFlat lit `data` dans `wire` (pointeur vers le parent converti en type sans méthode),
// puis range dans son champ Text les membres dont le nom est une langue de la liste. Un membre
// à `null` ou absent n'entre pas dans le texte ; un membre à "" y entre (le texte dit « langue
// connue, valeur vide »). Le Text repart de nil à chaque lecture.
func UnmarshalFlat(data []byte, wire any) error {
	field, err := textField(reflect.ValueOf(wire))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, wire); err != nil {
		return err
	}
	field.SetZero()
	flatFields.init()
	dst := reflect.New(flatFields.typ)
	if err := json.Unmarshal(data, dst.Interface()); err != nil {
		return err
	}
	var t Text
	for i, code := range flatFields.codes {
		if p, _ := dst.Elem().Field(i).Interface().(*string); p != nil {
			t.Set(code, *p)
		}
	}
	field.Set(reflect.ValueOf(t))
	return nil
}

// FlatMember est le nom JSON que porte, dans le tag d'un struct, le champ Text d'un parent à
// plat. Il n'est jamais écrit sur le fil (cf. MarshalFlat) : c'est le repère que `openapigen`
// remplace, dans le contrat, par une propriété par langue de la liste.
const FlatMember = "i18n_text"
