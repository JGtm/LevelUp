package mappings

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/i18n"
)

// WeaponNameSet porte la SOURCE UNIQUE des noms d'armes affiches d'un titre,
// data-driven (config/titles/{slug}/mappings/weapon_names.toml), keyee par
// weapon_key (identifiant canonique stable du registre weapons/weapon_ids), PAS par
// le nom EN brut. La resolution d'un kill fait weapon_id -> weapon_ids -> weapon_key
// -> i18n.Text depuis ce fichier (cf. internal/platform/duckdb/weapon_resolver.go +
// games/weapons.ReconcileNameLabels qui le seede en metadata).
//
// Motivation (V72-06) : avant, 3 sources de noms se marchaient dessus (weapon_labels
// keye par nom EN, name_fr invente du registre, priorite du resolver), keyees par le
// nom EN brut — fragile (mismatch "FRAG GRENADE" vs "Frag Grenade" cote H5). Ce
// fichier unifie tout par weapon_key.
type WeaponNameSet struct {
	titleSlug     string
	schemaVersion int
	names         map[string]i18n.Text // weapon_key -> nom par langue
}

// weaponNamesTOML — projection brute de weapon_names.toml.
type weaponNamesTOML struct {
	Meta metaSection `toml:"meta"`
	// Weapons : weapon_key -> nom par langue (table `{ en = "...", fr = "..." }`). Chaque
	// langue `required` de config/locales.toml est obligatoire ; quand aucun nom officiel
	// n'existe dans une langue, le fichier y met le nom EN (jamais de traduction inventee,
	// decision V72-06).
	Weapons map[string]i18n.Text `toml:"weapons"`
}

// Names retourne une copie de la table weapon_key -> nom par langue. nil-safe (map vide).
func (s *WeaponNameSet) Names() map[string]i18n.Text {
	out := make(map[string]i18n.Text)
	if s == nil {
		return out
	}
	for k, v := range s.names {
		out[k] = v.Clone()
	}
	return out
}

// TitleSlug retourne le slug declare.
func (s *WeaponNameSet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schema declaree.
func (s *WeaponNameSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadWeaponNamesFromFile lit et valide un weapon_names.toml a un chemin donne.
func LoadWeaponNamesFromFile(path string) (*WeaponNameSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadWeaponNamesFromBytes(path, raw)
}

// LoadWeaponNamesFromBytes parse et valide un payload TOML deja en memoire. Validation
// stricte (tout-ou-rien) : title_slug + schema_version obligatoires ; chaque entree a
// une cle non vide et un nom non vide dans chaque langue `required` (regle "EN dans les
// deux" quand aucun nom officiel n'est connu dans la langue).
func LoadWeaponNamesFromBytes(path string, raw []byte) (*WeaponNameSet, error) {
	var doc weaponNamesTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	names := make(map[string]i18n.Text, len(doc.Weapons))
	for rawKey, entry := range doc.Weapons {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, fmt.Errorf("%s: weapon_key vide", path)
		}
		text := entry.Trimmed()
		if err := text.Validate(); err != nil {
			return nil, fmt.Errorf("%s: weapon_key %q : %w (mettre le nom EN si aucun nom officiel)", path, key, err)
		}
		names[key] = text
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: aucune arme dans [weapons]", path)
	}
	return &WeaponNameSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		names:         names,
	}, nil
}
