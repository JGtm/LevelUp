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
 *    c'est une BORNE HAUTE. Même icône, CREUSE — l'incertitude est à l'écran, pas dans une note.
 *  - `dropped` — le drapeau est au sol, à la position publiée : il n'est porté par personne mais
 *    il est toujours en jeu.
 *  - `home` — il est à sa base.
 *
 * QUI CLIGNOTE, ET POURQUOI (retour utilisateur du 2026-08-27 : le drapeau doit « clignoter hors
 * de son socle »). Les TROIS états hors base — `carried`, `carried_open`, `dropped` — clignotent ;
 * `home` est stable. La lecture que cela sert est celle d'un match de CTF : ce qui compte, c'est
 * de repérer d'un coup d'œil qu'un drapeau est SORTI, sans avoir à comparer deux positions. Un
 * clignotement au socle dirait l'inverse — que la situation est en cours — alors qu'un drapeau
 * chez lui est justement le repos. `prefers-reduced-motion` éteint le clignotement : opacité
 * pleine et fixe, le glyphe reste localisable (même règle que tous les effets du rejeu).
 *
 * L'ATTÉNUATION NE PORTE PLUS L'INCERTITUDE, LE CREUX LA PORTE. `carried_open` clignote comme les
 * autres états hors base et garde son fanion CREUX : deux signaux pour deux choses différentes —
 * le clignotement dit « hors de sa base », le creux dit « fin non datée ». Une opacité faible en
 * plus les aurait confondus, et le clignotement l'aurait de toute façon effacée.
 *
 * LA BASE PORTE UN ÉTAT PRÉSENT / ABSENT, et c'est la lecture que le plan demande : quand le
 * drapeau est `home`, le glyphe se pose PLEIN sur sa base ; quand il est ailleurs, la base garde
 * un glyphe ATTÉNUÉ et FIXE — « le drapeau n'est pas là ». Ce rappel d'absence NE clignote PAS :
 * c'est un repère de lieu, pas un événement ; le faire battre ferait deux clignotements
 * concurrents à l'écran pour un seul drapeau. Rien de plus : ni anneau, ni compte à rebours.
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
import { type XY } from '../../../lib/replay/replayLogic'

import { type CanvasView, projectTo } from '../model/replayView'
import type { ReplayFlagCarryReady } from '../../../lib/replay/replayNormalize'

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
  /**
   * Encre du FOND, résolue par l'appelant elle aussi : c'est celle du liseré posé sous le
   * glyphe. Ce fichier ne connaît AUCUN token — il ne sait même pas quel thème est actif.
   */
  outline: string
  /** Mouvement réduit : le clignotement hors base devient une opacité constante. */
  reducedMotion: boolean
}

// Réglages du calque. Le glyphe reste volontairement PLUS PETIT que le marqueur d'un joueur :
// il se pose À CÔTÉ du porteur, il ne doit pas le remplacer.
//
// ÉLARGI DE 20 % LE 2026-08-26 (retour utilisateur : « le drapeau faudrait les faire un tout
// petit peu plus gros »), PUIS DE 21 % DE PLUS LE 2026-08-27 (même lecteur, même écran : « le
// rendre aussi plus gros » — 1,2 ne suffisait pas). C'est toujours un AJUSTEMENT, pas un
// changement d'échelle : à 13 px de hampe le fanion faisait 9 × 6,5 px, et sur un fond de carte
// chargé il se confondait avec les repères voisins. Les trois cotes du glyphe montent ensemble —
// les élargir séparément déformerait le dessin — et le décalage ne bouge PAS : le glyphe grandit
// vers le haut et la droite, donc son ancrage sur le point qu'il qualifie reste le même.
//
// LE RAYON DE SURVOL SUIT, et il le doit : il couvre la hampe et le fanion. Le laisser à 12
// aurait rendu insurvolable la part de glyphe gagnée par l'élargissement.
const FLAG_GLYPH_SCALE = 1.45
const FLAG_POLE_H = 13 * FLAG_GLYPH_SCALE
const FLAG_WING_W = 9 * FLAG_GLYPH_SCALE
const FLAG_WING_H = 6.5 * FLAG_GLYPH_SCALE
/** Décalage du glyphe par rapport au point qu'il qualifie (au-dessus et à droite du marqueur). */
const FLAG_OFFSET_X = 6
const FLAG_OFFSET_Y = 2
const FLAG_STROKE_WIDTH = 1.6
/**
 * LISERÉ À L'ENCRE DU FOND — débord de chaque côté du trait, en pixels d'écran.
 *
 * POURQUOI (retour utilisateur du 2026-08-27 : le drapeau veut « un contour »). Le glyphe se
 * pose sur un fond de carte photographique : selon la zone, l'encre d'équipe passe sur du clair
 * ou du sombre, et son bord se dissout. La MÊME silhouette reposée dessous, plus épaisse, à
 * l'encre du fond, lui rend un bord franc sans lui inventer de couleur — c'est la technique
 * déjà employée par les vignettes de socle, et la seule qui marche dans les DEUX thèmes.
 */
