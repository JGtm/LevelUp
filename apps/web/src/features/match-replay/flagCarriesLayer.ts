/**
 * flagCarriesLayer.ts — LA VIE DES DRAPEAUX DE CTF (schéma 15), image par image.
 *
 * POURQUOI IL VIT À CÔTÉ DU CALQUE STATIQUE, ET PAS DEDANS. Même partage que
 * `zoneStatesLayer.ts` : `objectivesLayer.ts` porte la GÉOMÉTRIE des objectifs (elle ne change
 * jamais, se cuit une fois hors écran et se recopie), celui-ci porte l'ÉTAT, qui change à chaque
 * image. C'est aussi la seule façon de garder les deux fichiers sous le seuil du dépôt.
 *
 * LES QUATRE ÉTATS SONT CEUX DE L'ARTEFACT, jamais des états inventés ici :
 *
 *  - `carried` — un fait DATÉ ferme le portage. Le drapeau est collé au marqueur de son porteur.
 *  - `carried_open` — RIEN ne ferme le portage : l'intervalle court jusqu'à la fin de l'axe et
 *    c'est une BORNE HAUTE. Même icône, ATTÉNUÉE — l'incertitude est à l'écran, pas dans une note.
 *  - `dropped` — le drapeau est au sol, à la position publiée, avec une RESPIRATION : il n'est
 *    porté par personne mais il est toujours en jeu.
 *  - `home` — il est à sa base.
 *
 * LA BASE PORTE UN ÉTAT PRÉSENT / ABSENT, et c'est la lecture que le plan demande : quand le
 * drapeau est `home`, le glyphe se pose PLEIN sur sa base ; quand il est ailleurs, la base garde
 * un glyphe ATTÉNUÉ — « le drapeau n'est pas là ». Rien de plus : ni anneau, ni compte à rebours.
 *
 * POURQUOI LA BASE VIENT DES SPANS `home` ET NON DE `mapObjectives`. Les deux disent la même
 * position — le serveur résout les socles `flag_spawn` par `map_id` avant de publier le calque —
 * mais `mapObjectives` est RECONSTRUIT À CHAQUE REQUÊTE, alors que les spans voyagent avec
 * l'artefact. Joindre les deux exposerait le lot au décalage que le plan a déjà mesuré (trois
 * `flag_spawn` par carte de CTF, dont un NEUTRE au centre ; deux catalogues qui ne nomment pas
 * les modules pareil). L'ancre interne ne peut pas se tromper de drapeau.
 *
 * AUCUNE VIGNETTE DU JEU, ET C'EST UNE MESURE, pas un renoncement : le drapeau n'est PAS un
 * `weap` (phase 0, item 0.3 — le marqueur de portage ne porte pas le suffixe d'identifiant
 * d'arme, 0 occurrence sur 83), il n'a donc pas de tag, et `weaponLabels` ne le nomme sur AUCUN
 * des trois témoins CTF. Le glyphe est donc tracé au canvas — hampe + fanion — à l'encre
 * d'équipe résolue par l'appelant. Aucune URL d'image n'est devinée.
 *
 * AUCUN TEXTE, comme les deux calques voisins : ce qui se dit se dit dans l'infobulle.
 */
import { worldToCanvas, type XY } from './replayLogic'

import type { CanvasView } from './objectivesLayer'
import type { ReplayFlagCarryReady } from './replayNormalize'

/** Les quatre états publiés par `flagCarries[].spans[].state` (schéma 15). */
export type FlagState = 'carried' | 'carried_open' | 'dropped' | 'home'

/** L'ordre dans lequel l'UI les nomme — et la liste que le contrat de texte doit couvrir. */
export const FLAG_STATES: readonly FlagState[] = ['carried', 'carried_open', 'dropped', 'home']

