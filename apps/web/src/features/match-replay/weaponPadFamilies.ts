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
 */

/** Les deux tailles d'icône de socle. `power` = grande, `classic` = petite. */
export type PadScale = 'power' | 'classic'

/**
 * LES ARMES DE PUISSANCE ET LES POWER-UPS, nommés un par un (demande utilisateur du 18/08 :
 * « liste EXPLICITE »). Ce sont les `weapon_key` du registre du titre
 * (`config/titles/halo_infinite/mappings/weapon_names.toml`) — un garde-rail les y vérifie.
 *
 * LES DEUX POWER-UPS SONT LÀ SANS QU'AUCUN SOCLE N'EN PORTE AUJOURD'HUI, et ce n'est pas un
 * oubli de mesure : le corpus des onze films ne compte qu'UNE pose de surbouclier et UNE de
 * camouflage, toutes deux LÂCHÉES À LA MORT (registre des reports, 2026-08-18) — elles
 * voyagent d'ailleurs par un autre canal du document (`equipmentPlacements`), pas par les
 * socles. L'utilisateur les a nommées explicitement dans la règle de taille : elles y figurent
 * donc comme règle ÉCRITE D'AVANCE, avec leurs identifiants de famille d'équipement, et le
 * jour où un film en portera une sur un socle elle sera grande sans qu'on y revienne.
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
  // Power-ups — familles d'équipement (cf. en-tête : aucun socle mesuré n'en porte).
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