const FLAG_OUTLINE_PAD = 1.6
/** Épaisseur du trait de liseré : le trait du glyphe, débordé des deux côtés. */
const FLAG_OUTLINE_WIDTH = FLAG_STROKE_WIDTH + 2 * FLAG_OUTLINE_PAD
/** Rayon de SURVOL, en pixels : il couvre la hampe et le fanion, sans déborder sur le voisin. */
export const FLAG_HIT_RADIUS = 12 * FLAG_GLYPH_SCALE

/** Opacité pleine : le drapeau à sa base, et le haut du clignotement. */
const ALPHA_SOLID = 0.95
/**
 * Opacité atténuée — elle ne sert PLUS QU'À LA BASE VIDE depuis le 2026-08-27 : c'est un rappel
 * d'absence, il doit rester en retrait du glyphe vivant. `carried_open`, lui, dit son incertitude
 * par son fanion creux (cf. l'en-tête) et non plus par une opacité faible.
 */
const ALPHA_FAINT = 0.38
/**
 * Bornes du CLIGNOTEMENT hors base. Le creux ne descend jamais sous `BLINK_MIN` : un glyphe qui
 * s'éteint tout à fait deviendrait introuvable une demi-seconde sur deux, et le lecteur perdrait
 * ce que le clignotement était censé lui montrer.
 */
const BLINK_MIN = 0.35
const BLINK_MAX = ALPHA_SOLID
/**
 * Période du clignotement, EN IMAGES — 10 images ≈ 1 s au pas de 100 ms documenté du rejeu.
 *
 * Elle est en images et non en millisecondes pour la même raison que l'ancienne respiration : le
 * calque ne reçoit que l'index d'image, jamais l'horloge du document. Conséquence assumée — un
 * film au pas différent clignoterait plus vite ou plus lentement ; aucun n'a été mesuré à ce
 * jour, et la lecture ne dépend pas de la période exacte.
 *
 * LE BATTEMENT EST UN COSINUS, ET C'EST MESURABLE : les images sont ENTIÈRES, et sur une période
 * paire un sinus ne tombe sur aucun de ses extrema (ils arrivent à la demi-image 2,5 et 7,5).
 * L'amplitude réellement affichée y perdait 5 % à chaque bord — 0,36..0,94 au lieu de 0,35..0,95.
 * Le cosinus place ses deux extrema sur des images entières : le clignotement montre toute sa
 * course, et les bornes ci-dessus sont ce qu'on voit, pas une intention.
 */
const BLINK_PERIOD_FRAMES = 10

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

/**
 * flagBlinkAlpha — l'opacité du glyphe à cette image : STABLE à la base, CLIGNOTANTE ailleurs.
 *
 * FONCTION PURE, exportée : le calque du crâne d'Oddball la réutilisera telle quelle (renvoi
 * écrit dans PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md, décision 7) — un objet porté hors de son socle
 * se lit de la même façon, et deux copies de cette formule battraient à des rythmes différents.
 *
 * UN ÉTAT INCONNU SUIT LE COMPORTEMENT « PRÉSENT » — opacité pleine et fixe. C'est la règle déjà
 * posée par l'en-tête pour les artefacts plus récents que ce code : l'objet EST quelque part,
 * l'artefact le dit, et faire clignoter ce qu'on ne sait pas nommer affirmerait « il est sorti ».
 *
 * Elle remplace l'ancienne respiration du `dropped` (2026-08-27) : les deux auraient dit la même
 * chose de deux façons, et l'utilisateur en a demandé une seule.
 */
