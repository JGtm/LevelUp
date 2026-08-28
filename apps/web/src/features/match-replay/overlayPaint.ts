/**
 * overlayPaint.ts — LES SURIMPRESSIONS DU REJEU, PEINTES DANS LA TOILE.
 *
 * # POURQUOI CE MODULE EXISTE
 *
 * L'écran de fin de match (`ReplayVictoryOverlay`) et le message inter-manche
 * (`ReplayRoundBreakOverlay`) sont du DOM posé PAR-DESSUS la toile. C'est le bon choix pour la
 * page : ils profitent du thème, de la typographie et de l'accessibilité sans une ligne de
 * dessin. Mais un encodeur vidéo ne voit QUE la toile — du DOM survolant, il ne sait rien. Sans
 * ce module, un export s'arrêterait sur un terrain muet, sans jamais dire comment le match
 * s'est terminé, ce qui est précisément ce que l'utilisateur regarde en dernier.
 *
 * # CE QUI EST PARTAGÉ AVEC LE DOM, ET CE QUI NE L'EST PAS
 *
 * PARTAGÉ : la VÉRITÉ. Ce module ne lit aucune donnée — il reçoit un panneau déjà décidé par
 * les mêmes modules de logique que les composants React (`victoryLogic`, `roundsLogic`,
 * `scoreBannerLogic`). Il ne peut donc pas annoncer un autre verdict ni un autre score que
 * l'écran : il n'a pas de quoi les calculer.
 *
 * PAS PARTAGÉ : la PEINTURE. Les classes Tailwind de `replayOverlayStyles.ts` n'ont pas
 * d'équivalent dans un contexte 2D ; les mesures ci-dessous les transcrivent en pixels, une
 * fois, avec la classe d'origine en commentaire. C'est la seule duplication du lot, et elle est
 * assumée : la fidélité se vérifie à l'œil sur l'export, pas par le typage.
 *
 * # CE QU'UN CANVAS NE SAIT PAS FAIRE, ET QU'ON N'IMITE PAS
 *
 * `backdrop-blur-sm` n'a pas d'équivalent : flouter ce qui est DÉJÀ peint dessous demanderait
 * de relire la toile, de la flouter et de la recomposer à chaque image — un coût par image pour
 * un effet que personne ne remarque sur un panneau opaque à 70 %. Il est volontairement absent.
 *
 * # LES COULEURS ARRIVENT RÉSOLUES, ELLES NE SE LISENT PAS ICI
 *
 * Un peintre qui appellerait `getComputedStyle` lirait le thème trente fois par seconde de
 * match. L'appelant résout une fois (`readInk`, `resolveToken` — cf. `useReplayInks`) et passe
 * les valeurs. C'est aussi ce qui rend ce fichier testable sans DOM.
 */

/** Les encres du thème dont les surimpressions ont besoin, déjà résolues par l'appelant. */
export interface OverlayInk {
  /** `--background` : la couleur du VOILE, posée à `VEIL_ALPHA`. */
  background: string
  /** `--foreground` : tout le texte. Jamais la couleur de camp — le contraste ne se négocie pas. */
  foreground: string
  /** `--muted-foreground` : le seul tiret séparateur du score. */
  muted: string
  /** `--border` et `--card` : le bloc NEUTRE (égalité, message inter-manche). */
  border: string
  card: string
}

/** Les familles de police de la page, lues une fois par l'appelant. */
export interface OverlayFonts {
  sans: string
  mono: string
}

/** Le fond et le bord du bloc de statut : couleur de camp résolue, ou tokens du thème. */
export interface OverlayStatusStyle {
  background: string
  border: string
}

/** Le score final, dans l'ordre du bandeau : allié à gauche, adverse à droite. */
export interface OverlayScore {
  ally: number
  enemy: number
}

