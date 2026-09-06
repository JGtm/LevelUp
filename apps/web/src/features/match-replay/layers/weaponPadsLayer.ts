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
 * UNE PILE VERTICALE, ET LE LOSANGE EST SON PIED (retour utilisateur du 2026-08-27 : « l'icône
 * doit être au-dessus du petit losange, pas dedans »). De bas en haut : le LOSANGE à la position
 * exacte du socle, la VIGNETTE posée au-dessus de lui (son bas au sommet du losange, augmenté de
 * `PAD_GAP_PX`), le COMPTEUR au-dessus. Trois marques, une seule colonne, un seul lieu.
 *
 * L'ANNEAU-BORDURE D'A13 N'EXISTE PLUS, et c'est le même retour : ce liseré losange, posé autour
 * de la vignette à l'encre de la nature, ENFERMAIT l'image — l'écran disait « l'icône dans le
 * losange » là où la donnée dit « une arme au-dessus d'un lieu ». Sa fonction — rendre la nature
 * du socle vive sur n'importe quel fond — revient au losange, désormais TOUJOURS PLEIN à cette
 * encre : un aplat de 6 px se lit mieux qu'un liseré de 1,4 px, et il ne cerne rien.
 *
 * LES TROIS ÉTATS SE LISENT AILLEURS (`weaponPadTime.ts`) : ce fichier ne fait que les
 * DESSINER, et il les dessine sur le LOSANGE, qui garde sa forme et son encre dans les trois
 * cas — plein à 0,95 (présence prouvée) ; plein atténué à 0,55, cerné d'un halo losange
 * POINTILLÉ (le film ne dit rien) ; plein très atténué à 0,35 et sans vignette (absence
 * prouvée).
 *
 * LE POINTILLÉ GARDE SON SENS — celui que `placementShapes` lui a donné : « cette limite n'est
 * pas affirmée ». Ici la limite est TEMPORELLE et non spatiale, mais le message est le même, et
 * c'est pour cela qu'il n'apparaît QUE sur l'état incertain : le losange plein, lui, dit une
 * présence ou une absence que la mesure tient.
 *
 * CE QUE CE CALQUE NE DESSINE JAMAIS :
 *  - LE RAMASSEUR. Le champ (`padPickups[].xuid`) est PUBLIÉ depuis le schéma 30 (2026-08-31),
 *    l'événement natif le portant. Aucune ligne d'ici ne le lit : ce calque dessine des LIEUX,
 *    pas des joueurs. C'est un choix de calque, plus une absence de donnée.
 *  - LES OBJETS LÂCHÉS. Ce sont les armes qu'un joueur relâche en mourant ; elles ne sont pas
 *    des socles et ne sont pas publiées ici (décision utilisateur du 18/08).
 *  - LA DIFFÉRENCE SOCLE AU SOL / RÂTELIER MURAL. La donnée ne porte qu'une position : rien
 *    ne les sépare, et l'écran dit « emplacement d'arme » plutôt que d'en choisir un.
 *  - UN COMPTE À REBOURS SANS SOURCE. Ni prochaine apparition mesurée, ni cycle établi : rien
 *    ne s'écrit — un tiret suggérerait qu'on saurait (cf. `padRespawnAt`).
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D, comme les calques voisins.
 * L'encre arrive de l'appelant, qui la tient des variables du thème.
 */
