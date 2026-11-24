/**
 * playerState.ts — CE QU'UN JOUEUR EST À UNE IMAGE, lu dans le film : vivant ou mort, bouclier, santé, délai de
 * réapparition. Le joueur (`ReplayPlayer`) et sa jointure à la base sont dans `rosterLogic.ts` ; les lectures par
 * SLOT (armes portées, capacité) sont dans `slotReadings.ts`.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import { heldReading, isAliveAt, trackWindow } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'

/**
 * PlayerState — ce qu'un joueur est à une image donnée, lu dans le film.
 *
 * `shield` est null quand aucune mesure n'existe dans la fenêtre de maintien : c'est
 * « on ne sait pas », à distinguer absolument d'un bouclier à zéro, qui est une mesure.
 */
export interface PlayerState {
  alive: boolean
  /** La vie en cours, quand il y en a une. */
  life: ReplayTrackReady | null
  /**
   * Bouclier : la dernière mesure de la vie avec son âge, ou 1,0 (âge 0) avant la première
   * mesure — on apparaît plein, et le flux différentiel ne retransmet que ce qui change.
   * Null UNIQUEMENT quand le document ne porte pas ce champ (cf. VitalityPresence) ou que
   * le joueur est mort.
   */
  shield: { value: number; age: number } | null
  /**
   * Santé, même contrat que le bouclier — y compris le plein d'apparition. Elle est
   * répliquée AU CHANGEMENT et rarement transmise (médiane zéro échantillon par vie sur le
   * film de référence) : dans Halo le bouclier encaisse d'abord, une vie sans mesure de
   * santé est une vie restée intacte.
   */
  health: { value: number; age: number } | null
  /** Nombre d'images écoulées depuis la fin de la dernière vie (mort) ; -1 s'il est en vie. */
  sinceDeath: number
  /**
   * Image de RÉAPPARITION, LUE : c'est l'image de départ de la vie suivante du même joueur, pas
   * une constante ajoutée à l'instant de la mort. -1 quand aucune vie ne suit — ce qui arrive
   * pour un survivant de fin de partie, et doit s'afficher comme une lacune, jamais comme un
   * délai deviné.
   */
  respawnFrame: number
  /**
   * Le joueur est PARTI : sa dernière vie est finie par un départ et sa présence au tableau n'est pas
   * encore fermée. Il n'est pas mort — la fiche dit qu'il a quitté la partie, sans compte à rebours.
   * Décidé par l'appelant (`departed` de `playerStateAt`) ; toujours faux en vie.
   */
  departed: boolean
}

/**
 * VitalityPresence — le document porte-t-il la donnée, champ par champ ? C'est la garde
 * multi-titre du « plein au spawn » : un titre dont le film ne transmet JAMAIS une
 * vitalité ne doit pas afficher des barres pleines inventées — il n'affiche pas de barre.
 * Dégradation par ABSENCE DE DONNÉE, jamais par comparaison de slug.
 */
export interface VitalityPresence {
  shield: boolean
  health: boolean
}

/** vitalityPresence mesure, une fois par document, quels champs de vitalité existent. */
export function vitalityPresence(doc: ReplayDocumentReady): VitalityPresence {
  let shield = false
  let health = false
  for (const t of doc.tracks) {
    for (const p of t.points) {
      if (p.sh !== undefined) shield = true
      if (p.hp !== undefined) health = true
      if (shield && health) return { shield, health }
    }
  }
  return { shield, health }
}

/**
 * playerStateAt lit l'état d'un joueur à une image.
 *
 * LE REPORT DE VITALITÉ COUVRE LA VIE ENTIÈRE, et c'est une lecture juste : le flux est
 * différentiel — non retransmis veut dire INCHANGÉ — et les points appartiennent à la vie,
 * donc le report ne franchit jamais une mort. Ce qui vieillit doit se VOIR : l'âge voyage
 * avec la valeur et l'affichage l'estompe (cf. barres).
 *
 * AVANT LA PREMIÈRE MESURE D'UNE VIE, LA VALEUR JUSTE EST 1,0 : on apparaît vie et bouclier
 * PLEINS (règle du jeu), et le film ne retransmet que ce qui CHANGE — « rien d'arrivé »
 * veut dire « plein », pas « inconnu ». C'est la lecture du POC, rétablie sur décision
 * utilisateur (2026-08-12). Elle est GARDÉE par VitalityPresence : un document qui ne
 * porte jamais le champ n'affiche rien (titre sans décodage film).
 *
 * LE DÉLAI DE RÉAPPARITION EST LU, PAS DÉDUIT. Mesuré sur le film de référence : 90 épisodes de
 * mort, 82 avec un retour lisible, médiane 8,0 s et 66 sur 82 exactement à 7,9-8,0 s. C'est un
 * palier net — mais mesuré sur UN match, ce qui n'en fait pas une constante du jeu. Les 8
 * épisodes sans retour restent sans délai affiché.
 *
 * `departed` : l'appelant dit que ce joueur a QUITTÉ la partie à cette image (cf. `PlayerState.departed`) ;
 * ce fichier ne lit pas la présence du tableau, il porte l'état.
 */
export function playerStateAt(
  player: ReplayPlayer,
  frame: number,
  presence: VitalityPresence,
  departed = false,
): PlayerState {
  const live = player.lives.find((l) => isAliveAt(l, frame)) ?? null
  if (live) {
    const spawnFull = { value: 1, age: 0 }
    const shield = heldReading(live.points, frame, (p) => p.sh, Number.POSITIVE_INFINITY)
    const health = heldReading(live.points, frame, (p) => p.hp, Number.POSITIVE_INFINITY)
    return {
      alive: true,
      life: live,
      shield: shield ?? (presence.shield ? spawnFull : null),
      health: health ?? (presence.health ? spawnFull : null),
      sinceDeath: -1,
      respawnFrame: -1,
      departed: false,
    }
  }
  // Mort : la dernière vie CLOSE avant cette image date la mort ; la suivante, si elle existe,
  // date le retour.
  let last: ReplayTrackReady | null = null
  let next = -1
  for (const l of player.lives) {
    const w = trackWindow(l)
    if (w.end < frame) last = l
    else if (w.start > frame && next === -1) next = w.start
  }
  return {
    alive: false,
    life: null,
    shield: null,
    health: null,
    sinceDeath: last ? frame - trackWindow(last).end : -1,
    respawnFrame: next,
    departed,
  }
}