/** Un panneau à peindre. Tout y est déjà décidé : ce module ne calcule aucune donnée. */
export interface OverlayPanel {
  /** Le verdict, ou « Manche N terminée ». Mis en capitales ici (le DOM le fait en CSS). */
  status: string
  statusStyle: OverlayStatusStyle
  /** Le nom de l'équipe du joueur de la page. Absent sur l'égalité et l'inter-manche. */
  label?: string | null
  /** Absent quand le mode ne publie pas de score — un « 0 - 0 » se lirait comme une mesure. */
  score?: OverlayScore | null
  /**
   * Le filigrane, DÉJÀ TEINTÉ par l'appelant (silhouette monochrome consommée en masque, cf.
   * `tintedIconCanvas`). Absent tant qu'il n'est pas chargé : le panneau n'a jamais eu besoin
   * du logo pour dire comment le match s'est terminé.
   */
  logo?: CanvasImageSource | null
  /** Le VOILE plein cadre : l'écran de fin en pose un, le message inter-manche non. */
  veil: boolean
}

/** La surface à couvrir, en pixels CSS — le même repère que `draw()` après sa mise à l'échelle. */
export interface OverlayView {
  width: number
  height: number
}

/** `bg-background/70` — l'opacité du voile de l'écran de fin. */
const VEIL_ALPHA = 0.7
/** `rounded-lg` = 0.5rem. */
const BLOCK_RADIUS = 8
/** `px-8 py-4` = 2rem / 1rem. */
const BLOCK_PAD_X = 32
const BLOCK_PAD_Y = 16
/** `border-2`. */
const BLOCK_BORDER = 2
/** `text-2xl` = 1.5rem, `font-bold` = 700. */
const STATUS_SIZE = 24
/** `tracking-wide` = 0.025em, exprimé en px à la taille du statut. */
const STATUS_TRACKING = STATUS_SIZE * 0.025
/** `text-sm` = 0.875rem, `font-semibold` = 600 (le nom d'équipe). */
const LABEL_SIZE = 14
const LABEL_TRACKING = LABEL_SIZE * 0.025
/** `text-3xl` = 1.875rem, `font-bold`, `font-mono` (le score final). */
const SCORE_SIZE = 30
/** `mt-2` = 0.5rem, entre le bloc, le nom et le score. */
const ROW_GAP = 8
/** `px-3` = 0.75rem de part et d'autre du tiret séparateur. */
const SCORE_SEP_PAD = 12
/** `h-[min(18rem,60vh)]` : 18rem, borné à 60 % de la hauteur de la toile. */
const LOGO_MAX = 288
const LOGO_VIEW_RATIO = 0.6
/** `opacity-20`. */
const LOGO_ALPHA = 0.2
/**
 * `shadow-lg` : `0 10px 15px -3px rgb(0 0 0 / 0.1)` — transcrit en ombre de contexte.
 *
 * LE NOIR EST ÉCRIT EN CLAIR, ET CE N'EST PAS UNE ENTORSE AUX TOKENS. Une ombre portée n'est
 * pas une couleur du thème : c'est un assombrissement de ce qu'il y a DESSOUS, identique en
 * thème clair et sombre — c'est d'ailleurs ainsi que Tailwind lui-même la définit. Même statut
 * que les contours de texte de `calloutsLayer.ts` et `zoneStatesLayer.ts`, écrits de la même
 * façon dans cette feature. Ce que la règle des tokens interdit, c'est une couleur qui DIT
 * quelque chose ; une ombre ne dit rien.
 */
const SHADOW_BLUR = 15
const SHADOW_OFFSET_Y = 10
const SHADOW_COLOR = 'rgba(0, 0, 0, 0.1)'

/** Les capitales du DOM (`uppercase`) sont une décision de style : elle se refait ici. */
function upper(text: string): string {
  return text.toLocaleUpperCase()
}

/**
 * setTracking pose l'interlettrage quand le navigateur le sait (`ctx.letterSpacing`, Chrome 99+).
 * Là où il ne le sait pas, le texte s'écrit sans — plus serré que le DOM, jamais illisible : un
 * interlettrage émulé lettre à lettre coûterait un `fillText` par caractère et par image.
 */
function setTracking(ctx: CanvasRenderingContext2D, px: number): void {
  if ('letterSpacing' in ctx) ctx.letterSpacing = `${px}px`
}

/** Largeur d'un texte à une police et un interlettrage donnés. */
function textWidth(ctx: CanvasRenderingContext2D, text: string, font: string, tracking: number): number {
  ctx.font = font
  setTracking(ctx, tracking)
  const w = ctx.measureText(text).width
  setTracking(ctx, 0)
  return w
}

