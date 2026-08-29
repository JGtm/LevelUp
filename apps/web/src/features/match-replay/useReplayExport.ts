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
import { useCallback, useEffect, useRef, useState, type RefObject } from 'react'

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
import { mixReplayAudio, soundUrlOf, type MixedTracks, type SoundFamily } from './replayAudioMix'
import { formatClock, frameToMs } from './replayLogic'
import { buildCaptureFilename, triggerDownload } from './replayCapture'
import { tintedIconCanvas } from './replayDraw'
import {
  END_HOLD_MS,
  buildExportPlan,
  defaultExportBounds,
  exportProgressPct,
  type ExportBounds,
  type ExportPlan,
} from './replayExportPlan'
import type { ReplayDocumentReady } from './replayNormalize'
import { EXPORT_FPS, canExportVideo, openVideoExport, type VideoExportSink } from './replayVideoEncoder'
import { displayClockMs, type ReplayWindowBounds } from './replayWindow'
import { EXPORT_SUPERSAMPLE, exportRenderScale } from './useReplayView'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplaySoundEvent } from './replaySoundVariants'
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
  /**
   * LA PISTE SONORE DU REJEU, telle que la page la joue (`useReplaySound.exportTrack`). Absente
   * = clip muet. Elle n'est PAS reconstruite par l'export : c'est ce qui garantit qu'un clip
   * n'invente aucun son et n'en manque aucun.
   */
  soundTrack?: () => {
    timeline: readonly ReplaySoundEvent[]
    endMatchStems: readonly string[]
    variationPercent: number
    distancePercent: number
  }
  /** Le volume réglé dans la page. Le mixage le suit, même haut-parleurs coupés (décision D6). */
  soundVolume?: number
}

/** Ce que l'appelant choisit au moment de lancer. Le son est INCLUS par défaut (décision D6). */
export interface ExportRunOptions {
  sound?: boolean
}

/** L'état que le dialogue affiche. `total` vaut 0 tant qu'aucun export n'a démarré. */
/**
 * LES CINQ ETATS D'UN EXPORT, et pourquoi ils ne se resument pas a « en cours / pas en cours ».
 *
 * `prepare` est celui qui manquait et qui coutait le plus cher a l'utilisateur : entre le clic
 * et la premiere image encodee, il faut attendre les polices, charger le logo, DECODER tous les
 * fichiers sons, RENDRE le mixage hors ligne et ENCODER la piste AAC. Sur un match charge en
 * sons, cela fait plusieurs secondes pendant lesquelles la barre affichait « Image 0 / 18000 »
 * sans bouger — c'est-a-dire exactement l'instant ou l'on se demande si ca a plante.
 *
 * `done` manquait aussi : le dialogue redevenait le formulaire et le fichier tombait dans les
 * telechargements sans qu'un mot ne le dise.
 */
export type ExportPhase = 'idle' | 'prepare' | 'encode' | 'done' | 'failed'

export interface ReplayExportState {
  phase: ExportPhase
  done: number
  total: number
  pct: number
  /**
   * Temps restant ESTIME, en millisecondes, ou `undefined` tant qu'aucune image n'est encodee.
   * Fiable ici parce que le cout par image est tres regulier : meme toile, meme encodeur, et
   * un dessin dont la charge ne depend pas de l'instant du match.
   */
  etaMs?: number
  /** Le nom du fichier depose, en phase `done`. */
  filename?: string
  /**
   * `true` = le son etait demande mais le navigateur a REFUSE la piste : le clip est muet, et
   * il faut le dire. Un clip silencieux qu'on croyait sonore se decouvre au montage.
   */
  mutedFallback?: boolean
  /** Le message technique de l'echec, pour la console et pour l'utilisateur averti. */
  message?: string
}

