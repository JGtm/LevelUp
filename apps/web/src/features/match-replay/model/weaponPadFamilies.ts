/**
 * weaponPadFamilies.ts — CE QUI MÉRITE UNE GRANDE ICÔNE SUR UN SOCLE, et pourquoi la liste
 * est écrite plutôt que devinée.
 *
 * LA DEMANDE EST UN ARBITRAGE D'ÉCRAN, mot pour mot (bilan du 2026-08-18, W4) : « des icônes
 * trop petites seraient inutiles mais des trop grosses risquent de polluer » — et la raison
 * du partage y est donnée dans la même phrase : « sur les socles au sol on a des power-ups et
 * des armes plus puissantes qui ont un intérêt très stratégique », les râteliers portant « des
 * armes "classiques" pas game changer ». La taille suit donc l'ENJEU de l'arme, pas sa place :
 * la donnée ne distingue pas un socle au sol d'un râtelier mural (position seule).
 *
 * LA CLÉ EST LE `weapon_key` DU TITRE, jamais l'hexadécimal du film ni un libellé. C'est le
 * même vocabulaire que la banque de sons (`weaponLabels[id].key`, posé à la requête par le
 * service) : une famille d'arme y entre par son identifiant canonique, stable et bilingue en
 * aval. Un identifiant que le titre ne catalogue pas n'a pas de clé — il reste donc CLASSIQUE,
 * jamais promu par défaut : agrandir ce qu'on ne sait pas nommer serait affirmer un enjeu que
 * rien n'établit.
 *
 * CETTE LISTE EST LA RÈGLE, PAS LE RELEVÉ D'UN CORPUS. Elle est écrite AVANT la mesure et vaut
 * pour les matchs à venir : un socle qui porterait une arme de puissance jamais vue dans les
 * quatre témoins doit être grand du premier coup.
 *
 * TOUT SOCLE N'EST PAS UNE ARME (schéma 17, 2026-08-19) : un socle de POWER-UP publie une
 * famille d'ÉQUIPEMENT et non un identifiant d'arme. La seconde table de ce fichier
 * (`PAD_EQUIPMENT_FAMILIES`) les nomme une par une, et c'est elle — jamais un test de préfixe —
 * qui décide qu'une clé n'est pas à chercher dans `weaponLabels`.
 */

/** Les deux tailles d'icône de socle. `power` = grande, `classic` = petite. */
export type PadScale = 'power' | 'classic'

/**
 * LES TROIS NATURES D'UN SOCLE (retour utilisateur du 2026-08-26 : « une couleur pour chaque
 * type, en respectant les couleurs accessibles »).
 *
 * ELLE N'EST PAS LA TAILLE, et c'est pour cela qu'elle a son propre type. `PadScale` répond à
 * « est-ce stratégique ? » et n'a que deux valeurs : un power-up et un sniper sont tous deux
 * GRANDS. La nature, elle, répond à « qu'est-ce que c'est ? » et en distingue trois — un
 * bonus qu'on ramasse, une arme de puissance, un râtelier ordinaire. Les confondre obligerait
 * à teinter pareil le surbouclier et l'épée.
 *
 * L'ORDRE DE RÉSOLUTION EST LE MÊME QUE PARTOUT DANS CE FICHIER : la table des familles
 * d'équipement d'abord (elle seule sait qu'une clé n'est pas une arme), le registre des armes
 * de puissance ensuite, le repli ordinaire enfin.
 */
export type PadFamily = 'powerup' | 'power' | 'classic'

/**
 * padFamilyOf — la nature d'un socle, d'après ce qu'il porte.
 *
 * `key` est la clé canonique du titre (`weaponLabels[id].key`), absente pour une arme hors
 * catalogue : un socle qu'on ne sait pas nommer reste `classic`, jamais promu — la même règle
 * prudente que `padScaleOf` juste en dessous.
 */
export function padFamilyOf(weapon: string, key: string | null | undefined): PadFamily {
  if (padEquipmentFamilyOf(weapon)) return 'powerup'
  return key && POWER_PAD_WEAPON_KEYS.includes(key) ? 'power' : 'classic'
}

/**
 * LES ARMES DE PUISSANCE ET LES POWER-UPS, nommés un par un (demande utilisateur du 18/08 :
 * « liste EXPLICITE »). Ce sont les `weapon_key` du registre du titre
 * (`config/titles/halo_infinite/mappings/weapon_names.toml`) — un garde-rail les y vérifie.
 *
 * LES DEUX POWER-UPS Y ÉTAIENT AVANT D'AVOIR UN MEMBRE MESURÉ, et le pari est tenu. Ils ont
 * été écrits le 2026-08-18 sur la seule demande de l'utilisateur — le corpus des onze films
 * ne montrait alors qu'UNE pose de surbouclier et UNE de camouflage, toutes deux LÂCHÉES À LA
 * MORT, publiées par un autre canal du document (`equipmentPlacements`). Le 2026-08-19, la
 * voie `ti=37` a trouvé leur socle : un `powerup_overshield` au centre de Catalyst, publié
 * dans `weaponPads` (schéma 17). Ils sont donc GRANDS sans qu'on ait eu à y revenir.
 *
 * CE SONT DES CLÉS DE FAMILLE D'ÉQUIPEMENT, pas des clés d'arme, et elles ne se joignent donc
 * à AUCUNE table de `weaponLabels` : `PAD_EQUIPMENT_FAMILIES`, plus bas, dit ce que le calque
 * en fait (taille, libellé, vignette).
 */
