/**
 * weaponPadsLayer.ts — LES SOCLES D'ARME sur la carte : ce que le calque affirme, et l'endroit
 * exact où il cesse d'affirmer.
 *
 * LA SOURCE est le document (schéma 11) : `weaponPads`, un socle par RÉCURRENCE SPATIALE
 * mesurée — des armes de même famille qui réapparaissent à moins d'un mètre, sans qu'aucune
 * vie ne s'achève à proximité. Chaque socle porte sa position monde, la FAMILLE d'arme (même
 * écriture qu'un loadout, donc même clé dans `weaponLabels`), ses apparitions, ses intervalles
 * de présence, et son CYCLE quand il est établi.
 *
 * TROIS ÉTATS, ET LE TROISIÈME EST L'HONNÊTETÉ DU CALQUE. Une occupation publie trois
 * instants : `t0` l'apparition (mesurée), `tLow` le dernier instant où l'arme est PROUVÉE
 * présente, `tHigh` le premier où son absence est prouvée. Entre les deux, le film ne dit
 * rien — les images-clés sont espacées de ~20 s. Une icône qui s'éteindrait pile à un instant
 * affirmerait une datation que la source n'a pas : le socle est donc PLEIN jusqu'à `tLow`,
 * INCERTAIN jusqu'à `tHigh` (icône fantôme, anneau pointillé), VIDE ensuite.
 *
 * LE POINTILLÉ GARDE SON SENS — celui que `placementShapes` lui a donné : « cette limite n'est
 * pas affirmée ». Ici la limite est TEMPORELLE et non spatiale, mais le message est le même, et
 * c'est pour cela que l'anneau du socle plein est plein : sa présence, elle, est prouvée.
 *
 * CE QUE CE CALQUE NE DESSINE JAMAIS :
 *  - LE RAMASSEUR. Le champ existe au contrat (`padPickups[].xuid`) et vaut `null` partout :
 *    l'oracle plafonne à 79,7 % contre 90 % exigés. Aucune ligne d'ici ne le lit.
 *  - LES OBJETS LÂCHÉS. Ce sont les armes qu'un joueur relâche en mourant ; elles ne sont pas
 *    des socles et ne sont pas publiées ici (décision utilisateur du 18/08).
 *  - LA DIFFÉRENCE SOCLE AU SOL / RÂTELIER MURAL. La donnée ne porte qu'une position : rien
 *    ne les sépare, et l'écran dit « emplacement d'arme » plutôt que d'en choisir un.
 *  - UN COMPTE À REBOURS SANS CYCLE. La clé `cycle` est ABSENTE quand il n'est pas établi
 *    (10 socles sur 31 seulement en portent un sur les quatre témoins) : ni chiffre, ni tiret
 *    qui suggérerait qu'on saurait.
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D, comme les calques voisins.
 * L'encre arrive de l'appelant, qui la tient des variables du thème.
 */
import type { ReplayWeaponPadReady } from './replayNormalize'
import { project, UNCERTAIN_DASH, type PlacementView } from './placementShapes'
import type { PadScale } from './weaponPadFamilies'
import type { XY } from './replayLogic'

/** Le cadrage est celui des poses : les deux calques projettent la même scène. */
export type { PlacementView as PadView } from './placementShapes'

/**
 * L'état d'un socle à un instant : plein (présence prouvée), incertain (le film ne dit rien),
 * vide (absence prouvée). `empty` couvre aussi l'avant-première-apparition.
 */
export type PadState = 'full' | 'uncertain' | 'empty'

/** Rayon de l'anneau du socle, en pixels d'ÉCRAN, par taille. */
const PAD_RING_PX: Record<PadScale, number> = { power: 9, classic: 5.5 }

/**
 * Hauteur de la vignette d'arme, en pixels d'écran, par taille.
 *
 * LES DEUX VALEURS SONT UN ARBITRAGE D'ÉCRAN, pas une mesure : « des icônes trop petites
 * seraient inutiles mais des trop grosses risquent de polluer » (bilan du 18/08). La grande
 * tient sous le nom d'un joueur (8,5 px) augmenté de son contour ; la petite reste au-dessus
 * du point neutre des objets non identifiés (2,5 px de rayon), qu'elle ne doit pas imiter.
 */
const PAD_ICON_H_PX: Record<PadScale, number> = { power: 13, classic: 8 }

/** Une vignette d'arme est large : au-delà de ce rapport, la largeur est bornée. */
const PAD_ICON_MAX_ASPECT = 3.2

/** Épaisseur de l'anneau, en pixels d'écran (elle suit la densité comme tout le reste). */
const PAD_RING_WIDTH = 1.1

/** Opacités de l'anneau et de la vignette, par état. `empty` n'a pas de vignette. */
const PAD_ALPHA: Record<PadState, { ring: number; icon: number }> = {
  full: { ring: 0.55, icon: 0.95 },
  uncertain: { ring: 0.4, icon: 0.3 },
  empty: { ring: 0.28, icon: 0 },
}

