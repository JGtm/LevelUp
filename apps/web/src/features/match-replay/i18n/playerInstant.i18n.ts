/**
 * playerInstant.i18n.ts — LES TEXTES DE LA FICHE DU JOUEUR « À L'INSTANT » : ce que le joueur a fait
 * du début du match jusqu'à l'instant lu, à côté des totaux du match. Le contrat (`PlayerInstantText`)
 * et ses deux langues.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de
 * taille du dépôt. Servi par `REPLAY_TEXT[locale].playerInstant` ; la parité FR/EN reste tenue par le
 * typage (`satisfies PlayerInstantText` dans chaque langue, `satisfies ReplayText` au-dessus).
 *
 * LES NOMS DE CLASSE D'ARME NE SONT PAS ICI : ils viennent du manifeste de l'application
 * (`frags.class.*`), le même que la page Synthèse.
 */

export interface PlayerInstantText {
  /** Le titre de l'état du joueur à l'instant lu, et l'en-tête de la colonne du même nom. */
  now: string
  /** Le tableau des éliminations, par classe d'arme puis par arme. */
  killsByWeapon: string
  /** En-tête de colonne : la part de chaque arme dans les éliminations du joueur à l'instant. */
  colShare: string
  /** En-tête de colonne : le total de fin de match. */
  colMatch: string
  /** Le tableau de l'équipement, par famille. */
  equipment: string
  /** Les trois colonnes de l'équipement : jamais « ramassé ». */
  equipmentTaken: string
  equipmentUsed: string
  equipmentLost: string
  medals: string
  /** La section des gestes du joueur (sprint, déplacement, sauts, rechargements). */
  gestures: string
  sprintTotal: string
  /** Le temps pendant lequel le joueur a poussé sa commande de déplacement, et le nombre de rechargements. */
  moveTotal: string
  jumps: string
  reloads: string
  /** Infobulle du temps de déplacement quand le film n'a pas publié toutes les commandes : le temps est une estimation. */
  moveHint: string
  /** Infobulle des rechargements quand le film perd des messages ou des rechargements sans joueur : le compte est un minimum. */
  reloadsHint: string
  matchTotals: string
  /** Les deux états d'une arme dans le bloc « À l'instant » : dégainée, et rangée. */
  weaponInHand: string
  weaponStowed: string
  /** Les charges restantes de l'équipement porté (« 1 charge », « 2 charges »). */
  equipmentCharges: (n: number) => string
  /** La section de ce que le joueur a fait depuis le début du match, et ses lignes. */
  sinceStart: string
  fda: string
  currentStreak: string
  bestStreak: string
  headshots: string
  perfects: string
  respawns: string
  currentLife: string
  averageLife: string
  /** Les trois lignes du bouclier : bris, recharges complètes, et leur différentiel (recharges moins bris). */
  shieldsBroken: string
  shieldRecharges: string
  shieldDifferential: string
  /** Le total du match qui suit la valeur à l'instant (« 31 sur le match »). */
  matchTotalFmt: (n: string) => string
  score: string
  /** La place dans le lobby (« 2e sur 8 »). */
  rankFmt: (place: number, of: number) => string
  /** La capsule des valeurs estimées, et son infobulle. */
  estimate: string
  estimateHint: string
  /** La section des tirs et des dégâts, et ses lignes. */
  shotsAndDamage: string
  shotAccuracy: string
  damageDealtTaken: string
  shotsFmt: (n: number) => string
  /** Les dégâts d'une arme, le nombre étant déjà écrit dans la langue de la page. */
  damageFmt: (written: string) => string
  /** La section des grenades lancées. */
  grenadesThrown: string
  /** La section des ramassages, l'infobulle de sa capsule (le compte du film est un minimum) et ses deux lignes. */
  pickups: string
  pickupsHint: string
  pickupWeapons: string
  pickupGrenades: string
  /** Une arme du tableau des éliminations dont aucun nom n'est connu : jamais sa clé. */
  unnamedWeapon: string
  /** Les éliminations à la tête, dans les totaux du match. */
  headshotKills: string
}

/** Le rang en français : « 1er », puis « 2e », « 3e »… */
const frOrdinal = (n: number): string => (n === 1 ? '1er' : `${n}e`)

/** Le rang en anglais : « 1st », « 2nd », « 3rd », « 4th », « 11th »… */
function enOrdinal(n: number): string {
  const lastTwo = n % 100
  if (lastTwo >= 11 && lastTwo <= 13) return `${n}th`
  const suffix = { 1: 'st', 2: 'nd', 3: 'rd' }[n % 10]
  return `${n}${suffix ?? 'th'}`
}

