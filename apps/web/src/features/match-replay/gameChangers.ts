/**
 * gameChangers.ts — CE QUI CHANGE UNE PARTIE, d'après le VOTE, et rien d'autre.
 *
 * PROVENANCE : vote du 2026-09-05 (artefact « Game changers », collection `votes` de sa base,
 * dix élus relus par le superviseur), commandé par l'utilisateur le 2026-09-04 (« replier ceux
 * qui ne sont pas game changer »). Cette liste est un JUGEMENT PRODUIT : elle gouverne la
 * HIÉRARCHIE D'AFFICHAGE des deux bilans de la fiche match (usages d'équipement, contrôle des
 * armes spéciales) — ce qui n'y est pas se REPLIE derrière « Voir plus (N) », il ne disparaît
 * jamais : les totaux, dénominateurs et notes de couverture comptent tout, replié compris.
 *
 * ELLE N'EST PAS `POWER_PAD_KEYS` (décision D1 du plan 2026-09-05). `POWER_PAD_KEYS`
 * (weaponPadFamilies.ts) répond à « quelle icône mérite d'être GRANDE sur un socle du rejeu
 * 2D » (arbitrage utilisateur du 2026-08-18) ; cette liste-ci répond à « quelle colonne mérite
 * d'être VISIBLE d'emblée dans un bilan » (vote du 2026-09-05). Même mot « game changer »,
 * deux jugements, deux surfaces : que leurs armes COÏNCIDENT aujourd'hui (depuis la promotion
 * du cindershot, décision utilisateur du 05/09) n'en fait pas une seule liste — l'une ne
 * dérive jamais de l'autre, et le garde-rail fige la coïncidence comme un FAIT daté.
 *
 * DEUX VOCABULAIRES POUR LE MÊME BONUS, ET UN PONT ÉCRIT (décision D5 — le piège n°1 du lot).
 * Un SOCLE de bonus publie une famille d'équipement (`powerup_camo`) ; l'ÉPISODE d'état actif
 * du même bonus publie la famille courte (`camo`, cf. `EPISODE_FAMILIES`). Le pont entre les
 * deux est la table `EPISODE_FAMILY_OF_POWERUP` — jamais un `includes`/`replace` sur le
 * préfixe : le manifeste du titre porte quinze familles, et rien ne garantit que les
 * prochaines suivront la convention de nom (même règle que `PAD_EQUIPMENT_FAMILIES`).
 *
 * LES ARMES SE JUGENT PAR `weapon_key` (décision D6) : `doc.weaponLabels[hex].key`, jamais
 * l'hexadécimal du film ni un libellé. Un label sans `key` (artefact ancien, arme hors
 * catalogue) est NON game changer — replié, jamais promu : même prudence que `padScaleOf`.
 */
import { EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD } from './equipmentFx'
import type { PadEquipmentFamilyKey } from './weaponPadFamilies'

/**
 * LES CINQ FAMILLES D'ÉQUIPEMENT ÉLUES (vote du 2026-09-05). Identifiants STABLES du manifeste
 * du titre (`replay_labels.toml`, vérifié par garde-rail). REPLIÉS par ce même vote :
 * `grapple`, `thruster`, `repulsor`, `wall`, `repair_field`, `translocator_beacon` — et toute
 * famille future, tant qu'un vote ne l'élit pas.
 */
export const GAME_CHANGER_EQUIPMENT_FAMILIES: readonly string[] = [
  'powerup_camo',
  'powerup_overshield',
  'sensor',
  'threat_seeker',
  'shroud_screen',
]

/**
 * LES ARMES ÉLUES (6 + 1 variante) : les `weapon_key` du registre du titre
 * (`weapon_names.toml`, vérifié par garde-rail). `hinf_fuel_rod_spnkr` est la variante Fiesta
 * du même socle SPNKr — élu avec lui. `hinf_cindershot` : voté NON le 05/09 au matin, PROMU
 * par décision utilisateur le 05/09 même (« le cindershot peut être un game changer ») — les
 * armes de cette liste et celles de `POWER_PAD_KEYS` coïncident donc à nouveau, sans que l'une
 * dérive de l'autre (deux jugements, deux surfaces — cf. en-tête).
 */
export const GAME_CHANGER_WEAPON_KEYS: readonly string[] = [
  'hinf_s7_sniper',
  'hinf_m41_spnkr',
  'hinf_fuel_rod_spnkr',
  'hinf_energy_sword',
  'hinf_gravity_hammer',
  'hinf_skewer',
  'hinf_cindershot',
]

/**
 * LE PONT D5 : famille de SOCLE de bonus -> famille d'ÉPISODE d'état actif. Table écrite et
 * testée par mutation (garde-rail : chaque valeur existe dans `EPISODE_FAMILIES`), jamais une
 * manipulation de préfixe. Les valeurs réutilisent les constantes du module de mesure — pas
 * une troisième copie du littéral.
 */
export const EPISODE_FAMILY_OF_POWERUP: Readonly<Record<PadEquipmentFamilyKey, string>> = {
  powerup_camo: EQUIP_FAMILY_CAMO,
  powerup_overshield: EQUIP_FAMILY_OVERSHIELD,
}

/**
 * Les familles d'ÉPISODE game changers, DÉRIVÉES du vote par le pont — jamais écrites une
 * seconde fois : si un vote futur replie `powerup_camo`, l'épisode `camo` se replie avec lui,
 * sans qu'on y pense.
 */
const GAME_CHANGER_EPISODE_FAMILIES: readonly string[] = Object.entries(EPISODE_FAMILY_OF_POWERUP)
  .filter(([powerup]) => GAME_CHANGER_EQUIPMENT_FAMILIES.includes(powerup))
  .map(([, episode]) => episode)

/**
 * isGameChangerFamily — cette famille est-elle élue ? Répond dans les DEUX vocabulaires :
 * la famille de socle/pose (`powerup_camo`, `sensor`) directement par la liste votée, la
 * famille d'épisode (`camo`) par le pont D5. Une famille inconnue n'est jamais promue.
 */
export function isGameChangerFamily(family: string): boolean {
  return (
    GAME_CHANGER_EQUIPMENT_FAMILIES.includes(family) ||
    GAME_CHANGER_EPISODE_FAMILIES.includes(family)
  )
}

/**
 * isGameChangerWeaponKey — cette clé canonique d'arme est-elle élue ?
 *
 * Clé absente (label sans `key` : artefact d'avant sa publication, ou arme hors catalogue du
 * titre) = NON game changer, donc REPLIÉ — dégradation VOULUE (décision D6) : promouvoir ce
 * qu'on ne sait pas nommer affirmerait un enjeu que rien n'établit.
 */
export function isGameChangerWeaponKey(key: string | null | undefined): boolean {
  return key != null && GAME_CHANGER_WEAPON_KEYS.includes(key)
}