import type { ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'
import { project, UNCERTAIN_DASH, type PlacementView } from './placementShapes'
import type { PadScale } from '../model/weaponPadFamilies'
import type { XY } from '../../../lib/replay/replayLogic'
import { padRespawnAt, padStateAt, type PadState } from '../model/weaponPadTime'

/** Le cadrage est celui des poses : les deux calques projettent la même scène. */
export type { PlacementView as PadView } from './placementShapes'

/**
 * Rayon du LOSANGE du socle, en pixels d'ÉCRAN, par taille (avant compensation d'aire).
 *
 * LE LOSANGE DIT LE LIEU ET L'ÉTAT, la vignette dit ce qu'on y trouve (verdict du 2026-08-18,
 * amendé le 2026-08-27) : la vignette est posée AU-DESSUS, libre, et plus jamais dans un
 * anneau — un contenu enfermé à 8 px n'est pas lisible.
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

/** Écart entre deux étages de la pile (losange, vignette, compteur), en pixels d'écran. */
const PAD_GAP_PX = 2.5

/**
 * LE HALO DE L'INCERTITUDE : un losange POINTILLÉ concentrique au socle, écarté de `GAP` pour
 * qu'on le distingue du plein qu'il cerne, tracé au plus fin qui se lise encore sur un fond de
 * carte imprimé. Il ne paraît QUE sur l'état incertain — c'est son seul rôle.
 */
const PAD_HALO_GAP_PX = 2
const PAD_HALO_WIDTH = 1.2

/**
 * LE SOCLE EST UN LOSANGE, PLUS UN ROND (A14, retour utilisateur du 2026-08-26 : « les socles
 * d'armes et de power up je veux pas de cercles, des points en losanges ce serait mieux, ça
 * facilite la lecture sinon on peut confondre avec des points de joueurs »).
 *
 * C'EST UNE FORME, PAS UNE TAILLE : le losange remplace le disque À LA MÊME PLACE, avec les
 * mêmes encres et les mêmes états. Un marqueur de JOUEUR reste un rond ; le vocabulaire de la
 * carte gagne donc une distinction franche, lisible même en petit et même en niveaux de gris.
 *
 * LA COMPENSATION D'AIRE N'EST PAS UN ORNEMENT. À demi-diagonale égale, un losange couvre
 * `2r²` quand un disque couvre `πr²` — soit 64 % seulement. Remplacer l'un par l'autre sans
 * corriger aurait AMAIGRI tous les socles d'un tiers, ce qui n'est pas ce qui a été demandé.
 * Le facteur est `sqrt(π/2)` ≈ 1,2533, arrondi : à ce grossissement, les deux formes pèsent le
 * même poids d'encre à l'écran.
 */
const PAD_DIAMOND_GROWTH = 1.25

/**
 * traceDiamond pose le contour d'un losange centré, de demi-diagonale `half` (sommet en haut).
 *
 * UNE SEULE COPIE POUR LES DEUX USAGES ICI — la marque et son halo d'incertitude. Deux tracés de
 * la même forme divergeraient au premier réglage, et l'écart serait invisible : un halo
 * légèrement désaligné de son losange reste crédible.
 *
 * EXPORTÉE depuis le lot véhicules (2026-09-02) : le repère « famille de châssis non résolue »
 * de `vehiclesLayer.ts` est le même losange neutre que ce fichier trace déjà pour un socle — le
 * vocabulaire du dépôt pour « un objet de la carte, pas un joueur » (cf. `replayMarkers.corePath`,
 * troisième copie du même tracé). RÉUTILISER cette fonction plutôt que d'en écrire une troisième
 * est ce que CLAUDE.md n°6 demande à la 3e copie — la centralisation complète des trois (dont
 * `replayMarkers.ts`) reste une dette pré-existante, hors périmètre de ce lot (cf. Découvertes).
 */
export function traceDiamond(ctx: CanvasRenderingContext2D, c: XY, half: number): void {
  ctx.beginPath()
  ctx.moveTo(c.x, c.y - half)
  ctx.lineTo(c.x + half, c.y)
  ctx.lineTo(c.x, c.y + half)
  ctx.lineTo(c.x - half, c.y)
  ctx.closePath()
}

/**
 * Épaisseur du LISERÉ de la vignette, en pixels d'écran, et le nombre de directions où on la
 * repose pour l'obtenir. Huit : à quatre, les diagonales laissent passer le fond.
 */
const PAD_OUTLINE_PX = 1.2
const PAD_OUTLINE_STEPS = 8

/** Opacités du losange et de la vignette, par état. `empty` n'a pas de vignette. */
const PAD_ALPHA: Record<PadState, { dot: number; icon: number }> = {
  full: { dot: 0.95, icon: 0.95 },
  uncertain: { dot: 0.55, icon: 0.3 },
  empty: { dot: 0.35, icon: 0 },
}

/** Corps du compte à rebours, en pixels d'écran, et son contour de lisibilité. */
const PAD_COUNTDOWN_FONT_PX = 8
const PAD_COUNTDOWN_STROKE_PX = 2.4

/**
 * Marge de confort autour de la pile, en pixels d'écran : on vise près, pas au pixel.
 *
 * IL N'Y A PLUS DE PLANCHER DE VISABILITÉ (2026-08-27, revue adversariale). La zone portait un
 * `Math.max(..., 9 px)` hérité du temps où elle ne couvrait que le losange — un losange de
 * 3,2 px ne se vise pas. Depuis qu'elle couvre la pile, son premier terme vaut 12,25 px pour la
 * PLUS PETITE taille (4 + 5,25 + 3) et 16,5 px pour la grande : le plancher ne pouvait plus
 * jamais l'emporter. Un `Math.max` dont une branche est inatteignable est du code mort qui
 * ment sur la règle, pas une sécurité.
 */
const PAD_HOVER_MARGIN_PX = 3

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
 * s'obtient en reposant la MÊME forme, teinte de l'encre du fond, tout autour du corps. Les
 * DEUX sont toujours servies depuis le 2026-08-27, y compris pour une image finie du jeu : un
 * liseré n'a besoin que de la SILHOUETTE, que la composition `source-in` rend de n'importe
 * quelle image à alpha (cf. `tintedIconCanvas`).
 */
export interface PadIcon {
  fill: CanvasImageSource
  outline: CanvasImageSource
}

/** Ce que le calque emprunte au thème et au catalogue du document. */
export interface PadStyle {
  /** Encre neutre du thème : un socle est un objet du terrain, il n'a pas de camp. */
  ink: string
  /**
   * Les DEUX encres du marquage : ce qui est REMPLI à l'encre du texte, ce qui est CERNÉ à
   * celle du fond. En thème sombre cela donne le « blanc rempli, contour sombre » ; en thème
   * clair, l'inverse.
   *
   * ELLES NE VALENT PLUS QUE POUR LE COMPTE À REBOURS depuis le 2026-08-28. La VIGNETTE, elle,
   * est cernée de l'encre de sa NATURE (celle de son losange) : sur un fond de carte en niveaux
   * de gris déjà cerné de noir, un liseré noir se confondait avec le décor et l'arme devenait
   * illisible (retour utilisateur). Le choix se fait à la cuisson (`useReplayWeaponPads`) — ce
   * calque reçoit les deux images déjà teintes et n'a pas à connaître la règle.
   */
  fill: string
  outline: string
  /** Vignette TEINTE de la famille, ou null : le socle garde alors son seul losange. */
  iconOf: (weapon: string) => PadIcon | null
  /** La taille à donner au socle, d'après ce qu'il porte (cf. weaponPadFamilies). */
  scaleOf: (weapon: string) => PadScale
  /**
   * L'encre de la NATURE du socle — power-up, arme de puissance, râtelier (A13, 2026-08-26).
   * RÉSOLUE PAR L'APPELANT comme toutes les autres (règle color-tokens) : ce fichier ne
   * connaît aucun token, il ne fait qu'employer la chaîne qu'on lui donne.
   */
  inkOf: (weapon: string) => string
  /**
   * Le compte à rebours déjà localisé, dans son écriture COMPACTE (celle de la carte).
   *
   * APPELÉ SUR TOUT SOCLE VIDE QUI A UNE SOURCE (D3, 2026-08-27) : une prochaine apparition
   * mesurée, ou à défaut un cycle établi. La précondition d'avant — « seulement quand un cycle
   * est établi » — est précisément le défaut que D3 corrige, la moitié des socles n'en portant
   * aucun. Ce libellé ne dit PAS d'où vient le chiffre : c'est l'infobulle qui le distingue.
   */
  countdownLabel: (seconds: number) => string
}

/** padRadiusPx — le rayon d'écran du LOSANGE d'un socle (sa taille suit ce qu'il porte). */
export function padRadiusPx(pad: ReplayWeaponPadReady, style: PadStyle, k: number): number {
  return PAD_DOT_PX[style.scaleOf(pad.weapon)] * k
}

/** Les trois mesures d'écran de la pile : le losange, la vignette, l'écart qui les sépare. */
interface PadStackPx {
  /** Demi-diagonale du losange, compensation d'aire comprise. */
  half: number
  /** Hauteur de la vignette posée au-dessus. */
  iconH: number
  /** Écart entre deux étages. */
  gap: number
}

/**
 * padStackPx — LA PILE D'UN SOCLE en pixels d'écran, à la densité courante. UNE SEULE SOURCE
 * POUR LE TRACÉ ET LE SURVOL : deux copies de ces trois lignes divergeraient au premier réglage,
 * et l'écart se paierait en infobulles qui n'apparaissent pas là où l'œil vise.
 */
function padStackPx(pad: ReplayWeaponPadReady, style: PadStyle, k: number): PadStackPx {
  return {
    half: padRadiusPx(pad, style, k) * PAD_DIAMOND_GROWTH,
    iconH: PAD_ICON_H_PX[style.scaleOf(pad.weapon)] * k,
    gap: PAD_GAP_PX * k,
  }
}

/**
 * padAt — le socle sous un point du canvas, ou null.
 *
 * LA ZONE EST UN DISQUE CENTRÉ SUR LA COLONNE DE LA PILE (2026-08-27), assez haut pour couvrir
 * du bas du losange au sommet de la vignette : viser l'image et n'attraper que le losange, dix
 * pixels plus bas, serait exactement le défaut que la nouvelle grammaire crée si on ne relève
 * pas la zone avec elle.
 *
 * CE QU'IL FAUT SAVOIR DE CE DISQUE, parce qu'un disque n'est pas une pile : il couvre la
 * COLONNE CENTRALE sur toute la hauteur, mais les FLANCS EXTRÊMES d'une vignette très large
 * (rapport proche de la borne `PAD_ICON_MAX_ASPECT`, soit ~3) en sortent — de quelques pixels à
 * mi-hauteur, d'une dizaine aux coins hauts. C'est ASSUMÉ (revue adversariale du 2026-08-27) :
 * un rectangle rendrait l'arbitrage entre socles voisins nettement moins simple pour un gain qui
 * ne concerne que les extrémités d'une image, là où l'œil ne vise pas.
 *
 * ELLE NE DÉPEND NI DE L'ÉTAT NI DU CHARGEMENT des vignettes : un socle vide occupe le même lieu
 * qu'un socle plein, et une cible qui bougerait avec l'instant lu serait impossible à viser.
 *
 * Le survol se rejoue sur la DONNÉE, jamais sur les pixels : ce sont les mêmes positions
 * projetées que le tracé. Le plus proche l'emporte quand deux socles se recouvrent — deux
 * armes peuvent partager un mètre carré sur une petite arène.
 *
 * DEUX DISTANCES, ET C'EST VOLONTAIRE : l'APPARTENANCE se juge sur la pile (centre relevé),
 * l'ARBITRAGE entre deux socles atteints se juge sur le LIEU. Départager sur les piles ferait
 * gagner le voisin le plus petit — sa pile est plus basse, donc son centre plus proche du sol —
 * alors même que le pointeur est pile sur le losange de l'autre. Un socle est sa position.
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
    const { half, iconH, gap } = padStackPx(pad, style, k)
    // La pile monte de `iconH + gap` au-dessus du losange : son centre monte donc de la
    // moitié, et son demi-hauteur vaut celle du losange augmentée de la même moitié.
    const rise = (iconH + gap) / 2
    const reach = half + rise + PAD_HOVER_MARGIN_PX * k
    const inPile = (at.x - c.x) ** 2 + (at.y - (c.y - rise)) ** 2 <= reach * reach
    const d2 = (at.x - c.x) ** 2 + (at.y - c.y) ** 2
    if (inPile && d2 < bestD2) {
      best = pad
      bestD2 = d2
    }
  }
  return best
}

/**
 * drawDot — LE LOSANGE du socle : TOUJOURS PLEIN à l'encre de sa nature, et c'est l'OPACITÉ qui
 * dit l'état (2026-08-27).
 *
 * POURQUOI PLEIN DANS LES TROIS CAS. Il portait un remplissage à l'état plein et un simple
 * contour aux deux autres : un trait de 1,2 px à 3 px de rayon, qui disparaissait sur un fond de
 * carte chargé — le lieu devenait invisible dès que l'arme était prise, c'est-à-dire au moment
 * précis où l'on cherche le socle. L'aplat tient dans tous les cas ; l'opacité le range derrière
 * la scène sans l'effacer.
 *
 * LE POINTILLÉ N'EST PAS PERDU, il est déplacé : l'état incertain gagne un HALO losange
 * pointillé concentrique, qui garde le sens que `placementShapes` lui donne partout ailleurs —
 * « cette limite n'est pas affirmée » — sans le payer de la lisibilité du lieu.
 */
function drawDot(
  ctx: CanvasRenderingContext2D,
  c: XY,
  half: number,
  state: PadState,
  k: number,
): void {
  ctx.globalAlpha = PAD_ALPHA[state].dot
  traceDiamond(ctx, c, half)
  ctx.fill()
  if (state !== 'uncertain') return
  ctx.lineWidth = PAD_HALO_WIDTH * k
  ctx.setLineDash(UNCERTAIN_DASH.map((d) => d * k))
  traceDiamond(ctx, c, half + PAD_HALO_GAP_PX * k)
  ctx.stroke()
  ctx.setLineDash([])
}

/**
 * drawPadIcon — la vignette de l'arme, posée AU-DESSUS du losange, remplie et cernée.
 *
 * Hauteur imposée, largeur déduite du rapport de l'image (bornée : une vignette d'arme est
 * très large). Le liseré est la même image reposée tout autour, à l'encre du fond : c'est ce
 * qui la détache d'un fond de carte clair comme d'un fond sombre.
 *
 * Sans vignette — famille hors catalogue du titre, ou visuel absent — RIEN ne prend sa place :
 * jamais l'icône d'une arme voisine, et pas davantage un glyphe de repli (retour utilisateur du
 * 2026-08-26 : « j'ai l'impression qu'il y en a deux »). Le losange porte déjà le lieu et
 * l'état ; le nom reste lisible au survol.
 */
function drawPadIcon(
  ctx: CanvasRenderingContext2D,
  centre: XY,
  h: number,
  icon: PadIcon,
  k: number,
): void {
  const source = icon.fill
  const natW = 'width' in source && typeof source.width === 'number' ? source.width : 0
  const natH = 'height' in source && typeof source.height === 'number' ? source.height : 0
  const aspect = natW > 0 && natH > 0 ? Math.min(natW / natH, PAD_ICON_MAX_ASPECT) : 1
  const w = h * aspect
  const x = centre.x - w / 2
  const y = centre.y - h / 2
  const d = PAD_OUTLINE_PX * k
  for (let i = 0; i < PAD_OUTLINE_STEPS; i++) {
    const a = (i / PAD_OUTLINE_STEPS) * Math.PI * 2
    ctx.drawImage(icon.outline, x + Math.cos(a) * d, y + Math.sin(a) * d, w, h)
  }
  ctx.drawImage(source, x, y, w, h)
}

/**
 * drawCountdown — le compte à rebours AU SOMMET DE LA PILE, rempli et cerné comme la vignette.
 *
 * SEULEMENT LE COMPTE (verdict du 2026-08-18) : ni médiane, ni nombre d'écarts, ni marge —
 * ces trois-là disaient la CONFIANCE dans le cycle, ce qui est une lecture d'analyse, pas un
 * repère de carte. Le compte, lui, répond à la seule question qu'on se pose en regardant un
 * socle vide : dans combien de temps.
 *
 * `topY` est le sommet de ce qui est RÉELLEMENT dessiné, et l'appelant n'a en pratique qu'un
 * candidat à lui passer : le sommet du LOSANGE. Un compte à rebours ne s'écrit que sur un socle
 * vide, et un socle vide n'a pas de vignette — ancrer plus haut ferait flotter le chiffre
 * au-dessus d'un trou, détaché du socle qu'il annonce.
 */
function drawCountdown(
  ctx: CanvasRenderingContext2D,
  c: XY,
  topY: number,
  text: string,
  style: { fill: string; outline: string; k: number },
): void {
  ctx.globalAlpha = 1
  ctx.font = `600 ${PAD_COUNTDOWN_FONT_PX * style.k}px ui-sans-serif, system-ui, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'bottom'
  const y = topY - PAD_GAP_PX * style.k
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
 * drawOnePad — LA PILE D'UN SOCLE, de bas en haut : le losange au lieu mesuré, la vignette
 * au-dessus (son bas au sommet du losange augmenté d'un écart, jamais sur lui), le compteur au
 * sommet. L'ordre vertical EST la règle du retour du 2026-08-27.
 */
function drawOnePad(
  ctx: CanvasRenderingContext2D,
  pad: ReplayWeaponPadReady,
  c: XY,
  time: PadTime,
  style: PadStyle,
): void {
  const { half, iconH, gap } = padStackPx(pad, style, time.k)
  const state = padStateAt(pad, time.frame)
  // LE LOSANGE, À L'ENCRE DE SA NATURE (A13, 2026-08-26 : « bordure et couleur plus vive, une
  // couleur pour chaque type »). Il portait l'encre neutre du terrain, la même pour les trois :
  // un socle de surbouclier ne se distinguait d'un râtelier que par sa taille. `style.ink` reste
  // servi et reste le NEUTRE — il ne teint plus la marque, et le râtelier ordinaire retombe sur
  // une encre propre par sa propre famille.
  const encre = style.inkOf(pad.weapon)
  ctx.strokeStyle = encre
  ctx.fillStyle = encre
  drawDot(ctx, c, half, state, time.k)
  const iconAlpha = PAD_ALPHA[state].icon
  const icon = iconAlpha > 0 ? style.iconOf(pad.weapon) : null
  const iconBottom = c.y - half - gap
  if (icon) {
    ctx.globalAlpha = iconAlpha
    drawPadIcon(ctx, { x: c.x, y: iconBottom - iconH / 2 }, iconH, icon, time.k)
  }
  // LE COMPTEUR NE DIT PAS SA SOURCE SUR LA CARTE, et c'est voulu : à 8 px, un chiffre suffit.
  // La distinction mesurée / attendue se lit au survol, dans l'infobulle (D3, 2026-08-27).
  //
  // IL S'ANCRE SUR LE LOSANGE, ET C'EST UNE CONSÉQUENCE, PAS UN CHOIX (revue adversariale du
  // 2026-08-27) : un compte à rebours n'existe que sur un socle VIDE (`padRespawnAt`), et un
  // socle vide n'a PAS de vignette (`PAD_ALPHA.empty.icon` vaut 0). Le sommet de la pile EST
  // donc toujours celui du losange ici. Le code portait un `icon ? … : …` dont la première
  // branche était inatteignable — il décrivait une pile que ce cas ne produit jamais.
  const respawn = padRespawnAt(pad, time.frame, time.frameMs)
  if (respawn === null) return
  drawCountdown(ctx, c, c.y - half, style.countdownLabel(respawn.seconds), {
    fill: style.fill,
    outline: style.outline,
    k: time.k,
  })
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
    drawOnePad(ctx, pad, project({ x: pad.x, y: pad.y }, view), time, style)
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