/** roundedRect : `ctx.roundRect` quand il existe, un rectangle net sinon (jamais rien). */
function roundedRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
): void {
  ctx.beginPath()
  if (typeof ctx.roundRect === 'function') ctx.roundRect(x, y, w, h, r)
  else ctx.rect(x, y, w, h)
}

const statusFont = (f: OverlayFonts) => `700 ${STATUS_SIZE}px ${f.sans}`
const labelFont = (f: OverlayFonts) => `600 ${LABEL_SIZE}px ${f.sans}`
const scoreFont = (f: OverlayFonts) => `700 ${SCORE_SIZE}px ${f.mono}`

/** Le texte du score, sans le tiret : il se peint à part, dans une autre encre. */
function scoreParts(score: OverlayScore): { ally: string; enemy: string; sep: string } {
  return { ally: String(score.ally), enemy: String(score.enemy), sep: '-' }
}

/**
 * panelHeight mesure la colonne complète, pour la centrer VERTICALEMENT comme le fait le
 * `flex items-center` du DOM. Sans cette mesure préalable, le panneau se peindrait depuis le
 * haut et flotterait au-dessus du centre dès qu'il porte un score.
 */
function panelHeight(panel: OverlayPanel): number {
  let h = statusBlockHeight()
  if (panel.label) h += ROW_GAP + LABEL_SIZE
  if (panel.score) h += ROW_GAP + SCORE_SIZE
  return h
}

/** paintVeil couvre toute la toile — c'est le fond sur lequel le nom et le score se lisent. */
function paintVeil(ctx: CanvasRenderingContext2D, view: OverlayView, ink: OverlayInk): void {
  ctx.save()
  ctx.globalAlpha = VEIL_ALPHA
  ctx.fillStyle = ink.background
  ctx.fillRect(0, 0, view.width, view.height)
  ctx.restore()
}

/**
 * paintLogo pose le filigrane, centré sur la colonne et SOUS le texte (l'ordre du DOM, où il
 * est en position absolue derrière les paragraphes).
 */
function paintLogo(
  ctx: CanvasRenderingContext2D,
  view: OverlayView,
  logo: CanvasImageSource,
): void {
  const side = Math.min(LOGO_MAX, view.height * LOGO_VIEW_RATIO)
  ctx.save()
  ctx.globalAlpha = LOGO_ALPHA
  ctx.drawImage(logo, (view.width - side) / 2, (view.height - side) / 2, side, side)
  ctx.restore()
}

/**
 * La hauteur de la carte de statut. UNE SEULE definition, consommee par la MESURE et par la
 * PEINTURE : les deux divergeaient de `BLOCK_BORDER * 2`, et le panneau se peignait 2 px
 * au-dessus du centre avec un ecart bloc-nom trop court de 4 px.
 */
function statusBlockHeight(): number {
  return STATUS_SIZE + BLOCK_PAD_Y * 2 + BLOCK_BORDER * 2
}

/** paintStatusBlock peint la carte colorée et le verdict qu'elle porte. */
function paintStatusBlock(
  ctx: CanvasRenderingContext2D,
  view: OverlayView,
  panel: OverlayPanel,
  fonts: OverlayFonts,
  ink: OverlayInk,
  top: number,
): number {
  const text = upper(panel.status)
  const font = statusFont(fonts)
  const w = textWidth(ctx, text, font, STATUS_TRACKING) + BLOCK_PAD_X * 2
  const h = statusBlockHeight()
  const x = (view.width - w) / 2
  ctx.save()
  ctx.shadowColor = SHADOW_COLOR
  ctx.shadowBlur = SHADOW_BLUR
  ctx.shadowOffsetY = SHADOW_OFFSET_Y
  ctx.fillStyle = panel.statusStyle.background
  roundedRect(ctx, x, top, w, h, BLOCK_RADIUS)
  ctx.fill()
  // L'OMBRE NE DOIT PAS DOUBLER SUR LE BORD : elle appartient à la carte, pas à son trait.
  ctx.shadowColor = 'transparent'
  ctx.lineWidth = BLOCK_BORDER
  ctx.strokeStyle = panel.statusStyle.border
  ctx.stroke()
  ctx.fillStyle = ink.foreground
  ctx.font = font
  setTracking(ctx, STATUS_TRACKING)
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText(text, view.width / 2, top + h / 2)
  setTracking(ctx, 0)
  ctx.restore()
  return top + h
}