/**
 * FlagNow — ce qu'un drapeau montre à une image donnée.
 *
 * `state` est rendu tel que l'artefact l'écrit — d'où le type `string` et non l'union fermée.
 * Un état INCONNU du client (artefact plus récent que ce code) se dessine comme un drapeau
 * PRÉSENT : glyphe plein à sa position publiée, base marquée absente. C'est le comportement le
 * moins trompeur — l'objet EST quelque part, l'artefact le dit — et l'infobulle, elle, omet la
 * ligne d'état plutôt que d'inventer un libellé.
 */
export interface FlagNow {
  /** L'équipe PROPRIÉTAIRE du drapeau (celle de sa base), telle que l'artefact la publie. */
  team: number
  state: string
  /** Le porteur, seulement pour les deux états portés — `null` partout ailleurs. */
  xuid: string | null
  /** Position publiée du span : celle du porteur au span, du lâcher, ou de la base. */
  x: number
  y: number
  /** Première image du span : c'est elle qui donne le « depuis quand » de l'infobulle. */
  t0: number
}

/**
 * flagSpanAt rend l'état du drapeau à l'image demandée, ou `null` quand aucun intervalle ne la
 * couvre — avant la première prise, l'artefact ne dit rien, et dessiner « à la base » par défaut
 * affirmerait ce qu'il ne dit pas.
 *
 * FONCTION PURE, testée à part : c'est elle que le rendu appelle à chaque image, et c'est elle
 * que le survol appelle pour composer l'infobulle.
 */
export function flagSpanAt(carry: ReplayFlagCarryReady, frame: number): FlagNow | null {
  for (const sp of carry.spans) {
    if (frame < sp.t0 || frame > sp.t1) continue
    return { team: carry.team, state: sp.state, xuid: sp.xuid ?? null, x: sp.x, y: sp.y, t0: sp.t0 }
  }
  return null
}

/**
 * homeAnchorOf rend la position de la BASE du drapeau : celle de ses intervalles `home`.
 *
 * `null` quand le drapeau n'est jamais rentré chez lui sur ce film — la base n'est alors pas
 * connue de l'artefact, et aucun marqueur d'absence ne se dessine. C'est une donnée manquante,
 * pas un zéro.
 */
export function homeAnchorOf(carry: ReplayFlagCarryReady): XY | null {
  for (const sp of carry.spans) {
    if (sp.state === 'home') return { x: sp.x, y: sp.y }
  }
  return null
}

/** Style du calque : l'encre d'équipe est RÉSOLUE par l'appelant (règle color-tokens). */
export interface FlagCarriesStyle {
  /** Encre du drapeau de cette équipe ; l'appelant sert le neutre du thème s'il ne sait pas. */
  colorOfTeam: (team: number) => string
  /** Mouvement réduit : la respiration du drapeau au sol devient une opacité constante. */
  reducedMotion: boolean
}

// Réglages du calque. Le glyphe reste volontairement PLUS PETIT que le marqueur d'un joueur :
// il se pose À CÔTÉ du porteur, il ne doit pas le remplacer.
//
// ÉLARGI DE 20 % LE 2026-08-26 (retour utilisateur : « le drapeau faudrait les faire un tout
// petit peu plus gros »). C'est un AJUSTEMENT, pas un changement d'échelle : à 13 px de hampe le
// fanion faisait 9 × 6,5 px, et sur un fond de carte chargé il se confondait avec les repères
// voisins. Les trois cotes du glyphe montent ensemble — les élargir séparément déformerait le
// dessin — et le décalage ne bouge PAS : le glyphe grandit vers le haut et la droite, donc son
// ancrage sur le point qu'il qualifie reste le même.
//
// LE RAYON DE SURVOL SUIT, et il le doit : il couvre la hampe et le fanion. Le laisser à 12
// aurait rendu insurvolable la part de glyphe gagnée par l'élargissement.
const FLAG_GLYPH_SCALE = 1.2
const FLAG_POLE_H = 13 * FLAG_GLYPH_SCALE
const FLAG_WING_W = 9 * FLAG_GLYPH_SCALE
const FLAG_WING_H = 6.5 * FLAG_GLYPH_SCALE
/** Décalage du glyphe par rapport au point qu'il qualifie (au-dessus et à droite du marqueur). */
const FLAG_OFFSET_X = 6
const FLAG_OFFSET_Y = 2
const FLAG_STROKE_WIDTH = 1.6
/** Rayon de SURVOL, en pixels : il couvre la hampe et le fanion, sans déborder sur le voisin. */
export const FLAG_HIT_RADIUS = 12 * FLAG_GLYPH_SCALE

