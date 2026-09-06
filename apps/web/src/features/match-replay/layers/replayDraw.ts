/**
 * replayDraw.ts — couches de DÉCOR et d'ÉVÉNEMENTS du rejeu 2D : sol reconstruit, props Forge
 * (en repli), tirs et lancers de grenade. Le joueur lui-même vit dans replayMarkers.ts.
 * Pas de React : uniquement un CanvasRenderingContext2D + de la géométrie pure
 * (replayLogic.ts). Les couleurs arrivent DÉJÀ RÉSOLUES depuis les tokens sémantiques
 * (getSeriesColors / resolveToken) — aucun littéral de couleur ici (règle color-tokens).
 */
import type { ReplayGrenade, ReplayMapObject } from '@/lib/api/types'

import type { FxInk } from './fxInk'
import { drawMuzzleFlash } from './muzzleFlash'
import type { ShotFxEntry } from '../model/shotFx'
import { drawDeathMarker, drawShotEffect } from './shotEffects'
import { MELEE_LINK_MAX_M, type KillFxEntry } from '../model/killFx'
import { drawMeleeStar, meleeStarProgress } from './meleeStar'
import { altitudeRatio, footprint } from '../model/replayLogic'
import { vehicleShotOrigin, type VehicleMountSpriteSize } from '../model/vehicleWeaponMounts'
import { type CanvasView, projectTo, scaleOf } from '../model/replayView'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
/** Amplitude verticale utilisée pour l'indication d'étage. */
interface ZRange {
  min: number
  max: number
}

// Fond de carte : props Forge de 0,25 m² en moyenne — sans plancher de taille ils sont
// invisibles. Opacités volontairement basses : le sujet du rendu reste les joueurs.
const OBJECT_MIN_PX = 2.5
const OBJECT_ALPHA_LOW = 0.14
const OBJECT_ALPHA_SPAN = 0.24

interface GeometryStyle {
  color: string
  z: ZRange
}

/**
 * drawGeometryLayer dessine les props Forge SOUS les trajectoires : rectangles orientés
 * (ou petits carrés quand l'emprise projetée est sous le seuil de lisibilité).
 * L'opacité monte avec l'altitude : indication d'étage discrète, sans couleur dédiée.
 */
export function drawGeometryLayer(
  ctx: CanvasRenderingContext2D,
  objects: ReplayMapObject[],
  view: CanvasView,
  style: GeometryStyle,
): void {
  const scale = scaleOf(view)
  ctx.fillStyle = style.color
  for (const o of objects) {
    ctx.globalAlpha =
      OBJECT_ALPHA_LOW + OBJECT_ALPHA_SPAN * altitudeRatio(o.z ?? 0, style.z.min, style.z.max)
    const corners = footprint(o)
    const wide = (o.dx ?? 0) * scale >= OBJECT_MIN_PX && (o.dy ?? 0) * scale >= OBJECT_MIN_PX
    if (corners.length === 4 && wide) {
      ctx.beginPath()
      corners.forEach((w, i) => {
        const c = projectTo(view, w)
        if (i === 0) ctx.moveTo(c.x, c.y)
        else ctx.lineTo(c.x, c.y)
      })
      ctx.closePath()
      ctx.fill()
      continue
    }
    const c = projectTo(view, o)
    ctx.fillRect(c.x - OBJECT_MIN_PX / 2, c.y - OBJECT_MIN_PX / 2, OBJECT_MIN_PX, OBJECT_MIN_PX)
  }
  ctx.globalAlpha = 1
}

// Les TRAJECTOIRES et les marqueurs de joueur vivent dans replayMarkers.ts : ce calque a
// gagné le cône de visée, le bouclier, les anneaux d'étage, l'apparition et la mort, et il
// aurait fait de ce fichier un god file.

// Événements ponctuels : un tir est un éclair de bouche (sa géométrie vit dans
// muzzleFlash.ts), un lancer une marque plus lisible.
const GRENADE_RADIUS = 4
const GRENADE_RING = 6.5

