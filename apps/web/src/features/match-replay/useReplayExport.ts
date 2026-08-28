/**
 * useReplayExport — LA BOUCLE QUI RECALCULE LE FILM, et la couture React autour d'elle.
 *
 * # LA DIFFÉRENCE AVEC L'ENREGISTREMENT, EN UNE PHRASE
 *
 * `useReplayCapture` FILME l'écran : il subit le temps réel. Ce hook RECALCULE le film — il
 * pose une image, la peint, l'encode, et passe à la suivante aussi vite que la machine suit.
 * Un match de dix minutes sort en une minute au lieu de dix, et la durée du fichier ne dépend
 * jamais du temps de calcul (les horodatages sont posés, cf. `replayVideoEncoder.ts`).
 *
 * Ce qui rend cela possible est une propriété du canvas, pas une astuce : `draw()` est une
 * FONCTION PURE de `frameRef.current`. Poser l'image puis appeler `draw()` peint exactement
 * cette image, autant de fois qu'on veut, dans n'importe quel ordre.
 *
 * # TROIS PIÈGES, TOUS MESURÉS PLUTÔT QUE SUPPOSÉS
 *
 * 1. NE JAMAIS RENDRE LA MAIN PAR UN MINUTEUR. `setTimeout` et `requestAnimationFrame` sont
 *    bridés en onglet caché — 673 ms de latence moyenne pour le premier, jamais déclenché pour
 *    le second — et un export dure des minutes pendant lesquelles l'utilisateur va ailleurs.
 *    C'est `yieldToEvents()` (cf. `eventLoopYield.ts`), et rien d'autre.
 * 2. LES COULEURS DOIVENT ÊTRE RÉSOLUES AVANT D'ATTEINDRE LA TOILE. Un `fillStyle` qui reçoit
 *    `var(--primary)` est SILENCIEUSEMENT IGNORÉ : le contexte garde la couleur précédente,
 *    sans erreur, et le panneau sort dans une teinte au hasard. Vérifié dans le navigateur le
 *    2026-08-28. D'où `resolveToken` / `readInk` ici, et un peintre qui ne lit rien.
 * 3. LA LECTURE DOIT S'ARRÊTER. La boucle d'animation écrit `frameRef` elle aussi ; les deux
 *    en même temps se disputeraient le curseur et l'export peindrait n'importe quoi.
 *
 * # L'ÉCRAN N'EST PAS ÉPARGNÉ, ET C'EST VOULU
 *
 * La boucle peint dans la TOILE VISIBLE. L'utilisateur voit donc le match défiler très vite
 * pendant l'export — ce n'est pas un défaut mais la seule preuve visible que quelque chose se
 * passe, et cela évite une seconde toile de la taille de la première. À la fin, quoi qu'il
 * arrive, l'image d'avant l'export est reposée et repeinte.
 */
import { useCallback, useRef, useState, type RefObject } from 'react'

import { teamTintStyles } from '@/features/match-view/teamColor'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { teamLogoPath } from '@/lib/halo/teamNames'
import type { MatchScoreboardRow } from '@/lib/api/types'
import type { ReplayScoreDocument } from '@/lib/replay/scoreTimeline'

import { readInk } from './canvasInk'
import { yieldToEvents } from './eventLoopYield'
import {
  buildOverlayPanelSource,
  type ExportOutcome,
  type OverlayPanelSource,
} from './exportOverlayPanels'
import { paintOverlayPanel, type OverlayFonts, type OverlayInk } from './overlayPaint'
import { triggerDownload } from './replayCapture'
import { tintedIconCanvas } from './replayDraw'
import {
  buildExportPlan,
  defaultExportBounds,
  exportProgressPct,
  type ExportBounds,
} from './replayExportPlan'
import type { ReplayDocumentReady } from './replayNormalize'
import { canExportVideo, openVideoExport } from './replayVideoEncoder'
import type { ReplayWindowBounds } from './replayWindow'
import type { ReplayLocale } from './i18n'
import { readVictory } from './victoryLogic'

/** Cadence de publication de la progression, en images encodées. */
const PROGRESS_EVERY = 15

