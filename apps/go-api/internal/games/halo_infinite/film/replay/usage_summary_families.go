package replay

// usage_summary_families.go — LA CLASSIFICATION DES FAMILLES D'ÉQUIPEMENT pour le
// résumé d'usage (usage_summary.go), et elle seule.
//
// # POURQUOI UNE TABLE ÉCRITE, ET PAS UN TEST DE PRÉFIXE
//
// Le vocabulaire des familles vient du manifeste du titre
// (`[[equipment_objects]].family`, liste FERMÉE validée par
// games/mappings/loader_replay_labels_equipment.go). Le client web classe déjà ces
// familles (PLACEMENT_RENDER, PLACEMENT_DROPPED_FAMILIES) par des tables écrites,
// jamais par un préfixe — « reconnaître ça commence par powerup_ aurait marché
// aujourd'hui et menti demain » (weaponPadFamilies.ts). Même règle ici : les
// familles à traitement SPÉCIAL (grenade, capacité portée, bonus) sont classées
// NOMMÉMENT, et tout le reste du manifeste est déployable PAR DÉFAUT. Le garde-rail
// usage_summary_families_guard_test.go vérifie la COHÉRENCE de ces listes avec le
// manifeste (une famille listée ici qui n'y existe plus échoue), PAS l'exhaustivité :
// une famille NOUVELLE du manifeste tombe dans « déployable » sans décision — si
// c'est une capacité portée, l'ajouter à usageCarriedCapacityFamilies fait partie
// de son intégration (revue adversariale 2026-09-04).
//
// # LA FRONTIÈRE SOCLE D'ARME / SOCLE DE BONUS N'EST PAS ICI
//
// Elle est [PadWeaponFamilyKey] (pad_pickup_dating.go), l'UNIQUE écriture du test
// « cette valeur est-elle une famille d'arme » du dépôt. Les tables ci-dessous ne
// servent que les POSES d'équipement (`equipmentPlacements`), jamais les socles.

import "levelup/go-api/internal/domain/equipmentusage"

// Clés de famille du manifeste dupliquées 4 fois ou plus dans le paquet (corpus de
// test compris) : golangci-lint (goconst, min-occurrences: 4) exige une constante
// nommée plutôt que le littéral répété. Les autres membres des mêmes maps
// (grenade_plasma, grenade_dynamo, grenade_spike, repulsor) restent en littéral :
// sous le seuil, aucune obligation.
//
// LES QUATRE CLÉS DU BILAN VIENNENT DE LEUR MAISON (item 2.5.f, 2026-09-17) : mur,
// capteur et les deux bonus portent une ligne d'issue, et ce périmètre a quitté le
// paquet pour `internal/domain/equipmentusage` — l'agrégat de session le lit sans
// importer un titre (D9). Elles sont RELUES ici plutôt que réécrites : deux
// orthographes de la même famille finiraient par diverger.
const (
	usageFamilyGrenadeFrag       = "grenade_frag"
	usageFamilyGrapple           = "grapple"
	usageFamilyThruster          = "thruster"
	usageFamilyPowerupCamo       = equipmentusage.EquipmentFamilyPowerupCamo
	usageFamilyPowerupOvershield = equipmentusage.EquipmentFamilyPowerupOvershield
	usageFamilyWall              = equipmentusage.EquipmentFamilyWall
	usageFamilySensor            = equipmentusage.EquipmentFamilySensor
	// Ajoutée le 2026-09-09 (étape E3) : la table de reconnaissance des issues et
	// ses tests font franchir le seuil de goconst à cette clé à son tour.
	usageFamilyRepulsor = "repulsor"
)

// UsageFamilyWallKey — LA clé de famille du mur de protection, exportée pour les
// lecteurs de `deployed_json` hors de ce paquet (bloc « formes retenues » de la page
// Escouade, 2026-09-13 : le mur y est l'un des cinq gestes d'équipement mesurés).
// Exportée plutôt que recopiée : deux orthographes de la même famille finiraient par
// diverger, et le compilateur ne dirait rien.
const UsageFamilyWallKey = usageFamilyWall

