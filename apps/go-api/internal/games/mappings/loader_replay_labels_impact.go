package mappings

// loader_replay_labels_impact.go — LA TABLE `[shot_impact]` de replay_labels.toml : les armes A PIED
// dont le projectile EXPLOSE a l'impact.
//
// POURQUOI UNE TABLE A PART. Ni la forme de rendu (`[shot_effects]`) ni la teinte ne decident d'une
// explosion : le mortier du Wraith est `plasma` comme le pistolet plasma, qui n'explose pas. La
// decision est un fait de l'arme, declare explicitement par le titre. Les armes de VEHICULE portent
// le meme fait dans leur propre registre (`vehicle_weapons.toml`, champ `impact`).

import (
	"fmt"
	"strings"
)

// parseShotImpacts valide `[shot_impact]` : chaque weapon_key declare doit avoir une forme de
// rendu dans `[shot_effects]` — une cle qui n'y figure pas est une faute de frappe, qui laisserait
// l'arme sans explosion EN SILENCE. Seules les entrees `true` entrent dans la table rendue.
func parseShotImpacts(path string, rows map[string]bool, effects map[string]string) (map[string]bool, error) {
	out := make(map[string]bool, len(rows))
	for rawKey, explodes := range rows {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, fmt.Errorf("%s: weapon_key vide dans [shot_impact]", path)
		}
		if _, ok := effects[key]; !ok {
			return nil, fmt.Errorf("%s: [shot_impact] déclare %q, absent de [shot_effects]", path, key)
		}
		if explodes {
			out[key] = true
		}
	}
	return out, nil
}

// ShotImpacts retourne l'ensemble des weapon_key dont le projectile explose a l'impact (copie).
func (s *ReplayLabelSet) ShotImpacts() map[string]bool {
	out := make(map[string]bool)
	if s == nil {
		return out
	}
	for k, v := range s.shotImpacts {
		out[k] = v
	}
	return out
}