/** Opacité pleine : un fait daté (drapeau porté, ou à sa base). */
const ALPHA_SOLID = 0.95
/** Opacité atténuée : `carried_open` (fin non datée) et la base d'un drapeau absent. */
const ALPHA_FAINT = 0.38
/** Bornes de la respiration du drapeau au sol, et sa période en images. */
const BREATH_MIN = 0.55
const BREATH_MAX = 0.95
const BREATH_PERIOD_FRAMES = 22

/**
 * ReplayFlagPaint — ce que le calque reçoit de l'appelant (`useReplayFlagCarries`).
 *
 * `posOf` est LA raison d'être de ce champ : un drapeau `carried` se dessine sur le marqueur de
 * son PORTEUR à l'image courante, pas à la position figée du span. Le calque ne sait pas relire
 * une trajectoire — c'est l'appelant qui lui passe la lecture, comme les pulses d'objectif.
 */
export interface FlagCarriesInput {
  style: FlagCarriesStyle
  /** Position monde d'un joueur à une image, ou `null` s'il n'est pas localisable. */
  posOf: (xuid: string, frame: number) => XY | null
}

/**
 * flagPointAt rend le point MONDE où le drapeau se dessine à cette image, et rien d'autre.
 *
 * LE REPLI EST UNE DONNÉE, PAS UNE INVENTION : quand le porteur n'est pas localisable (vie non
 * publiée, image hors de ses trajectoires), le span porte lui-même une position mesurée — celle
 * que le serveur a retenue pour ce portage. On la sert plutôt que de faire disparaître le
 * drapeau, et l'infobulle dit de toute façon qui le porte.
 */
export function flagPointAt(now: FlagNow, frame: number, posOf: FlagCarriesInput['posOf']): XY {
  if (now.state === 'carried' || now.state === 'carried_open') {
    const p = now.xuid ? posOf(now.xuid, frame) : null
    if (p) return p
  }
  return { x: now.x, y: now.y }
}

/** Opacité du glyphe pour un état — la respiration du `dropped` comprise. */
function alphaOf(state: string, frame: number, reducedMotion: boolean): number {
  if (state === 'carried_open') return ALPHA_FAINT
  if (state !== 'dropped') return ALPHA_SOLID
  if (reducedMotion) return (BREATH_MIN + BREATH_MAX) / 2
  const phase = (2 * Math.PI * frame) / BREATH_PERIOD_FRAMES
  return BREATH_MIN + (BREATH_MAX - BREATH_MIN) * (0.5 + 0.5 * Math.sin(phase))
}

/**
 * drawFlagCarries peint, PAR-DESSUS le calque statique, ce que le film dit de chaque drapeau à
 * l'image courante : le glyphe sur son porteur, au sol, ou à sa base — et la base ATTÉNUÉE quand
 * le drapeau n'y est pas.
 *
 * Un drapeau sans état à cette image n'est PAS dessiné : l'artefact ne dit rien de lui, le
 * calque non plus.
 */
