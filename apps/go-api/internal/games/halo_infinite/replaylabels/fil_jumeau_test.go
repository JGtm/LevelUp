package replaylabels_test

// fil_jumeau_test.go — LE CODEC « À PLAT » DU DÉCODEUR ÉCRIT ET LIT LE FIL DE `i18n` (plan D2,
// item 2.9 ; règle 6 de CLAUDE.md ; ADR 0040 D-9).
//
// Le décodeur de film porte une copie du codec des libellés par langue (`film/internal/flatjson`),
// réécrite sans liste de langues ; LevelUp garde l'original (`i18n.MarshalFlat` /
// `i18n.UnmarshalFlat`). Les artefacts cuits sont relus sans recuisson : les deux codecs doivent
// écrire LES MÊMES OCTETS et se relire l'un l'autre. Ce test le vérifie pour chaque type du document
// qui porte un libellé, donc pour les cinq dispositions de `replay/document_labels_json.go` (début,
// après `name`, après `family`, après `vehicle`, fin sans les langues vides), sur des textes complets,
// partiels, vides, nil, à échappements et à clé hors liste :
//
//  1. le type du rejeu, rempli par TexteVersFilm, s'écrit comme son MIROIR (mêmes champs, le texte
//     en `i18n.Text`) écrit par `i18n.MarshalFlat` sous la disposition déclarée ci-dessous ;
//  2. ce fil relu par le décodeur, puis converti par TexteDepuisFilm, rend le texte que
//     `i18n.UnmarshalFlat` en relit, et les mêmes autres champs ;
//  3. le fil écrit par le décodeur, relu par `i18n.UnmarshalFlat`, rend le même texte.
//
// Les dispositions sont RÉÉCRITES ici, en `i18n.FlatLayout` : ce sont celles du fil d'avant D2. Une
// disposition changée d'un seul côté fait rougir l'étape 1.