/** paintLabel écrit le nom de l'équipe, en texte libre sur le voile (jamais dans la carte). */
function paintLabel(
  ctx: CanvasRenderingContext2D,
  view: OverlayView,
  label: string,
  fonts: OverlayFonts,
  ink: OverlayInk,
  top: number,
): number {
  ctx.save()
  ctx.fillStyle = ink.foreground
  ctx.font = labelFont(fonts)
  setTracking(ctx, LABEL_TRACKING)
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.fillText(upper(label), view.width / 2, top + ROW_GAP)
  setTracking(ctx, 0)
  ctx.restore()
  return top + ROW_GAP + LABEL_SIZE
}

/**
 * paintScore écrit `ally - enemy`, le tiret dans l'encre atténuée comme dans le DOM.
 *
 * L'ORDRE EST CELUI DU BANDEAU (allié à gauche) et il ne se discute pas : sur un export, le
 * bandeau n'est pas là pour arbitrer, mais l'habitude prise sur la page, si.
 */
function paintScore(
  ctx: CanvasRenderingContext2D,
  view: OverlayView,
  score: OverlayScore,
  fonts: OverlayFonts,
  ink: OverlayInk,
  top: number,
): void {
  const { ally, enemy, sep } = scoreParts(score)
  const font = scoreFont(fonts)
  const wAlly = textWidth(ctx, ally, font, 0)
  const wSep = textWidth(ctx, sep, font, 0)
  const wEnemy = textWidth(ctx, enemy, font, 0)
  const total = wAlly + SCORE_SEP_PAD * 2 + wSep + wEnemy
  let x = (view.width - total) / 2
  const y = top + ROW_GAP
  ctx.save()
  ctx.font = font
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  ctx.fillStyle = ink.foreground
  ctx.fillText(ally, x, y)
  x += wAlly + SCORE_SEP_PAD
  ctx.fillStyle = ink.muted
  ctx.fillText(sep, x, y)
  x += wSep + SCORE_SEP_PAD
  ctx.fillStyle = ink.foreground
  ctx.fillText(enemy, x, y)
  ctx.restore()
}

/**
 * paintOverlayPanel — LE SEUL POINT D'ENTRÉE. Peint le voile (si demandé), le filigrane, la
 * carte du statut, le nom et le score, dans l'ordre du DOM : du fond vers le texte.
 *
 * Ne rend rien : l'appelant sait déjà s'il y avait un panneau à peindre (c'est lui qui l'a
 * construit). Un panneau sans statut ne se peint pas — une carte vide au milieu de l'écran
 * serait pire que pas de carte.
 */
export function paintOverlayPanel(
  ctx: CanvasRenderingContext2D,
  view: OverlayView,
  panel: OverlayPanel,
  fonts: OverlayFonts,
  ink: OverlayInk,
): void {
  if (!panel.status.trim()) return
  if (panel.veil) paintVeil(ctx, view, ink)
  if (panel.logo) paintLogo(ctx, view, panel.logo)
  let y = (view.height - panelHeight(panel)) / 2
  y = paintStatusBlock(ctx, view, panel, fonts, ink, y)
  if (panel.label) y = paintLabel(ctx, view, panel.label, fonts, ink, y)
  if (panel.score) paintScore(ctx, view, panel.score, fonts, ink, y)
}

/**
 * neutralStatusStyle — le bloc NEUTRE (`OVERLAY_STATUS_NEUTRAL` : `border-2 border-border
 * bg-card`), employé par l'égalité et par le message inter-manche. Les deux le partagent dans
 * le DOM ; ils le partagent ici aussi, et par la même fonction.
 */
export function neutralStatusStyle(ink: OverlayInk): OverlayStatusStyle {
  return { background: ink.card, border: ink.border }
}
