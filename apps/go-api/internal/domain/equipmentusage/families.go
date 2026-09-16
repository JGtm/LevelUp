// Package equipmentusage porte LE VOCABULAIRE DES FAMILLES D'ÉQUIPEMENT du bilan
// « pris / utilisé / gardé / lâché » : le périmètre du bilan, et le canal sur lequel
// se lit le côté « utilisé » de chaque famille.
//
// POURQUOI CE PAQUET EXISTE (item 2.5.f du PLAN_DECODEUR_FILM_2026-09-13, décision
// V15 (5) ; ADR 0034 D-1). Ce vocabulaire vivait dans le décodeur de film
// (`internal/games/halo_infinite/film/replay`), et l'agrégat de session
// (`internal/analysis/sessionusage`) l'y lisait : c'était le SEUL franchissement de
// PRODUCTION de l'allowlist du garde-rail D9 — un algorithme title-agnostic qui
// importait un paquet de titre (ADR 0012 / ADR 0025). Le sens des dépendances est
// maintenant celui de l'architecture : `replay` et `sessionusage` lisent TOUS LES DEUX
// ce paquet, aucun des deux ne lit l'autre.
//
// POURQUOI SOUS `domain/` ET NON SOUS `games/canonical/`. `games/canonical` est la
// lingua franca INTER-TITRES : y poser ces clés affirmerait que « wall », « sensor »
// ou « powerup_camo » ont un sens dans tout titre, ce qui est faux (Halo 5 n'a aucun
// équipement de ce genre). Ces clés sont le vocabulaire d'un DOCUMENT : elles
// voyagent telles quelles jusqu'au contrat public — `domain.SessionUsageMetric.Key`
// vaut `equipment_<famille>`, `domain.EquipmentUsageFamilyLine.Family` porte la
// famille brute. C'est exactement le chemin qu'a pris `domain/replaydoc` pour le
// document de rejeu : un voisin de `domain/`, feuille, sans aucun import du dépôt.
//
// CE PAQUET NE LIT PAS LE MANIFESTE, et c'est voulu. La source de ces tables est
// `config/titles/halo_infinite/mappings/replay_labels.toml` ; leur RECOLLEMENT au
// manifeste reste un garde-rail du côté du titre
// (`film/replay/usage_summary_families_guard_test.go`), là où le manifeste est. Une
// feuille de `domain/` qui irait lire le manifeste d'un titre rouvrirait la frontière
// que ce déplacement vient de fermer.
package equipmentusage

import "sort"

// Clés de famille du manifeste portées ici parce que les deux tables ci-dessous les
// nomment : le périmètre du bilan et la table des familles à pièce engendrée. Les
// autres clés du manifeste (grenades, capacités portées, répulseur) n'ont pas de
// ligne d'issue et restent du côté du titre, avec les tables qui les classent.
const (
	usageFamilyPowerupCamo       = "powerup_camo"
	usageFamilyPowerupOvershield = "powerup_overshield"
	usageFamilyWall              = "wall"
	usageFamilySensor            = "sensor"
)

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

// EquipmentFamilyPowerupCamo / EquipmentFamilyPowerupOvershield — les deux familles
// dont le côté « utilisé » est un ÉPISODE et non une pose (décision P2). Exportées
// parce que l'agrégat de session doit faire la MÊME bascule sur une ligne de base :
// des littéraux recopiés là-bas feraient une seconde vérité (CLAUDE.md n°6).
const (
	EquipmentFamilyPowerupCamo       = usageFamilyPowerupCamo
	EquipmentFamilyPowerupOvershield = usageFamilyPowerupOvershield
)

