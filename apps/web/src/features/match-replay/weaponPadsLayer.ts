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
 * LA POSITION PEUT VENIR D'AILLEURS depuis le 2026-08-19, et c'est la seule chose qui le
 * peut : quand la réponse porte `mapWeaponPads`, l'appelant a remplacé le centroïde des
 * apparitions par la position du SPAWNER lue dans le fichier de carte (cf.
 * `crossedWeaponPads`). Ce calque ne le sait pas et n'a pas à le savoir — il dessine ce
 * qu'on lui donne, et tout le reste (présence, états, cycle) reste la mesure du match.
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

/**
 * Rayon du POINT du socle, en pixels d'ÉCRAN, par taille.
 *
 * UN POINT, PLUS UN ANNEAU (verdict du 2026-08-18) : « point disponible + icône de l'arme en
 * dessous + compteur au-dessus ». L'anneau enfermait la vignette, ce qui donnait à chaque
 * socle l'emprise d'une cible et faisait de l'icône un contenu illisible à 8 px. Le point dit
 * le LIEU et l'ÉTAT ; l'icône, posée dessous et libre, dit ce qu'on y trouve.
 */
const PAD_DOT_PX: Record<PadScale, number> = { power: 4.6, classic: 3.2 }

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

/** Épaisseur du contour du point (états incertain et vide), en pixels d'écran. */
const PAD_DOT_WIDTH = 1.2

/** Écart entre le bord du point et ce qu'on pose au-dessus ou en dessous, en pixels d'écran. */
const PAD_GAP_PX = 2.5
/**
 * LA BORDURE DU SOCLE (A13, 2026-08-26) : l'anneau qui enferme la marque, à l'encre de sa
 * nature. `GAP` l'écarte de la vignette pour qu'il la cerne sans la mordre ; `WIDTH` est le
 * plus fin qui se lise encore sur un fond de carte imprimé.
 */
const PAD_BORDER_GAP_PX = 2
const PAD_BORDER_WIDTH = 1.4

/**
 * Épaisseur du LISERÉ de la vignette, en pixels d'écran, et le nombre de directions où on la
 * repose pour l'obtenir. Huit : à quatre, les diagonales laissent passer le fond.
 */
const PAD_OUTLINE_PX = 1.2
const PAD_OUTLINE_STEPS = 8

/** Opacités du point et de la vignette, par état. `empty` n'a pas de vignette. */
const PAD_ALPHA: Record<PadState, { dot: number; icon: number }> = {
  full: { dot: 0.95, icon: 0.95 },
  uncertain: { dot: 0.55, icon: 0.3 },
  empty: { dot: 0.35, icon: 0 },
}

/** Corps du compte à rebours, en pixels d'écran, et son contour de lisibilité. */
const PAD_COUNTDOWN_FONT_PX = 8
const PAD_COUNTDOWN_STROKE_PX = 2.4

/** Rayon minimal de la zone sensible au survol : un point de 3,2 px ne se vise pas. */
const PAD_HOVER_MIN_RADIUS_PX = 9

/** Ce que le calque a besoin de savoir de l'instant courant. */
export interface PadTime {
  frame: number
  /** Durée RÉELLE d'une image (`frameToMs(1, doc)`) : le compte à rebours bat en temps réel. */
  frameMs: number
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
}

/**
 * Une vignette d'arme prête à poser : son CORPS et son LISERÉ, déjà teints hors écran.
 *
 * DEUX IMAGES ET NON UNE, parce qu'un canvas ne sait pas cerner une image : le liseré
 * s'obtient en reposant la MÊME forme, teinte de l'encre du fond, tout autour du corps.
 * `outline` vaut null quand la source n'est pas un masque (image finie du jeu) : on ne peut
 * alors ni la reteindre ni la cerner, et elle se pose telle quelle.
 */
export interface PadIcon {
  fill: CanvasImageSource
  outline: CanvasImageSource | null
}

