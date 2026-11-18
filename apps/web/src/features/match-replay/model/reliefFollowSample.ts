/**
 * reliefFollowSample.ts — CE QUE LE SUIVI DE L'ÉTAGE LIT DU JOUEUR OUVERT à une image : sa vie, sa position,
 * la hauteur de ses pieds, l'instant en millisecondes du REJEU. Logique pure, sans React ni canvas.
 *
 * C'est la jonction entre le document du film et `stepFollow` (`lib/replay/relief/follow.ts`), qui ne connaît
 * ni pistes ni véhicules. Trois règles, toutes celles de l'anneau du joueur ouvert (`openPlayerRing.ts`) :
 *
 *  - VIVANT = une vie du joueur couvre l'image (`isAliveAt`). Mort, pas encore réapparu : l'échantillon est
 *    « silencieux » (`alive: false`), la coupe ne bouge pas. La fenêtre après-mort des croix de mort ne compte
 *    pas ici : un joueur mort n'est nulle part ;
 *  - EMBARQUÉ : la position est celle du véhicule (`buildEmbarkedPosAt`), et la hauteur AUSSI
 *    (`buildEmbarkedAltitudeAt`) — jamais le z du bipède, qui ne réplique plus pendant l'épisode et nommerait
 *    l'étage du point d'embarquement. Un véhicule sans hauteur lue rend un échantillon silencieux : mieux vaut
 *    laisser la coupe où elle est que la poser sur un étage inventé ;
 *  - LA VIE EST SON RANG dans `player.lives` : une autre vie est une « première image d'une vie », donc une
 *    prise d'étage immédiate à la réapparition (décision D.1, choix 5).
 *
 * LE TEMPS EST CELUI DU REJEU (`frameToMs`), pas celui de l'horloge : le délai de 300 ms de `stepFollow` se
 * compte en temps de match. À ×4 comme à ×0,5, la même chute change l'étage au même endroit du match ; en
 * pause le délai ne s'écoule pas.
 */
import { altitudeAt, frameToMs, isAliveAt, positionAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { FollowSample } from '../../../lib/replay/relief'
import { buildEmbarkedAltitudeAt, buildEmbarkedPosAt } from './carrierPosition'

/** La lecture du joueur ouvert à une image (fractionnaire admise). */
export type FollowSampler = (frame: number) => FollowSample

/** Rang de vie des échantillons silencieux : aucune vie ne le porte. */
const NO_LIFE = -1

/** Un échantillon qui dit « je ne lis rien » : `stepFollow` laisse la coupe où elle est. */
function silent(atMs: number): FollowSample {
  return { alive: false, lifeKey: NO_LIFE, x: 0, y: 0, atMs }
}

/**
 * buildFollowSampler bâtit la lecture d'un joueur pour ce document : les index d'épisodes d'occupation se
 * construisent UNE fois, la lecture elle-même ne fait que chercher la vie et interpoler.
 */
export function buildFollowSampler(doc: ReplayDocumentReady, player: ReplayPlayer): FollowSampler {
  const embarkedAt = buildEmbarkedPosAt(doc.vehicles)
  const embarkedAltitudeAt = buildEmbarkedAltitudeAt(doc.vehicles)
  return (frame) => {
    const atMs = frameToMs(frame, doc)
    const life = player.lives.findIndex((l) => isAliveAt(l, frame))
    if (life < 0) return silent(atMs)
    const points = player.lives[life].points
    const at = embarkedAt(player.xuid, frame) ?? positionAt(points, frame)
    // `undefined` = pas embarqué (le bipède répond) ; `null` = embarqué sur une monture sans hauteur lue.
    const onBoard = embarkedAltitudeAt(player.xuid, frame)
    if (!at || onBoard === null) return silent(atMs)
    return { alive: true, lifeKey: life, x: at.x, y: at.y, z: onBoard ?? altitudeAt(points, frame), atMs }
  }
}
