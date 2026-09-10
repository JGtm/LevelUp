package replay

// usage_summary_outcomes.go — LES TROIS ISSUES D'UN OBJET PRIS, au grain du résumé
// (étape E3 du PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// # LA RÈGLE, ET OÙ ELLE EST DÉJÀ ÉCRITE
//
// Tout objet d'équipement ramassé sur la carte finit d'UNE SEULE de trois façons
// (décision P1) : utilisé, lâché en mourant, ou gardé sans l'utiliser. Le web la
// sert déjà sur la vue match depuis l'étape E2 (`equipmentUsageLogic.ts`) ; ce
// fichier en est le JUMEAU pour le résumé persisté, à la formule près :
//
//	kept = max(0, taken - utilisé - lâché)
//
// « Utilisé » a DEUX définitions et le lecteur ne voit pas la différence (P2) : un
// équipement d'ACTIVATION sert quand il est ACTIVÉ (les deux bonus, par le canal
// des épisodes), un DÉPLOYABLE sert quand il est POSÉ (par les poses `deployed`).
// Le clamp à zéro absorbe deux écarts mesurés et attendus : les 2,45 % de fenêtres
// que la mesure E0.4 ne referme pas, et le fait qu'une POSE EST UNE CHARGE, PAS UN
// OBJET (un capteur pris une fois et lancé quatre fois donne 4 poses pour 1 objet —
// piège d'unité n°1 de `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`).
//
// # LA JOINTURE RANG -> FAMILLE SE FAIT SUR LA FAMILLE PUBLIÉE (schéma 51, lot 4.3)
//
// Elle se faisait sur la RACINE DU LIBELLÉ, et c'était un pis-aller : le manifeste
// porte la table exacte (`AbilityPalette.Families`, `replay_labels.toml`) mais elle
// ne traversait pas le document cuit, or [BuildUsageSummary] est une fonction PURE
// DU DOCUMENT DÉJÀ CUIT — c'est ce qui permet au backfill
// (`levelup backfill-usage-summary`) de re-résumer les artefacts sur disque SANS
// re-décoder un seul film. Le document publie désormais `abilityLabels[].family`
// (cf. Label.Family) : la reconstruction par racine a DISPARU d'ici, et avec elle la
// deuxième copie de la table que la règle CLAUDE.md n°6 plafonnait.
//
// CE QUI RESTE ÉCRIT ICI EST UNE DÉCISION PRODUIT, PAS UNE RECONNAISSANCE : la liste
// des familles qui PORTENT une ligne d'issue. Le répulseur (négatif mesuré, décision
// P4), le grappin et le propulseur ont une famille au manifeste et n'ont pas de bilan
// — ce sont des capacités portées, pas des objets qu'on garde ou qu'on gâche.
//
// LES DEUX BONUS ONT REÇU LEUR `family` AU MANIFESTE dans le même lot (rangs 8 et 9,
// `powerup_camo` / `powerup_overshield`) : leur objet de manifeste n'est pas un
// emplacement de capacité, mais la palette peut nommer leur famille comme les autres,
// et sans elle la bascule aurait perdu les deux familles les plus lues du bilan.
//
// # LE VOCABULAIRE DE CLÉ EST CELUI DES POSES, ET C'EST UN CHOIX
//
// Le web nomme un bonus par son ÉPISODE (`camo`) dans `kept` et par sa POSE
// (`powerup_camo`) dans `dropped`, puis ponte les deux (`droppedFamilyOf`). Ici les
// QUATRE ventilations persistées (taken / spent / kept / dropped) parlent le
// vocabulaire des POSES — celui que `DeployedByFamily` et `DroppedByFamily`
// employaient déjà. L'agrégat de session joint donc les quatre sur UNE clé, sans
// pont ni dictionnaire ; le seul endroit qui traduit est ici, au moment de lire le
// côté « utilisé » des deux bonus (leur compte d'épisodes).

import "strconv"