/**
 * Couleurs héritées du sol reconstruit, SUPPRIMÉ le 2026-09-03 (décision utilisateur : ce repli
 * ne devait plus pouvoir tourner par inadvertance et cuire un calque de ~45 000 cellules).
 * Seul `edge` survit — il sert d'encre fine aux ZONES NOMMÉES. `fill` reste porté par
 * `useReplayInks` : c'est la même paire de tokens, et la séparer pour un seul champ coûterait
 * plus de câblage qu'elle n'en économiserait.
 */
export interface FloorStyle {
  fill: string
  edge: string
}

/** Fenêtre d'affichage d'un événement ponctuel, en frames. */
export interface EventWindow {
  frame: number
  /** Nombre de frames pendant lesquelles l'événement reste visible après son instant. */
  hold: number
  /**
   * Durée RÉELLE d'une frame, en ms. Les effets dont la mise en scène a sa propre horloge —
   * aujourd'hui l'étoile de mêlée, 400 ms — la lisent ici : une durée écrite en frames
   * changerait de sens le jour où la cadence d'échantillonnage change (même règle que la fin
   * de vol des grenades, qui porte déjà ce champ).
   */
  frameMs: number
}

/**
 * drawShotsLayer dessine les ÉCLAIRS DE BOUCHE de la fenêtre courante.
 *
 * CE QUE LE POINT SIGNIFIE, et il faut que le rendu le respecte : le film n'enregistre que les
 * tirs qui INFLIGENT un dégât. Un tir dessiné a donc touché. Sa DIRECTION est celle du REGARD
 * du tireur à cet instant (relu dans sa trajectoire, cf. shotFx.ts) — sans lecture, une
 * bouffée ronde, jamais une direction inventée. SUR UN TIR EN VÉHICULE, le point ET la
 * direction viennent du MONTAGE de l'arme plutôt que du regard (`vehicleShotOrigin`, demande
 * utilisateur du 2026-09-03 — « si ça vient du passager, faut que le tir vienne du siège
 * passager qui a une tourelle »).
 *
 * CE CALQUE A CHANGÉ DE NATURE le 2026-08-15 (étape 2 du plan des effets de tirs) : il
 * dessinait une TRACE de 62 px dans la couleur du TIREUR ; il dessine désormais un éclair
 * court à la bouche de l'arme, teinté par la NATURE DE LA DÉCHARGE et par elle seule
 * (décision utilisateur). Le cône de visée, lui, n'a pas bougé — le flash s'y ajoute.
 */
export function drawShotsLayer(
  ctx: CanvasRenderingContext2D,
  shots: ShotFxEntry[],
  view: CanvasView,
  win: EventWindow,
  style: ShotStyle,
): void {
  for (const s of shots) {
    const age = win.frame - s.frame
    if (age < 0 || age > win.hold) continue
    const c = projectTo(view, s)
    const { origin, angle } = vehicleShotOrigin({
      h: s.h, vehicleShot: s.vehicleShot, center: c, sizeOf: style.vehicleSizeOf, k: style.k,
    })
    drawMuzzleFlash(ctx, s.fam, s.tint, {
      x: origin.x,
      y: origin.y,
      angle,
      fade: 1 - age / Math.max(win.hold, 1),
      reduced: style.reducedMotion,
      seed: s.seed,
      k: style.k,
    }, style.ink)
  }
  ctx.globalAlpha = 1
}

/** Style du calque des tirs : les encres du thème et la densité de l'écran. */
export interface ShotStyle {
  /**
   * Teintes de décharge résolues depuis le thème (fxInk.ts). AUCUNE couleur de joueur
   * n'entre ici : correction utilisateur du 2026-08-15 — « les couleurs des effets de tirs
   * [...] prennent seulement l'ARME en compte ».
   */
  ink: FxInk
  /** Densité du canevas : l'éclair s'adresse à l'œil, sa taille est en pixels d'écran. */
  k: number
  reducedMotion: boolean
  /**
   * Taille du sprite d'une famille de véhicule, ou `null` (pas encore chargé). Optionnel :
   * un appelant qui ne câble pas le calque véhicules (tests, ou futur écran sans véhicules)
   * garde le repli centre pour tout tir en véhicule, jamais une exception.
   */
  vehicleSizeOf?: (family: string) => VehicleMountSpriteSize | null
}

