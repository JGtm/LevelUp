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
	"strings"
)

// equipmentObjectEntry — une ligne de [[equipment_objects]] : l'identifiant du tag, la famille
// de pose qu'il designe, l'identifiant de chaine du jeu, la PROVENANCE du nom et la NATURE
// mesuree de l'objet.
type equipmentObjectEntry struct {
	ID         string `toml:"id"`
	Family     string `toml:"family"`
	NameID     string `toml:"name_id"`
	Provenance string `toml:"provenance"`
	Kind       string `toml:"kind"`
}

// equipmentKinds — LA NATURE MESUREE de l'objet. Liste FERMEE, ajoutee le 2026-08-18
// (PLAN_ORIGINE_POSES_ET_FAMILLES, G.3).
//
// POURQUOI ELLE EXISTE A COTE DE LA FAMILLE, alors que chaque pose porte deja son `origin`
// mesure. La famille dit CE QUE l'objet est (un mur, un capteur) ; l'origine dit ce qu'une
// POSE PARTICULIERE est (un lacher, un deploiement). Ni l'une ni l'autre ne dit si un
// identifiant vaut la peine d'etre dessine — et c'est la question que le rendu pose. La
// mesure y repond identifiant par identifiant, et l'ecart est franc : les PANNEAUX du mur
// sont deployes dans 97,7 et 97,9 % de leurs poses (0 lacher sur 48 pour `0x528fce46`),
// quand l'APPAREIL du meme mur ne l'est que dans 13,0 a 29,4 % — il est porte, donc lache.
//
// LA NATURE N'EST PAS UNE OPINION : `verifieProvenanceEquipement` exige que `deployed` soit
// justifie par la provenance `sofa_parent`, la seule que la structure du jeu rattache a une
// piece ENGENDREE par un equipement. Les deux lectures — la chaine des tags et la mesure des
// poses — designent le meme couple, et l'invariant les tient ensemble.
var equipmentKinds = map[string]bool{
	// PORTE par le joueur : ses poses sont majoritairement des LACHERS a la mort. Les
	// appareils d'equipement, les grenades, les bonus, la balise.
	equipKindCarried: true,
	// N'EXISTE QU'UNE FOIS DEPLOYE : la piece qu'un equipement engendre (panneaux du mur).
	equipKindDeployed: true,
}

// Les valeurs sur lesquelles le CODE branche — famille, provenance, nature. Constantes parce
// qu'elles vivent a la fois dans la liste fermee, dans l'invariant et dans ses tests : trois
// copies d'un meme litteral, c'est la limite que le depot fixe (regle des <= 2 copies).
const (
	// equipFamilyOther : la famille des objets dont la nature n'est pas etablie.
	equipFamilyOther = "other"
	// equipProvSofaStringID : l'identifiant de chaine du `sofa` a ete casse.
	equipProvSofaStringID = "sofa_string_id"
	// equipProvSofaAnonyme : rattache a un `sofa` dont le `string_id` resiste au dictionnaire.
	equipProvSofaAnonyme = "sofa_anonyme"
	// equipProvAucune : aucun rattachement structurel trouve.
	equipProvAucune = "aucune"
	// equipProvSofaParent : engendre par l'`eqip` d'un `sofa` nomme — LA provenance qui
	// autorise `kind = deployed`, et la seule.
	equipProvSofaParent = "sofa_parent"
	// equipKindCarried / equipKindDeployed : cf. equipmentKinds.
	equipKindCarried  = "carried"
	equipKindDeployed = "deployed"
)

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
	// ECRAN OCCULTANT, nomme le 2026-08-27 : cf. la provenance `banque_nommee` ci-dessous.
	"shroud_screen": true,
	// Bonus ramasses au sol.
	"powerup_overshield": true, "powerup_camo": true,
	// Grenades au sol : les quatre entrees de la liste `gggl` du jeu, dans son ordre.
	"grenade_frag": true, "grenade_plasma": true, "grenade_dynamo": true,
	"grenade_spike": true,
	// Pose mesuree, nature non etablie.
	equipFamilyOther: true,
}

