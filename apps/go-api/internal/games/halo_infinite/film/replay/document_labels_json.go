package replay

import "levelup/go-api/internal/games/halo_infinite/film/internal/flatjson"

// document_labels_json.go — LA FORME DE FIL DES LIBELLÉS PAR LANGUE.
//
// Chaque type ci-dessous porte un [Text] et s'écrit sur le fil comme avant l'ouverture à N
// langues : un membre par langue, AU MÊME NIVEAU que les autres champs, à la place où `en` / `fr`
// étaient déclarés (artefacts cuits relus sans re-cuisson, ADR 0040 : `flatjson.MarshalFlat` /
// `UnmarshalFlat`, copie du codec `i18n` de LevelUp). Le jumeau servi est
// `domain/replaydoc/labels_json.go` ; les deux formes sont gardées égales par
// `replayview/parity_test.go`, par le golden de forme de ce paquet et, pour les deux codecs, par
// `replaylabels/fil_jumeau_test.go` (mêmes octets dans les cinq dispositions ci-dessous).

type (
	labelWire           Label
	weaponLabelWire     WeaponLabel
	vehicleLabelWire    VehicleLabel
	calloutZoneWire     CalloutZone
	objectiveObjectWire ObjectiveObjectLife
	vehicleWeaponWire   VehicleWeapon
)

// Les langues de Label, WeaponLabel, CalloutZone, ObjectiveObjectLife et VehicleWeapon sont
// toutes émises, vides comprises (c'était `json:"en"` / `json:"fr"` sans omitempty) : toutes celles
// que porte le texte, et le producteur (`replaylabels.TexteVersFilm`) y pose chaque langue de
// l'application.
var (
	labelFlat           = flatjson.FlatLayout{}
	weaponLabelFlat     = flatjson.FlatLayout{}
	calloutZoneFlat     = flatjson.FlatLayout{After: "name"}
	objectiveObjectFlat = flatjson.FlatLayout{After: "family"}
	vehicleWeaponFlat   = flatjson.FlatLayout{After: "vehicle"}
	// VehicleLabel les écrivait en DERNIER avec omitempty : une langue vide n'est pas émise.
	vehicleLabelFlat = flatjson.FlatLayout{After: flatjson.FlatEnd, OmitEmpty: true}
)

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (l Label) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](labelWire(l), labelFlat)
}

// UnmarshalJSON lit les langues à plat.
func (l *Label) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*labelWire)(l))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (w WeaponLabel) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](weaponLabelWire(w), weaponLabelFlat)
}

// UnmarshalJSON lit les langues à plat.
func (w *WeaponLabel) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*weaponLabelWire)(w))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (v VehicleLabel) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](vehicleLabelWire(v), vehicleLabelFlat)
}

// UnmarshalJSON lit les langues à plat.
func (v *VehicleLabel) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*vehicleLabelWire)(v))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (z CalloutZone) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](calloutZoneWire(z), calloutZoneFlat)
}

// UnmarshalJSON lit les langues à plat.
func (z *CalloutZone) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*calloutZoneWire)(z))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (o ObjectiveObjectLife) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](objectiveObjectWire(o), objectiveObjectFlat)
}

// UnmarshalJSON lit les langues à plat.
func (o *ObjectiveObjectLife) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*objectiveObjectWire)(o))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (v VehicleWeapon) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](vehicleWeaponWire(v), vehicleWeaponFlat)
}

// UnmarshalJSON lit les langues à plat.
func (v *VehicleWeapon) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*vehicleWeaponWire)(v))
}