// equipmentOutcomeFamilies — LES FAMILLES QUI PORTENT UNE LIGNE D'ISSUE, dans
// l'ordre où le bilan les cite. Une LISTE et non une map : l'ordre d'itération
// d'une map n'est pas un ordre, et l'agrégat de session publie ces familles dans
// celui-ci.
//
// CE N'EST PLUS UNE TABLE DE RECONNAISSANCE (la famille est publiée par le document
// depuis le schéma 51) : c'est le PÉRIMÈTRE DU BILAN. Une famille du manifeste
// absente d'ici reste hors bilan sans qu'on ait à l'exclure — le répulseur (négatif
// mesuré, décision P4), le grappin et le propulseur portent une famille, ils n'ont
// simplement aucune ligne « pris / utilisé / gardé / lâché ».
var equipmentOutcomeFamilies = []string{
	usageFamilyWall,
	usageFamilySensor,
	"translocator_beacon",
	"shroud_screen",
	"threat_seeker",
	"repair_field",
	usageFamilyPowerupCamo,
	usageFamilyPowerupOvershield,
}

// estFamilleDuBilan dit si cette famille porte une ligne d'issue.
func estFamilleDuBilan(family string) bool {
	for _, f := range equipmentOutcomeFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// EquipmentFamilyPowerupCamo / EquipmentFamilyPowerupOvershield — les deux familles
// dont le côté « utilisé » est un ÉPISODE et non une pose (décision P2). Exportées
// parce que l'agrégat de session doit faire la MÊME bascule sur une ligne de base :
// des littéraux recopiés là-bas feraient une seconde vérité (CLAUDE.md n°6).
const (
	EquipmentFamilyPowerupCamo       = usageFamilyPowerupCamo
	EquipmentFamilyPowerupOvershield = usageFamilyPowerupOvershield
)

// EquipmentOutcomeFamilies rend les familles qui portent une ligne d'issue, dans
// l'ordre de la table ci-dessus. Exportée pour l'agrégat de session, qui doit
// pouvoir citer une famille du bilan même quand AUCUNE prise ne l'a nommée sur le
// scope (un déployable posé depuis l'équipement de réapparition, jamais `taken`).
func EquipmentOutcomeFamilies() []string {
	out := make([]string, len(equipmentOutcomeFamilies))
	copy(out, equipmentOutcomeFamilies)
	return out
}

// equipmentOutcomeFamilyOf rend la famille du bilan que nomme ce rang de palette,
// et `named` dit si le rang porte un libellé DANS CE FILM.
//
// Les deux « non » ne disent pas la même chose et c'est tout l'intérêt du second
// retour : un rang SANS libellé est une MESURE MANQUANTE (palette du film non
// classée, ou rang non établi — 25,21 % des lectures au parc, mesure E0.2), un rang
// libellé mais hors bilan est une EXCLUSION PRODUIT. Le premier se compte à la
// couverture, le second se tait.
func equipmentOutcomeFamilyOf(labels map[string]Label, rank int) (family string, named bool) {
	label, ok := labels[strconv.Itoa(rank)]
	if !ok {
		return "", false
	}
	if !estFamilleDuBilan(label.Family) {
		// Rang NOMMÉ dont la famille n'a pas de ligne d'issue — ou que le manifeste ne
		// classe pas du tout. Les deux se taisent, et aucun des deux n'est une mesure
		// manquante : c'est ce que dit le second retour.
		return "", true
	}
	return label.Family, true
}

// UsageChangeCoverage — ce que le canal des ramassages n'a pas su rattacher. Ces
// comptes ne sont PAS persistés et n'alimentent AUCUNE métrique : ils servent aux
// journaux de production et au `--dry-run` du backfill (décision utilisateur du
// 2026-09-09 : les objets pris sans famille connue ne s'affichent nulle part, leur
// identification passera par un relevé Theater guidé).
type UsageChangeCoverage struct {
	// UnnamedRankTaken : prises dont le rang n'a AUCUN libellé dans ce film.
	UnnamedRankTaken int
	// UnattributedSlot : changements dont le slot n'ouvre aucune ligne de joueur
	// (vie anonyme, bot, slot sans trajectoire publiée).
	UnattributedSlot int
	// SpentUnreliableFrom : consommations dont la chaîne du compteur de rotation
	// est trouée (`Gap > 0`) : leur rang précédent n'est pas une identité fiable,
	// elles ne ventilent donc aucune famille.
	SpentUnreliableFrom int
}

// Total dit s'il y a matière à journaliser — un compte à zéro ne se dit pas.
func (c UsageChangeCoverage) Total() int {
	return c.UnnamedRankTaken + c.UnattributedSlot + c.SpentUnreliableFrom
}

// tallyUsageEquipmentChanges ventile les ramassages et les consommations par
// famille, et compte ce qu'il n'a pas su rattacher.
//
// L'ATTRIBUTION PASSE PAR LA VIE QUI COUVRE L'INSTANT (`at`), comme les tractions
// et les épisodes : un changement d'équipement se produit pendant une vie, jamais
// après elle — contrairement à une pose lâchée à la mort, qui porte `t0 = finVie+1`
// et exige `atOrJustBefore`.
func tallyUsageEquipmentChanges(
	doc *ReplayDocument, players *usageTallies, slotOwner usageOwners,
) UsageChangeCoverage {
	var cov UsageChangeCoverage
	for i := range doc.EquipmentChanges {
		c := &doc.EquipmentChanges[i]
		t := players.of(slotOwner.at(c.Slot, c.T))
		if t == nil {
			cov.UnattributedSlot++
			continue
		}
		switch c.Kind {
		case EquipmentTaken:
			family, named := equipmentOutcomeFamilyOf(doc.AbilityLabels, c.R)
			if !named {
				// Rang muet : COMPTÉ, jamais rangé dans une famille inventée.
				cov.UnnamedRankTaken++
				continue
			}
			if family == "" {
				continue // libellé connu, hors bilan (P4) : exclusion produit
			}
			bumpFamily(&t.TakenByFamily, family)
		case EquipmentSpent:
			// Le rang consommé est sur `From` : `R` vaut NoAbilityRank sur un
			// `spent` (l'emplacement est vide). Une chaîne trouée rend `From`
			// non fiable : le document le dit lui-même par `Gap`.
			if c.Gap > 0 {
				cov.SpentUnreliableFrom++
				continue
			}
			family, named := equipmentOutcomeFamilyOf(doc.AbilityLabels, c.From)
			if !named || family == "" {
				continue
			}
			bumpFamily(&t.SpentByFamily, family)
		}
	}
	return cov
}

// deriveUsageKept pose le troisième segment sur chaque ligne, APRÈS que les prises,
// les poses, les lâchers et les épisodes y soient : `max(0, taken - utilisé -
// lâché)`, famille par famille du bilan.
func deriveUsageKept(players *usageTallies) {
	for _, t := range players.byXUID {
		for _, family := range equipmentOutcomeFamilies {
			taken := t.TakenByFamily[family]
			if taken == 0 {
				continue
			}
			kept := taken - usageUsedOf(t, family) - t.DroppedByFamily[family]
			if kept < 0 {
				kept = 0 // une pose est une CHARGE, pas un objet : jamais de gardé négatif
			}
			if t.KeptByFamily == nil {
				t.KeptByFamily = map[string]int{}
			}
			t.KeptByFamily[family] = kept
		}
	}
}

// usageUsedOf — le côté « utilisé » d'une famille pour CETTE ligne (décision P2) :
// le compte d'ÉPISODES pour les deux bonus, les poses DÉPLOYÉES pour tout le reste.
// C'est le seul endroit du fichier qui traduit entre les deux vocabulaires.
func usageUsedOf(t *UsagePlayerSummary, family string) int {
	switch family {
	case usageFamilyPowerupCamo:
		return t.CamoEpisodes
	case usageFamilyPowerupOvershield:
		return t.OvershieldEpisodes
	default:
		return t.DeployedByFamily[family]
	}
}

// bumpFamily incrémente une ventilation créée à la demande (nil quand aucune).
func bumpFamily(m *map[string]int, family string) {
	if *m == nil {
		*m = map[string]int{}
	}
	(*m)[family]++
}