/** Ce que le calque emprunte au thème et au catalogue du document. */
export interface PadStyle {
  /** Encre neutre du thème : un socle est un objet du terrain, il n'a pas de camp. */
  ink: string
  /**
   * Les DEUX encres du marquage (verdict du 2026-08-18) : la vignette et le compte à rebours
   * sont REMPLIS de l'encre du texte et CERNÉS de celle du fond. En thème sombre cela donne
   * le « blanc rempli, contour noir » demandé ; en thème clair, l'inverse — c'est la même
   * règle, et c'est la seule qui reste lisible sur les deux fonds de carte.
   */
  fill: string
  outline: string
  /** Vignette TEINTE de la famille, ou null : le socle garde alors un glyphe neutre. */
  iconOf: (weapon: string) => PadIcon | null
  /** La taille à donner au socle, d'après ce qu'il porte (cf. weaponPadFamilies). */
  scaleOf: (weapon: string) => PadScale
  /**
   * L'encre de la NATURE du socle — power-up, arme de puissance, râtelier (A13, 2026-08-26).
   * RÉSOLUE PAR L'APPELANT comme toutes les autres (règle color-tokens) : ce fichier ne
   * connaît aucun token, il ne fait qu'employer la chaîne qu'on lui donne.
   */
  inkOf: (weapon: string) => string
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

/** padRadiusPx — le rayon d'écran du POINT d'un socle (sa taille suit ce qu'il porte). */
export function padRadiusPx(pad: ReplayWeaponPadReady, style: PadStyle, k: number): number {
  return PAD_DOT_PX[style.scaleOf(pad.weapon)] * k
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

/**
 * drawDot — LE POINT du socle : plein quand l'arme est prouvée là, pointillé quand le film ne
 * dit rien, discret quand l'absence est prouvée.
 *
 * L'ÉTAT SE LIT SANS LA COULEUR (le point n'en a qu'une, l'encre neutre du terrain) : c'est le
 * REMPLISSAGE qui dit « disponible », le POINTILLÉ qui dit « on ne sait pas » — la même
 * grammaire que `placementShapes`, où le pointillé a toujours voulu dire « non affirmé ».
 */
function drawDot(
  ctx: CanvasRenderingContext2D,
  c: XY,
  radius: number,
  state: PadState,
  time: PadTime,
): void {
  ctx.globalAlpha = PAD_ALPHA[state].dot
  ctx.beginPath()
  ctx.arc(c.x, c.y, radius, 0, Math.PI * 2)
  if (state === 'full') {
    ctx.fill()
    return
  }
  ctx.lineWidth = PAD_DOT_WIDTH * time.k
  ctx.setLineDash(state === 'uncertain' ? UNCERTAIN_DASH.map((d) => d * time.k) : [])
  ctx.stroke()
  ctx.setLineDash([])
}

/**
 * drawPadIcon — la vignette de l'arme, posée SOUS le point, remplie et cernée.
 *
 * Hauteur imposée, largeur déduite du rapport de l'image (bornée : une vignette d'arme est
 * très large). Le liseré est la même image reposée tout autour, à l'encre du fond : c'est ce
 * qui la détache d'un fond de carte clair comme d'un fond sombre.
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
  icon: PadIcon | null,
): void {
  const h = PAD_ICON_H_PX[scale] * time.k
  // PAS DE GLYPHE DE REPLI (retour utilisateur du 2026-08-26 : « j'ai l'impression qu'il y en
  // a deux »). Sans vignette, ce bloc posait un DISQUE PLEIN du même rayon que le point, à une
  // dizaine de pixels sous lui : deux ronds empilés pour un seul socle, que l'œil lisait comme
  // deux socles. Le point porte déjà la position ET l'état — il se suffit.
  if (!icon) return
  const source = icon.fill
  const natW = 'width' in source && typeof source.width === 'number' ? source.width : 0
  const natH = 'height' in source && typeof source.height === 'number' ? source.height : 0
  const aspect = natW > 0 && natH > 0 ? Math.min(natW / natH, PAD_ICON_MAX_ASPECT) : 1
  const w = h * aspect
  const x = c.x - w / 2
  const y = c.y - h / 2
  if (icon.outline) {
    const d = PAD_OUTLINE_PX * time.k
    for (let i = 0; i < PAD_OUTLINE_STEPS; i++) {
      const a = (i / PAD_OUTLINE_STEPS) * Math.PI * 2
      ctx.drawImage(icon.outline, x + Math.cos(a) * d, y + Math.sin(a) * d, w, h)
    }
  }
  ctx.drawImage(source, x, y, w, h)
}

/**
 * drawCountdown — le compte à rebours AU-DESSUS du point, rempli et cerné comme la vignette.
 *
 * SEULEMENT LE COMPTE (verdict du 2026-08-18) : ni médiane, ni nombre d'écarts, ni marge —
 * ces trois-là disaient la CONFIANCE dans le cycle, ce qui est une lecture d'analyse, pas un
 * repère de carte. Le compte, lui, répond à la seule question qu'on se pose en regardant un
 * socle vide : dans combien de temps.
 */
function drawCountdown(
  ctx: CanvasRenderingContext2D,
  c: XY,
  radius: number,
  text: string,
  style: { fill: string; outline: string; k: number },
): void {
  ctx.globalAlpha = 1
  ctx.font = `600 ${PAD_COUNTDOWN_FONT_PX * style.k}px ui-sans-serif, system-ui, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'bottom'
  const y = c.y - radius - PAD_GAP_PX * style.k
  if (style.outline) {
    ctx.lineJoin = 'round'
    ctx.lineWidth = PAD_COUNTDOWN_STROKE_PX * style.k
    ctx.strokeStyle = style.outline
    ctx.strokeText(text, c.x, y)
  }
  ctx.fillStyle = style.fill
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
  for (const pad of pads) {
    const c = project({ x: pad.x, y: pad.y }, view)
    const scale = style.scaleOf(pad.weapon)
    const radius = PAD_DOT_PX[scale] * time.k
    const state = padStateAt(pad, time.frame)
    // LE POINT, À L'ENCRE DE SA NATURE (A13, 2026-08-26 : « bordure et couleur plus vive, une
    // couleur pour chaque type »). Il portait l'encre neutre du terrain, la même pour les
    // trois : un socle de surbouclier ne se distinguait d'un râtelier que par sa taille.
    // `style.ink` reste servi et reste le NEUTRE — il ne teint plus le point, mais l'appelant
    // le passe toujours, et le râtelier ordinaire y retombe par sa propre famille.
    const encre = style.inkOf(pad.weapon)
    ctx.strokeStyle = encre
    ctx.fillStyle = encre
    drawDot(ctx, c, radius, state, time)
    // LA VIGNETTE SUR LE POINT, PLUS SOUS LUI (retour utilisateur du 2026-08-26). Elle était
    // posée une dizaine de pixels plus bas — un point ici, une image là — et les deux se
    // lisaient comme DEUX socles voisins plutôt que comme un seul. Centrée, elle coiffe le
    // point : UN socle, UNE marque. Le point reste dessous et garde son rôle, qui n'a jamais
    // été d'être vu pour lui-même : il porte l'ÉTAT (plein / incertain / vide) par sa forme.
    const iconAlpha = PAD_ALPHA[state].icon
    if (iconAlpha > 0) {
      ctx.globalAlpha = iconAlpha
      ctx.fillStyle = style.fill
      drawPadIcon(ctx, c, scale, time, style.iconOf(pad.weapon))
    }
    // LA BORDURE, par-dessus tout le reste : un anneau à l'encre de la nature, qui ENFERME la
    // marque (A13). C'est elle qui fait la « couleur plus vive » demandée — un liseré se lit sur
    // n'importe quel fond de carte, là où un aplat se dilue. Elle suit l'opacité du POINT, pas
    // celle de la vignette : elle appartient au socle, pas à ce qu'il porte, et un socle vide
    // doit garder un contour visible.
    ctx.globalAlpha = PAD_ALPHA[state].dot
    ctx.strokeStyle = encre
    ctx.lineWidth = PAD_BORDER_WIDTH * time.k
    ctx.beginPath()
    ctx.arc(c.x, c.y, (PAD_ICON_H_PX[scale] / 2 + PAD_BORDER_GAP_PX) * time.k, 0, Math.PI * 2)
    ctx.stroke()
    // LE COMPTE À REBOURS, AU-DESSUS du point.
    const left = padRespawnSecondsAt(pad, time.frame, time.frameMs)
    if (left !== null) {
      drawCountdown(ctx, c, radius, style.countdownLabel(left), {
        fill: style.fill,
        outline: style.outline,
        k: time.k,
      })
    }
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