export interface ReplayExport {
  /** `false` = pas de WebCodecs : c'est l'enregistrement temps réel qui reprend la main. */
  supported: boolean
  state: ReplayExportState
  /** La plage proposée par défaut : la fenêtre de gameplay entière. */
  defaultBounds: () => ExportBounds
  run: (bounds: ExportBounds, options?: ExportRunOptions) => Promise<void>
  cancel: () => void
  /**
   * L'HORLOGE DE MATCH d'une image, prête à afficher. Portée par l'export et non par le
   * dialogue : le recalage sur la fenêtre de gameplay demande le document ET la fenêtre, que
   * le dialogue n'a aucune raison de recevoir — et qui coûteraient deux props au canvas, déjà
   * à son plafond de taille.
   */
  clockOf: (frame: number) => string
  /** La durée du CLIP pour ces bornes, formatée. Même raison, même endroit. */
  lengthClock: (bounds: ExportBounds) => string
}

const IDLE: ReplayExportState = { phase: 'idle', done: 0, total: 0, pct: 0 }

/** Nombre d'images encodees avant de risquer une estimation : sous ce seuil, elle danserait. */
const ETA_MIN_FRAMES = 30

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
async function buildSource(o: ReplayExportOptions, ink: OverlayInk): Promise<OverlayPanelSource> {
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
  paint: OverlayPaintContext,
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
  // MEME ECHELLE QUE LE TRACE, surechantillonnage compris : une surimpression posee a
  // l'echelle de l'ecran serait deux fois trop petite dans un export sureechantillonne.
  const dpr = (window.devicePixelRatio || 1) * exportRenderScale.current
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  paintOverlayPanel(ctx, { width: canvas.width / dpr, height: canvas.height / dpr }, panel, paint.fonts, paint.ink)
}