// equipmentProvenances — COMMENT le nom a ete obtenu. Liste FERMEE, et l'invariant qui la rend
// utile est plus bas : une famille nommee EXIGE une provenance de structure, et `sofa_anonyme`
// / `aucune` EXIGENT la famille `other`. Sans cela, la table redeviendrait une liste d'opinions.
var equipmentProvenances = map[string]bool{
	// L'identifiant de chaine du `sofa` a ete casse : le nom est celui du jeu.
	equipProvSofaStringID: true,
	// L'identifiant de chaine du `sofa` n'est pas casse, mais l'`eqip` partage son MODELE
	// (`hlmt`) avec un `eqip` dont le `sofa` est nomme : c'est le meme objet, autre reglage.
	"sofa_modele": true,
	// L'`eqip` est ENGENDRE par un autre `eqip` (dependance `eqip -> eqip`) qui appartient a un
	// `sofa` nomme : c'est une piece deployee par l'equipement (les panneaux d'un mur).
	equipProvSofaParent: true,
	// L'`eqip` est une entree de la liste des grenades du jeu (`gggl`), dont l'ordre EST le rang
	// de type de grenade.
	"gggl_entree": true,
	// Rattache a un `sofa` dont l'identifiant de chaine resiste au dictionnaire, sans modele
	// commun avec un `sofa` nomme. La pose est mesuree, la nature ne l'est pas.
	equipProvSofaAnonyme: true,
	// La BANQUE SONORE de l objet porte son nom, et ce nom se casse. Voie ouverte le
	// 2026-08-26 : l identifiant Wwise d une banque est le FNV-1 32 bits de son nom de
	// fichier en minuscules, verifie sur 647 banques temoins avant qu un seul nom ne soit
	// casse. Elle nomme ce que les trois voies `sofa` ne nommaient pas — le rang 10 de la
	// palette A tenait sa banque `92c830f5` depuis le 18/08, mais pas son nom ; celui-ci est
	// `sb_007_abl_shroud`, l ecran occultant.
	//
	// C EST UNE VOIE DE STRUCTURE, pas une opinion : elle passe donc l invariant qui exige
	// une provenance de structure pour toute famille nommee.
	"banque_nommee": true,
	// Aucun rattachement structurel trouve.
	equipProvAucune: true,
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
		id, err := tagGlobalID32(path, raw, "identifiant d'objet d'équipement")
		if err != nil {
			return nil, err
		}
		fam := strings.TrimSpace(e.Family)
		if !equipmentFamilies[fam] {
			return nil, fmt.Errorf("%s: famille %q inconnue pour %q (admises : %s)",
				path, fam, raw, clesTriees(equipmentFamilies))
		}
		if err := verifieProvenanceEquipement(path, raw, fam, e); err != nil {
			return nil, err
		}
		if prev, dup := out[id]; dup {
			return nil, fmt.Errorf("%s: objet d'équipement %q déclaré deux fois (%q puis %q)",
				path, raw, prev, fam)
		}
		out[id] = fam
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
	echec := prov == equipProvSofaAnonyme || prov == equipProvAucune
	if echec && fam != equipFamilyOther {
		return fmt.Errorf("%s: %q porte la famille %q avec la provenance %q — une nature non"+
			" etablie se publie `other`", path, raw, fam, prov)
	}
	if !echec && fam == equipFamilyOther {
		return fmt.Errorf("%s: %q porte la famille `other` avec la provenance %q — une chaine"+
			" structurelle etablie doit nommer une famille", path, raw, prov)
	}
	if nid := strings.TrimSpace(e.NameID); nid != "" {
		if _, err := tagGlobalID32(path, nid, fmt.Sprintf("name_id de %q", raw)); err != nil {
			return err
		}
	} else if prov == equipProvSofaStringID {
		return fmt.Errorf("%s: %q declare la provenance `sofa_string_id` sans name_id —"+
			" l'identifiant de chaine casse EST la piece", path, raw)
	}
	return verifieNatureEquipement(path, raw, prov, e)
}

// verifieNatureEquipement fait tenir ensemble la NATURE et la PROVENANCE.
//
// L'INVARIANT, ET CE QU'IL EMPECHE. `kind = deployed` dit « cet objet n'existe qu'une fois
// deploye », et c'est ce qui autorise le rendu a le dessiner. Le declarer sur un identifiant
// que la structure du jeu ne rattache pas comme une piece ENGENDREE (`sofa_parent`) ferait
// dessiner un mur a l'endroit ou un joueur est mort en portant son appareil — exactement le
// defaut que ce lot a mesure. La reciproque est verifiee aussi : un `sofa_parent` est deploye
// par construction, et le declarer `carried` contredirait les deux lectures a la fois.
func verifieNatureEquipement(path, raw, prov string, e equipmentObjectEntry) error {
	kind := strings.TrimSpace(e.Kind)
	if !equipmentKinds[kind] {
		return fmt.Errorf("%s: nature %q inconnue pour %q (admises : %s) — chaque objet declare"+
			" sa nature MESUREE, sans quoi le rendu ne sait pas s'il doit le dessiner",
			path, kind, raw, clesTriees(equipmentKinds))
	}
	if (kind == equipKindDeployed) != (prov == equipProvSofaParent) {
		return fmt.Errorf("%s: %q croise la nature %q avec la provenance %q — `deployed` exige"+
			" `sofa_parent` (une piece ENGENDREE par un equipement) et reciproquement ;"+
			" les deux lectures designent le meme couple, l'invariant les tient ensemble",
			path, raw, kind, prov)
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
