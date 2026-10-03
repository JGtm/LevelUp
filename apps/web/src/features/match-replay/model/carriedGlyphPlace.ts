/**
 * carriedGlyphPlace.ts — LA RÈGLE UNIQUE : où se dessine un objet d'objectif PORTÉ quand son
 * porteur n'a PAS de position à l'image servie.
 *
 * LE FAIT, MESURÉ. Une période de portage est un INTERVALLE ; la trajectoire du porteur, elle,
 * est une suite d'échantillons qui peut s'interrompre (vie non répliquée, trou de réplication du
 * bipède). L'audit du 2026-09-10 a compté ces images : **533 sur 32 464** pour le drapeau
 * (1,64 %, §3.5) et **694** pour le crâne d'Oddball (7,0 % du parc, rapport 6.2 §2.1). Le
 * portage est VRAI sur ces images — c'est la position qui manque, pas le fait.
 *
 * CE QUE LES TROIS CALQUES FAISAIENT, ET QUI ÉTAIT FAUX DE DEUX FAÇONS DIFFÉRENTES :
 *  - crâne (`skullCarrierLayer`) et bombe (`bombCarrierLayer`) : `if (!w) continue` — l'objet
 *    DISPARAÎT de l'écran, alors qu'il est quelque part et que le film le sait ;
 *  - drapeau (`flagCarriesLayer`) : repli sur `{x: now.x, y: now.y}`, l'ANCRE DU SPAN — une
 *    position PÉRIMÉE, dessinée avec l'habillage d'un objet PORTÉ. Un objet immobile faux se lit
 *    comme un fait, ce qui est pire qu'un trou.
 *
 * LA RÈGLE, LA MÊME POUR LES TROIS : sans position du porteur, l'objet est rendu **LIBRE à sa
 * DERNIÈRE POSITION CONNUE** — la dernière position du porteur depuis le début du portage, à
 * défaut le dernier repos connu de l'objet (l'appelant le sert : socle du crâne, ancre du span
 * du drapeau). « Libre » n'est pas un aveu d'ignorance : c'est la seule chose vraie qu'on puisse
 * dire — l'objet est à cet endroit-là, et personne n'est dessiné dessous.
 *
 * POURQUOI L'ÉTAT VISUEL CHANGE, ET PAS SEULEMENT LA POSITION. L'habillage « porté » désigne un
 * JOUEUR : le décalage du drapeau existe pour ne pas recouvrir le pion du porteur
 * (`glyphIsOffset`), le crâne se pose au-dessus du marqueur. Sans porteur à l'écran, ces
 * décorations pointent le vide. L'habillage « libre » désigne un LIEU, et c'est ce que la
 * position servie ici est devenue.
 *
 * UN SEUL GLYPHE PAR OBJET ET PAR IMAGE, c'est l'invariant que cette règle protège : l'état
 * rendu ici est EXCLUSIF (`carried` XOR `free` XOR `absent`), et les calques d'objet libre s'y
 * branchent (cf. `skullPresenceAt`, précédence du portage) plutôt que de deviner chacun de leur
 * côté. Deux glyphes du même objet à la même image seraient deux affirmations contradictoires.
 *
 * LE BALAYAGE ARRIÈRE NE SORT JAMAIS DU PORTAGE : il s'arrête à `t0`. Remonter au-delà servirait
 * la position d'AVANT la prise — c'est-à-dire le dernier repos, que l'appelant sert déjà
 * explicitement s'il le connaît. Coût : le trou mesuré le plus long du parc tient en quelques
 * dizaines d'images, et un portage dont le porteur n'est JAMAIS localisable sort au premier tour
 * par le repli (`xuid` nul) ou au terme d'un seul balayage de la période.
 */
import type { XY } from '../../../lib/replay/replayLogic'

/** La lecture de position d'un porteur à une image — celle de `carrierPosition.ts`. */
export type CarrierPosAt = (xuid: string, frame: number) => XY | null

/**
 * CarriedGlyphPlace — où et COMMENT se dessine l'objet à cette image. Les trois états sont
 * EXCLUSIFS : c'est ce qui interdit deux glyphes du même objet à la même image.
 */
export type CarriedGlyphPlace =
  /** Le porteur est localisable : l'objet se dessine SUR lui, habillage « porté ». */
  | { state: 'carried'; at: XY }
  /** Le porteur n'a pas de position : l'objet se dessine LIBRE, à sa dernière place connue. */
  | { state: 'free'; at: XY }
  /** Aucune position connue, ni du porteur ni de l'objet : on n'invente rien. */
  | { state: 'absent' }

/** Le portage réduit à ce dont la règle a besoin : QUI porte, et DEPUIS QUELLE image. */
export interface CarriedSpan {
  /** Le porteur. `null` = l'artefact ne le nomme pas : aucune trajectoire à relire. */
  xuid: string | null
  /** Première image du portage : borne du balayage arrière, jamais franchie. */
  t0: number
}

/**
 * carriedGlyphPlaceAt applique la règle, et c'est son UNIQUE écriture (garde-rail :
 * `carriedGlyphPlace.guard.test.ts`).
 *
 * `fallback` est le dernier REPOS connu de l'objet lui-même, servi par l'appelant quand il en
 * connaît un : socle du crâne, ancre du span pour le drapeau. `null` quand l'objet n'a aucun
 * canal de position propre — c'est le cas de la bombe (`document_bomb_carries.go` : l'objet
 * bombe n'a pas de canal mesuré côté Go).
 */
export function carriedGlyphPlaceAt(
  posOf: CarrierPosAt,
  span: CarriedSpan,
  frame: number,
  fallback: XY | null,
): CarriedGlyphPlace {
  const here = span.xuid ? posOf(span.xuid, frame) : null
  if (here) return { state: 'carried', at: here }
  const back = span.xuid ? lastKnownCarrierPos(posOf, span.xuid, span.t0, frame) : null
  if (back) return { state: 'free', at: back }
  return fallback ? { state: 'free', at: fallback } : { state: 'absent' }
}

/**
 * lastKnownCarrierPos remonte image par image de `frame - 1` jusqu'à `since` (inclus) et rend la
 * première position trouvée, ou `null`. Le pas est l'image parce que c'est la seule unité que le
 * calque connaisse — il ne reçoit jamais l'horloge du document.
 */
function lastKnownCarrierPos(
  posOf: CarrierPosAt,
  xuid: string,
  since: number,
  frame: number,
): XY | null {
  for (let f = frame - 1; f >= since; f -= 1) {
    const p = posOf(xuid, f)
    if (p) return p
  }
  return null
}
