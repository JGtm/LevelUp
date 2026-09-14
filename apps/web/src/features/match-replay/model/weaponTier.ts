/**
 * weaponTier.ts — À QUEL NIVEAU SE LIT UNE ARME DE SOCLE : base, terrain, puissance, non classé.
 *
 * CE QUE LE BLOC NE SAVAIT PAS DIRE. « Contrôle des armes spéciales » comptait toutes les prises
 * de socle à égalité : reprendre son fusil d'assaut posé sur un râtelier y pesait autant que
 * rafler le lance-roquettes du socle central. Ce n'est pas ce qu'un match raconte. Demande
 * utilisateur du 2026-09-13 : « faire la distinction entre les armes de base, les armes de
 * terrain et les armes spéciales […] sur les râteliers ce sont des armes de terrain, sur les
 * socles des armes de puissance ».
 *
 * LES TROIS SOURCES, ET AUCUNE N'EST UN NOM D'ARME :
 *
 *   BASE       l'arme est dans l'équipement de DÉPART d'une vie du match — canal `loadouts`,
 *              PREMIÈRE émission de chaque `slot` (un slot = une vie).
 *   TERRAIN    l'emplacement de la carte qui confirme le socle est un RÂTELIER (`rack`).
 *   PUISSANCE  cet emplacement est un SOCLE DE PUISSANCE (`power`).
 *   NON CLASSÉ aucun emplacement ne confirme le socle. Il reste VISIBLE avec son compte,
 *              jamais fondu dans un autre niveau.
 *
 * LA NATURE D'UN EMPLACEMENT VIENT DE LA CARTE, JAMAIS DE L'ARME. Mesure du 2026-09-14 sur 76
 * artefacts : 70 socles sur 669 portent une arme de rôle « lourd » (Hydra, Needler, Sentinel
 * Beam, Shock Rifle) sur un RÂTELIER, et c'est nominal. Juger au nom de l'arme produirait donc
 * 10 % de faux niveaux ; la même Hydra est de terrain sur une carte et de puissance sur une
 * autre.
 *
 * JUMEAU DE `internal/analysis/weapontier` (Go), QUI SERT LES AGRÉGATS. Les deux existent parce
 * que la résolution vit de deux côtés : ici la vue match, qui a la catégorie de mode dans son
 * en-tête ; là-bas les pages Sessions / Escouade / Timeseries, qui agrègent côté serveur. Les
 * deux partagent les mêmes seuils écrits et les mêmes témoins — toute divergence est un bug,
 * pas une variante.
 *
 * Pur : aucun React, aucune couleur, AUCUNE LANGUE. Les intertitres vivent dans l'i18n du rejeu.
 */
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

/** Les niveaux, dans l'ORDRE DE LECTURE du bloc. */
export const PAD_TIER_ORDER = ['base', 'ground', 'power', 'powerup', 'unclassified'] as const

export type PadTier = (typeof PAD_TIER_ORDER)[number]

/**
 * Part minimale des vies du match qu'une arme doit occuper AU DÉPART pour être tenue pour une
 * arme de base.
 *
 * POURQUOI UN SEUIL, ET POURQUOI CELUI-LÀ (mesure du 2026-09-14, 4 406 vies en classé et partie
 * rapide). Les trois armes de départ réelles pèsent 94,4 % des équipements de première émission
 * (MA40 45,5 %, Sidekick 29,5 %, BR75 19,4 %) ; une queue de 5,6 % porte des armes qui n'en sont
 * pas — Skewer, Sniper, Épée — parce que la première émission publiée d'une vie arrive parfois
 * après un ramassage. Sans seuil, un Skewer vu une fois au départ d'une vie ferait passer TOUTES
 * ses prises en « base » et viderait le niveau « puissance » de son sens. À 5 % la queue tombe
 * entièrement, et les trois vraies armes de base passent avec quatre à neuf fois la marge.
 *
 * La valeur est celle de `weapontier.BaseShareMin` côté Go. Elle ne retire jamais une prise du
 * bloc : une arme recalée retombe sur le niveau de son emplacement.
 */
export const BASE_SHARE_MIN = 0.05

/**
 * Catégories de mode à DÉPARTS ALÉATOIRES : le niveau « base » n'y a aucun sens et n'est pas
 * publié. Les valeurs sont celles de la taxonomie du titre (`mode_category`, posée par le
 * serveur) — jamais un nom de mode lu à l'écran, jamais un slug de titre. Un titre sans
 * taxonomie laisse la catégorie vide, donc départs NON aléatoires : le repli sûr, puisque le
 * niveau se mesure alors et se vérifie tout seul.
 */
const CATEGORIES_DEPARTS_ALEATOIRES = new Set(['Fiesta', 'Super Fiesta', 'Husky Raid'])

