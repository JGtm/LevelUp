/**
 * deadTimeLogic.ts — LE TEMPS MORT D'UN JOUEUR, lu dans les vies que le film publie déjà.
 *
 * CE QU'ON MESURE, ET RIEN D'AUTRE : la somme des intervalles entre la FIN d'une vie (la mort)
 * et le DÉBUT de la vie suivante du même joueur. Ni intervalle de tête — avant sa première vie
 * un joueur n'est pas mort, il n'est pas encore entré — ni intervalle de queue : un survivant
 * de fin de partie, comme un joueur qui quitte le match, n'accumule rien après sa dernière vie.
 * Ces deux bornes ne sont pas un détail d'implémentation, ce sont les deux endroits où un
 * compteur naïf inventerait des minutes que personne n'a passées à terre.
 *
 * ET SURTOUT : UN TROU ENTRE DEUX VIES NOMMÉES N'EST PAS UNE MORT, c'est une ABSENCE DE VIE
 * NOMMÉE. Le pont slot -> xuid du film est incomplet sur une partie du corpus : des traces
 * réelles arrivent SANS xuid (`buildPlayers` ne les attribue à personne, à dessein — on ne
 * joint jamais sur un rang). Quand une de ces vies anonymes vit à l'intérieur du trou d'un
 * joueur, le film montre quelqu'un qui court là où ce module lirait « à terre ». Mesuré sur
 * les artefacts servis : `64e8adfa` / flamesamurai, 5:11 « de temps mort » dont ~63 % couverts
 * par la trace anonyme du slot 607 ; `000d5950` / JGtm, le plus long trou (54,2 s) contient la
 * trace anonyme du slot 588 ; 6 artefacts sur 21 à huit joueurs sont touchés.
 *
 * LA RÈGLE QUI EN SORT — on n'affiche jamais un nombre non prouvé (même doctrine que le
 * bandeau de score, masqué quand son horloge n'est pas recalée) : dès qu'une trace SANS xuid
 * chevauche un seul des trous d'un joueur, sa mesure est REFUSÉE (`null`), et la fiche écrit
 * « non mesurable ». Pas d'attribution devinée, pas de soustraction de la portion douteuse :
 * rafistoler un chiffre faux le rendrait invérifiable, pas juste. Même refus pour un joueur du
 * roster SANS AUCUNE vie : le film ne l'a jamais situé, « 00:00 » se lirait « jamais à terre ».
 *
 * AUCUNE CONSTANTE DE RÉAPPARITION N'ENTRE ICI. Le délai de retour mesuré sur le film de
 * référence (médiane 8,0 s, 66 épisodes sur 82 à 7,9-8,0 s) est un PALIER OBSERVÉ sur un match,
 * pas une règle du jeu : le multiplier par le nombre de morts donnerait un chiffre lisse et
 * faux. On additionne des intervalles LUS, ce qui rend au passage sa juste valeur au cas
 * intéressant — la mort dont personne ne revient.
 *
 * POURQUOI LES VIES ET PAS LES MORTS DU FIL. Les vies par joueur sont déjà dérivées par
 * `buildPlayers` (rosterLogic) : elles servent la vitalité, les croix de mort et la marque du
 * tireur. Repartir du fil d'éliminations serait une deuxième source pour la même grandeur — et
 * elle serait moins bonne, puisqu'elle ne date pas les retours.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import { frameToMs, trackWindow } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'

/** Une fenêtre de temps bornée à la partie, en images. */
interface Span {
  start: number
  end: number
}

/**
 * deadTimeByPlayer rend, par xuid, le temps mort cumulé en MILLISECONDES de temps réel —
 * ou `null` quand la mesure est REFUSÉE (cf. l'en-tête : vie anonyme dans un trou, ou joueur
 * sans aucune vie). `null` n'est pas zéro : c'est « le film ne permet pas de le dire ».
 *
 * Un joueur mesurable et jamais mort vaut 0, et c'est bien une MESURE : la fiche l'écrit
 * « 00:00 ». La conversion en temps réel passe par `frameToMs`, donc par l'échelle temporelle
 * de l'artefact (`frameIntervalMs`) : sans elle, l'axe T est un index et la cadence de repli
 * s'applique — la même que celle de toute la page.
 */
