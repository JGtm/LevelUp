package replay

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"levelup/go-api/internal/i18n"
)

// document_labels_json_test.go — LA FORME DE FIL DES LIBELLÉS PAR LANGUE EST CELLE D'AVANT.
//
// Trois preuves : (1) chaque type face à son jumeau « d'avant » (deux champs `En` / `Fr` déclarés à
// la même place, mêmes tags) : même corps octet pour octet, vides et omitempty compris ; (2) un
// artefact RÉEL du parc (les calques à libellés d'un film d'Oddball, copie réduite lue du cache de
// films) se relit et se ré-écrit à l'identique, sans re-cuisson ; (3) le champ Text de chaque type
// porte le nom que `openapigen` remplace par une propriété par langue.

type (
	legacyLabel struct {
		En     string `json:"en"`
		Fr     string `json:"fr"`
		Img    string `json:"img,omitempty"`
		Tinted bool   `json:"tinted,omitempty"`
		Family string `json:"family,omitempty"`
	}
	legacyWeaponLabel struct {
		En     string `json:"en"`
		Fr     string `json:"fr"`
		Fx     string `json:"fx,omitempty"`
		Key    string `json:"key,omitempty"`
		Role   string `json:"role,omitempty"`
		Tint   string `json:"tint,omitempty"`
		Impact bool   `json:"impact,omitempty"`
		Img    string `json:"img,omitempty"`
		Tinted bool   `json:"tinted,omitempty"`
	}
	legacyVehicleLabel struct {
		Img    string `json:"img,omitempty"`
		Tinted bool   `json:"tinted,omitempty"`
		Kind   string `json:"kind,omitempty"`
		En     string `json:"en,omitempty"`
		Fr     string `json:"fr,omitempty"`
	}
	legacyCalloutZone struct {
		VolumeIndex int            `json:"volume_index"`
		Name        string         `json:"name"`
		EN          string         `json:"en"`
		FR          string         `json:"fr"`
		X           float64        `json:"x"`
		Y           float64        `json:"y"`
		Z           float64        `json:"z"`
		ZBottom     float64        `json:"z_bottom"`
		ZTop        float64        `json:"z_top"`
		Big         bool           `json:"big,omitempty"`
		Polygon     [][2]float64   `json:"polygon,omitempty"`
		Parts       [][][2]float64 `json:"parts,omitempty"`
		Holes       [][][2]float64 `json:"holes,omitempty"`
	}
	legacyObjectiveObjectLife struct {
		Family string                 `json:"family"`
		En     string                 `json:"en"`
		Fr     string                 `json:"fr"`
		T0     int                    `json:"t0"`
		T1     int                    `json:"t1"`
		Pts    []ObjectiveObjectPoint `json:"pts"`
	}
	legacyVehicleWeapon struct {
		Vehicle     string              `json:"vehicle"`
		En          string              `json:"en"`
		Fr          string              `json:"fr"`
		Fire        string              `json:"fire"`
		Fx          string              `json:"fx"`
		Tint        string              `json:"tint"`
		Sound       string              `json:"sound,omitempty"`
		Loop        string              `json:"loop,omitempty"`
		Impact      bool                `json:"impact,omitempty"`
		ImpactSound string              `json:"impactSound,omitempty"`
		Mount       *VehicleWeaponMount `json:"mount,omitempty"`
	}
)

func marshalOrFatal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T : %v", v, err)
	}
	return string(b)
}

