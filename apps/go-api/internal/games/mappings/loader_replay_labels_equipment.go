package mappings

// loader_replay_labels_equipment.go — LES OBJETS D'EQUIPEMENT POSES, extraits de
// `loader_replay_labels.go` le 2026-08-18.
//
// POURQUOI UN FICHIER A PART : le lot de nommage
// (`.ai/V7.5/replay2d/PLAN_NOMMAGE_EQIP_TRANSLOCATEUR.md`) fait passer cette section de deux
// familles a quinze et lui ajoute un invariant de PROVENANCE ; le fichier hote depassait alors
// le seuil de 500 lignes. La section a de toute facon sa propre raison d'etre — c'est la seule
// du manifeste dont la cle est un GlobalID de tag du jeu et non un libelle.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// equipmentObjectEntry — une ligne de [[equipment_objects]] : l'identifiant du tag, la famille
// de pose qu'il designe, l'identifiant de chaine du jeu et la PROVENANCE du nom.
type equipmentObjectEntry struct {
	ID         string `toml:"id"`
	Family     string `toml:"family"`
	NameID     string `toml:"name_id"`
	Provenance string `toml:"provenance"`
}

// equipmentFamilies — les FAMILLES DE POSE admises. Liste FERMEE : une famille ne s'ajoute
// qu'avec au moins un identifiant que la STRUCTURE DU JEU y rattache (cf. equipmentProvenances).
// `other` est le defaut de tout identifiant hors table — un objet non prouve se publie sans
// nom, jamais sous le nom d'un voisin.
//
// ELARGIE le 2026-08-18 (PLAN_NOMMAGE_EQIP_TRANSLOCATEUR gate 0) : jusque-la le nommage passait
// par une DIAGONALE statistique (identifiant x rang de capacite du poseur >= 85 %) et seules
// deux familles la franchissaient. La chaine `sofd -> sofa -> eqip` du jeu nomme 20 des
// 21 identifiants du corpus sans statistique — la diagonale n'est plus la source, elle est le
// CONTROLE (6 identifiants sur 6 mesurables : accord).
var equipmentFamilies = map[string]bool{
	// Capacites d'armure deployees par un joueur.
	"wall": true, "sensor": true, "threat_seeker": true, "grapple": true,
	"thruster": true, "repulsor": true, "translocator_beacon": true, "repair_field": true,
	// Bonus ramasses au sol.
	"powerup_overshield": true, "powerup_camo": true,
	// Grenades au sol : les quatre entrees de la liste `gggl` du jeu, dans son ordre.
	"grenade_frag": true, "grenade_plasma": true, "grenade_dynamo": true,
	"grenade_spike": true,
	// Pose mesuree, nature non etablie.
	"other": true,
}

// equipmentProvenances — COMMENT le nom a ete obtenu. Liste FERMEE, et l'invariant qui la rend
// utile est plus bas : une famille nommee EXIGE une provenance de structure, et `sofa_anonyme`
// / `aucune` EXIGENT la famille `other`. Sans cela, la table redeviendrait une liste d'opinions.
var equipmentProvenances = map[string]bool{
	// L'identifiant de chaine du `sofa` a ete casse : le nom est celui du jeu.
	"sofa_string_id": true,
	// L'identifiant de chaine du `sofa` n'est pas casse, mais l'`eqip` partage son MODELE
	// (`hlmt`) avec un `eqip` dont le `sofa` est nomme : c'est le meme objet, autre reglage.
	"sofa_modele": true,
	// L'`eqip` est ENGENDRE par un autre `eqip` (dependance `eqip -> eqip`) qui appartient a un
	// `sofa` nomme : c'est une piece deployee par l'equipement (les panneaux d'un mur).
	"sofa_parent": true,
	// L'`eqip` est une entree de la liste des grenades du jeu (`gggl`), dont l'ordre EST le rang
	// de type de grenade.
	"gggl_entree": true,
	// Rattache a un `sofa` dont l'identifiant de chaine resiste au dictionnaire, sans modele
	// commun avec un `sofa` nomme. La pose est mesuree, la nature ne l'est pas.
	"sofa_anonyme": true,
	// Aucun rattachement structurel trouve.
	"aucune": true,
}

