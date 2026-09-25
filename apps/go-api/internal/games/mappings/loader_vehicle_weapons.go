package mappings

// loader_vehicle_weapons.go — LE REGISTRE DES ARMES DE VEHICULE du titre
// (`config/titles/{slug}/mappings/vehicle_weapons.toml`, schema 69 du document de rejeu, lot M4a
// des retours du rejeu du 2026-09-23).
//
// UNE ENTREE PAR TAG `weap` OBSERVE DANS UN FILM, avec ce que le rejeu en dessine (forme, teinte,
// ancre sur le sprite), ce qu il en joue (son, ou silence DECIDE et motive) et sa PREUVE. Les tags
// observes que rien n identifie encore vivent dans `[[unknown]]`, avec leur raison — jamais
// absents en silence. Le garde-rail qui tient la cle sur le parc est dans le test de ce fichier.
//
// LISTES FERMEES, validees au chargement : une valeur libre ferait tomber le client sur son rendu
// neutre EN SILENCE, ce qui est indistinguable d une arme volontairement non stylee.

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Les deux regimes de tir, publies tels quels (valeurs ANGLAISES comme toute enumeration du
// contrat de rejeu).
const (
	VehicleWeaponFireSingle     = "single"
	VehicleWeaponFireContinuous = "continuous"
)

// Les deux classes de visee d un montage.
const (
	VehicleWeaponAimFixed  = "fixed"
	VehicleWeaponAimTurret = "turret"
)

var (
	vehicleWeaponFires = map[string]bool{VehicleWeaponFireSingle: true, VehicleWeaponFireContinuous: true}
	vehicleWeaponAims  = map[string]bool{VehicleWeaponAimFixed: true, VehicleWeaponAimTurret: true}
)

// vehicleWeaponMountBound : une ancre est une fraction du sprite, centre a 0.
const vehicleWeaponMountBound = 0.5

// VehicleWeaponMount — l ancre d une arme sur le sprite de son vehicule.
type VehicleWeaponMount struct {
	Aim string  `toml:"aim"`
	AX  float64 `toml:"ax"`
	AY  float64 `toml:"ay"`
}

// VehicleWeapon — une entree du registre.
type VehicleWeapon struct {
	Tag      string              `toml:"tag"`
	Vehicle  string              `toml:"vehicle"`
	En       string              `toml:"en"`
	Fr       string              `toml:"fr"`
	Fire     string              `toml:"fire"`
	Fx       string              `toml:"fx"`
	Tint     string              `toml:"tint"`
	Sound    string              `toml:"sound"`
	Loop     string              `toml:"loop"`
	Silence  string              `toml:"silence"`
	Mount    *VehicleWeaponMount `toml:"mount"`
	Proof    string              `toml:"proof"`
	Decision string              `toml:"decision"`
}

// VehicleWeaponUnknown — un tag observe que rien n identifie encore, et pourquoi.
type VehicleWeaponUnknown struct {
	Tag    string `toml:"tag"`
	Reason string `toml:"reason"`
}

type vehicleWeaponsDoc struct {
	Meta struct {
		TitleSlug     string `toml:"title_slug"`
		SchemaVersion int    `toml:"schema_version"`
	} `toml:"meta"`
	Weapons []VehicleWeapon        `toml:"weapons"`
	Unknown []VehicleWeaponUnknown `toml:"unknown"`
}

// VehicleWeaponSet — le registre charge et valide, keye par tag (8 chiffres hex majuscules).
type VehicleWeaponSet struct {
	weapons map[string]VehicleWeapon
	unknown map[string]string
}

// LoadVehicleWeaponsFromFile charge et valide le registre d un titre. Un fichier ABSENT est une
// erreur `os.ErrNotExist` que l appelant traite en degradation (un titre sans registre n a pas
// d arme de vehicule nommee) ; un fichier invalide est une erreur de configuration.
func LoadVehicleWeaponsFromFile(path string) (*VehicleWeaponSet, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin du manifeste du titre, resolu par l appelant
	if err != nil {
		return nil, err
	}
	return LoadVehicleWeaponsFromBytes(path, raw)
}

// LoadVehicleWeaponsFromBytes valide un registre deja lu (`path` ne sert qu aux messages).
func LoadVehicleWeaponsFromBytes(path string, raw []byte) (*VehicleWeaponSet, error) {
	var doc vehicleWeaponsDoc
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	set := &VehicleWeaponSet{weapons: map[string]VehicleWeapon{}, unknown: map[string]string{}}
	for _, w := range doc.Weapons {
		if err := validateVehicleWeapon(w); err != nil {
			return nil, fmt.Errorf("%s: arme %q : %w", path, w.Tag, err)
		}
		if set.has(w.Tag) {
			return nil, fmt.Errorf("%s: tag %q declare deux fois", path, w.Tag)
		}
		set.weapons[w.Tag] = w
	}
	for _, u := range doc.Unknown {
		if !isVehicleWeaponTag(u.Tag) || strings.TrimSpace(u.Reason) == "" {
			return nil, fmt.Errorf("%s: [[unknown]] %q : tag de 8 chiffres hex majuscules et raison requis", path, u.Tag)
		}
		if set.has(u.Tag) {
			return nil, fmt.Errorf("%s: tag %q declare deux fois", path, u.Tag)
		}
		set.unknown[u.Tag] = u.Reason
	}
	return set, nil
}

// validateVehicleWeapon tient les listes fermees et les champs obligatoires d une entree.
func validateVehicleWeapon(w VehicleWeapon) error {
	switch {
	case !isVehicleWeaponTag(w.Tag):
		return fmt.Errorf("tag : 8 chiffres hexadecimaux MAJUSCULES attendus")
	case strings.TrimSpace(w.Vehicle) == "":
		return fmt.Errorf("vehicle vide")
	case strings.TrimSpace(w.En) == "" || strings.TrimSpace(w.Fr) == "":
		return fmt.Errorf("libelle en/fr incomplet")
	case !vehicleWeaponFires[w.Fire]:
		return fmt.Errorf("fire %q inconnu (admis : single, continuous)", w.Fire)
	case !shotEffectFamilies[w.Fx]:
		return fmt.Errorf("fx %q inconnu (liste de [shot_effects])", w.Fx)
	case !shotTintKinds[w.Tint]:
		return fmt.Errorf("tint %q inconnue (liste de [shot_tints])", w.Tint)
	case (w.Sound == "") == (strings.TrimSpace(w.Silence) == ""):
		return fmt.Errorf("exactement un de sound / silence (un silence est DECIDE et motive)")
	case strings.TrimSpace(w.Proof) == "":
		return fmt.Errorf("proof vide : une entree sans preuve n entre pas au registre")
	case w.Loop != "" && (w.Fire != VehicleWeaponFireContinuous || w.Sound == ""):
		return fmt.Errorf("loop : le son TENU d une rafale n existe que pour une arme continuous qui sonne")
	}
	return validateVehicleWeaponMount(w.Mount)
}

func validateVehicleWeaponMount(m *VehicleWeaponMount) error {
	if m == nil {
		return nil
	}
	if !vehicleWeaponAims[m.Aim] {
		return fmt.Errorf("mount.aim %q inconnu (admis : fixed, turret)", m.Aim)
	}
	if m.AX < -vehicleWeaponMountBound || m.AX > vehicleWeaponMountBound ||
		m.AY < -vehicleWeaponMountBound || m.AY > vehicleWeaponMountBound {
		return fmt.Errorf("mount (%v, %v) hors du sprite [-0,5 ; 0,5]", m.AX, m.AY)
	}
	return nil
}

// isVehicleWeaponTag : 8 chiffres hexadecimaux MAJUSCULES — l ecriture de `Shot.w`.
func isVehicleWeaponTag(tag string) bool {
	if len(tag) != 8 {
		return false
	}
	for _, c := range tag {
		if (c < '0' || c > '9') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func (s *VehicleWeaponSet) has(tag string) bool {
	_, w := s.weapons[tag]
	_, u := s.unknown[tag]
	return w || u
}

// Weapon rend l entree d un tag, ou faux.
func (s *VehicleWeaponSet) Weapon(tag string) (VehicleWeapon, bool) {
	if s == nil {
		return VehicleWeapon{}, false
	}
	w, ok := s.weapons[tag]
	return w, ok
}

// Tags rend les tags des entrees, tries.
func (s *VehicleWeaponSet) Tags() []string {
	if s == nil {
		return nil
	}
	return sortedKeys(s.weapons)
}

// Unknown rend les tags observes non identifies et leur raison (copie).
func (s *VehicleWeaponSet) Unknown() map[string]string {
	if s == nil {
		return nil
	}
	out := make(map[string]string, len(s.unknown))
	for k, v := range s.unknown {
		out[k] = v
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