export function useReplayExport(o: ReplayExportOptions): ReplayExport {
  const [state, setState] = useState<ReplayExportState>(IDLE)
  const cancelRef = useRef(false)
  const runningRef = useRef(false)
  // LE DÉMONTAGE ANNULE, ET NE TÉLÉCHARGE RIEN. Même invariant que l'enregistrement temps réel
  // juste à côté (`useReplayCapture`, `liveRef`) : quitter la page d'un match pendant un export
  // ne doit pas déposer, trois minutes plus tard, le fichier d'un match qu'on a quitté.
  const liveRef = useRef(true)
  useEffect(() => {
    liveRef.current = true
    return () => {
      liveRef.current = false
      cancelRef.current = true
    }
  }, [])

  const cancel = useCallback(() => {
    cancelRef.current = true
  }, [])

  const defaultBounds = useCallback(
    () => defaultExportBounds(o.doc, o.playWindow),
    [o.doc, o.playWindow],
  )

  const run = useCallback(
    async (bounds: ExportBounds, options?: ExportRunOptions) => {
      const canvas = o.canvasRef.current
      // `runningRef` est posé SYNCHRONEMENT, avant le moindre `await` : c'est ce qui rend le
      // double-clic sur « Exporter » inoffensif.
      if (!canvas || runningRef.current || !canExportVideo()) return
      runningRef.current = true
      cancelRef.current = false
      o.pause()
      // L'IMAGE D'AVANT L'EXPORT est mise de côté MAINTENANT : la boucle va écrire `frameRef`
      // des milliers de fois, et l'utilisateur doit retrouver son rejeu où il l'avait laissé.
      const resume = o.frameRef.current
      // Un premier plan SANS maintien : il donne la duree de la plage, dont le mixage a besoin.
      let plan = buildExportPlan(bounds, o.doc)
      // PHASE 1 : la preparation. La barre n'affiche pas « image 0 » — elle dit ce qu'on fait.
      setState({ phase: 'prepare', done: 0, total: plan.frames.length, pct: 0 })
      let sink: VideoExportSink | null = null
      try {
        // Les polices AVANT la première image : sans cette attente, le début du clip écrirait
        // les surimpressions dans une police de repli, et le reste dans la bonne.
        await document.fonts.ready
        const ink = readOverlayInk()
        const fonts = readOverlayFonts()
        const source = await buildSource(o, ink)
        // LE SON SE MIXE AVANT D'OUVRIR LE CONTENEUR : le MP4 annonce ses pistes une fois pour
        // toutes, donc il faut savoir AVANT s'il y en aura une. Le mixage hors ligne est rapide
        // (il ne joue rien, il calcule), et il rend `null` s'il n'y a rien a mixer.
        // LA TOILE EST RENDUE PLUS GRANDE LE TEMPS DE L'EXPORT (cf. `exportRenderScale`) : la
        // perte de nettete d'un clip vient du sous-echantillonnage chroma de H.264, qu'aucun
        // debit ne rachete. Le redessin suivant applique la nouvelle taille au backing store,
        // et c'est ELLE que l'encodeur doit recevoir — d'ou l'ordre : echelle, trace, ouverture.
        exportRenderScale.current = EXPORT_SUPERSAMPLE
        o.redraw()
        const mix = options?.sound === false ? null : await mixExportAudio(o, bounds, plan)
        // LE MAINTIEN SE CALCULE UNE FOIS LE SON CONNU (cf. `holdMsFor`), et le plan se refait :
        // la derniere image doit tenir assez longtemps pour qu'on lise le verdict, et au moins
        // aussi longtemps que le son continue.
        plan = buildExportPlan(bounds, o.doc, EXPORT_FPS, holdMsFor(o, bounds, plan, mix))
        sink = await openVideoExport({
          width: canvas.width,
          height: canvas.height,
          audioTracks: mix ? trackNames(mix, o.locale) : undefined,
        })
        if (!sink) {
          // Configuration refusée par le navigateur : ce n'est pas une panne, c'est une
          // capacité absente — mais l'utilisateur doit le voir, pas regarder une barre
          // disparaître sans un mot.
          fail(setState, 'configuration video refusee par le navigateur')
          return
        }
        // LE SON DEMANDE MAIS IMPOSSIBLE NE SE TAIT PAS : le navigateur peut avoir l'API et
        // REFUSER la configuration AAC. Le clip sort muet — mieux qu'un export perdu — et la
        // phase finale le dit.
        const muet = mix !== null && !sink.audioEnabled
        if (muet) console.warn('[replay-export] piste sonore refusee par le navigateur, clip muet')
        if (mix && sink.audioEnabled) await sink.addAudioTracks(exportTracks(mix, o.locale))
        // PHASE 2 : l'encodage. C'est seulement ici que le compte d'images veut dire quelque
        // chose, et que le temps restant peut s'estimer.
        const total = plan.frames.length
        const debut = performance.now()
        setState({ phase: 'encode', done: 0, total, pct: 0 })
        const ok = await encodeAll(canvas, o, source, { ink, fonts }, plan.frames, {
          cancelled: () => cancelRef.current,
          progress: (done) =>
            setState({
              phase: 'encode',
              done,
              total,
              pct: exportProgressPct(done, total),
              etaMs: etaFor(done, total, performance.now() - debut),
            }),
          sink,
        })
        const clip = ok ? await sink.finish() : null
        // `sink` n'est lâché QUE si `finish()` l'a refermé. Sur annulation il reste ici, et
        // c'est le `finally` qui l'abandonne — sans quoi l'encodeur et son muxeur restaient
        // ouverts après un clic sur « Annuler ».
        if (ok) sink = null
        // LE DÉMONTAGE NE TÉLÉCHARGE PAS : le clip est assemblé (l'encodeur doit être vidé
        // proprement quoi qu'il arrive) mais il ne part pas vers un onglet qui a changé de page.
        const filename = exportFilename(o, bounds)
        if (clip && liveRef.current) triggerDownload(clip, filename)
        // PHASE 3 : le fichier est depose, et on le DIT avec son nom. Une annulation, elle,
        // ramene au formulaire — il n'y a rien a annoncer d'un geste qu'on vient de faire.
        setState(clip ? { phase: 'done', done: total, total, pct: 100, filename, mutedFallback: muet } : IDLE)
      } catch (err) {
        // JAMAIS D'ÉCHEC MUET (règle 3 du dépôt : logger AVANT toute dégradation). Sans ce
        // bloc, une panne d'encodeur faisait disparaître la barre de progression sans un mot,
        // sans fichier et sans trace — et laissait l'encodeur ouvert jusqu'à la fermeture de
        // l'onglet.
        console.error('[replay-export] export interrompu', err)
        fail(setState, err instanceof Error ? err.message : String(err))
      } finally {
        // L'ECHELLE REDESCEND AVANT LE DERNIER TRACE, sur TOUS les chemins : la laisser levee
        // rendrait la page entiere en double resolution jusqu'au prochain remontage.
        exportRenderScale.current = 1
        // L'ENCODEUR SE REFERME SUR TOUS LES CHEMINS : `sink` n'est remis à `null` qu'une fois
        // le fichier assemblé. S'il est encore là, c'est qu'on sort par une erreur ou une
        // annulation, et il retient un muxeur entier en mémoire.
        sink?.abort()
        runningRef.current = false
        o.frameRef.current = resume
        o.redraw()
      }
    },
    [o],
  )

  const clockOf = useCallback(
    (frame: number) => formatClock(displayClockMs(frameToMs(frame, o.doc), o.playWindow)),
    [o.doc, o.playWindow],
  )
  const lengthClock = useCallback(
    (b: ExportBounds) => formatClock(frameToMs(b.endFrame, o.doc) - frameToMs(b.startFrame, o.doc)),
    [o.doc],
  )

  return { supported: canExportVideo(), state, defaultBounds, run, cancel, clockOf, lengthClock }
}