// usageGrenadeFamilies — les familles de GRENADE du manifeste (liste `gggl` du jeu).
// Un lâcher de grenade à la mort n'est PAS un « objet lâché au sol » au sens du
// résumé (décision utilisateur du 2026-09-04 : les grenades ne sont pas des
// équipements), et un « déploiement » de grenade est un LANCER, déjà compté par
// `grenades_thrown` — le compter deux fois ferait deux colonnes d'un même geste.
var usageGrenadeFamilies = map[string]bool{
	usageFamilyGrenadeFrag: true, "grenade_plasma": true, "grenade_dynamo": true,
	"grenade_spike": true,
}

// usageCarriedCapacityFamilies — les capacités qui agissent SUR LEUR PORTEUR :
// ce que le film publie sous ces familles est l'APPAREIL (porté, donc lâché à la
// mort), jamais un objet posé sur le terrain. Leur déploiement ne se compte pas
// (le grappin a son propre canal de tractions ; le propulseur son canal
// d'impulsions ; le répulseur n'a aucun canal mesuré) — même décision que
// PLACEMENT_RENDER côté web (valeurs `null` explicites).
var usageCarriedCapacityFamilies = map[string]bool{
	usageFamilyGrapple: true, usageFamilyThruster: true, usageFamilyRepulsor: true,
}

// usagePowerupFamilies — les BONUS ramassés au sol. Un power-up n'est jamais un
// objet « déployé » (aucun membre mesuré au corpus — cf. la note en fin de
// PLACEMENT_RENDER côté web) ; son lâcher à la mort, lui, compte dans les objets
// lâchés (un surbouclier au sol change l'échange suivant).
var usagePowerupFamilies = map[string]bool{
	usageFamilyPowerupCamo: true, usageFamilyPowerupOvershield: true,
}

// usageWallPanelIDs — les identifiants `eqip` des PANNEAUX du mur, les seuls sur
// lesquels un déploiement de mur se compte. Un mur déployé publie DEUX poses
// `deployed` (l'appareil qui vole ET ses panneaux) : compter les deux ferait deux
// murs pour un seul geste. La mesure désigne les panneaux sans ambiguïté (97,7 %
// et 97,9 % de déploiements contre 13,0 % et 29,4 % pour les appareils — manifeste
// replay_labels.toml, `kind = "deployed"`). Même écriture que EquipmentPlacement.ID
// (`0x` + huit hexa minuscules) ; même table que WALL_PANEL_IDS côté web
// (placementWall.ts), deuxième et dernière copie tolérée (règle des <= 2).
var usageWallPanelIDs = map[string]bool{
	"0x528fce46": true, "0x686b40c9": true,
}

// equipmentIsSpawnedPiece dit si CET objet est une PIÈCE ENGENDRÉE par un autre
// équipement — aujourd'hui les deux panneaux du mur, et eux seuls.
//
// POURQUOI UNE PIÈCE ENGENDRÉE NE PASSE PAS PAR LES LECTURES DE L'APPAREIL PORTÉ
// (item H.2 du 2026-09-13, découverte D-F1 ; ordre de cascade fixé au lot 1.9.1).
// « Mort du porteur » et « prise du porteur » répondent à la question « cette création
// tombe-t-elle de quelqu'un ? ». Elle a un sens sur un objet PORTÉ, qui peut être posé de
// son vivant ou lâché à sa mort. Elle n'en a AUCUN sur une pièce engendrée : le manifeste
// la déclare `kind = "deployed"`, c'est-à-dire qu'elle N'EXISTE QUE DÉPLOYÉE — elle n'est
// jamais dans un inventaire, donc jamais lâchée à la mort, et l'absence de poseur mesuré
// ne dit rien de son origine. La classer sur le temps produisait le défaut SYMÉTRIQUE de
// celui que F.1 a corrigé : 7 panneaux du parc sur 216 sortaient en `dropped` (3) ou
// `unknown` (4) — un panneau né à l'instant où son poseur meurt (le mur déployé au dernier
// souffle), ou dont le poseur n'a pas été mesuré.
//
// CE PRÉDICAT N'EST PLUS LA PREMIÈRE RÈGLE, ET C'EST TOUT CE QUE LE LOT 1.9.1 LUI A FAIT :
// la LECTURE du film passe devant (l'événement 103 désigne 115 des 124 poses de panneau du
// corpus mesuré), et ce prédicat devient le REPLI nommé `repli_piece_engendree_sans_evenement`
// pour les 9 autres — toutes sur les deux films de build les plus anciens, dont la liste
// d'événements ne se lit pas. Cf. `equipment_origin.go`.
//
// LA MESURE QUI FERME LA QUESTION (rapport F.0 du 2026-09-13, §2.3) : l'événement 103
// `EquipmentSpawnedObject` — le fait « une pièce a été engendrée » — désigne 216 des 216
// poses de panneau publiées du parc, dans les TROIS origines (206 `deployed`,
// 3 `dropped`, 4 `unknown`), et AUCUNE pose d'un appareil porté. Le film dit donc de ces
// 7 poses exactement ce qu'il dit des 209 autres.
//
// ELLE VIT ICI, AVEC SA TABLE, et pas dans `equipment_origin.go` : la source est le
// manifeste, et c'est ce fichier qui en porte la transcription et ses DEUX garde-rails —
// les FAMILLES et, depuis le lot 1.9.1, les IDENTIFIANTS
// (`usage_summary_families_guard_test.go`). Aucune troisième copie n'est créée.
func equipmentIsSpawnedPiece(id string) bool {
	return usageWallPanelIDs[id]
}

