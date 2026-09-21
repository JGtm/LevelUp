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

/**
 * Les niveaux, dans l'ORDRE DE LECTURE du bloc.
 *
 * PUISSANCE EN TÊTE DEPUIS LE 2026-09-21 (décision utilisateur, amendant D2) : puissance,
 * puis terrain, puis base dans un dépliable fermé. Ce qui décide un match ouvre le bloc ;
 * reprendre son fusil d'assaut ferme la marche.
 *
 * `powerup` ET `unclassified` FERMENT LA LISTE ET NE SE RENDENT PLUS (même décision, cf.
 * `MatchPadControlSection`) : un socle de BONUS est un ÉQUIPEMENT, il se lit dans « Usages
 * d'équipement » (`EPISODE_FAMILIES`, `equipmentUsageLogic.ts`) et n'a rien à faire dans le
 * contrôle des ARMES ; « non identifié » est une absence de mesure, pas un niveau de jeu. Le
 * CLASSEMENT (`padTierOf`) est inchangé — seul l'affichage retire ces deux groupes.
 *
 * IL NE DÉPARTAGE PLUS RIEN : voir `PAD_TIER_TIEBREAK` ci-dessous.
 */
export const PAD_TIER_ORDER = ['power', 'ground', 'base', 'powerup', 'unclassified'] as const

export type PadTier = (typeof PAD_TIER_ORDER)[number]

/**
 * ORDRE DE DÉPARTAGE d'une arme vue à deux niveaux dans le même match (`tierOfWeaponOf`) —
 * DISTINCT de l'ordre d'affichage, et c'est tout l'objet de cette constante.
 *
 * DÉCOUPLAGE DU 2026-09-21 (lot G), sur découverte du lot D. Les deux usages lisaient
 * `PAD_TIER_ORDER` : inverser l'ordre de LECTURE du bloc (base → puissance en tête, commit
 * a9733a985) a donc silencieusement inversé le DÉPARTAGE d'un conflit à compte égal, sans
 * qu'aucune décision produit ne le demande. Un ordre d'écran est un choix de mise en page ;
 * un ordre de départage est une règle de classement — ils n'ont pas à bouger ensemble.
 *
 * La valeur ci-dessous RESTAURE le départage d'avant a9733a985 (`base, ground, power,
 * powerup, unclassified`). Le conflit est rare et mesuré : UNE arme sur 59 matchs et 2 881
 * prises (0,17 %), `terrain` contre `non classé` (cf. `padControlLogic.ts`).
 */
export const PAD_TIER_TIEBREAK: readonly PadTier[] = [
  'base',
  'ground',
  'power',
  'powerup',
  'unclassified',
]

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

/**
 * Assemble le classement d'un match.
 *
 * LE CARACTÈRE ALÉATOIRE DES DÉPARTS VIENT DU SERVEUR (`doc.weaponTiers.randomStarts`), jamais
 * d'une liste de catégories côté web. Cette liste a existé une semaine et a divergé : la règle
 * vit dans le TOML du titre, elle gouverne aussi l'écriture en base, et deux copies auraient
 * fini par ranger le même match dans deux niveaux différents selon la page (revue 2026-09-14).
 *
 * Champ absent = le serveur ne sait pas (mode inconnu, titre sans règle) : on ne publie alors
 * PAS la note « départs aléatoires », et on ne l'invente pas non plus — le niveau « base » se
 * mesure, et il se vérifie de lui-même.
 */
export function buildPadTierMatch(doc: ReplayDocumentReady): PadTierMatch {
  const familyByPad: (string | null)[] = doc.weaponPads.map(() => null)
  for (const spot of doc.mapWeaponPads?.pads ?? []) {
    if (spot.pad >= 0 && spot.pad < familyByPad.length) familyByPad[spot.pad] = spot.family
  }
  const randomStarts = doc.weaponTiers?.randomStarts === true
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