func TestLibellesParLangue_MemeFilQueLAncienneForme(t *testing.T) {
	langues := []struct{ nom, en, fr string }{
		{"deux langues", "Rocket Launcher", "Lance-roquettes"},
		{"fr manquant", "Sentinel", ""},
		{"en manquant", "", "Sentinelle"},
		{"aucune", "", ""},
		{"echappement", "A <b> & \"c\"", "é x"},
	}
	for _, c := range langues {
		t.Run(c.nom, func(t *testing.T) {
			tx := i18n.Text{}
			if c.en != "" {
				tx[i18n.LocaleEN] = c.en
			}
			if c.fr != "" {
				tx[i18n.LocaleFR] = c.fr
			}
			poly := [][2]float64{{0, 0}, {5, 0}, {5, 5}}
			mount := &VehicleWeaponMount{Aim: "turret", AX: 0.5, AY: 0.25, Reach: 0.12}
			pts := []ObjectiveObjectPoint{{T: 1, X: 1.5, Y: -2}}
			paires := []struct {
				nom          string
				neuf, ancien any
			}{
				{"Label", Label{Text: tx, Img: "i.png", Tinted: true, Family: "wall"}, legacyLabel{c.en, c.fr, "i.png", true, "wall"}},
				{"Label nu", Label{Text: tx}, legacyLabel{En: c.en, Fr: c.fr}},
				{"WeaponLabel", WeaponLabel{Text: tx, Fx: "plasma", Key: "k", Role: "power", Tint: "t", Impact: true, Img: "i", Tinted: true},
					legacyWeaponLabel{c.en, c.fr, "plasma", "k", "power", "t", true, "i", true}},
				{"WeaponLabel nu", WeaponLabel{Text: tx}, legacyWeaponLabel{En: c.en, Fr: c.fr}},
				{"VehicleLabel", VehicleLabel{Img: "v.png", Tinted: true, Kind: "map_element", Text: tx},
					legacyVehicleLabel{"v.png", true, "map_element", c.en, c.fr}},
				{"VehicleLabel nu", VehicleLabel{Text: tx}, legacyVehicleLabel{En: c.en, Fr: c.fr}},
				{"CalloutZone", CalloutZone{VolumeIndex: 3, Name: "n", Text: tx, X: 1, Y: 2, Z: 3, ZBottom: 0, ZTop: 4, Big: true, Polygon: poly, Parts: [][][2]float64{poly}, Holes: [][][2]float64{poly}},
					legacyCalloutZone{3, "n", c.en, c.fr, 1, 2, 3, 0, 4, true, poly, [][][2]float64{poly}, [][][2]float64{poly}}},
				{"CalloutZone nue", CalloutZone{Text: tx}, legacyCalloutZone{EN: c.en, FR: c.fr}},
				{"ObjectiveObjectLife", ObjectiveObjectLife{Family: "ball", Text: tx, T0: 1, T1: 1, Pts: pts},
					legacyObjectiveObjectLife{"ball", c.en, c.fr, 1, 1, pts}},
				{"VehicleWeapon", VehicleWeapon{Vehicle: "wraith", Text: tx, Fire: "single", Fx: "plasma", Tint: "t", Sound: "s", Loop: "l", Impact: true, ImpactSound: "is", Mount: mount},
					legacyVehicleWeapon{"wraith", c.en, c.fr, "single", "plasma", "t", "s", "l", true, "is", mount}},
				{"VehicleWeapon nue", VehicleWeapon{Text: tx}, legacyVehicleWeapon{En: c.en, Fr: c.fr}},
			}
			for _, p := range paires {
				if got, want := marshalOrFatal(t, p.neuf), marshalOrFatal(t, p.ancien); got != want {
					t.Errorf("%s : fil %s\n   attendu  %s", p.nom, got, want)
				}
			}
		})
	}
}

// TestLibellesParLangue_ArtefactReelRelu : les calques à libellés d'un artefact cuit du parc
// (film d'Oddball, copie réduite) se décodent et se ré-écrivent sans un octet de différence.
func TestLibellesParLangue_ArtefactReelRelu(t *testing.T) {
	raw, err := os.ReadFile("testdata/labels_flat_artefact_1d2a26d8.json")
	if err != nil {
		t.Fatalf("copie réduite de l'artefact illisible : %v", err)
	}
	var calques struct {
		WeaponLabels     map[string]WeaponLabel `json:"weaponLabels"`
		AbilityLabels    map[string]Label       `json:"abilityLabels"`
		GrenadeLabels    []Label                `json:"grenadeLabels"`
		ObjectiveObjects []ObjectiveObjectLife  `json:"objectiveObjects"`
	}
	if err := json.Unmarshal(raw, &calques); err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if len(calques.WeaponLabels) == 0 || len(calques.AbilityLabels) == 0 || len(calques.GrenadeLabels) == 0 || len(calques.ObjectiveObjects) == 0 {
		t.Fatalf("la copie réduite ne porte pas les quatre calques : %+v", calques)
	}
	if got := calques.ObjectiveObjects[0].Text; got[i18n.LocaleEN] != "Oddball" || got[i18n.LocaleFR] != "Crane" {
		t.Errorf("libellé de l'objet = %v", got)
	}
	out, err := json.Marshal(calques)
	if err != nil {
		t.Fatalf("ré-écriture : %v", err)
	}
	if want := bytes.TrimSpace(raw); !bytes.Equal(out, want) {
		t.Errorf("l'artefact relu ne se ré-écrit pas à l'identique :\n got %s\nwant %s", out, want)
	}
}

// TestLibellesParLangue_ChampTextSousLeNomDuContrat : le champ Text de chaque type du document porte
// le nom JSON `i18n.FlatMember` (celui que `openapigen` remplace par une propriété par langue), et le
// type sait se relire (un type qui oublierait ses méthodes JSON écrirait ce nom sur le fil).
func TestLibellesParLangue_ChampTextSousLeNomDuContrat(t *testing.T) {
	for _, v := range []any{Label{}, WeaponLabel{}, VehicleLabel{}, CalloutZone{}, ObjectiveObjectLife{}, VehicleWeapon{}} {
		typ := reflect.TypeOf(v)
		var found int
		for f := range typ.Fields() {
			if f.Type != reflect.TypeFor[i18n.Text]() {
				continue
			}
			found++
			if tag := f.Tag.Get("json"); tag != i18n.FlatMember+",omitempty" {
				t.Errorf("%s.%s : tag json %q, attendu %q", typ.Name(), f.Name, tag, i18n.FlatMember+",omitempty")
			}
		}
		if found != 1 {
			t.Errorf("%s : %d champ(s) Text, attendu 1", typ.Name(), found)
		}
		if _, ok := reflect.PointerTo(typ).MethodByName("UnmarshalJSON"); !ok {
			t.Errorf("%s n'a pas d'UnmarshalJSON : ses langues ne se liraient pas", typ.Name())
		}
	}
}