/** Corps du compte à rebours, en pixels d'écran, et son contour de lisibilité. */
const PAD_COUNTDOWN_FONT_PX = 8
const PAD_COUNTDOWN_STROKE_PX = 2.4
const PAD_COUNTDOWN_GAP_PX = 2.5

/** Rayon minimal de la zone sensible au survol : un anneau de 5,5 px ne se vise pas. */
const PAD_HOVER_MIN_RADIUS_PX = 9

/** Ce que le calque a besoin de savoir de l'instant courant. */
export interface PadTime {
  frame: number
  /** Durée RÉELLE d'une image (`frameToMs(1, doc)`) : le compte à rebours bat en temps réel. */
  frameMs: number
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
}

/** Ce que le calque emprunte au thème et au catalogue du document. */
export interface PadStyle {
  /** Encre neutre du thème : un socle est un objet du terrain, il n'a pas de camp. */
  ink: string
  /** Contour du compte à rebours (sombre dans les deux thèmes) ; vide = pas de contour. */
  labelStroke: string
  /** Vignette TEINTE de la famille, ou null : le socle garde alors un glyphe neutre. */
  iconOf: (weapon: string) => CanvasImageSource | null
  /** La taille à donner au socle, d'après ce qu'il porte (cf. weaponPadFamilies). */
  scaleOf: (weapon: string) => PadScale
  /** Le compte à rebours déjà localisé ; appelé seulement quand un cycle est établi. */
  countdownLabel: (seconds: number) => string
}

/**
 * padOccupancyAt — l'occupation en cours à cette image, c'est-à-dire la DERNIÈRE dont
 * l'apparition a eu lieu. Null avant la première : le socle n'a alors rien porté du tout.
 */
export function padOccupancyAt(
  pad: ReplayWeaponPadReady,
  frame: number,
): ReplayWeaponPadReady['presence'][number] | null {
  let found: ReplayWeaponPadReady['presence'][number] | null = null
  for (const occ of pad.presence) {
    if (occ.t0 > frame) break
    found = occ
  }
  return found
}

/**
 * padStateAt — l'état du socle à cette image.
 *
 * L'ordre des comparaisons EST la règle : plein tant que la présence est prouvée, incertain
 * tant que l'absence ne l'est pas, vide ensuite.
 *
 * LE CAS « JAMAIS VIDÉ » EST À PART, et il est fréquent (8 occupations sur 28 sur un des
 * témoins) : quand l'arme est encore recensée à la DERNIÈRE image-clé, `tHigh` ne dépasse pas
 * `tLow` — aucune absence n'a jamais été prouvée. Le socle reste alors PLEIN jusqu'au bout.
 * L'écrire vide, fût-ce une image, affirmerait un ramassage que rien n'a observé.
 */
export function padStateAt(pad: ReplayWeaponPadReady, frame: number): PadState {
  const occ = padOccupancyAt(pad, frame)
  if (!occ) return 'empty'
  if (frame < occ.tLow || occ.tHigh <= occ.tLow) return 'full'
  return frame < occ.tHigh ? 'uncertain' : 'empty'
}

/**
 * padRespawnSecondsAt — les secondes restantes avant la réapparition ATTENDUE, ou null.
 *
 * TROIS CONDITIONS, ET AUCUNE N'EST NÉGOCIABLE : le socle est VIDE (avant `tHigh`, rien n'est
 * fini), il porte un CYCLE ÉTABLI (clé absente sinon — jamais un chiffre instable), et le
 * compte n'est pas déjà épuisé. Le départ est `tHigh`, la borne HAUTE de la disparition :
 * partir de `tLow` avancerait la prédiction d'un intervalle que la source ne date pas.
 */
export function padRespawnSecondsAt(
  pad: ReplayWeaponPadReady,
  frame: number,
  frameMs: number,
): number | null {
  const cycle = pad.cycle
  if (!cycle || !(frameMs > 0) || !(cycle.medianS > 0)) return null
  if (padStateAt(pad, frame) !== 'empty') return null
  const occ = padOccupancyAt(pad, frame)
  if (!occ) return null
  const elapsedS = ((frame - occ.tHigh) * frameMs) / 1000
  const left = cycle.medianS - elapsedS
  return left > 0 ? left : null
}

/** padRadiusPx — le rayon d'écran de l'anneau d'un socle (sa taille suit ce qu'il porte). */
export function padRadiusPx(pad: ReplayWeaponPadReady, style: PadStyle, k: number): number {
  return PAD_RING_PX[style.scaleOf(pad.weapon)] * k
}

/**
 * padAt — le socle sous un point du canvas, ou null.
 *
 * Le survol se rejoue sur la DONNÉE, jamais sur les pixels : ce sont les mêmes positions
 * projetées que le tracé. Le plus proche l'emporte quand deux socles se recouvrent — deux
 * armes peuvent partager un mètre carré sur une petite arène.
 */