// usageFamilyIsDroppable dit si un LÂCHER de cette famille compte dans
// `dropped_objects` : tout SAUF les grenades (§3 du handoff — « hors familles de
// grenade »). Les appareils de capacité (grappin, propulseur…) et les bonus lâchés
// comptent : ils gisent au sol, ramassables.
func usageFamilyIsDroppable(family string) bool {
	return !usageGrenadeFamilies[family]
}

// usageFamilyIsDeployable dit si un DÉPLOIEMENT de cette famille compte dans
// `deployed_by_family` : ni grenade (un « déploiement » de grenade est un lancer),
// ni capacité portée (l'appareil n'est pas un objet posé), ni bonus (aucun
// déploiement mesuré n'existe). `other` COMPTE : la pose est réelle, sa nature
// seule n'est pas établie — même lecture que le bilan d'équipement web
// (isDeployableFamily : PLACEMENT_RENDER['other'] non nul).
func usageFamilyIsDeployable(family string) bool {
	return !usageGrenadeFamilies[family] &&
		!usageCarriedCapacityFamilies[family] &&
		!usagePowerupFamilies[family]
}

// usageDeployedCounts dit si CETTE pose est un déploiement à compter : origine
// STRICTEMENT `deployed`, famille déployable, et — pour le mur seulement —
// l'identifiant des PANNEAUX. L'ÉGALITÉ STRICTE N'EST PAS UN DÉTAIL (revue
// adversariale 2026-09-04) : sur le fil de l'eau un poseur mesuré a toujours une
// origine (equipmentOrigin), mais le BACKFILL projette aussi des artefacts d'un
// schéma < 10 où `origin` n'existe pas — une origine absente est NON MESURÉE et ne
// compte nulle part (« le client lit unknown, JAMAIS deployed », equipment_placements.go
// — et 88,6 % des poses mesurées sont des lâchers : le repli inverse gonflerait tout).
func usageDeployedCounts(p *EquipmentPlacement) bool {
	if p.Origin != OriginDeployed || !usageFamilyIsDeployable(p.Family) {
		return false
	}
	if p.Family == usageFamilyWall {
		return usageWallPanelIDs[p.ID]
	}
	return true
}

// LES FAMILLES QUI ENGENDRENT UNE PIÈCE DISTINCTE — celles dont le canal des POSES
// voit le déploiement — ONT QUITTÉ CE FICHIER le 2026-09-17 (item 2.5.f) pour
// `internal/domain/equipmentusage` : leur lecteur n'est pas que le résumé, c'est
// aussi l'agrégat de session, qui ne peut pas importer un paquet de titre (D9). La
// table y est inchangée, et son RECOLLEMENT au manifeste reste ici
// (usage_summary_families_guard_test.go), là où le manifeste du titre est.