/** Vrai quand la catégorie de mode distribue des équipements de départ aléatoires. */
export function hasRandomStarts(modeCategory: string | null | undefined): boolean {
  return !!modeCategory && CATEGORIES_DEPARTS_ALEATOIRES.has(modeCategory)
}

/** Ce qu'un match apprend une fois pour toutes sur ses niveaux. */
export interface PadTierMatch {
  /** Famille d'emplacement par index de socle (`weaponPads`) ; `null` = non confirmé. */
  familyByPad: (string | null)[]
  /** Familles d'arme retenues comme armes de départ (hexadécimal, clé de `weaponPads[].weapon`). */
  baseWeapons: Set<string>
  /** Le mode distribue des départs aléatoires : pas de niveau « base ». */
  randomStarts: boolean
  /** Nombre de vies dont l'équipement de départ a été lu (dénominateur de `BASE_SHARE_MIN`). */
  lives: number
  /**
   * FAUX quand AUCUN emplacement de la carte n'a confirmé de socle — carte hors référence, ou
   * repère décalé. Le bloc doit alors dire « niveaux non établis », et surtout PAS ranger tout
   * le match sous « Non classé » : l'un est une absence de mesure, l'autre un résultat de
   * mesure, et les confondre ferait lire un défaut de carte comme un fait de jeu.
   */
  tiersMeasured: boolean
}

/** Assemble le classement d'un match. `modeCategory` vient de `header.mode_category`. */
export function buildPadTierMatch(
  doc: ReplayDocumentReady,
  modeCategory: string | null | undefined,
): PadTierMatch {
  const familyByPad: (string | null)[] = doc.weaponPads.map(() => null)
  for (const spot of doc.mapWeaponPads?.pads ?? []) {
    if (spot.pad >= 0 && spot.pad < familyByPad.length) familyByPad[spot.pad] = spot.family
  }
  const randomStarts = hasRandomStarts(modeCategory)
  const { weapons, lives } = baseWeaponsOf(doc)
  return {
    familyByPad,
    baseWeapons: randomStarts ? new Set<string>() : weapons,
    randomStarts,
    lives,
    tiersMeasured: familyByPad.some((f) => f !== null),
  }
}

/**
 * Le niveau d'UNE prise : le socle où elle a eu lieu, et l'arme qui s'y trouvait.
 *
 * L'ORDRE DES TESTS EST LA RÈGLE : base > terrain > puissance > non classé. Une arme de départ
 * ramassée sur un râtelier se lit « base » — reprendre son fusil d'assaut n'est pas contrôler
 * la carte. Un socle de BONUS échappe à cet ordre : ce n'est pas une arme.
 */
export function padTierOf(match: PadTierMatch, padIndex: number, weapon: string): PadTier {
  const family = match.familyByPad[padIndex] ?? null
  if (family === 'powerup') return 'powerup'
  if (!match.randomStarts && match.baseWeapons.has(weapon)) return 'base'
  if (family === 'rack') return 'ground'
  if (family === 'power') return 'power'
  return 'unclassified'
}

/**
 * baseWeaponsOf — les armes de départ du match.
 *
 * UNE SEULE ÉMISSION COMPTE PAR SLOT, LA PREMIÈRE, et c'est mesuré : un `slot` est une VIE (100
 * slots distincts pour 113 pistes sur le match témoin `01e1f945`), et le canal RÉ-ÉMET en cours
 * de vie après un changement d'arme. Prendre tout le canal dilue les trois armes de base de
 * 94,4 % à 85,6 % en classé et fait monter le S7 Sniper de 0,73 % à 3,15 % — autrement dit, il
 * transforme une arme de puissance en arme de base.
 */
function baseWeaponsOf(doc: ReplayDocumentReady): { weapons: Set<string>; lives: number } {
  const premiere = new Map<number, string[]>()
  for (const lo of doc.loadouts) {
    if (!premiere.has(lo.slot)) premiere.set(lo.slot, lo.w)
  }
  const lives = premiere.size
  const weapons = new Set<string>()
  if (lives === 0) return { weapons, lives }
  const compte = new Map<string, number>()
  for (const w of premiere.values()) {
    // Une même arme deux fois dans le même équipement de départ ne vaut qu'UNE vie : le
    // dénominateur est la vie, pas l'emplacement d'inventaire.
    for (const arme of new Set(w)) {
      if (!arme) continue
      compte.set(arme, (compte.get(arme) ?? 0) + 1)
    }
  }
  const seuil = lives * BASE_SHARE_MIN
  for (const [arme, n] of compte) {
    if (n >= seuil) weapons.add(arme)
  }
  return { weapons, lives }
}
