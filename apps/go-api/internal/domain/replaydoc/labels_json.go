package replaydoc

import "levelup/go-api/internal/i18n"

// labels_json.go — LA FORME DE FIL DES LIBELLÉS PAR LANGUE.
//
// Jumeau servi de `film/replay/document_labels_json.go` : chaque type ci-dessous porte un
// `i18n.Text` et s'écrit sur le fil avec un membre par langue, AU MÊME NIVEAU que les autres
// champs, à la place où `en` / `fr` étaient déclarés (ADR 0040, `i18n.MarshalFlat`). Le membre
// Go `i18n_text` n'est jamais écrit ; `openapigen` le remplace dans le contrat par une propriété
// par langue de `config/locales.toml`.

type (
	labelWire           Label
	weaponLabelWire     WeaponLabel
	vehicleLabelWire    VehicleLabel
	calloutZoneWire     CalloutZone
	objectiveObjectWire ObjectiveObjectLife
	vehicleWeaponWire   VehicleWeapon
)

// Toutes les langues sont émises (vides comprises) sauf pour VehicleLabel, qui les écrivait en
// dernier avec omitempty.
var (
	labelFlat           = i18n.FlatLayout{}
	weaponLabelFlat     = i18n.FlatLayout{}
	calloutZoneFlat     = i18n.FlatLayout{After: "name"}
	objectiveObjectFlat = i18n.FlatLayout{After: "family"}
	vehicleWeaponFlat   = i18n.FlatLayout{After: "vehicle"}
	vehicleLabelFlat    = i18n.FlatLayout{After: i18n.FlatEnd, OmitEmpty: true}
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