/**
 * fail pose l'état d'échec. L'export s'arrête, mais le dialogue RESTE ouvert et le dit — c'est
 * la seule différence qui compte entre « ça n'a pas marché » et « il ne s'est rien passé ».
 */
function fail(setState: (s: ReplayExportState) => void, message: string): void {
  setState({ phase: 'failed', done: 0, total: 0, pct: 0, message })
}

/**
 * etaFor estime le temps restant depuis la cadence DEJA constatee.
 *
 * `undefined` sous `ETA_MIN_FRAMES` : une estimation batie sur trois images danserait d'un
 * facteur dix a chaque rafraichissement, et une estimation qui saute est pire que pas
 * d'estimation.
 */
function etaFor(done: number, total: number, elapsedMs: number): number | undefined {
  if (done < ETA_MIN_FRAMES || elapsedMs <= 0) return undefined
  return Math.max(0, Math.round((elapsedMs / done) * (total - done)))
}

/**
 * Les deux moities de la peinture, REGROUPEES : elles voyagent toujours ensemble, et les
 * separer poussait quatre signatures a six parametres (seuil du depot : cinq).
 */
interface OverlayPaintContext {
  ink: OverlayInk
  fonts: OverlayFonts
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
  paint: OverlayPaintContext,
  frames: readonly number[],
  hooks: EncodeHooks,
): Promise<boolean> {
  for (let i = 0; i < frames.length; i++) {
    if (hooks.cancelled()) return false
    paintExportFrame(canvas, o, source, paint, frames[i])
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

/**
 * exportFilename nomme le clip sur SES DEUX BORNES (décision D9 du plan).
 *
 * `filenameFor` de la capture ne connaît qu'un instant — celui de `frameRef`, qui vaut la borne
 * de FIN au moment où le fichier part. Deux exports de plages différentes qui partagent leur
 * fin porteraient alors le même nom et s'écraseraient dans le dossier de téléchargements.
 */
function exportFilename(o: ReplayExportOptions, bounds: ExportBounds): string {
  return buildCaptureFilename(
    o.doc.matchId,
    frameToMs(bounds.startFrame, o.doc),
    'mp4',
    new Date(),
    frameToMs(bounds.endFrame, o.doc),
  )
}

/**
 * mixExportAudio rend la piste sonore de la plage, ou `null` s'il n'y a rien à mixer.
 *
 * LES BORNES PASSENT EN MILLISECONDES DE FILM, pas en images : la piste sonore est horodatée
 * sur l'horloge du rejeu (celle du fil et des fiches), jamais sur l'axe des images.
 */
async function mixExportAudio(
  o: ReplayExportOptions,
  bounds: ExportBounds,
  plan: ExportPlan,
): Promise<MixedTracks | null> {
  // LA PISTE SE LIT ICI, au lancement : elle porte des reglages d'instance qui vivent dans des
  // refs, et les lire au rendu rendrait des valeurs arbitraires.
  const track = o.soundTrack?.()
  if (!track || track.timeline.length === 0) return null
  const startMs = frameToMs(bounds.startFrame, o.doc)
  return mixReplayAudio(
    track.timeline,
    { startMs, endMs: startMs + plan.durationMs },
    {
      variationPercent: track.variationPercent,
      distancePercent: track.distancePercent,
      // LA CONCLUSION N'EST FOURNIE QUE SI LA PLAGE ATTEINT VRAIMENT LA FIN DU MATCH. Sans
      // cette garde, un extrait des minutes 2 a 4 se terminait sur la voix d'annonceur et la
      // fanfare de victoire : le son affirmait un fait faux sur l'extrait.
      endMatchStems: reachesMatchEnd(o, bounds) ? track.endMatchStems : [],
      volume: o.soundVolume ?? 1,
      urlOf: soundUrlOf,
    },
  )
}

/**
 * holdMsFor — combien de temps la DERNIERE image doit rester a l'ecran, apres la plage.
 *
 * DEUX RAISONS, et on prend la plus longue. (1) L'ecran de fin de match ne se peint qu'A la
 * borne : sans maintien, le clip se terminait sur un eclair de 1/30 s de verdict. (2) Le son
 * deborde souvent la borne — une fanfare de fin dure jusqu'a 11 s — et une piste sonore qui
 * joue apres la derniere image laisse le lecteur sur un ecran mort.
 *
 * Aucune des deux ne s'applique a un extrait muet de milieu de match : il s'arrete net, ce qui
 * est le comportement attendu d'un extrait.
 */
function holdMsFor(
  o: ReplayExportOptions,
  bounds: ExportBounds,
  plan: ExportPlan,
  mix: MixedTracks | null,
): number {
  const verdict = reachesMatchEnd(o, bounds) ? END_HOLD_MS : 0
  const son = mix ? mix.full.duration * 1000 - plan.durationMs : 0
  return Math.max(verdict, son, 0)
}

/**
 * reachesMatchEnd — la plage va-t-elle jusqu'au bout du MATCH ?
 *
 * Sans fenetre de gameplay etablie, on ne sait pas ou le match finit : on repond NON. Poser une
 * fanfare de victoire sur une plage dont on ignore si elle est la fin serait affirmer ce qu'on
 * ne sait pas.
 */
function reachesMatchEnd(o: ReplayExportOptions, bounds: ExportBounds): boolean {
  if (!o.playWindow) return false
  return bounds.endFrame >= o.playWindow.endFrame
}

/**
 * LE NOM DES PISTES, ET LEUR ORDRE. Le MIXAGE COMPLET EN PREMIER — c'est la seule que joue un
 * lecteur ordinaire, et un navigateur n'expose meme pas les autres. Les familles suivent, pour
 * qui ouvre le clip dans un montage.
 */
function exportTracks(mix: MixedTracks, locale: ReplayLocale): { name: string; buffer: AudioBuffer }[] {
  const t = REPLAY_TEXT[locale]
  const nom: Record<SoundFamily, string> = {
    sfx: t.exportTrackSfx,
    voice: t.exportTrackVoice,
    music: t.exportTrackMusic,
  }
  return [
    { name: t.exportTrackMix, buffer: mix.full },
    ...mix.families.map((f) => ({ name: nom[f.family], buffer: f.buffer })),
  ]
}

/** Les seuls NOMS, pour declarer les pistes avant d'avoir quoi que ce soit a y ecrire. */
function trackNames(mix: MixedTracks, locale: ReplayLocale): string[] {
  return exportTracks(mix, locale).map((p) => p.name)
}
