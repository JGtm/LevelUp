/**
 * replayAimCone.ts — LE CÔNE DE VISÉE d'un marqueur : son cap, son ÉLÉVATION, et le trait qui
 * dit de quel côté le joueur regarde.
 *
 * EXTRAIT DE `replayMarkers.ts` LE 2026-08-18 (lot R2-V) : le calque des marqueurs avait
 * franchi le seuil de taille du dépôt (CLAUDE.md n°5) et le lot lui ajoutait le double contour
 * du joueur de la page. La découpe tombe sur une frontière nette — le cône ne dessine PAS le
 * joueur, il dessine ce qu'il REGARDE, et il est le seul bloc du fichier à lire deux champs
 * (`h` et `p`) plutôt qu'une position.
 */
import type { MarkerStyle } from './replayMarkers'
import { freshness, heldReading, type XY } from './replayLogic'
import type { ReplayTrackReady } from './replayNormalize'

const AIM_LENGTH = 52
const AIM_HALF_ANGLE = 0.42
const AIM_CONE_ALPHA = 0.55
/** Une visée de 5 s ne vaut pas une visée de l'instant : elle perd 62 % de son opacité. */
const AIM_FADE = 0.62
/**
 * L'ÉLÉVATION DE VISÉE (schéma 13) : le cône garde son ANGLE et raccourcit.
 *
 * `AIM_LENGTH` reste la longueur MAXIMALE — celle d'un joueur qui vise à plat. Le facteur est
 * `cos(p)`, la part horizontale d'un regard incliné, borné en bas pour que le marqueur reste
 * lisible quand la visée est verticale.
 *
 * Ce que la mesure dit du champ (lot E, 3 films) : médiane −4,7 / −3,4 / −3,6°, 67 à 77 % des
 * visées vers le BAS, extrêmes −85,5 à +82°. Le cône passe donc son temps à 99,5 % de sa
 * longueur, et se contracte franchement dans les instants qui comptent — un tir en plongée,
 * un joueur qui couvre une passerelle.
 */
const AIM_PITCH_FLOOR = 0.35
/** Trait de sens, à la pointe du cône : vers l'extérieur = vers le haut, intérieur = vers le bas. */
const AIM_TICK_LENGTH = 6
const AIM_TICK_WIDTH = 1.4
/** Zone morte du trait : sous 2°, la visée se lit à plat et le cône n'a pas bougé (cos 2° = 0,9994). */
const AIM_TICK_DEAD_DEG = 2

/**
 * drawAimCone dessine la DIRECTION DU REGARD, décodée du même record que la position.
 *
 * Le cône se dégrade du centre vers le bord — dense à l'origine, où il faut lire QUI vise,
 * transparent au bout, où il ne faut pas masquer le décor. Il pâlit avec l'âge de la mesure et
 * n'est PAS dessiné au-delà du maintien : passé ce délai, on ne sait plus où le joueur regarde,
 * et une direction périmée affirmerait ce qu'on ignore.
 *
 * IL N'Y A PLUS D'AXE (décision D3, 2026-08-16) : les deux traits qui prolongeaient le point
 * — « le bâton » — ont été supprimés à la demande de l'utilisateur. Ce que le cône seul perd
 * en précision d'angle, la carte le regagne en lisibilité : huit bâtons sur un 4v4 se
 * croisaient au-dessus des noms.
 *
 * SES DIMENSIONS SONT CELLES DE LA PLANCHE, « un peu plus prononcées » (verdict du soir du
 * 2026-08-16, §1bis du plan) : rayon 52, demi-ouverture 0,42 rad, alpha 0,55. Le cône avait
 * d'abord été rétréci à 30 px / 0,30 — trop timide pour se lire une fois les noms posés.
 *
 * DEPUIS LE SCHÉMA 13, IL DIT LES DEUX AXES DU REGARD. Le cap oriente le secteur ; l'ÉLÉVATION
 * (`p`, degrés, positif = vers le haut) le RACCOURCIT — `AIM_LENGTH × max(0,35 ; cos p)` — et
 * un trait posé à sa pointe dit de quel côté. Il faut les deux : le cosinus est pair, donc la
 * longueur seule confond « vise le ciel » et « vise ses pieds ». Un artefact antérieur au
 * schéma 13 ne porte pas `p` : le cône y garde sa pleine longueur, sans tick, et c'est le
 * comportement voulu — absent se lit « à plat », jamais « inconnu ».
 */
