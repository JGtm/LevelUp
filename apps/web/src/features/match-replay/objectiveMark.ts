/**
 * objectiveMark.ts — QUI PORTE L'OBJECTIF, à une image donnée. Logique pure, pas de React.
 *
 * CE QUE LA FICHE NE DISAIT PAS. La carte montre depuis longtemps le drapeau porté, le crâne
 * d'Oddball et la couronne VIP — mais sur leur MARQUEUR, au milieu des traces. La colonne des
 * fiches, elle, ne disait rien : on lisait « qui est vivant », jamais « qui tient l'objet qui
 * décide de la manche ». Ce module répond à cette question, et lui seule ; l'habillage
 * (filigrane, encre) vit dans `ReplayObjectiveMark.tsx`, la phrase d'infobulle dans
 * `playerCardFx.titleOf`.
 *
 * DEUX RÉGIMES, PARCE QUE LA DONNÉE EN A DEUX (décision utilisateur du 2026-08-29) :
 *
 *  - un PORTAGE est une PÉRIODE attribuée — `flagCarries[].spans[]` (schéma 15),
 *    `skullCarries[]` (23), `vipCrown[]` (22) portent tous un `xuid` et un intervalle. La
 *    marque dure exactement l'intervalle : c'est un état, il se lit comme tel.
 *  - un ÉVÉNEMENT INSTANTANÉ n'a PAS de période attribuée. `zoneStates[].spans[]` ne publie
 *    que le propriétaire (une ÉQUIPE) et la jauge — aucun xuid ; seuls les ÉVÉNEMENTS
 *    `zone_captures` et `zone_secures` ont un auteur. La marque est alors un INSTANT tenu
 *    quelques secondes, jamais un « est en train de capturer » : le déduire de la position
 *    serait affirmer ce que le film ne dit pas (règle « une valeur non lue s'affiche comme
 *    une lacune »).
 *
 * LA BOMBE A LES DEUX RÉGIMES depuis le schéma 30 : le PORT est une période attribuée
 * (`bombCarries[]`, canal des armes tenues — jusque-là « rien n'attribue le PORT de la
 * bombe » était vrai, et ne l'est plus), et l'EXPLOSION reste l'instant `bomb_detonations`
 * (le point de mode d'Assaut). Le portage prime, comme partout : un porteur marqué reste
 * marqué pendant tout son portage, l'explosion tient sa marque quelques secondes après le
 * geste — même glyphe, deux moments d'un même enjeu.
 *
 * KOTH EST PRÊT ET ATTEND SA SOURCE. Le mode n'a AUCUNE donnée attribuée aujourd'hui : les
 * emplacements de statistiques `hill` ne sont pas encore nommés — `NamedEvents` rend `nil`
 * pour KOTH — et les périodes de colline du document sont, comme les zones, au niveau de
 * l'équipe. Le genre `hill` existe donc ici avec son glyphe, son encre, son libellé et son
 * test (il passe par le MÊME résolveur de périodes que le crâne et le VIP, `markFromPeriods`),
 * mais sa source est vide : le jour où le décodage KOTH publiera des occupations par joueur,
 * c'est UNE ligne à ajouter dans `carrySourcesOf` — rien d'autre à écrire.
 *
 * UNE SEULE ENCRE POUR LES SIX (règle color-tokens) : `extreme`, « rare et intense », le
 * sommet d'une rampe d'intensité — et le seul token de la palette qu'AUCUN autre effet de
 * fiche n'emploie (`legendary` est le surbouclier, `success` le champ de réparation,
 * `destructive` le capteur et la mort, `info` la jauge de bouclier). Porter l'objectif est un
 * état de jeu, pas un camp : c'est le GLYPHE qui dit lequel des six, pas la couleur. La carte,
 * elle, garde l'encre d'ÉQUIPE sur ses glyphes — elle doit distinguer deux drapeaux sur un même
 * fond, quand une fiche n'a jamais qu'un seul porteur.
 */