/** Style du calque des morts : la couleur du tueur, et le repli quand il n'a pas de trace. */
export interface KillFxStyle {
  /**
   * Couleur du TUEUR, résolue par slot ET par image : elle est demandée à l'INSTANT DU KILL
   * (`e.frame`), pas à l'image de dessin — un slot est réattribué entre manches, et l'effet de
   * mort persiste plusieurs frames après le coup, pendant lesquelles le slot pourrait déjà
   * appartenir à un autre joueur.
   */
  colorOfSlot: (slot: number, frame: number) => string | null
  fallback: string
  reducedMotion: boolean
  /** Densité du canevas : l'étoile de mêlée est déclarée en pixels d'ÉCRAN. */
  k: number
}

/**
 * drawKillFxLayer dessine les EFFETS DE MORT de la fenêtre courante.
 *
 * ORIENTÉ TUEUR -> VICTIME seulement quand le couple est complet (règle POC 89/93, portée
 * par `buildKillFx` : `vx` null = pas d'axe) ; sinon un marqueur pointillé non orienté.
 * L'extrémité est RÉELLE (`target`) : c'est la seule différence de nature avec les tirs,
 * dont la longueur n'est qu'une trace. La COULEUR est celle du tueur — la famille d'arme
 * se lit dans la FORME (arbitrage du lot 3.2, conservé).
 *
 * LA MÊLÉE FATALE FAIT EXCEPTION depuis le 2026-08-18 (D3/R2-3) : elle ne trace ni arc ni
 * marqueur, elle pose une ÉTOILE au lieu de la mort (meleeStar.ts). Le corps à corps est
 * justement le cas où l'axe n'existe presque jamais — tueur et victime tombent sous le seuil
 * de 1,5 px ci-dessous — et où le rendu générique se réduisait donc à un anneau pointillé,
 * indiscernable d'une mort dont on ignore tout.
 */
export function drawKillFxLayer(
  ctx: CanvasRenderingContext2D,
  fx: KillFxEntry[],
  view: CanvasView,
  win: EventWindow,
  style: KillFxStyle,
): void {
  for (const e of fx) {
    const age = win.frame - e.frame
    if (age < 0 || age > win.hold) continue
    const fade = 1 - age / (win.hold + 1)
    const c = projectTo(view, e)
    const color = (e.slot !== null ? style.colorOfSlot(e.slot, e.frame) : null) ?? style.fallback
    if (e.fam === 'melee') {
      // AU LIEU DE LA MORT : la victime quand elle est relue, l'origine sinon (elle vaut
      // alors la position du tueur, à un pas de corps près — c'est un corps à corps).
      const mort =
        e.deathX !== null && e.deathY !== null
          ? projectTo(view, { x: e.deathX, y: e.deathY })
          : c
      const p = meleeStarProgress(age * win.frameMs, style.reducedMotion)
      if (p !== null) drawMeleeStar(ctx, mort, p, style.k, color)
      continue
    }
    let angle: number | null = null
    let length = 0
    if (e.vx !== null && e.vy !== null) {
      const v = projectTo(view, { x: e.vx, y: e.vy })
      const dx = v.x - c.x
      const dy = v.y - c.y
      const d = Math.hypot(dx, dy)
      // < 1,5 px : tueur et victime au même endroit à l'écran (corps à corps). Aucun axe
      // fiable à en tirer — le marqueur non orienté vaut mieux qu'un angle calculé sur du
      // bruit d'arrondi (règle POC).
      if (d > 1.5) {
        angle = Math.atan2(dy, dx)
        length = d
      }
    }
    const shape = {
      x: c.x,
      y: c.y,
      angle,
      length,
      fade,
      reduced: style.reducedMotion,
      seed: e.seed,
      target: angle !== null,
      meleeLink: e.dist !== null && e.dist < MELEE_LINK_MAX_M,
    }
    if (angle === null) drawDeathMarker(ctx, shape, color)
    else drawShotEffect(ctx, e.fam, shape, color)
  }
  ctx.globalAlpha = 1
}

/** Côté de la vignette de type posée au-dessus de l'anneau d'un lancer (POC : 18 px). */
const GRENADE_ICON_PX = 18

