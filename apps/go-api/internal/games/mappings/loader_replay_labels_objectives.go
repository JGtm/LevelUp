package mappings

// loader_replay_labels_objectives.go — LES OBJETS D'OBJECTIF du rejeu : les entites du monde
// qui FONT le mode de jeu, et que rien d'autre dans le manifeste ne nomme.
//
// POURQUOI CETTE TABLE N'EST PAS `equipment_objects`. Les deux sont keyees par un GlobalID de
// tag du jeu, et c'est leur seul point commun. `equipment_objects` porte l'archetype 37 (les
// poses d'equipement) et son nom vient de la chaine `sofd -> sofa -> eqip` ; celle-ci porte
// l'archetype 42 — le MEME que les armes au sol — et son identifiant a ete etabli par la
// GEOMETRIE (naissance au socle) et l'HORLOGE (coincidence avec l'oracle des evenements
// nommes), pas par une chaine de tags. Les melanger ferait passer un drapeau pour une pose.
//
// LA VALEUR DE LA TABLE EST DOUBLE, et la seconde est la plus importante : elle NOMME
// l'objet dans les deux langues du produit, et elle dit a la chaine des socles ce qu'elle
// doit REFUSER de publier comme arme. Sans elle, un drapeau reste ecarte par accident
// (« identite hors du catalogue d'armes ») — et le jour ou son identifiant entrerait dans le
// catalogue, il deviendrait un socle de fusil a la base de chaque equipe.

import (
	"fmt"
	"strings"
)

// ObjectiveFamilyFlag — LE DRAPEAU de CTF. C'est LA valeur sur laquelle la couche titre
// branche pour projeter la table vers le catalogue de rejeu ; elle est declaree ici, chez le
// proprietaire de la liste fermee, et nulle part ailleurs.
const ObjectiveFamilyFlag = "flag"

// ObjectiveFamilyBall — LE CRANE d'Oddball, admis le 2026-08-27 (phase D4 du plan des objectifs
// vivants). Son identifiant a ete etabli par la MEME recette que le drapeau, sur quatre films :
// naissance a 0,0 m du socle `oddball_spawn` unique de la carte, et coincidence a 3-6 ms d'un
// evenement `th=10` de crane.
const ObjectiveFamilyBall = "ball"

// objectiveFamilies — les familles d'objet d'objectif ADMISES. Liste FERMEE, pour la meme
// raison que les familles de pose : une valeur libre ferait tomber l'objet dans le rendu
// neutre en silence, ce qui est indistinguable d'un objet volontairement non nomme.
//
// CE QUE L'ENTREE `ball` APPORTE, ET CE QU'ELLE N'APPORTE PAS. Elle apporte l'IDENTITE et
// l'EXCLUSION : le crane cesse d'echapper aux socles d'armes par accident (« identifiant hors
// du catalogue d'armes ») pour en etre ecarte parce qu'on sait ce que c'est — c'est exactement
// la raison d'etre de cette table. Elle n'apporte PAS le portage : la mesure D4 du seuil (2) a
// REFUTE l'oracle du score personnel (40,6 a 66,7 % de trous a porteur unique contre un seuil de
// 90 %, temoin hors trou a 66,7 et 71,4 %), et rien ici ne publie qui porte le crane.
var objectiveFamilies = map[string]bool{
	ObjectiveFamilyFlag: true,
	ObjectiveFamilyBall: true,
}

// ObjectiveObject — un objet d'objectif du titre : sa famille et son nom bilingue.
type ObjectiveObject struct {
	// Family appartient a [objectiveFamilies] — c'est ce que l'objet EST.
	Family string
	// Label est son nom dans les deux langues du produit.
	Label BilingualLabel
}

// objectiveObjectEntry — une ligne de [[objective_objects]] du TOML.
type objectiveObjectEntry struct {
	ID     string `toml:"id"`
	Family string `toml:"family"`
	En     string `toml:"en"`
	Fr     string `toml:"fr"`
}

// parseObjectiveObjects valide la table des objets d'objectif.
//
// TROIS INVARIANTS, tous FATAUX, et ce sont ceux de `parseEquipmentObjects` pour les memes
// raisons : l'identifiant doit etre un GlobalID de tag 32 bits (il est lu tel quel dans le
// film) ; la famille doit appartenir a la liste fermee, sans quoi une faute de frappe
// desactiverait le calque EN SILENCE ; et un identifiant declare deux fois rendrait la table
// dependante de l'ordre de lecture. Le nom bilingue est exige par `bilingual` — un objet
// nomme dans une seule langue interdit l'autre.
func parseObjectiveObjects(path string, rows []objectiveObjectEntry) (map[uint32]ObjectiveObject, error) {
	out := make(map[uint32]ObjectiveObject, len(rows))
	for _, e := range rows {
		raw := strings.TrimSpace(e.ID)
		if raw == "" {
			return nil, fmt.Errorf("%s: objet d'objectif sans id", path)
		}
		id, err := tagGlobalID32(path, raw, "identifiant d'objet d'objectif")
		if err != nil {
			return nil, err
		}
		fam := strings.TrimSpace(e.Family)
		if !objectiveFamilies[fam] {
			return nil, fmt.Errorf("%s: famille d'objectif %q inconnue pour %q (admises : %s)",
				path, fam, raw, clesTriees(objectiveFamilies))
		}
		lbl, err := bilingual(path, fmt.Sprintf("objet d'objectif %q", raw),
			bilingualEntry{En: e.En, Fr: e.Fr})
		if err != nil {
			return nil, err
		}
		if prev, dup := out[id]; dup {
			return nil, fmt.Errorf("%s: objet d'objectif %q declare deux fois (%q puis %q)",
				path, raw, prev.Family, fam)
		}
		out[id] = ObjectiveObject{Family: fam, Label: lbl}
	}
	return out, nil
}