export interface ReplayExportOptions {
  canvasRef: RefObject<HTMLCanvasElement | null>
  /** L'image courante, PARTAGÉE avec le dessin : la boucle l'écrit, `draw()` la lit. */
  frameRef: RefObject<number>
  /** Le tracé d'une image, tel que le canvas le publie. */
  redraw: () => void
  /** Met la lecture en pause. Appelé AVANT la boucle (cf. piège 3 de l'en-tête). */
  pause: () => void
  doc: ReplayDocumentReady & ReplayScoreDocument
  playWindow: ReplayWindowBounds | null
  scoreboard: readonly MatchScoreboardRow[]
  xuidMeta?: XuidMeta
  /** Le verdict du backend. `null` = pas d'écran de fin, exactement comme dans le DOM. */
  outcome: ExportOutcome | null
  titleSlug: string
  locale: ReplayLocale
  /** Nomme le fichier sur l'instant de match courant (partagé avec la capture). */
  filenameFor: (ext: string) => string
}

/** L'état que le dialogue affiche. `total` vaut 0 tant qu'aucun export n'a démarré. */
export interface ReplayExportState {
  running: boolean
  done: number
  total: number
  pct: number
}

export interface ReplayExport {
  /** `false` = pas de WebCodecs : c'est l'enregistrement temps réel qui reprend la main. */
  supported: boolean
  state: ReplayExportState
  /** La plage proposée par défaut : la fenêtre de gameplay entière. */
  defaultBounds: () => ExportBounds
  run: (bounds: ExportBounds) => Promise<void>
  cancel: () => void
}

const IDLE: ReplayExportState = { running: false, done: 0, total: 0, pct: 0 }

/** Les encres du thème, résolues UNE fois par export (cf. piège 2 de l'en-tête). */
function readOverlayInk(): OverlayInk {
  return {
    background: readInk('--background'),
    foreground: readInk('--foreground'),
    muted: readInk('--muted-foreground'),
    border: readInk('--border'),
    card: readInk('--card'),
  }
}

/** Les familles de police de la page. Le repli n'est atteint que si le thème perd ses vars. */
function readOverlayFonts(): OverlayFonts {
  const root = getComputedStyle(document.documentElement)
  const sans = root.getPropertyValue('--sans').trim()
  const mono = root.getPropertyValue('--mono').trim()
  return { sans: sans || 'system-ui, sans-serif', mono: mono || 'ui-monospace, monospace' }
}

/** Charge le filigrane et le TEINTE. `null` partout ailleurs : le panneau reste entier. */
async function loadTeamLogo(
  titleSlug: string,
  teamID: number | null | undefined,
  color: string,
): Promise<HTMLCanvasElement | null> {
  const src = teamLogoPath(titleSlug, teamID)
  if (!src) return null
  const img = await new Promise<HTMLImageElement | null>((resolve) => {
    const el = new Image()
    el.onload = () => resolve(el)
    // Un `team_id` sans asset publié répond 404 : silence, pas de filigrane, pas d'erreur.
    el.onerror = () => resolve(null)
    el.src = src
  })
  return img ? tintedIconCanvas(img, color) : null
}

/** Prépare tout ce qui ne change pas d'une image à l'autre. Hors React : aucun état ici. */
async function buildSource(o: ReplayExportOptions): Promise<OverlayPanelSource> {
  const ink = readOverlayInk()
  // LA COULEUR EST CELLE QUE L'UTILISATEUR A RÉGLÉE (décision D1 du DOM) : l'écran de fin est
  // TOUJOURS celui de son camp, donc toujours `team-ally`, surchargeable par l'accessibilité.
  const teamColor = resolveToken('team-ally')
  const victory = readVictory(o.scoreboard, o.outcome?.code)
  const logo = victory?.mine
    ? await loadTeamLogo(o.titleSlug, victory.mine.teamID, teamColor)
    : null
  const tint = teamTintStyles(teamColor)
  return buildOverlayPanelSource({
    doc: o.doc,
    scoreboard: o.scoreboard,
    xuidMeta: o.xuidMeta,
    playWindow: o.playWindow,
    outcome: o.outcome,
    locale: o.locale,
    ink,
    teamStyle: { background: tint.background, border: tint.border },
    logo,
  })
}