/**
 * tintedIconCanvas — un masque du HUD (blanc/gris + alpha) TEINT à une encre du thème,
 * une fois pour toutes dans un canvas hors écran. Un canvas ne connaît pas le
 * `mask-image` CSS de WeaponIcon : la teinte se fait par composition `source-in`, qui
 * préserve l'alpha et suit le thème par re-teinture (l'appelant re-teint au changement).
 *
 * LE MIROIR EST CUIT ICI, PAS AU TRACÉ (2026-08-28) : les vignettes d'arme des socles sont
 * désormais les icônes PLEINES des fiches, et les deux atlas du jeu pointent vers la GAUCHE
 * quand le kill feed — donc la fiche, donc le socle — pointe à droite. Retourner à chaque image
 * demanderait un `save`/`scale`/`restore` par socle et par frame ; retourner une fois à la
 * cuisson ne coûte rien et laisse le calque poser une image comme les autres.
 *
 * `tint: false` LAISSE L'IMAGE TELLE QUELLE (couleurs comprises) : c'est le cas d'un dessin fini
 * qu'on ne fait que retourner. Sans cette porte, une image finie retournée perdrait ses couleurs.
 */
export interface TintedIconOptions {
  /** Retourner l'image horizontalement (le sens du kill feed du jeu). */
  mirrored?: boolean
  /** Teindre à l'encre donnée (défaut) ou garder les couleurs de l'image. */
  tinted?: boolean
  /**
   * Mode de composition de la teinte (2026-09-02, lot véhicules) : `'source-in'` (DÉFAUT,
   * comportement STRICTEMENT INCHANGÉ) — la teinte REMPLACE la couleur du sprite, c'est le
   * mode des masques HUD (blanc/gris + alpha). `'multiply'` — la teinte se MULTIPLIE avec les
   * couleurs du sprite : le blanc devient l'encre, le NOIR RESTE NOIR (les traits de volume des
   * sprites véhicules, cf. `.ai/V7.5/film_re/V4_RAPPORT_SPRITES_2026-08-31.md` §10.1).
   * `multiply` ne respecte PAS l'alpha à lui seul (`fillRect` en mode multiply rend opaque tout
   * pixel de fond transparent) : un second passage `destination-in` avec l'image d'origine
   * réapplique le canal alpha du sprite après le `fillRect`.
   */
  composite?: 'source-in' | 'multiply'
}

export function tintedIconCanvas(
  img: HTMLImageElement,
  color: string,
  options: TintedIconOptions = {},
): HTMLCanvasElement {
  const { mirrored = false, tinted = true, composite = 'source-in' } = options
  const off = document.createElement('canvas')
  off.width = Math.max(1, img.naturalWidth)
  off.height = Math.max(1, img.naturalHeight)
  const octx = off.getContext('2d')
  if (!octx) return off
  if (mirrored) {
    octx.translate(off.width, 0)
    octx.scale(-1, 1)
  }
  octx.drawImage(img, 0, 0)
  if (!tinted) return off
  octx.globalCompositeOperation = composite
  octx.fillStyle = color
  // `source-in`/`multiply` s'appliquent au canvas ENTIER : le rectangle se pose donc dans le
  // repère de départ, sinon le miroir le décalerait hors du cadre.
  octx.setTransform(1, 0, 0, 1, 0, 0)
  octx.fillRect(0, 0, off.width, off.height)
  if (composite === 'multiply') {
    // `multiply` a rendu tout le canvas opaque (y compris le fond, auparavant transparent) :
    // on redessine le sprite dans le MÊME repère que le premier tracé (miroir compris, pour
    // rester aligné pixel à pixel) et on ne garde, via `destination-in`, que ce que son canal
    // alpha recouvre — le fond redevient transparent, les traits noirs gardent leur alpha quasi
    // pleine, le remplissage garde son alpha d'ombrage (0,80..1).
    if (mirrored) {
      octx.translate(off.width, 0)
      octx.scale(-1, 1)
    }
    octx.globalCompositeOperation = 'destination-in'
    octx.drawImage(img, 0, 0)
    octx.setTransform(1, 0, 0, 1, 0, 0)
  }
  return off
}

/** Style du calque des lancers : la couleur des marques, et la vignette du TYPE par rang. */
export interface GrenadeStyle {
  color: string
  /** Vignette teintée du rang, ou null : l'anneau seul reste juste — jamais la vignette
   *  d'un type voisin. */
  iconOf: (rank: number) => CanvasImageSource | null
}