import { filmClockTrusted } from '@/lib/replay/scoreTimeline'

import { msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/**
 * Les six marques, nommées par l'OBJET porté, l'endroit tenu ou le geste qui vient d'être fait
 * — jamais par le mode : un même objet se retrouve d'un mode à l'autre, et la fiche ne connaît
 * pas le mode.
 */
export type ObjectiveMarkKind = 'flag' | 'skull' | 'vip' | 'hill' | 'zone' | 'bomb'

/**
 * Les états de `flagCarries[].spans[].state` qui veulent dire PORTÉ. `carried_open` est le
 * drapeau porté « à découvert » (le porteur est visible de tous) : c'est un portage, avec la
 * même conséquence pour qui lit la fiche. `dropped` et `home` n'ont pas de porteur — leur
 * `xuid` est celui du DERNIER porteur, et le prendre pour un portage collerait le drapeau à un
 * joueur qui ne l'a plus.
 */
const CARRIED_STATES = new Set(['carried', 'carried_open'])

/**
 * Tenue de la marque de PRISE DE BASE, en temps réel — un instant qui doit se voir sans
 * s'installer. 2,5 s : le temps de lire un nom sur une colonne de huit fiches, bien au-delà de
 * l'onde de capture du drapeau (600 ms, `flagCaptureFx`) qui, elle, est peinte SUR la carte à
 * l'endroit où l'œil regarde déjà. Déclarée en millisecondes, jamais en images : la cadence du
 * film peut changer au build sans que la lecture change (même règle que `useReplayTiming`).
 */
export const ZONE_MARK_HOLD_MS = 2_500

/**
 * Les ÉVÉNEMENTS INSTANTANÉS attribués à un joueur, par genre de marque (cf.
 * objectiveevents/named.go). Ce sont les seuls modes dont la donnée ne publie AUCUNE période :
 * la marque est un instant tenu quelques secondes, jamais un état.
 *
 * `bomb_detonations` (Assaut, depuis le 2026-08-31) : le point de mode d'une manche d'Assaut
 * vaut une EXPLOSION, et rien d'autre ne fait bouger le score. Le joueur crédité est celui que
 * le moteur crédite ; le film ne distingue pas l'armement de la détonation, et la fiche ne
 * l'affirme donc pas.
 */
const EVENT_STATS: Record<'zone' | 'bomb', ReadonlySet<string>> = {
  zone: new Set(['zone_captures', 'zone_secures']),
  bomb: new Set(['bomb_detonations']),
}

/** Une période de portage attribuée : la forme commune du crâne, du VIP et (demain) de la colline. */
interface CarryPeriod {
  xuid: string
  t0: number
  t1: number
}

/**
 * markFromPeriods — le résolveur de PÉRIODES, partagé par tous les portages à intervalle plat.
 *
 * Bornes INCLUSES aux deux bouts, comme `skullPresenceAt` : une période d'une seule image
 * (prise et perte dans le même pas) doit se voir, et c'est la convention de tout le rejeu.
 */
function markFromPeriods(
  periods: readonly CarryPeriod[],
  xuid: string,
  frame: number,
): boolean {
  for (const p of periods) {
    if (p.xuid === xuid && p.t0 <= frame && frame <= p.t1) return true
  }
  return false
}

/**
 * Les sources de PORTAGE du document, dans l'ordre où elles se disputent la fiche. L'ordre est
 * arbitraire dans les faits — aucun mode ne publie deux de ces listes à la fois — mais il est
 * FIXE, pour qu'un document mal formé rende toujours la même marque plutôt qu'une au hasard.
 */
function carrySourcesOf(
  doc: ReplayDocumentReady,
): readonly { kind: ObjectiveMarkKind; periods: readonly CarryPeriod[] }[] {
  return [
    { kind: 'skull', periods: doc.skullCarries },
    { kind: 'vip', periods: doc.vipCrown },
    // LA BOMBE PORTÉE (schéma 30) : le même résolveur que le crâne — le kind `bomb` sert
    // aussi l'ÉVÉNEMENT d'explosion (EVENT_STATS), et le portage prime par construction :
    // les périodes se lisent AVANT les événements dans `objectiveMarkAt`.
    { kind: 'bomb', periods: doc.bombCarries },
    // KOTH : la source manque encore (cf. l'en-tête). Une ligne `{ kind: 'hill', periods: ... }`
    // ici, et le filigrane de colline s'allume — tout le reste est déjà en place.
  ]
}

/**
 * flagCarriedBy dit si CE joueur tient un drapeau à cette image, quel que soit le camp du
 * drapeau : sur une fiche, le camp est déjà dit par la colonne.
 */
function flagCarriedBy(doc: ReplayDocumentReady, xuid: string, frame: number): boolean {
  for (const carry of doc.flagCarries) {
    for (const sp of carry.spans) {
      if (sp.xuid !== xuid) continue
      if (!CARRIED_STATES.has(sp.state)) continue
      if (sp.t0 <= frame && frame <= sp.t1) return true
    }
  }
  return false
}

/**
 * eventMarkedBy dit si ce joueur a accompli l'un des événements du genre donné dans la fenêtre
 * qui précède cette image.
 *
 * LA GARDE D'HORLOGE EST LA MÊME QUE CELLE DE L'ONDE DE CAPTURE (`flagCaptureFx`) : les
 * `objectives[]` sont datés par l'horloge du FILM puis recalés sur la grille du document ; sans
 * origine résolue, l'écart est inconnu (mesuré de 3,6 s à 50,8 s) et la marque s'allumerait sur
 * la mauvaise image. Muet vaut mieux que faux.
 */
function eventMarkedBy(
  doc: ReplayDocumentReady,
  xuid: string,
  frame: number,
  hold: number,
  stats: ReadonlySet<string>,
): boolean {
  if (!filmClockTrusted(doc)) return false
  for (const a of doc.objectives) {
    if (a.xuid !== xuid) continue
    if (!stats.has(a.stat)) continue
    const age = frame - a.t
    if (age >= 0 && age <= hold) return true
  }
  return false
}

/**
 * objectiveMarkAt — la marque d'objectif d'un joueur à une image, ou `null` s'il n'en porte
 * aucune.
 *
 * LE PORTAGE PRIME SUR L'ÉVÉNEMENT : un porteur de drapeau qui vient de sécuriser une base
 * garde le drapeau à l'écran — c'est l'état qui dure qui décide de la manche, l'instant est
 * déjà passé.
 *
 * L'APPELANT NE L'APPELLE QUE POUR UNE FICHE VIVANTE (même règle que les zones et l'équipement) :
 * une fiche morte ne porte que la mort, et un mort a lâché ce qu'il tenait.
 */
export function objectiveMarkAt(
  doc: ReplayDocumentReady,
  xuid: string,
  frame: number,
): ObjectiveMarkKind | null {
  if (flagCarriedBy(doc, xuid, frame)) return 'flag'
  for (const src of carrySourcesOf(doc)) {
    if (markFromPeriods(src.periods, xuid, frame)) return src.kind
  }
  const hold = Math.max(1, msToFrames(ZONE_MARK_HOLD_MS, doc))
  if (eventMarkedBy(doc, xuid, frame, hold, EVENT_STATS.zone)) return 'zone'
  if (eventMarkedBy(doc, xuid, frame, hold, EVENT_STATS.bomb)) return 'bomb'
  return null
}

/**
 * Exporté POUR LE GENRE `hill` ET SON TEST, le temps que la source KOTH existe : c'est ce qui
 * fait de la colline un genre réellement résolu par le même code que les autres, et non un
 * décor qui divergerait avant d'avoir servi. À l'arrivée de la source, `carrySourcesOf` s'en
 * sert comme des deux autres et cet export perd sa raison d'être.
 */
export { markFromPeriods as objectiveMarkFromPeriods }