export function drawAimCone(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: MarkerStyle,
  color: string,
): void {
  const read = heldReading(track.points, style.frame, (p) => p.h, style.timing.aimHold)
  if (!read) return
  const fresh = freshness(read.age, style.timing.aimHold, AIM_FADE)
  // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi.
  const ang = (-read.value * Math.PI) / 180
  const pitch = heldPitch(track.points, style, read.age)
  const R = AIM_LENGTH * pitchScale(pitch) * style.k
  const gradient = ctx.createRadialGradient(c.x, c.y, 0, c.x, c.y, R)
  gradient.addColorStop(0, color)
  gradient.addColorStop(1, 'transparent')
  ctx.globalAlpha = AIM_CONE_ALPHA * fresh
  ctx.beginPath()
  ctx.moveTo(c.x, c.y)
  ctx.arc(c.x, c.y, R, ang - AIM_HALF_ANGLE, ang + AIM_HALF_ANGLE)
  ctx.closePath()
  ctx.fillStyle = gradient
  ctx.fill()
  drawPitchTick(ctx, c, ang, R, pitch, style, color)
  ctx.globalAlpha = 1
}

/**
 * heldPitch rend l'ÉLÉVATION en vigueur, ou 0 (à plat).
 *
 * LA RÈGLE D'ÂGE N'EST PAS UNE PRÉCAUTION DÉCORATIVE. `p` est omis quand la visée s'arrondit
 * à plat (contrat du champ, cf. `Point.P` côté Go), et `heldReading` remonterait alors
 * jusqu'à un point PLUS ANCIEN qui, lui, porte une élévation : le marqueur afficherait une
 * plongée périmée sur une visée à plat actuelle. Les deux angles venant du MÊME
 * enregistrement, une élévation trouvée plus loin dans le passé que le cap appartient
 * forcément à une autre visée — on la refuse, et « absent » redevient ce qu'il doit être.
 */
function heldPitch(
  points: ReplayTrackReady['points'],
  style: MarkerStyle,
  headingAge: number,
): number {
  const read = heldReading(points, style.frame, (p) => p.p, style.timing.aimHold)
  return read && read.age <= headingAge ? read.value : 0
}

/**
 * pitchScale : ce que l'élévation fait à la LONGUEUR du cône.
 *
 * Le cône est la projection au sol d'un regard qui, lui, vit en trois dimensions. Plus le
 * joueur pique ou lève la tête, moins ce regard porte LOIN SUR LE PLAN — d'où le cosinus, qui
 * est exactement la part horizontale d'une direction inclinée. L'ANGLE, lui, ne bouge pas :
 * c'est toujours le même cap.
 *
 * LE PLANCHER EXISTE POUR QUE LE MARQUEUR RESTE LISIBLE : à ±90° le cosinus s'annule et le
 * cône disparaîtrait, alors qu'un joueur qui vise ses pieds ou le ciel est précisément ce
 * qu'on veut voir. On s'arrête donc à 35 % de la longueur maximale.
 */
function pitchScale(pitchDeg: number): number {
  return Math.max(AIM_PITCH_FLOOR, Math.cos((pitchDeg * Math.PI) / 180))
}

/**
 * drawPitchTick dit le SENS de l'élévation, que la longueur ne peut pas dire.
 *
 * Le cosinus est PAIR : viser 30° au-dessus et 30° en dessous raccourcissent le cône
 * exactement pareil. Un repère de sens est donc nécessaire, et c'est un trait court posé sur
 * l'axe du regard, au BORD du cône — vers l'EXTÉRIEUR quand le joueur lève la tête, vers
 * l'INTÉRIEUR quand il pique. Il ne part JAMAIS du point : « le bâton » reste supprimé
 * (décision D3), et ce trait-ci vit à la pointe du cône, à des dizaines de pixels de là.
 *
 * LA ZONE MORTE (2°) N'EST PAS UN ARRONDI DE CONFORT : sous 2° le cône perd 0,06 % de sa
 * longueur, donc l'œil ne peut RIEN vérifier de ce que le trait affirmerait, et l'affirmation
 * changerait de sens à chaque image. Une visée est à plat quand elle se lit à plat.
 */
function drawPitchTick(
  ctx: CanvasRenderingContext2D,
  c: XY,
  ang: number,
  R: number,
  pitchDeg: number,
  style: MarkerStyle,
  color: string,
): void {
  if (Math.abs(pitchDeg) < AIM_TICK_DEAD_DEG) return
  const len = AIM_TICK_LENGTH * style.k
  const far = R + (pitchDeg > 0 ? len : -len)
  ctx.beginPath()
  ctx.moveTo(c.x + Math.cos(ang) * R, c.y + Math.sin(ang) * R)
  ctx.lineTo(c.x + Math.cos(ang) * far, c.y + Math.sin(ang) * far)
  ctx.strokeStyle = color
  ctx.lineWidth = AIM_TICK_WIDTH * style.k
  ctx.stroke()
}