// EquipmentFamilyWall / EquipmentFamilySensor — les deux autres clés du bilan que le
// décodeur nomme encore chez lui : le MUR classe ses poses (`usageDeployedCounts`,
// `usageWallPanelIDs`, et la clé publique `replay.UsageFamilyWallKey` que lit la page
// Escouade), le CAPTEUR sert de témoin aux tests du résumé. Exportées pour la même
// raison que les deux ci-dessus, et pas une de plus : deux orthographes de la même
// famille finiraient par diverger, et le compilateur ne dirait rien.
const (
	EquipmentFamilyWall   = usageFamilyWall
	EquipmentFamilySensor = usageFamilySensor
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

// usageFamiliesWithSpawnedPiece — LES FAMILLES QUI ENGENDRENT UNE PIÈCE DISTINCTE,
// et donc les seules dont le canal des POSES voit le déploiement.
//
// Cette liste est la transcription d'une DONNÉE ÉCRITE : les familles dont le
// manifeste (`config/titles/halo_infinite/mappings/replay_labels.toml`) porte au
// moins un objet `kind = "deployed"` — la nature « n'existe qu'une fois déployé »,
// que le valideur n'autorise qu'avec la provenance `sofa_parent` (l'`eqip` engendré
// par un autre équipement). Le manifeste n'en désigne aujourd'hui que DEUX, les deux
// panneaux du mur, donc UNE famille. Le résumé ne peut pas lire le manifeste
// lui-même — `replay.BuildUsageSummary` est une fonction PURE DU DOCUMENT CUIT,
// c'est ce qui permet au backfill de re-projeter sans re-décoder un film — d'où la
// transcription ; le garde-rail usage_summary_families_guard_test.go la RECOLLE au
// manifeste à chaque test, une famille ajoutée là-bas échoue ici.
//
// CE QUE LA FRONTIÈRE DÉCIDE (rapport E0 du 2026-09-10, question 5). Une pose
// `origin: deployed` sur un objet PORTÉ ne mesure pas un déploiement : elle mesure un
// LÂCHER VOLONTAIRE à mi-vie (l'objet qui tombe parce que son porteur en ramasse un
// autre). Mesure décisive : sur 202 consommations de charge annoncées par les films,
// ZÉRO n'est couverte par une pose de la même famille du même joueur à moins de 2 s —
// contre 84 % pour le mur, dont le `spent` tombe sur la pose de PANNEAU (149 sur 242)
// et JAMAIS sur la création de l'appareil porté (0 sur 31).
var usageFamiliesWithSpawnedPiece = map[string]bool{
	usageFamilyWall: true,
}

// usageFamilySpawnsPiece dit si le canal des POSES voit le déploiement de cette
// famille — donc si son côté « utilisé » se lit sur `deployed` (mur) ou sur les
// CONSOMMATIONS (tout le reste : capteur, traqueur, écran, champ, balise).
func usageFamilySpawnsPiece(family string) bool {
	return usageFamiliesWithSpawnedPiece[family]
}

// UsageFamilySpawnsPiece — LA MÊME QUESTION, POUR L'AGRÉGAT DE SESSION.
//
// Exportée le 2026-09-10 (correction C1 de la revue de la vague 5) : l'agrégat de
// session décide du même côté « utilisé » sur une ligne de BASE et non sur une
// ligne de projection (`sessionusage.equipmentUsedOf`). Il ne peut donc pas
// appeler `replay.usageUsedOf`, mais il ne doit pas non plus RECOPIER la liste des
// familles à pièce engendrée : cette liste est recollée au manifeste par un
// garde-rail (usage_summary_families_guard_test.go) et une deuxième écriture
// re-divergerait au premier objet `kind = "deployed"` ajouté là-bas — c'est
// exactement ce qui s'est produit avec le passage de `us5` à `us6`, où seul le
// résumé avait suivi. Garde-rail du côté appelant :
// sessionusage/usage_outcomes_guard_test.go.
func UsageFamilySpawnsPiece(family string) bool {
	return usageFamilySpawnsPiece(family)
}

// UsageFamiliesSpawningPiece rend la table ci-dessus ÉNUMÉRÉE, triée pour que la
// sortie d'un garde-rail soit stable.
//
// Exportée pour le recollement au manifeste, et pour lui seul : le garde-rail du
// titre parcourt la table DANS LES DEUX SENS — « toute famille `kind = deployed` du
// manifeste est ici » se répond avec le prédicat, mais « toute famille d'ici est
// encore `kind = deployed` au manifeste » exige l'énumération. La déduire du
// périmètre du bilan filtré par le prédicat affaiblirait le garde-rail : une entrée
// hors bilan y échapperait aux deux contrôles.
func UsageFamiliesSpawningPiece() []string {
	out := make([]string, 0, len(usageFamiliesWithSpawnedPiece))
	for family := range usageFamiliesWithSpawnedPiece {
		out = append(out, family)
	}
	sort.Strings(out)
	return out
}