export function drawFlagCarries(
  ctx: CanvasRenderingContext2D,
  layer: FlagCarriesInput,
  carries: readonly ReplayFlagCarryReady[],
  view: CanvasView,
  frame: number,
): void {
  const px = (p: XY) => worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
  for (const carry of carries) {
    const now = flagSpanAt(carry, frame)
    if (!now) continue
    const ink = layer.style.colorOfTeam(carry.team)
    const anchor = homeAnchorOf(carry)
    // LA BASE D'ABORD, et seulement quand le drapeau n'y est pas : c'est le fond de la lecture,
    // le glyphe vivant se pose par-dessus (ils coïncident quand l'état est `home`).
    if (anchor && now.state !== 'home') {
      drawFlagGlyph(ctx, px(anchor), { ink, alpha: ALPHA_FAINT, hollow: true })
    }
    const at = px(flagPointAt(now, frame, layer.posOf))
    drawFlagGlyph(ctx, at, {
      ink,
      alpha: alphaOf(now.state, frame, layer.style.reducedMotion),
      hollow: now.state === 'carried_open',
    })
  }
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'un glyphe a besoin de savoir (règle des 5 paramètres). */
interface FlagGlyphPaint {
  ink: string
  alpha: number
  /** Fanion CREUX (liseré seul) : l'état est incertain, ou la base est vide. */
  hollow: boolean
}

/**
 * drawFlagGlyph trace le drapeau : une HAMPE verticale et un FANION triangulaire.
 *
 * Le point servi est le pied de la hampe, décalé pour se poser à côté du marqueur qu'il
 * qualifie plutôt que dessus. Tracé au canvas, sans image : cf. l'en-tête du fichier.
 */
export function drawFlagGlyph(
  ctx: CanvasRenderingContext2D,
  at: XY,
  paint: FlagGlyphPaint,
): void {
  const x = at.x + FLAG_OFFSET_X
  const foot = at.y - FLAG_OFFSET_Y
  const top = foot - FLAG_POLE_H
  ctx.globalAlpha = paint.alpha
  ctx.strokeStyle = paint.ink
  ctx.fillStyle = paint.ink
  ctx.lineWidth = FLAG_STROKE_WIDTH
  ctx.beginPath()
  ctx.moveTo(x, foot)
  ctx.lineTo(x, top)
  ctx.stroke()
  ctx.beginPath()
  ctx.moveTo(x, top)
  ctx.lineTo(x + FLAG_WING_W, top + FLAG_WING_H / 2)
  ctx.lineTo(x, top + FLAG_WING_H)
  ctx.closePath()
  if (paint.hollow) ctx.stroke()
  else ctx.fill()
}

/** Ce qu'un survol a trouvé : le drapeau, son état LU À CET INSTANT, et où poser l'infobulle. */
export interface FlagHit {
  now: FlagNow
  /** Point CANVAS du glyphe — l'infobulle se pose là, pas sous le pointeur brut. */
  at: XY
}

/**
 * flagAt rend le drapeau dont le glyphe se trouve sous le point CANVAS servi, ou `null`.
 *
 * LE SURVOL REJOUE EXACTEMENT LA MÊME GÉOMÉTRIE QUE LE TRACÉ (`flagPointAt`, même décalage) :
 * viser une forme dessinée ailleurs ne toucherait rien, et ce genre d'écart ne se voit pas.
 * Seul le glyphe VIVANT est survolable — la base atténuée n'est qu'un rappel d'absence.
 */
export function flagAt(
  carries: readonly ReplayFlagCarryReady[],
  layer: FlagCarriesInput,
  view: CanvasView,
  frame: number,
  at: XY,
): FlagHit | null {
  let best: FlagHit | null = null
  let bestD = FLAG_HIT_RADIUS * FLAG_HIT_RADIUS
  for (const carry of carries) {
    const now = flagSpanAt(carry, frame)
    if (!now) continue
    const w = flagPointAt(now, frame, layer.posOf)
    const c = worldToCanvas(w, view.bounds, view.width, view.height, view.pad)
    const cx = c.x + FLAG_OFFSET_X
    const cy = c.y - FLAG_OFFSET_Y - FLAG_POLE_H / 2
    const d = (cx - at.x) * (cx - at.x) + (cy - at.y) * (cy - at.y)
    if (d <= bestD) {
      bestD = d
      best = { now, at: { x: cx, y: cy } }
    }
  }
  return best
}
