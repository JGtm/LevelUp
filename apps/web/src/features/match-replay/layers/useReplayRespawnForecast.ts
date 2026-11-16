/**
 * useReplayRespawnForecast — LA PROBABILITÉ DE RÉAPPARITION, câblée au canvas : le chargement (seulement
 * quand le calque est actif), la jointure des zones de la réponse avec leurs formes (`/replay/callouts`,
 * même `volume_index`), les textes dans la langue de la page et les DEUX peintres que la scène ordonne
 * (la teinte sous les pions, les repères et la légende au-dessus).
 *
 * UN REFUS EST UN CALQUE ABSENT, pas une erreur ni une action impossible : réponse `unavailable`,
 * échec de chargement, carte sans zone nommée, aucune mort exploitable — le calque ne se peint pas et
 * son interrupteur n'est pas proposé. Tant que la réponse n'est pas connue, l'interrupteur n'est offert
 * que si le calque est ÉTEINT (sans lui, un lecteur qui l'a éteint ne pourrait jamais le rallumer) ; allumé
 * et en cours de chargement, il attend.
 *
 * SEUL / MASQUER : la probabilité APPARTIENT au joueur mort. `shownAuthor` (`focusFilter.author`) la retire
 * avec ses autres traces, et le sans-joueur n'existe pas ici : chaque mort porte son xuid.
 *
 * LE ROGNAGE AU PLAN est un paramètre (`clipMask`) : voir `respawnForecastZones.ts`.
 */
import { useMemo, useRef } from 'react'

import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import { intlLocale } from '@/lib/formatters/intlLocale'
import { durationFrameMs } from '../../../lib/replay/replayLogic'
import { useReplaySpawnForecast } from '../../../lib/replay/queries'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { CanvasView } from '../model/replayView'
import {
  momentsAt,
  normalizeSpawnForecast,
  primaryMoment,
  type ForecastMoment,
} from '../model/respawnForecast.logic'
import type { NamedLayerPainter } from './replayCompose'
import { drawForecastLegend } from './respawnForecastLegend'
import { drawForecastMarks } from './respawnForecastMarks'
import type { ForecastInk, ForecastPaint, ForecastWording, ForecastZone } from './respawnForecastPaint'
import { createForecastScratch, drawForecastZones, type ZoneClip, type ZonesPaint } from './respawnForecastZones'
import type { SlotIdentity } from './useSlotIdentity'

/** Le nom d'un joueur que le film ne nomme pas. */
const UNKNOWN_NAME = '—'
const NO_MOMENTS: readonly ForecastMoment[] = []

export interface RespawnForecastOptions {
  /** Le pas du document : la durée d'une image. */
  doc: Parameters<typeof durationFrameMs>[0]
  view: CanvasView
  /** L'interrupteur du calque : faux = rien n'est chargé ni peint. */
  enabled: boolean
  /** Le match dont on lit la probabilité ; absents = pas de calque (page sans match). */
  playerSlug?: string
  matchId?: string
  locale: ReplayLocale
  /** Les zones nommées de la carte (formes) : sans elles, la probabilité n'a rien à teinter. */
  callouts: readonly CalloutZoneReady[]
  identity: Pick<SlotIdentity, 'colorOfXuid' | 'nameOfXuid'>
  /** Voile et texte sur voile (jetons `scrim` / `on-scrim`), liseré sombre, repli neutre. */
  ink: { scrim: string; onScrim: string; labelStroke: string; neutral: string }
  /** Seul et Masquer : faux = la probabilité de ce joueur est retirée de la carte. Absent = rien n'est filtré. */
  shownAuthor?: (xuid: string) => boolean
  /** Retire de la teinte ce qui sort du plan de la carte ; absent = zones non rognées. */
  clipMask?: ZoneClip['mask']
}

export interface ReplayRespawnForecast {
  /** Les deux peintres, à passer à `bindPainters`. */
  layers: readonly [NamedLayerPainter<'reapparition-zones'>, NamedLayerPainter<'reapparition-reperes'>]
  /** L'interrupteur du calque. */
  show: boolean
  /** La réponse est prête et la carte a de quoi teinter : le calque se peint. */
  ready: boolean
  /** L'interrupteur se propose au panneau de la vue (cf. l'en-tête). */
  available: boolean
}