export const FR_PLAYER_INSTANT = {
  now: "À l'instant",
  killsByWeapon: 'Éliminations par arme',
  colShare: 'Part',
  colMatch: 'Match',
  equipment: 'Équipement',
  equipmentTaken: 'Pris',
  equipmentUsed: 'Utilisé',
  equipmentLost: 'Perdu',
  medals: 'Médailles',
  gestures: 'Gestes',
  sprintTotal: 'Sprint, temps cumulé',
  moveTotal: 'Déplacement, temps cumulé',
  jumps: 'Sauts',
  reloads: 'Rechargements',
  moveHint: "Le film ne montre pas toutes les commandes de déplacement : ce temps est une estimation",
  reloadsHint: 'Le film ne montre pas tous les rechargements : ce compte est un minimum',
  matchTotals: 'Totaux du match',
  weaponInHand: 'en main',
  weaponStowed: 'en réserve',
  equipmentCharges: (n) => (n === 1 ? '1 charge' : `${n} charges`),
  sinceStart: 'Depuis le début du match',
  fda: 'FDA',
  currentStreak: 'Série en cours',
  bestStreak: 'Meilleure série',
  headshots: 'Tirs à la tête',
  perfects: 'Éliminations parfaites',
  respawns: 'Réapparitions',
  currentLife: 'Vie en cours',
  averageLife: "Durée moyenne d'une vie",
  shieldsBroken: 'Boucliers brisés',
  shieldRecharges: 'Recharges complètes',
  shieldDifferential: 'Différentiel',
  matchTotalFmt: (n) => `${n} sur le match`,
  score: 'Score',
  rankFmt: (place, of) => `${frOrdinal(place)} sur ${of}`,
  estimate: 'estimation',
  estimateHint: "Valeur estimée à partir du film, pas un total officiel",
  shotsAndDamage: 'Tirs et dégâts',
  shotAccuracy: 'Précision',
  damageDealtTaken: 'Dégâts infligés / subis',
  shotsFmt: (n) => (n === 1 ? '1 tir' : `${n} tirs`),
  damageFmt: (written) => `${written} dégâts`,
  grenadesThrown: 'Grenades lancées',
  pickups: 'Ramassages',
  pickupsHint: 'Le film ne montre pas tous les ramassages : ce compte est un minimum',
  pickupWeapons: 'Armes',
  pickupGrenades: 'Grenades',
  unnamedWeapon: 'Sans arme nommée',
  headshotKills: 'Éliminations à la tête',
} satisfies PlayerInstantText

export const EN_PLAYER_INSTANT = {
  now: 'Right now',
  killsByWeapon: 'Kills by weapon',
  colShare: 'Share',
  colMatch: 'Match',
  equipment: 'Equipment',
  equipmentTaken: 'Taken',
  equipmentUsed: 'Used',
  equipmentLost: 'Lost',
  medals: 'Medals',
  gestures: 'Moves',
  sprintTotal: 'Sprint, total time',
  moveTotal: 'Movement, total time',
  jumps: 'Jumps',
  reloads: 'Reloads',
  moveHint: 'The film does not show every movement command: this time is an estimate',
  reloadsHint: 'The film does not show every reload: this count is a minimum',
  matchTotals: 'Match totals',
  weaponInHand: 'in hand',
  weaponStowed: 'in reserve',
  equipmentCharges: (n) => (n === 1 ? '1 charge' : `${n} charges`),
  sinceStart: 'Since the start of the match',
  fda: 'KDA',
  currentStreak: 'Current streak',
  bestStreak: 'Best streak',
  headshots: 'Headshots',
  perfects: 'Perfect kills',
  respawns: 'Respawns',
  currentLife: 'Current life',
  averageLife: 'Average life',
  shieldsBroken: 'Shields broken',
  shieldRecharges: 'Full recharges',
  shieldDifferential: 'Differential',
  matchTotalFmt: (n) => `${n} in the match`,
  score: 'Score',
  rankFmt: (place, of) => `${enOrdinal(place)} of ${of}`,
  estimate: 'estimate',
  estimateHint: 'Value estimated from the film, not an official total',
  shotsAndDamage: 'Shots and damage',
  shotAccuracy: 'Accuracy',
  damageDealtTaken: 'Damage dealt / taken',
  shotsFmt: (n) => (n === 1 ? '1 shot' : `${n} shots`),
  damageFmt: (written) => `${written} damage`,
  grenadesThrown: 'Grenades thrown',
  pickups: 'Pickups',
  pickupsHint: 'The film does not show every pickup: this count is a minimum',
  pickupWeapons: 'Weapons',
  pickupGrenades: 'Grenades',
  unnamedWeapon: 'No named weapon',
  headshotKills: 'Headshot kills',
} satisfies PlayerInstantText