/** Peint UNE image de l'export : le terrain, puis la surimpression s'il y en a une. */
function paintExportFrame(
  canvas: HTMLCanvasElement,
  o: ReplayExportOptions,
  source: OverlayPanelSource,
  ink: OverlayInk,
  fonts: OverlayFonts,
  frame: number,
): void {
  o.frameRef.current = frame
  o.redraw()
  const panel = source.panelAt(frame)
  if (!panel) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  // LA MISE À L'ÉCHELLE SE REPOSE ICI, elle ne se suppose pas : `draw()` la pose au début de
  // son tracé, mais rien ne garantit dans quel état ses calques la laissent.
  const dpr = window.devicePixelRatio || 1
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  paintOverlayPanel(ctx, { width: canvas.width / dpr, height: canvas.height / dpr }, panel, fonts, ink)
}

export function useReplayExport(o: ReplayExportOptions): ReplayExport {
  const [state, setState] = useState<ReplayExportState>(IDLE)
  const cancelRef = useRef(false)
  const runningRef = useRef(false)

  const cancel = useCallback(() => {
    cancelRef.current = true
  }, [])

  const defaultBounds = useCallback(
    () => defaultExportBounds(o.doc, o.playWindow),
    [o.doc, o.playWindow],
  )

  const run = useCallback(
    async (bounds: ExportBounds) => {
      const canvas = o.canvasRef.current
      if (!canvas || runningRef.current || !canExportVideo()) return
      runningRef.current = true
      cancelRef.current = false
      o.pause()
      // L'IMAGE D'AVANT L'EXPORT est mise de côté MAINTENANT : la boucle va écrire `frameRef`
      // des milliers de fois, et l'utilisateur doit retrouver son rejeu où il l'avait laissé.
      const resume = o.frameRef.current
      const plan = buildExportPlan(bounds, o.doc)
      setState({ running: true, done: 0, total: plan.frames.length, pct: 0 })
      try {
        // Les polices AVANT la première image : sans cette attente, le début du clip écrirait
        // les surimpressions dans une police de repli, et le reste dans la bonne.
        await document.fonts.ready
        const source = await buildSource(o)
        const ink = readOverlayInk()
        const fonts = readOverlayFonts()
        const sink = await openVideoExport({ width: canvas.width, height: canvas.height })
        if (!sink) return
        const ok = await encodeAll(canvas, o, source, ink, fonts, plan.frames, {
          cancelled: () => cancelRef.current,
          progress: (done) =>
            setState({ running: true, done, total: plan.frames.length, pct: exportProgressPct(done, plan.frames.length) }),
          sink,
        })
        if (ok) triggerDownload(await sink.finish(), o.filenameFor('mp4'))
        else sink.abort()
      } finally {
        runningRef.current = false
        setState(IDLE)
        o.frameRef.current = resume
        o.redraw()
      }
    },
    [o],
  )

  return { supported: canExportVideo(), state, defaultBounds, run, cancel }
}

/** Ce dont la boucle a besoin en plus des images : où pousser, et quand s'arrêter. */
interface EncodeHooks {
  cancelled: () => boolean
  progress: (done: number) => void
  sink: { addFrame: (c: HTMLCanvasElement, i: number) => Promise<void> }
}

/**
 * encodeAll parcourt le plan. HORS du hook (règle des 80 lignes du dépôt) et hors React : elle
 * ne touche à aucun état, elle publie par rappel.
 *
 * Rend `false` si l'utilisateur a annulé — c'est ce qui décide, chez l'appelant, entre remettre
 * un fichier et n'en remettre aucun.
 */
async function encodeAll(
  canvas: HTMLCanvasElement,
  o: ReplayExportOptions,
  source: OverlayPanelSource,
  ink: OverlayInk,
  fonts: OverlayFonts,
  frames: readonly number[],
  hooks: EncodeHooks,
): Promise<boolean> {
  for (let i = 0; i < frames.length; i++) {
    if (hooks.cancelled()) return false
    paintExportFrame(canvas, o, source, ink, fonts, frames[i])
    await hooks.sink.addFrame(canvas, i)
    if (i % PROGRESS_EVERY !== 0) continue
    hooks.progress(i)
    // LA MAIN REVIENT À LA PAGE ICI, et par le canal non bridé : sans cela la barre de
    // progression ne se peindrait jamais, et « Annuler » ne serait jamais lu.
    await yieldToEvents()
  }
  hooks.progress(frames.length)
  return true
}