// parseEquipmentObjects valide la table des objets d'equipement poses.
//
// TROIS INVARIANTS, tous FATAUX. L'identifiant doit etre un entier 32 bits (c'est un GlobalID
// de tag, lu tel quel dans le film) ; la famille doit appartenir a la liste fermee, sans quoi
// une faute de frappe ferait tomber un mur dans le rendu neutre en silence — indistinguable
// d'un objet volontairement non nomme ; et un identifiant declare deux fois rendrait la table
// dependante de l'ordre de lecture, donc son resultat arbitraire.
func parseEquipmentObjects(path string, rows []equipmentObjectEntry) (map[uint32]string, error) {
	out := make(map[uint32]string, len(rows))
	for _, e := range rows {
		raw := strings.TrimSpace(e.ID)
		if raw == "" {
			return nil, fmt.Errorf("%s: objet d'équipement sans id", path)
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("%s: identifiant d'objet d'équipement %q illisible (attendu : GlobalID hexadécimal 32 bits, ex. \"0x8e2dc574\")",
				path, raw)
		}
		fam := strings.TrimSpace(e.Family)
		if !equipmentFamilies[fam] {
			return nil, fmt.Errorf("%s: famille %q inconnue pour %q (admises : %s)",
				path, fam, raw, clesTriees(equipmentFamilies))
		}
		if err := verifieProvenanceEquipement(path, raw, fam, e); err != nil {
			return nil, err
		}
		if prev, dup := out[uint32(id)]; dup {
			return nil, fmt.Errorf("%s: objet d'équipement %q déclaré deux fois (%q puis %q)",
				path, raw, prev, fam)
		}
		out[uint32(id)] = fam
	}
	return out, nil
}

// verifieProvenanceEquipement fait tenir ensemble la famille et la PROVENANCE du nom.
//
// C'EST L'INVARIANT QUI PORTE LA REGLE DU CHANTIER — « un nom vient de la structure du jeu ou
// n'existe pas ». Sans lui, `provenance` serait un commentaire : on pourrait nommer `wall` un
// identifiant dont rien n'etablit la nature, et la table redeviendrait une liste d'opinions
// impossible a auditer. Les deux sens sont verifies : une famille nommee exige une provenance
// de structure, et une provenance d'echec exige la famille `other`.
func verifieProvenanceEquipement(path, raw, fam string, e equipmentObjectEntry) error {
	prov := strings.TrimSpace(e.Provenance)
	if !equipmentProvenances[prov] {
		return fmt.Errorf("%s: provenance %q inconnue pour %q (admises : %s)",
			path, prov, raw, clesTriees(equipmentProvenances))
	}
	echec := prov == "sofa_anonyme" || prov == "aucune"
	if echec && fam != "other" {
		return fmt.Errorf("%s: %q porte la famille %q avec la provenance %q — une nature non"+
			" etablie se publie `other`", path, raw, fam, prov)
	}
	if !echec && fam == "other" {
		return fmt.Errorf("%s: %q porte la famille `other` avec la provenance %q — une chaine"+
			" structurelle etablie doit nommer une famille", path, raw, prov)
	}
	if nid := strings.TrimSpace(e.NameID); nid != "" {
		if _, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(nid, "0x"), "0X"), 16, 32); err != nil {
			return fmt.Errorf("%s: name_id %q illisible pour %q (attendu : identifiant de chaine"+
				" hexadecimal 32 bits, ex. \"0xedebd7b7\")", path, nid, raw)
		}
	} else if prov == "sofa_string_id" {
		return fmt.Errorf("%s: %q declare la provenance `sofa_string_id` sans name_id —"+
			" l'identifiant de chaine casse EST la piece", path, raw)
	}
	return nil
}

// clesTriees rend les cles d'un ensemble, triees, pour un message d'erreur qui ne se perime
// pas quand la liste fermee s'allonge.
func clesTriees(m map[string]bool) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