export const POWER_PAD_KEYS: readonly string[] = [
  // Armes de puissance — le registre d'armes du titre.
  'hinf_s7_sniper',
  'hinf_energy_sword',
  'hinf_gravity_hammer',
  'hinf_m41_spnkr',
  'hinf_fuel_rod_spnkr',
  'hinf_skewer',
  'hinf_cindershot',
  // Power-ups — familles d'équipement (cf. `PAD_EQUIPMENT_FAMILIES` : un socle mesuré en porte).
  'powerup_overshield',
  'powerup_camo',
]

/** Les clés de POWER_PAD_KEYS qui viennent du registre d'ARMES du titre (garde-rail). */
export const POWER_PAD_WEAPON_KEYS: readonly string[] = POWER_PAD_KEYS.filter(
  (k) => !k.startsWith('powerup_'),
)

/**
 * padScaleOf — la taille d'un socle d'après la clé canonique de ce qu'il porte.
 *
 * Clé absente (arme hors catalogue du titre, ou artefact servi sans catalogue lisible) =
 * `classic` : le défaut est la petite taille, dans les deux sens de la règle.
 */
export function padScaleOf(key: string | null | undefined): PadScale {
  return key && POWER_PAD_KEYS.includes(key) ? 'power' : 'classic'
}

// --- LES FAMILLES NON-ARME D'UN SOCLE -------------------------------------------------------

/**
 * CE QU'UN SOCLE PORTE QUAND CE N'EST PAS UNE ARME — et pourquoi il fallait une seconde table.
 *
 * DEPUIS LE SCHÉMA 17 (2026-08-19), `weaponPads[].weapon` n'est plus toujours l'hexadécimal
 * d'une famille d'arme : un socle de POWER-UP publie la FAMILLE du manifeste d'équipement
 * (`powerup_overshield`), parce que `weaponLabels` est une table d'ARMES où aucun identifiant
 * d'équipement n'entre (cf. `gwPadWeaponID`, côté Go). Le calque joignait cette clé à
 * `weaponLabels` : elle n'y était pas, donc le socle restait PETIT, sans vignette, et
 * l'infobulle affichait la chaîne brute « powerup_overshield ».
 *
 * UNE TABLE ÉCRITE, PAS UN TEST DE PRÉFIXE. Reconnaître « ça commence par `powerup_` » aurait
 * marché aujourd'hui et menti demain : le manifeste du titre porte quinze familles
 * d'équipement (`wall`, `sensor`, `grenade_*`…), et rien ne dit que les prochaines à monter sur
 * un socle s'appelleront `powerup_*`. Une clé absente de cette table reste donc traitée comme
 * un identifiant d'arme — le comportement d'avant, inchangé.
 */
export interface PadEquipmentFamily {
  /**
   * Stem de la vignette de HUD, servie sous `static/weapons-assets/{slug}/hud/`.
   *
   * C'EST L'IMAGE QUE LE MANIFESTE DU TITRE NOMME DÉJÀ pour la capacité de même nom
   * (`icon = "hud/Overshield"` dans `replay_labels.toml`) : le manifeste ne déclare aucune
   * icône sur ses `[[equipment_objects]]`, mais il nomme celle-ci ailleurs, et c'est le même
   * masque. Le garde-rail rejoue les deux — le fichier livré ET la ligne du manifeste.
   */
  icon: string
}

/** Les familles NON-ARME connues du calque des socles. Le typage porte la parité FR/EN. */
export type PadEquipmentFamilyKey = 'powerup_overshield' | 'powerup_camo'

/**
 * LA TABLE. Ses clés sont les familles publiées par le document, et elles figurent toutes
 * dans `POWER_PAD_KEYS` (garde-rail) : un power-up de socle est GRAND, sans règle en double.
 */
export const PAD_EQUIPMENT_FAMILIES: Readonly<Record<PadEquipmentFamilyKey, PadEquipmentFamily>> = {
  powerup_overshield: { icon: 'Overshield' },
  powerup_camo: { icon: 'ActiveCamouflage' },
}

/**
 * padEquipmentFamilyOf — la famille non-arme d'un socle, ou null quand la clé n'en est pas une.
 *
 * `hasOwnProperty` et non `in` : `'constructor'` est une clé de socle parfaitement possible sur
 * un titre futur, et un identifiant hérité du prototype rendrait une famille qui n'existe pas.
 */
export function padEquipmentFamilyOf(weapon: string): PadEquipmentFamilyKey | null {
  return Object.prototype.hasOwnProperty.call(PAD_EQUIPMENT_FAMILIES, weapon)
    ? (weapon as PadEquipmentFamilyKey)
    : null
}
