package replay

import "levelup/go-api/internal/i18n"

// document_labels_json.go — LA FORME DE FIL DES LIBELLÉS PAR LANGUE.
//
// Chaque type ci-dessous porte un `i18n.Text` et s'écrit sur le fil comme avant l'ouverture à N
// langues : un membre par langue, AU MÊME NIVEAU que les autres champs, à la place où `en` / `fr`
// étaient déclarés (artefacts cuits relus sans re-cuisson : `i18n.MarshalFlat` / `UnmarshalFlat`,
// ADR 0040). Le jumeau servi est `domain/replaydoc/labels_json.go` ; les deux formes sont gardées
// égales par `replayview/parity_test.go` et par le golden de forme de ce paquet.

type (
	labelWire           Label
	weaponLabelWire     WeaponLabel
	vehicleLabelWire    VehicleLabel
	calloutZoneWire     CalloutZone
	objectiveObjectWire ObjectiveObjectLife
	vehicleWeaponWire   VehicleWeapon
)

// Les langues de Label, WeaponLabel, CalloutZone, ObjectiveObjectLife et VehicleWeapon sont
// toujours émises (vides comprises) : c'était `json:"en"` / `json:"fr"` sans omitempty.
var (
	labelFlat           = i18n.FlatLayout{}
	weaponLabelFlat     = i18n.FlatLayout{}
	calloutZoneFlat     = i18n.FlatLayout{After: "name"}
	objectiveObjectFlat = i18n.FlatLayout{After: "family"}
	vehicleWeaponFlat   = i18n.FlatLayout{After: "vehicle"}
	// VehicleLabel les écrivait en DERNIER avec omitempty : une langue vide n'est pas émise.
	vehicleLabelFlat = i18n.FlatLayout{After: i18n.FlatEnd, OmitEmpty: true}
)

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (l Label) MarshalJSON() ([]byte, error) { return i18n.MarshalFlat(labelWire(l), labelFlat) }

// UnmarshalJSON lit les langues à plat.
func (l *Label) UnmarshalJSON(b []byte) error { return i18n.UnmarshalFlat(b, (*labelWire)(l)) }

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (w WeaponLabel) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(weaponLabelWire(w), weaponLabelFlat)
}

// UnmarshalJSON lit les langues à plat.
func (w *WeaponLabel) UnmarshalJSON(b []byte) error {
	return i18n.UnmarshalFlat(b, (*weaponLabelWire)(w))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (v VehicleLabel) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(vehicleLabelWire(v), vehicleLabelFlat)
}

// UnmarshalJSON lit les langues à plat.
func (v *VehicleLabel) UnmarshalJSON(b []byte) error {
	return i18n.UnmarshalFlat(b, (*vehicleLabelWire)(v))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (z CalloutZone) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(calloutZoneWire(z), calloutZoneFlat)
}

// UnmarshalJSON lit les langues à plat.
func (z *CalloutZone) UnmarshalJSON(b []byte) error {
	return i18n.UnmarshalFlat(b, (*calloutZoneWire)(z))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (o ObjectiveObjectLife) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(objectiveObjectWire(o), objectiveObjectFlat)
}

// UnmarshalJSON lit les langues à plat.
func (o *ObjectiveObjectLife) UnmarshalJSON(b []byte) error {
	return i18n.UnmarshalFlat(b, (*objectiveObjectWire)(o))
}

// MarshalJSON écrit les langues à plat (cf. l'en-tête du fichier).
func (v VehicleWeapon) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(vehicleWeaponWire(v), vehicleWeaponFlat)
}

// UnmarshalJSON lit les langues à plat.
func (v *VehicleWeapon) UnmarshalJSON(b []byte) error {
	return i18n.UnmarshalFlat(b, (*vehicleWeaponWire)(v))
}
