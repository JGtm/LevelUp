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
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'

/**
 * deadTimeByPlayer rend, par xuid, le temps mort cumulé en MILLISECONDES de temps réel.
 *
 * Un joueur sans mort — une seule vie, ou aucune — vaut 0 : c'est une mesure, pas une lacune,
 * et la fiche l'écrit « 00:00 ». La conversion en temps réel passe par `frameToMs`, donc par
 * l'échelle temporelle de l'artefact (`frameIntervalMs`) : sans elle, l'axe T est un index et
 * la cadence de repli s'applique — la même que celle de toute la page.
 */
export function deadTimeByPlayer(
  players: readonly ReplayPlayer[],
  doc: ReplayDocumentReady,
): Map<string, number> {
  const lastFrame = Math.max(0, doc.frameCount - 1)
  const out = new Map<string, number>()
  for (const p of players) {
    out.set(p.xuid, frameToMs(deadFrames(p, lastFrame), doc))
  }
  return out
}

/**
 * deadFrames additionne les TROUS entre les vies d'un joueur, en images.
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
function deadFrames(player: ReplayPlayer, lastFrame: number): number {
  const windows = player.lives
    .map((life) => trackWindow(life))
    .map((w) => ({ start: clamp(w.start, 0, lastFrame), end: clamp(w.end, 0, lastFrame) }))
    .sort((a, b) => a.start - b.start)
  let total = 0
  // -1 = aucune vie vue : le premier tour ne compte donc AUCUN intervalle de tête.
  let covered = -1
  for (const w of windows) {
    if (covered >= 0 && w.start > covered) total += w.start - covered
    if (w.end > covered) covered = w.end
  }
  return total
}

function clamp(v: number, min: number, max: number): number {
  return v < min ? min : v > max ? max : v
}

/**
 * formatDeadTime rend un cumul en `mm:ss`, minutes COMPLÉTÉES par un zéro.
 *
 * POURQUOI PAS `formatClock` (replayLogic), qui rend `m:ss` : ce n'est pas un instant du
 * chronomètre mais un CUMUL, affiché en colonne sous d'autres fiches — sans le zéro de tête,
 * « 9:04 » et « 12:07 » ne s'alignent pas, et l'oeil compare mal. Le zéro n'est donc pas un
 * ornement, il est ce qui rend la colonne lisible. Un match sans mort s'écrit « 00:00 ».
 *
 * Troncature à la seconde, comme le chronomètre : on affiche les secondes ÉCOULÉES.
 */
export function formatDeadTime(ms: number): string {
  const total = Math.floor(Math.max(0, ms) / 1000)
  return `${pad(Math.floor(total / 60))}:${pad(total % 60)}`
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}
