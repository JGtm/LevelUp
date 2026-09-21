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
 *              PREMIÈRE émission de chaque `slot` (un slot = une vie), et SEULEMENT si aucune
 *              prise d'arme de cette vie ne la précède (cf. `prisesEnVie`).
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
 * L'ordre sert aussi à DÉPARTAGER une arme vue à deux niveaux (`tierOfWeaponOf`) : à compte
 * égal, elle se lit désormais au niveau le plus disputé, ce qui est la même règle de lecture.
 */
export const PAD_TIER_ORDER = ['power', 'ground', 'base', 'powerup', 'unclassified'] as const

export type PadTier = (typeof PAD_TIER_ORDER)[number]

/**
 * Part minimale des vies du match qu'une arme doit occuper AU DÉPART pour être tenue pour une
 * arme de base.
 *
 * POURQUOI UN SEUIL SUBSISTE APRÈS LE CORRECTIF DU 2026-09-21. La queue d'armes qui n'en sont
 * pas venait d'une CAUSE, désormais traitée en amont : le canal `loadouts` est publié sur une
 * GRILLE D'IMAGES-CLÉS GLOBALE (une émission toutes les 200 frames = 20 s ; témoin `b1ad85eb` :
 * t = 12, 212, 412 … 5014), si bien que la première émission d'une vie accuse 0 à 19,2 s de
 * retard sur l'apparition — le temps d'un ramassage. `baseWeaponsOf` écarte maintenant toute
 * émission POSTÉRIEURE à une prise d'arme de la même vie.
 *
 * Le seuil reste parce que les canaux de prise ne couvrent pas TOUTES les prises (b1ad85eb
 * publie 118 prises d'arme pour 87 vies, et deux vies y prennent un Empaleur sans qu'aucun
 * canal ne l'écrive). Mesure du 2026-09-21 sur les 92 artefacts rangés : après le correctif, la
 * queue plafonne à 4,94 % des vies retenues et la plus faible vraie arme de base tient 5,06 %.
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
 * prisesEnVie — l'instant de la PREMIÈRE prise d'arme de chaque vie, par slot.
 *
 * POURQUOI TROIS CANAUX, ET POURQUOI LEUR UNION. Aucun ne couvre seul le match : sur le témoin
 * `b1ad85eb`, `pickups` écrit 118 prises d'arme, `weaponChanges` 29 prises et 6 échanges, et la
 * chaîne des objets au sol nomme 13 preneurs. Les trois disent la même chose — « cette vie a
 * acquis une arme à cet instant » — et l'union est le négatif le plus large que le film serve.
 * Elle ne sert qu'à DISQUALIFIER une émission d'équipement de départ ; elle n'invente jamais
 * une arme de base.
 *
 * LA DOTATION DE RÉAPPARITION EST ÉCARTÉE ICI : le jeu remet les armes de départ en main à
 * l'apparition, et `pickups` l'écrit comme une prise datée du début de la vie (huit prises à
 * t = 0 pour les huit premières vies de b1ad85eb). Sans ce filtre, les vies qui partent le plus
 * proprement seraient les premières écartées.
 *
 * JUMEAU de `prisesPour` (Go, internal/sync/replayartifacts/padtiers.go) : mêmes canaux, même
 * comparaison au début de la vie. Toute divergence est un bug.
 */
function prisesEnVie(doc: ReplayDocumentReady): Map<number, number> {
  const debut = new Map<number, number>()
  for (const t of doc.tracks) {
    const t0 = t.points[0]?.t
    if (t0 === undefined) continue
    const vu = debut.get(t.slot)
    if (vu === undefined || t0 < vu) debut.set(t.slot, t0)
  }
  const prises = new Map<number, number>()
  const ajoute = (slot: number, instant: number) => {
    const d = debut.get(slot)
    if (d !== undefined && instant <= d) return
    const vu = prises.get(slot)
    if (vu === undefined || instant < vu) prises.set(slot, instant)
  }
  for (const p of doc.pickups) if (p.kind === 'weapon') ajoute(p.slot, p.t)
  for (const c of doc.weaponChanges) {
    if (c.kind === 'taken' || c.kind === 'swapped') ajoute(c.slot, c.t)
  }
  for (const g of doc.groundWeapons) if (g.picker >= 0) ajoute(g.picker, g.t1)
  return prises
}

/**
 * baseWeaponsOf — les armes de départ du match.
 *
 * UNE SEULE ÉMISSION COMPTE PAR SLOT, LA PREMIÈRE, et c'est mesuré : un `slot` est une VIE (100
 * slots distincts pour 113 pistes sur le match témoin `01e1f945`), et le canal RÉ-ÉMET en cours
 * de vie après un changement d'arme. Prendre tout le canal dilue les trois armes de base de
 * 94,4 % à 85,6 % en classé et fait monter le S7 Sniper de 0,73 % à 3,15 % — autrement dit, il
 * transforme une arme de puissance en arme de base.
 *
 * ET CETTE PREMIÈRE ÉMISSION N'EST PAS UNE LECTURE D'APPARITION : la grille d'images-clés est
 * GLOBALE (cf. `BASE_SHARE_MIN`). Une vie dont la première émission SUIT une prise d'arme
 * n'apprend rien sur les départs — elle quitte le numérateur ET le dénominateur, parce que le
 * silence n'est pas un zéro.
 */
function baseWeaponsOf(doc: ReplayDocumentReady): { weapons: Set<string>; lives: number } {
  const prises = prisesEnVie(doc)
  const premiere = new Map<number, string[]>()
  const ecartees = new Set<number>()
  for (const lo of doc.loadouts) {
    if (premiere.has(lo.slot) || ecartees.has(lo.slot)) continue
    const prise = prises.get(lo.slot)
    if (prise !== undefined && lo.t >= prise) {
      ecartees.add(lo.slot)
      continue
    }
    premiere.set(lo.slot, lo.w)
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