import (
	"encoding/json"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/i18n"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

// dispositionDuFil : un type du rejeu, rempli hors texte, et la disposition sous laquelle `i18n`
// l'écrivait.
type dispositionDuFil struct {
	nom    string
	valeur any
	layout i18n.FlatLayout
}

func dispositionsDuFil() []dispositionDuFil {
	poly := [][2]float64{{0, 0}, {5, 0}, {5, 5}}
	return []dispositionDuFil{
		{"Label", replay.Label{Img: "i.png", Tinted: true, Family: "wall"}, i18n.FlatLayout{}},
		{"Label nu", replay.Label{}, i18n.FlatLayout{}},
		{"WeaponLabel", replay.WeaponLabel{Fx: "plasma", Key: "k", Role: "power", Tint: "t", Impact: true, Img: "i", Tinted: true}, i18n.FlatLayout{}},
		{"CalloutZone", replay.CalloutZone{VolumeIndex: 3, Name: "n", X: 1, Y: 2, Z: 3, ZTop: 4, Big: true, Polygon: poly}, i18n.FlatLayout{After: "name"}},
		{"CalloutZone nue", replay.CalloutZone{}, i18n.FlatLayout{After: "name"}},
		{"ObjectiveObjectLife", replay.ObjectiveObjectLife{Family: "ball", T0: 1, T1: 2}, i18n.FlatLayout{After: "family"}},
		{"VehicleWeapon", replay.VehicleWeapon{Vehicle: "wraith", Fire: "single", Fx: "plasma", Tint: "t", Sound: "s", Impact: true}, i18n.FlatLayout{After: "vehicle"}},
		{"VehicleWeapon nue", replay.VehicleWeapon{}, i18n.FlatLayout{After: "vehicle"}},
		{"VehicleLabel", replay.VehicleLabel{Img: "v.png", Tinted: true, Kind: "map_element"}, i18n.FlatLayout{After: i18n.FlatEnd, OmitEmpty: true}},
		{"VehicleLabel nu", replay.VehicleLabel{}, i18n.FlatLayout{After: i18n.FlatEnd, OmitEmpty: true}},
	}
}

// textesDuFil : les libellés éprouvés, construits sur la liste de l'application (aucune langue
// écrite en dur).
func textesDuFil() map[string]i18n.Text {
	codes := i18n.FlatCodes()
	complet, vides := i18n.Text{}, i18n.Text{}
	for _, c := range codes {
		complet[c] = "Libellé <" + string(c) + "> & \"q\" é"
		vides[c] = ""
	}
	premier, dernier := codes[0], codes[len(codes)-1]
	return map[string]i18n.Text{
		"nil":            nil,
		"vide":           {},
		"complet":        complet,
		"langues vides":  vides,
		"premier seul":   {premier: "A"},
		"dernier seul":   {dernier: "Z"},
		"hors liste":     {premier: "a", i18n.Locale("zz"): "hors liste"},
		"premier vide":   {premier: "", dernier: "z"},
		"retour ligne":   {premier: "ligne\nsuivante\t "},
		"html et unicod": {dernier: "<b>&amp;</b> é日"},
	}
}

func TestFilJumeau_MemesOctetsEtRelectureCroisee(t *testing.T) {
	for _, d := range dispositionsDuFil() {
		for nomTexte, tx := range textesDuFil() {
			t.Run(d.nom+"/"+nomTexte, func(t *testing.T) {
				verifierFilJumeau(t, d, tx)
			})
		}
	}
}

func verifierFilJumeau(t *testing.T, d dispositionDuFil, tx i18n.Text) {
	t.Helper()
	film := avecTexte(t, d.valeur, replaylabels.TexteVersFilm(tx))
	miroir := miroirI18n(t, d.valeur, tx)

	// 1. Mêmes octets.
	filI18n, err := i18n.MarshalFlat(miroir.Interface(), d.layout)
	if err != nil {
		t.Fatalf("i18n.MarshalFlat : %v", err)
	}
	filFilm, err := json.Marshal(film.Interface())
	if err != nil {
		t.Fatalf("écriture par le décodeur : %v", err)
	}
	if string(filFilm) != string(filI18n) {
		t.Fatalf("fils différents :\n  décodeur %s\n  i18n     %s", filFilm, filI18n)
	}

	// 2. Le fil d'i18n relu par le décodeur rend le texte et les champs qu'i18n en relit.
	luFilm := relireParLeDecodeur(t, d.valeur, filI18n)
	luI18n := relireParI18n(t, d.valeur, filI18n)
	if got, want := replaylabels.TexteDepuisFilm(texteDe(luFilm)), texteI18nDe(luI18n); !got.Equal(want) {
		t.Errorf("fil d'i18n relu : décodeur %v, i18n %v", got, want)
	}
	comparerHorsTexte(t, luFilm, luI18n)

	// 3. Le fil du décodeur relu par i18n rend le même texte.
	luI18nDuFilm := relireParI18n(t, d.valeur, filFilm)
	if got, want := texteI18nDe(luI18nDuFilm), replaylabels.TexteDepuisFilm(texteDe(relireParLeDecodeur(t, d.valeur, filFilm))); !got.Equal(want) {
		t.Errorf("fil du décodeur relu : i18n %v, décodeur %v", got, want)
	}
}

// TestFilJumeau_MembresQueSeulI18nIgnore : un fil réel peut porter une clé de langue hors liste ou
// un membre inconnu ; relu par le décodeur puis converti, il rend ce que lit `i18n`.
func TestFilJumeau_MembresQueSeulI18nIgnore(t *testing.T) {
	codes := i18n.FlatCodes()
	premier := string(codes[0])
	fils := []string{
		`{"` + premier + `":"a","zz":"b","img":"i.png","inconnu":"x","de":null}`,
		`{"` + premier + `":""}`,
		`{}`,
		`null`,
	}
	label := replay.Label{}
	for _, fil := range fils {
		luFilm := relireParLeDecodeur(t, label, []byte(fil))
		luI18n := relireParI18n(t, label, []byte(fil))
		if got, want := replaylabels.TexteDepuisFilm(texteDe(luFilm)), texteI18nDe(luI18n); !got.Equal(want) {
			t.Errorf("fil %s : décodeur %v, i18n %v", fil, got, want)
		}
		comparerHorsTexte(t, luFilm, luI18n)
	}
}

// TestFilJumeau_MembreDuContrat : le champ texte de chaque type du rejeu porte le nom JSON que
// `openapigen` remplace par une propriété par langue (`i18n.FlatMember`).
func TestFilJumeau_MembreDuContrat(t *testing.T) {
	for _, d := range dispositionsDuFil() {
		champ := champTexte(t, reflect.TypeOf(d.valeur))
		if tag := champ.Tag.Get("json"); tag != i18n.FlatMember+",omitempty" {
			t.Errorf("%s.%s : tag json %q, attendu %q", d.nom, champ.Name, tag, i18n.FlatMember+",omitempty")
		}
	}
}

// TestTexteVersFilm_ChaqueLangueDeLaListe : la conversion vers le décodeur pose chaque langue de
// la liste, vide comprise, et aucune autre ; la conversion inverse ne garde que les langues de la
// liste.
func TestTexteVersFilm_ChaqueLangueDeLaListe(t *testing.T) {
	codes := i18n.FlatCodes()
	film := replaylabels.TexteVersFilm(i18n.Text{codes[0]: "a", i18n.Locale("zz"): "b"})
	if len(film) != len(codes) {
		t.Fatalf("TexteVersFilm = %v, attendu les %d langues de la liste", film, len(codes))
	}
	for _, c := range codes {
		if _, ok := film[string(c)]; !ok {
			t.Errorf("langue %q absente de %v", c, film)
		}
	}
	if film[string(codes[0])] != "a" {
		t.Errorf("valeur perdue : %v", film)
	}
	retour := replaylabels.TexteDepuisFilm(replay.Text{string(codes[0]): "a", "zz": "b"})
	if len(retour) != 1 || retour[codes[0]] != "a" {
		t.Errorf("TexteDepuisFilm = %v, attendu la seule langue de la liste", retour)
	}
	if replaylabels.TexteDepuisFilm(nil) != nil {
		t.Error("TexteDepuisFilm(nil) doit rester nil")
	}
}

// --- outillage par réflexion ---------------------------------------------------------------------

var (
	typeTexteFilm = reflect.TypeFor[replay.Text]()
	typeTexteI18n = reflect.TypeFor[i18n.Text]()
)

// champTexte rend le champ de type replay.Text d'un type du rejeu (exactement un).
func champTexte(t *testing.T, typ reflect.Type) reflect.StructField {
	t.Helper()
	var trouves []reflect.StructField
	for f := range typ.Fields() {
		if f.Type == typeTexteFilm {
			trouves = append(trouves, f)
		}
	}
	if len(trouves) != 1 {
		t.Fatalf("%s : %d champ(s) %s, attendu 1", typ, len(trouves), typeTexteFilm)
	}
	return trouves[0]
}

// avecTexte rend une copie adressable de `valeur` dont le champ texte vaut `tx`.
func avecTexte(t *testing.T, valeur any, tx replay.Text) reflect.Value {
	t.Helper()
	v := reflect.New(reflect.TypeOf(valeur)).Elem()
	v.Set(reflect.ValueOf(valeur))
	v.FieldByIndex(champTexte(t, v.Type()).Index).Set(reflect.ValueOf(tx))
	return v
}

// typeMiroir rend le type sans méthode de mêmes champs et mêmes tags, le texte en i18n.Text.
func typeMiroir(typ reflect.Type) reflect.Type {
	champs := make([]reflect.StructField, 0, typ.NumField())
	for f := range typ.Fields() {
		if f.Type == typeTexteFilm {
			f.Type = typeTexteI18n
		}
		f.Index, f.Offset = nil, 0
		champs = append(champs, f)
	}
	return reflect.StructOf(champs)
}

// miroirI18n rend le miroir de `valeur`, champs recopiés, le texte valant `tx`.
func miroirI18n(t *testing.T, valeur any, tx i18n.Text) reflect.Value {
	t.Helper()
	src := reflect.ValueOf(valeur)
	m := reflect.New(typeMiroir(src.Type())).Elem()
	for i := range src.NumField() {
		if src.Type().Field(i).Type == typeTexteFilm {
			m.Field(i).Set(reflect.ValueOf(tx))
			continue
		}
		m.Field(i).Set(src.Field(i))
	}
	return m
}

func relireParLeDecodeur(t *testing.T, modele any, fil []byte) reflect.Value {
	t.Helper()
	p := reflect.New(reflect.TypeOf(modele))
	if err := json.Unmarshal(fil, p.Interface()); err != nil {
		t.Fatalf("relecture par le décodeur de %s : %v", fil, err)
	}
	return p.Elem()
}

func relireParI18n(t *testing.T, modele any, fil []byte) reflect.Value {
	t.Helper()
	p := reflect.New(typeMiroir(reflect.TypeOf(modele)))
	if err := i18n.UnmarshalFlat(fil, p.Interface()); err != nil {
		t.Fatalf("relecture par i18n de %s : %v", fil, err)
	}
	return p.Elem()
}

func texteDe(v reflect.Value) replay.Text {
	for i := range v.NumField() {
		if tx, ok := v.Field(i).Interface().(replay.Text); ok {
			return tx
		}
	}
	return nil
}

func texteI18nDe(v reflect.Value) i18n.Text {
	for i := range v.NumField() {
		if tx, ok := v.Field(i).Interface().(i18n.Text); ok {
			return tx
		}
	}
	return nil
}

// comparerHorsTexte exige que les deux relectures portent les mêmes valeurs hors du texte.
func comparerHorsTexte(t *testing.T, film, miroir reflect.Value) {
	t.Helper()
	for i := range film.NumField() {
		if film.Type().Field(i).Type == typeTexteFilm {
			continue
		}
		if a, b := film.Field(i).Interface(), miroir.Field(i).Interface(); !reflect.DeepEqual(a, b) {
			t.Errorf("champ %s : décodeur %v, i18n %v", film.Type().Field(i).Name, a, b)
		}
	}
}