export function flagBlinkAlpha(state: string, frame: number, reducedMotion: boolean): number {
  if (reducedMotion) return ALPHA_SOLID
  if (state !== 'carried' && state !== 'carried_open' && state !== 'dropped') return ALPHA_SOLID
  const phase = (2 * Math.PI * frame) / BLINK_PERIOD_FRAMES
  return BLINK_MIN + (BLINK_MAX - BLINK_MIN) * (0.5 + 0.5 * Math.cos(phase))
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
  const px = (p: XY) => projectTo(view, p)
  for (const carry of carries) {
    const now = flagSpanAt(carry, frame)
    if (!now) continue
    const ink = layer.style.colorOfTeam(carry.team)
    const outline = layer.style.outline
    const anchor = homeAnchorOf(carry)
    // LA BASE D'ABORD, et seulement quand le drapeau n'y est pas : c'est le fond de la lecture,
    // le glyphe vivant se pose par-dessus (ils coïncident quand l'état est `home`).
    if (anchor && now.state !== 'home') {
      drawFlagGlyph(ctx, px(anchor), { ink, outline, alpha: ALPHA_FAINT, hollow: true })
    }
    const at = px(flagPointAt(now, frame, layer.posOf))
    drawFlagGlyph(ctx, at, {
      ink,
      outline,
      alpha: flagBlinkAlpha(now.state, frame, layer.style.reducedMotion),
      hollow: now.state === 'carried_open',
    })
  }
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'un glyphe a besoin de savoir (règle des 5 paramètres). */
export interface FlagGlyphPaint {
  ink: string
  /** Encre du FOND : celle du liseré posé sous la silhouette (cf. FLAG_OUTLINE_PAD). */
  outline: string
  alpha: number
  /** Fanion CREUX (liseré seul) : l'état est incertain, ou la base est vide. */
  hollow: boolean
}

/** Chemin de la HAMPE — tracé deux fois par glyphe (liseré puis trait), jamais recopié. */
function traceFlagPole(ctx: CanvasRenderingContext2D, x: number, foot: number, top: number): void {
  ctx.beginPath()
  ctx.moveTo(x, foot)
  ctx.lineTo(x, top)
}

/**
 * Contour du FANION posé sur le chemin COURANT, sans l'ouvrir : c'est la seule copie de cette
 * géométrie du fichier. Elle sert seule (`traceFlagWing`) et composée avec l'emprise du glyphe
 * (`strokeWingOutlineOutside`) — un troisième triangle écrit à la main dériverait des deux autres.
 */
function wingSubpath(ctx: CanvasRenderingContext2D, x: number, top: number): void {
  ctx.moveTo(x, top)
  ctx.lineTo(x + FLAG_WING_W, top + FLAG_WING_H / 2)
  ctx.lineTo(x, top + FLAG_WING_H)
  ctx.closePath()
}

/** Chemin du FANION — même raison que la hampe : une seule géométrie pour les deux passes. */
function traceFlagWing(ctx: CanvasRenderingContext2D, x: number, top: number): void {
  ctx.beginPath()
  wingSubpath(ctx, x, top)
}

/**
 * strokeWingOutlineOutside dépose le liseré du fanion CREUX en n'en gardant que la moitié
 * EXTÉRIEURE, par écrêtage.
 *
 * POURQUOI CE DÉTOUR (revue adversariale R1 du 2026-08-27, P1). Un trait de canvas est CENTRÉ sur
 * son chemin : à 4,8 px, le liseré déborde de 2,4 px DE CHAQUE CÔTÉ, donc 2,4 px vers l'intérieur
 * du fanion, dont le trait d'encre (1,6 px, soit 0,8 px vers l'intérieur) ne recouvre qu'un tiers.
 * Sur un fanion de rayon inscrit 3,31 px, il ne restait que 4,64 px² de creux visible contre
 * 21,1 px² avant le lot — le creux était comblé à ~87 %, et c'est lui, depuis ce même lot, qui
 * porte SEUL la fin non datée de `carried_open` (l'atténuation lui a été retirée).
 *
 * LA RÈGLE `evenodd` FAIT LE TRAVAIL : le chemin d'écrêtage compose l'emprise du glyphe ET le
 * fanion ; un point intérieur au fanion est entouré par DEUX sous-chemins — compte pair, donc
 * HORS région. Le liseré ne peut alors se déposer que dehors, et le creux reste entier.
 *
 * `save`/`restore` encadrent strictement l'écrêtage : une région d'écrêtage laissée ouverte
 * rognerait TOUS les calques suivants, et cela ne se verrait pas ici.
 */
function strokeWingOutlineOutside(
  ctx: CanvasRenderingContext2D,
  x: number,
  top: number,
  foot: number,
): void {
  ctx.save()
  ctx.beginPath()
  const m = FLAG_OUTLINE_WIDTH
  ctx.rect(x - m, top - m, FLAG_WING_W + 2 * m, foot - top + 2 * m)
  wingSubpath(ctx, x, top)
  ctx.clip('evenodd')
  traceFlagWing(ctx, x, top)
  ctx.stroke()
  ctx.restore()
}

/**
 * drawFlagGlyph trace le drapeau : une HAMPE verticale et un FANION triangulaire, posés sur un
 * LISERÉ à l'encre du fond — la même silhouette, dessous, plus épaisse (2026-08-27).
 *
 * LE LISERÉ SE DÉPOSE PAR DEUX VOIES, ET LE FANION CREUX EXIGE LA SECONDE (revue R1 du
 * 2026-08-27, P1) :
 *
 *  - FANION PLEIN et HAMPE — voie simple : le liseré est tracé AVANT, puis le remplissage (ou,
 *    pour la hampe, le trait d'encre) recouvre ce qui déborde vers l'intérieur. Une hampe n'a
 *    pas d'intérieur, un fanion plein est repeint : rien ne subsiste du débord intérieur.
 *  - FANION CREUX — voie écrêtée (`strokeWingOutlineOutside`) : là, RIEN ne repeint l'intérieur.
 *    Le trait de canvas étant CENTRÉ sur son chemin, la moitié intérieure du liseré (2,4 px) s'y
 *    déposait et le trait d'encre n'en recouvrait que 0,8 px : il restait 4,64 px² de creux sur
 *    les 21,1 px² d'avant le lot — comblé à ~87 %. L'écrêtage `evenodd` retire l'intérieur du
 *    fanion de la région de dessin, et le creux redevient entier.
 *
 * CE N'EST PAS UN DÉTAIL DE STYLE : depuis ce même lot, le creux porte SEUL la fin non datée de
 * `carried_open` (l'atténuation lui a été retirée au profit du clignotement). Un creux comblé,
 * c'est un état qui ne se distingue plus de `carried`.
 *
 * IL SUIT L'OPACITÉ DU GLYPHE : un rappel d'absence atténué ne doit pas être cerné d'un trait
 * franc, sinon c'est le liseré qu'on voit et l'atténuation ne dit plus rien.
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
  // La pointe du fanion est un angle aigu : en `miter` le liseré y pousserait une aiguille
  // plusieurs fois plus longue que le débord demandé. `round` la borne au débord.
  ctx.lineJoin = 'round'
  ctx.strokeStyle = paint.outline
  ctx.lineWidth = FLAG_OUTLINE_WIDTH
  traceFlagPole(ctx, x, foot, top)
  ctx.stroke()
  // Le fanion CREUX écrête son liseré ; le plein n'en a pas besoin (son remplissage recouvre le
  // débord intérieur), et l'écrêter pour rien coûterait un save/clip/restore par image et par
  // drapeau.
  if (paint.hollow) strokeWingOutlineOutside(ctx, x, top, foot)
  else {
    traceFlagWing(ctx, x, top)
    ctx.stroke()
  }
  ctx.strokeStyle = paint.ink
  ctx.fillStyle = paint.ink
  ctx.lineWidth = FLAG_STROKE_WIDTH
  traceFlagPole(ctx, x, foot, top)
  ctx.stroke()
  traceFlagWing(ctx, x, top)
  if (paint.hollow) ctx.stroke()
  else ctx.fill()
  // L'état du contexte ne fuit pas vers les calques suivants (même règle que `globalAlpha`).
  ctx.lineJoin = 'miter'
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
    const c = projectTo(view, w)
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