/**
 * drawGrenadesLayer dessine les lancers de grenade.
 *
 * CE QUI EST DESSINÉ EST LE POINT DE DÉPART, pas une trajectoire : l'arc et le point de chute
 * ne sont pas décodés, et rien ici ne les invente. L'anneau distingue le lancer d'un tir ;
 * la VIGNETTE au-dessus dit le TYPE (item 2.4 — le rang est écrit dans le film, la table
 * grenadeLabels le nomme).
 */
export function drawGrenadesLayer(
  ctx: CanvasRenderingContext2D,
  grenades: ReplayGrenade[],
  view: CanvasView,
  win: EventWindow,
  style: GrenadeStyle,
): void {
  ctx.strokeStyle = style.color
  ctx.fillStyle = style.color
  for (const g of grenades) {
    const age = win.frame - g.t
    if (age < 0 || age > win.hold) continue
    const fade = 1 - age / Math.max(win.hold, 1)
    const c = projectTo(view, g)
    ctx.globalAlpha = fade
    ctx.beginPath()
    ctx.arc(c.x, c.y, GRENADE_RADIUS, 0, Math.PI * 2)
    ctx.fill()
    ctx.beginPath()
    ctx.arc(c.x, c.y, GRENADE_RING, 0, Math.PI * 2)
    ctx.lineWidth = 1.5
    ctx.stroke()
    const icon = style.iconOf(g.rank)
    if (icon) {
      ctx.globalAlpha = Math.min(1, 1.2 * fade)
      ctx.drawImage(
        icon,
        c.x - GRENADE_ICON_PX / 2,
        c.y - GRENADE_ICON_PX / 2 - 13,
        GRENADE_ICON_PX,
        GRENADE_ICON_PX,
      )
    }
  }
  ctx.globalAlpha = 1
}

/**
 * drawRotatedSprite — LA ROTATION BITMAP DU REJEU, sans précédent avant le lot véhicules
 * (2026-09-02). Repris du seul repère du dépôt qui tourne déjà une image autour d'un point —
 * `glow()` dans `muzzleFlash.ts:159-165` (translate + rotate + scale, puis restore) — plutôt
 * que d'inventer une seconde mécanique pour le même besoin.
 *
 * `angle` EST DÉJÀ EN RADIANS CANEVAS, pas en cap monde : la conversion (inversion de signe,
 * cf. `drawShotsLayer` : « monde -> canevas, l'axe Y est inversé, donc l'angle l'est aussi »)
 * est la responsabilité de L'APPELANT, qui seul sait de quel cap il part. Ce helper ne fait
 * QUE la géométrie d'écran : centrer l'image sur `(x, y)`, la tourner, la mettre à l'échelle.
 *
 * `scale` EST UN FACTEUR MULTIPLICATIF DES PIXELS NATURELS DE L'IMAGE, pas une taille cible en
 * pixels d'écran : c'est l'appelant (`vehiclesLayer.ts`) qui calcule ce facteur à partir de la
 * taille voulue à l'écran et de la résolution du sprite — exactement le rôle que `ctx.scale`
 * joue déjà dans `glow()`. `drawImage` est ensuite posé à SA taille NATURELLE, centrée sur
 * l'origine du repère mis à l'échelle : le canevas fait le reste.
 *
 * SANS IMAGE VALIDE (chargement pas encore abouti, dimensions nulles), RIEN NE SE DESSINE —
 * jamais un rectangle de repli qui prétendrait montrer un véhicule.
 */
export function drawRotatedSprite(
  ctx: CanvasRenderingContext2D,
  img: CanvasImageSource,
  x: number,
  y: number,
  angle: number,
  scale: number,
): void {
  const w = 'width' in img && typeof img.width === 'number' ? img.width : 0
  const h = 'height' in img && typeof img.height === 'number' ? img.height : 0
  if (w <= 0 || h <= 0 || scale <= 0) return
  ctx.save()
  ctx.translate(x, y)
  ctx.rotate(angle)
  ctx.scale(scale, scale)
  ctx.drawImage(img, -w / 2, -h / 2, w, h)
  ctx.restore()
}