export function padAt(
  pads: readonly ReplayWeaponPadReady[],
  view: PlacementView,
  style: PadStyle,
  k: number,
  at: XY,
): ReplayWeaponPadReady | null {
  let best: ReplayWeaponPadReady | null = null
  let bestD2 = Infinity
  for (const pad of pads) {
    const c = project({ x: pad.x, y: pad.y }, view)
    const reach = Math.max(padRadiusPx(pad, style, k) + 3 * k, PAD_HOVER_MIN_RADIUS_PX * k)
    const d2 = (at.x - c.x) ** 2 + (at.y - c.y) ** 2
    if (d2 <= reach * reach && d2 < bestD2) {
      best = pad
      bestD2 = d2
    }
  }
  return best
}

/** drawRing — l'anneau du socle : plein quand la présence est prouvée, pointillé sinon. */
function drawRing(
  ctx: CanvasRenderingContext2D,
  c: XY,
  radius: number,
  state: PadState,
  time: PadTime,
): void {
  ctx.globalAlpha = PAD_ALPHA[state].ring
  ctx.lineWidth = PAD_RING_WIDTH * time.k
  ctx.setLineDash(state === 'full' ? [] : UNCERTAIN_DASH.map((d) => d * time.k))
  ctx.beginPath()
  ctx.arc(c.x, c.y, radius, 0, Math.PI * 2)
  ctx.stroke()
  ctx.setLineDash([])
}

/**
 * drawPadIcon — la vignette de l'arme, centrée sur l'anneau, à hauteur imposée et largeur
 * déduite du rapport de l'image (bornée : une vignette d'arme est très large).
 *
 * Sans vignette — famille hors catalogue du titre, ou visuel absent — un GLYPHE NEUTRE prend
 * sa place : jamais l'icône d'une arme voisine. Le nom (ou, à défaut, l'hexadécimal) reste
 * lisible au survol.
 */
function drawPadIcon(
  ctx: CanvasRenderingContext2D,
  c: XY,
  scale: PadScale,
  time: PadTime,
  icon: CanvasImageSource | null,
): void {
  const h = PAD_ICON_H_PX[scale] * time.k
  if (!icon) {
    ctx.beginPath()
    ctx.arc(c.x, c.y, PAD_RING_PX[scale] * time.k * 0.38, 0, Math.PI * 2)
    ctx.fill()
    return
  }
  const natW = 'width' in icon && typeof icon.width === 'number' ? icon.width : 0
  const natH = 'height' in icon && typeof icon.height === 'number' ? icon.height : 0
  const aspect = natW > 0 && natH > 0 ? Math.min(natW / natH, PAD_ICON_MAX_ASPECT) : 1
  const w = h * aspect
  ctx.drawImage(icon, c.x - w / 2, c.y - h / 2, w, h)
}

/** drawCountdown — le compte à rebours sous l'anneau ; cerné pour rester lisible partout. */
function drawCountdown(
  ctx: CanvasRenderingContext2D,
  c: XY,
  radius: number,
  text: string,
  style: { ink: string; labelStroke: string; k: number },
): void {
  ctx.globalAlpha = 1
  ctx.font = `600 ${PAD_COUNTDOWN_FONT_PX * style.k}px ui-sans-serif, system-ui, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  const y = c.y + radius + PAD_COUNTDOWN_GAP_PX * style.k
  if (style.labelStroke) {
    ctx.lineJoin = 'round'
    ctx.lineWidth = PAD_COUNTDOWN_STROKE_PX * style.k
    ctx.strokeStyle = style.labelStroke
    ctx.strokeText(text, c.x, y)
  }
  ctx.fillStyle = style.ink
  ctx.fillText(text, c.x, y)
}

/**
 * drawWeaponPadsLayer trace TOUS les socles du match à l'image courante.
 *
 * Ils se dessinent tous, quel que soit leur état : un socle vide reste un lieu du terrain, et
 * l'effacer priverait le lecteur de l'information la plus utile — « l'arme de puissance n'est
 * plus là ». C'est aussi ce qui donne son ancrage au compte à rebours.
 */
export function drawWeaponPadsLayer(
  ctx: CanvasRenderingContext2D,
  pads: readonly ReplayWeaponPadReady[],
  view: PlacementView,
  time: PadTime,
  style: PadStyle,
): void {
  if (pads.length === 0 || view.width === 0) return
  ctx.save()
  ctx.strokeStyle = style.ink
  ctx.fillStyle = style.ink
  for (const pad of pads) {
    const c = project({ x: pad.x, y: pad.y }, view)
    const scale = style.scaleOf(pad.weapon)
    const radius = PAD_RING_PX[scale] * time.k
    const state = padStateAt(pad, time.frame)
    drawRing(ctx, c, radius, state, time)
    const iconAlpha = PAD_ALPHA[state].icon
    if (iconAlpha > 0) {
      ctx.globalAlpha = iconAlpha
      drawPadIcon(ctx, c, scale, time, style.iconOf(pad.weapon))
    }
    const left = padRespawnSecondsAt(pad, time.frame, time.frameMs)
    if (left !== null) {
      drawCountdown(ctx, c, radius, style.countdownLabel(left), {
        ink: style.ink,
        labelStroke: style.labelStroke,
        k: time.k,
      })
      ctx.strokeStyle = style.ink
      ctx.fillStyle = style.ink
    }
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