export function deadTimeByPlayer(
  players: readonly ReplayPlayer[],
  doc: ReplayDocumentReady,
): Map<string, number | null> {
  const lastFrame = Math.max(0, doc.frameCount - 1)
  // LES VIES QUE LE PONT N'A PAS NOMMÉES, une fois pour tout le document : ce sont elles qui
  // invalident les trous. Une trace sans xuid n'appartient à personne, mais elle prouve que
  // QUELQU'UN était en jeu à ce moment-là.
  const orphans = doc.tracks.filter((t) => !t.xuid).map((t) => spanOf(t, lastFrame))
  const out = new Map<string, number | null>()
  for (const p of players) {
    const frames = deadFrames(p, lastFrame, orphans)
    out.set(p.xuid, frames === null ? null : frameToMs(frames, doc))
  }
  return out
}

/**
 * deadFrames additionne les TROUS entre les vies d'un joueur, en images — ou rend `null` si
 * l'un d'eux n'est pas interprétable.
 *
 * TRI DÉFENSIF ET COUVERTURE COURANTE. `buildPlayers` trie déjà les vies par départ, mais ce
 * module ne s'appuie pas dessus : il retrie, et il suit la fin de couverture la plus lointaine
 * atteinte (`covered`) au lieu de comparer chaque vie à la seule précédente. Sans cela, deux
 * vies qui se CHEVAUCHENT — anomalie que le film peut produire quand un slot est réattribué à
 * cheval — fabriqueraient un trou là où le joueur était vivant tout du long.
 *
 * BORNAGE À LA FENÊTRE DU MATCH : une vie dont le film déborde le nombre d'images publié
 * (`frameCount`) est ramenée dans la fenêtre. Un temps mort ne peut pas dépasser la partie.
 */
function deadFrames(
  player: ReplayPlayer,
  lastFrame: number,
  orphans: readonly Span[],
): number | null {
  // AUCUNE VIE : le film n'a jamais situé ce joueur du roster. Refus, pas zéro.
  if (player.lives.length === 0) return null
  const windows = player.lives
    .map((life) => spanOf(life, lastFrame))
    .sort((a, b) => a.start - b.start)
  let total = 0
  // -1 = aucune vie vue : le premier tour ne compte donc AUCUN intervalle de tête.
  let covered = -1
  for (const w of windows) {
    if (covered >= 0 && w.start > covered) {
      if (overlapsAny(orphans, covered, w.start)) return null
      total += w.start - covered
    }
    if (w.end > covered) covered = w.end
  }
  return total
}

/**
 * overlapsAny — une vie anonyme vit-elle DANS le trou `]start, end[` ?
 *
 * INTERSECTION STRICTEMENT POSITIVE : une trace qui ne fait que toucher la borne du trou (elle
 * finit là où il commence, ou commence là où il finit) ne prouve rien sur son intérieur, et
 * refuser la mesure pour un contact de largeur nulle rendrait la ligne muette partout.
 */
function overlapsAny(orphans: readonly Span[], start: number, end: number): boolean {
  return orphans.some((o) => Math.min(end, o.end) - Math.max(start, o.start) > 0)
}

/** spanOf borne la vie d'une trace à la fenêtre publiée du match. */
function spanOf(track: ReplayTrackReady, lastFrame: number): Span {
  const w = trackWindow(track)
  return { start: clamp(w.start, 0, lastFrame), end: clamp(w.end, 0, lastFrame) }
}

function clamp(v: number, min: number, max: number): number {
  return v < min ? min : v > max ? max : v
}

/**
 * DEAD_TIME_UNMEASURABLE — ce qu'écrit une mesure REFUSÉE. Un tiret, jamais un zéro ni une
 * estimation : la fiche dit son ignorance à la place de la combler. Le POURQUOI, lui, est du
 * texte traduit et vit dans les tables i18n (`deadTimeUnmeasurable`), pas ici.
 */
export const DEAD_TIME_UNMEASURABLE = '—'

/**
 * formatDeadTime rend un cumul en `mm:ss`, minutes COMPLÉTÉES par un zéro — ou le tiret du
 * refus quand la mesure est `null`.
 *
 * POURQUOI PAS `formatClock` (replayLogic), qui rend `m:ss` : ce n'est pas un instant du
 * chronomètre mais un CUMUL, affiché en colonne sous d'autres fiches — sans le zéro de tête,
 * « 9:04 » et « 12:07 » ne s'alignent pas, et l'oeil compare mal. Le zéro n'est donc pas un
 * ornement, il est ce qui rend la colonne lisible. Un match sans mort s'écrit « 00:00 ».
 *
 * Troncature à la seconde, comme le chronomètre : on affiche les secondes ÉCOULÉES.
 */
export function formatDeadTime(ms: number | null): string {
  if (ms === null) return DEAD_TIME_UNMEASURABLE
  const total = Math.floor(Math.max(0, ms) / 1000)
  return `${pad(Math.floor(total / 60))}:${pad(total % 60)}`
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}