/** Les textes de la page, résolus : une probabilité s'écrit selon la langue (`35 %`, `35%`). */
export function forecastWording(locale: ReplayLocale): ForecastWording {
  const t = REPLAY_TEXT[locale].respawnForecast
  const percent = new Intl.NumberFormat(intlLocale(locale), { style: 'percent', maximumFractionDigits: 0 })
  return {
    title: t.title,
    chip: t.chipFmt,
    rank: t.rankFmt,
    share: (p) => (p > 0 && p < 0.005 ? t.lessThanOnePercent : percent.format(p)),
    legendCountdown: t.legendCountdownFmt,
    appearedIn: t.appearedInFmt,
    appearedOutside: t.appearedOutsideFmt,
    appearedRank: t.appearedRankFmt,
    appearedUnlikely: t.appearedUnlikely,
    legendScale: t.legendScale,
    legendFooter: t.legendFooter,
  }
}

export function useReplayRespawnForecast(o: RespawnForecastOptions): ReplayRespawnForecast {
  const { doc, view, enabled, playerSlug = '', matchId = '', locale, callouts, identity, ink, shownAuthor, clipMask } = o
  const hasZones = callouts.length > 0
  const query = useReplaySpawnForecast(playerSlug, matchId, enabled && hasZones, locale)
  const forecast = useMemo(() => normalizeSpawnForecast(query.data), [query.data])
  const zones = useMemo(() => {
    const byIndex = new Map(callouts.map((c) => [c.volumeIndex, c]))
    const out = new Map<number, ForecastZone>()
    for (const z of forecast?.zones ?? []) {
      const shape = byIndex.get(z.volumeIndex)
      if (shape) out.set(z.volumeIndex, { name: z.name, shape })
    }
    return out
  }, [forecast, callouts])
  const frameMs = durationFrameMs(doc)
  const { colorOfXuid, nameOfXuid } = identity
  const { scrim, onScrim, labelStroke, neutral } = ink
  const wording = useMemo(() => forecastWording(locale), [locale])
  const clip = useMemo<ZoneClip | undefined>(
    () => (clipMask ? { mask: clipMask, scratch: createForecastScratch() } : undefined),
    [clipMask],
  )
  const inputs = useMemo(() => {
    const paintInk: ForecastInk = { panel: scrim, onPanel: onScrim, underlay: labelStroke, neutral }
    return {
      ink: paintInk,
      wording,
      zones,
      colorOf: (xuid: string) => colorOfXuid(xuid) ?? neutral,
      nameOf: (xuid: string) => nameOfXuid(xuid) ?? UNKNOWN_NAME,
    } satisfies Omit<ForecastPaint, 'k'>
  }, [scrim, onScrim, labelStroke, neutral, wording, zones, colorOfXuid, nameOfXuid])

  const ready = forecast !== null && zones.size > 0
  // Les deux peintres lisent les mêmes morts à la même image : une lecture par image, pas deux. La mémoire
  // porte ce qui la rend valide (réponse, filtre, pas du document) : un changement de l'un l'invalide.
  const memo = useRef<{ forecast: unknown; shown: unknown; frameMs: number; frame: number; moments: readonly ForecastMoment[] } | null>(null)
  const layers = useMemo(() => {
    const momentsOf = (frame: number): readonly ForecastMoment[] => {
      if (!forecast) return NO_MOMENTS
      const m = memo.current
      if (m && m.forecast === forecast && m.shown === shownAuthor && m.frameMs === frameMs && m.frame === frame) return m.moments
      const moments = momentsAt(forecast, frame, frameMs, shownAuthor)
      memo.current = { forecast, shown: shownAuthor, frameMs, frame, moments }
      return moments
    }
    return [
      {
        id: 'reapparition-zones',
        paint: (ctx, frame, k) => {
          const p: ZonesPaint = { ...inputs, k, clip }
          drawForecastZones(ctx, momentsOf(frame), view, p)
        },
      },
      {
        id: 'reapparition-reperes',
        paint: (ctx, frame, k) => {
          const moments = momentsOf(frame)
          drawForecastMarks(ctx, moments, view, { ...inputs, k })
          drawForecastLegend(ctx, primaryMoment(moments), view, { ...inputs, k })
        },
      },
    ] as const satisfies ReplayRespawnForecast['layers']
  }, [forecast, frameMs, shownAuthor, inputs, clip, view])

  const known = query.data !== undefined || query.isError
  const available = !!playerSlug && !!matchId && hasZones && (known ? ready : !enabled)
  return useMemo(() => ({ layers, show: enabled, ready, available }), [layers, enabled, ready, available])
}
