package mappings

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// loader_replay_labels.go — libelles PROPRES AU REJEU 2D d'un titre
// (config/titles/{slug}/mappings/replay_labels.toml).
//
// Trois catalogues, trois raisons d'etre ici plutot qu'en Go :
//   - les RANGS DE GRENADE etaient nommes a DEUX endroits et differemment (« Dynamo »
//     contre « Shock » pour le meme rang) ;
//   - les CAPACITES etaient en FRANCAIS DANS DU GO, ce qui interdit l'anglais autant
//     qu'un second titre ;
//   - l'EFFET DE TIR etait un catalogue Halo code en dur cote web (22 noms d'armes).
//
// Ce loader ne resout AUCUN nom d'arme : ceux-la vivent dans weapon_names.toml, keyes
// par weapon_key. Ici on ne porte que l'effet, qui n'appartient a aucune langue.

// BilingualLabel — un libelle affichable dans les deux langues du produit. en ET fr sont
// obligatoires ; quand aucun FR officiel n'est connu, le fichier met le EN dans les deux
// (jamais de FR invente — meme regle que weapon_names.toml).
type BilingualLabel struct {
	En string
	Fr string
}

// ReplayLabelSet porte les libelles de rejeu d'un titre.
type ReplayLabelSet struct {
	titleSlug     string
	schemaVersion int
	grenades      []BilingualLabel // index = rang lu dans le film
	abilities     map[int]BilingualLabel
	shotEffects   map[string]string // weapon_key -> famille de rendu
}

// replayLabelsTOML — projection brute du fichier.
type replayLabelsTOML struct {
	Meta        metaSection               `toml:"meta"`
	Grenades    []bilingualEntry          `toml:"grenades"`
	Abilities   map[string]bilingualEntry `toml:"abilities"`
	ShotEffects map[string]string         `toml:"shot_effects"`
}

type bilingualEntry struct {
	En string `toml:"en"`
	Fr string `toml:"fr"`
}

// shotEffectFamilies — les familles de rendu ADMISES. La liste est fermee a dessein :
// une valeur libre ferait tomber l'arme sur le rendu neutre en silence, ce qui est
// indistinguable d'une arme volontairement non cataloguee.
var shotEffectFamilies = map[string]bool{
	"ballistic": true, "plasma": true, "light": true, "shock": true,
	"explosive": true, "melee": true, "needles": true,
}

// GrenadeRanks retourne les libelles de grenade DANS L'ORDRE DES RANGS (copie).
func (s *ReplayLabelSet) GrenadeRanks() []BilingualLabel {
	if s == nil {
		return nil
	}
	out := make([]BilingualLabel, len(s.grenades))
	copy(out, s.grenades)
	return out
}

// Abilities retourne la table index -> libelle (copie). nil-safe (map vide).
func (s *ReplayLabelSet) Abilities() map[int]BilingualLabel {
	out := make(map[int]BilingualLabel)
	if s == nil {
		return out
	}
	for k, v := range s.abilities {
		out[k] = v
	}
	return out
}

// ShotEffects retourne la table weapon_key -> famille de rendu (copie).
func (s *ReplayLabelSet) ShotEffects() map[string]string {
	out := make(map[string]string)
	if s == nil {
		return out
	}
	for k, v := range s.shotEffects {
		out[k] = v
	}
	return out
}

// TitleSlug retourne le slug declare.
func (s *ReplayLabelSet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schema declaree.
func (s *ReplayLabelSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadReplayLabelsFromFile lit et valide un replay_labels.toml a un chemin donne.
func LoadReplayLabelsFromFile(path string) (*ReplayLabelSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadReplayLabelsFromBytes(path, raw)
}

// LoadReplayLabelsFromBytes parse et valide un payload TOML deja en memoire.
//
// Validation STRICTE (tout-ou-rien) : un titre dont le fichier est incoherent ne doit pas
// produire un rejeu a moitie nomme — un libelle manquant se lit comme une donnee absente,
// alors que c'est une erreur de configuration.
func LoadReplayLabelsFromBytes(path string, raw []byte) (*ReplayLabelSet, error) {
	var doc replayLabelsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	grenades, err := parseGrenadeRanks(path, doc.Grenades)
	if err != nil {
		return nil, err
	}
	abilities, err := parseAbilities(path, doc.Abilities)
	if err != nil {
		return nil, err
	}
	effects, err := parseShotEffects(path, doc.ShotEffects)
	if err != nil {
		return nil, err
	}
	return &ReplayLabelSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		grenades:      grenades,
		abilities:     abilities,
		shotEffects:   effects,
	}, nil
}

// parseGrenadeRanks valide les rangs : l'ORDRE est la donnee, un trou la detruirait.
func parseGrenadeRanks(path string, rows []bilingualEntry) ([]BilingualLabel, error) {
	out := make([]BilingualLabel, 0, len(rows))
	for i, e := range rows {
		lbl, err := bilingual(path, fmt.Sprintf("grenade rang %d", i), e)
		if err != nil {
			return nil, err
		}
		out = append(out, lbl)
	}
	return out, nil
}

// parseAbilities valide la table des capacites. Les cles sont des index NUMERIQUES lus
// dans le film : une cle non numerique ne designerait aucune capacite.
func parseAbilities(path string, rows map[string]bilingualEntry) (map[int]BilingualLabel, error) {
	out := make(map[int]BilingualLabel, len(rows))
	for rawKey, e := range rows {
		idx, err := strconv.Atoi(strings.TrimSpace(rawKey))
		if err != nil {
			return nil, fmt.Errorf("%s: index de capacité %q non numérique", path, rawKey)
		}
		lbl, err := bilingual(path, fmt.Sprintf("capacité %d", idx), e)
		if err != nil {
			return nil, err
		}
		out[idx] = lbl
	}
	return out, nil
}

// parseShotEffects valide les familles de rendu contre la liste fermee.
func parseShotEffects(path string, rows map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(rows))
	for rawKey, rawFam := range rows {
		key := strings.TrimSpace(rawKey)
		fam := strings.TrimSpace(rawFam)
		if key == "" {
			return nil, fmt.Errorf("%s: weapon_key vide dans [shot_effects]", path)
		}
		if !shotEffectFamilies[fam] {
			return nil, fmt.Errorf("%s: effet %q inconnu pour %q (admis : ballistic, plasma, light, shock, explosive, melee, needles)",
				path, fam, key)
		}
		out[key] = fam
	}
	return out, nil
}

// bilingual valide qu'un libelle porte bien ses deux langues.
func bilingual(path, what string, e bilingualEntry) (BilingualLabel, error) {
	en := strings.TrimSpace(e.En)
	fr := strings.TrimSpace(e.Fr)
	if en == "" {
		return BilingualLabel{}, fmt.Errorf("%s: %s sans en (nom EN obligatoire)", path, what)
	}
	if fr == "" {
		return BilingualLabel{}, fmt.Errorf("%s: %s sans fr (mettre le EN si aucun FR officiel)", path, what)
	}
	return BilingualLabel{En: en, Fr: fr}, nil
}
