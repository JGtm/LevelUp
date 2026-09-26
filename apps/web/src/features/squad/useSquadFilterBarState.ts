/**
 * useSquadFilterBarState — état TRANSITOIRE de la barre de filtres Escouade.
 *
 * Pourquoi ce hook existe (symptôme utilisateur du 2026-09-20 : « dès que je
 * touche à un filtre, la page a l'air de recharger sans rien changer ») :
 * jusqu'ici cet état vivait dans `SquadLayout`, c'est-à-dire dans le PARENT de
 * `<Outlet />` et du `SquadContext.Provider`. Cocher une case de cascade, ouvrir
 * un popover ou recevoir la réponse du preview re-rendait donc tout l'arbre de
 * la page — jusqu'aux `ChartCard`, qui reconstruisent alors leur option ECharts
 * et REJOUENT leur animation d'entrée, sans qu'aucune donnée n'ait bougé.
 *
 * La règle de découpe est celle des autres pages (FilterOmnibar est un FRÈRE du
 * contenu, jamais son parent) : tout ce qui n'est commité qu'au clic
 * « Analyser » — période, cascade, popover ouvert, preview live — vit ici, donc
 * sous `SquadFilterBar`. Ce qui s'applique en direct (coéquipiers, sessions
 * pickées, composition stricte) reste chez `SquadLayout`, qui en a besoin pour
 * sa requête `useTeammates`, et descend ici par props — les sessions pickées
 * étant lues dans le store escouade, leur source unique (lot perf L4a, D4.1).
 *
 * Ce module ne rend rien : il tient l'état et les dérivés. Le rendu est dans
 * `SquadFilterBar.tsx`.
 */
import { useMemo, useState, type ReactNode } from 'react'

import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useFiltersPreview } from '@/features/filters/queries'
import {
  DEFAULT_CASCADE,
  DEFAULT_SESSIONS,
  computePendingHash,
} from '@/components/shell/FilterOmnibar'
import type {
  CascadeInput,
  FilterContextInput,
  LabelValue,
  PeriodInput,
  PeriodPresetCount,
  SeasonCount,
  SessionLabelEntry,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { SeasonEntry } from '@/lib/i18n/fieldMappings'

import { getSquadText } from './i18n'
import { deriveSquadPending } from './squadPending'
import { buildCompositionGapHint } from './squadCompositionGapHint'
import {
  resolveSquadSessionFallback,
  squadSessionCount,
  squadSessionShownCount,
} from './squadSessionCounts'
import { useActiveSeason } from './useActiveSeason'

/** Popover actuellement ouvert dans la barre (un seul à la fois). */
export type SquadActivePopover = 'filtres' | 'periode' | 'saison' | null

/** Options de cascade nettoyées (forme attendue par FiltresPill). */
export interface SquadCascadeOptions {
  playlists: LabelValue[]
  modes: LabelValue[]
  maps: LabelValue[]
  experience_types: LabelValue[]
}

/** Compte affiché par le rail L2 pour une session : « 4 sur 7 » + explication. */
export interface SquadRailSessionCount {
  shown: number
  total: number
  /** Contenu explicatif de l'écart (liste des matchs écartés), rendu au survol par la L2. */
  hint: ReactNode
}

export interface SquadFilterBarStateInput {
  playerSlug: string
  locale: Locale
  /** Sessions pickées (store escouade, `filterContext.sessions.picked_sessions`). */
  pickedSquadSessionLabels: string[]
  /** Sessions de la composition courante (réponse teammates, source unique ADR 0033). */
  compositionSessions: SessionLabelEntry[]
}

export interface SquadFilterBarState {
  pendingCascade: CascadeInput
  pendingPeriod: PeriodInput | undefined
  setPendingCascade: (c: CascadeInput) => void
  setPendingPeriod: (p: PeriodInput) => void
  activePopover: SquadActivePopover
  togglePopover: (which: Exclude<SquadActivePopover, null>) => void
  closeAllPopovers: () => void
  available: SquadCascadeOptions | undefined
  presetCounts: PeriodPresetCount[] | undefined
  seasons: SeasonEntry[]
  activeSeason: SeasonEntry | null
  seasonCounts: SeasonCount[] | undefined
  cascadeCount: number
  isDirty: boolean
  analyser: () => void
  /** Libellés playlists/modes du filtre EN ATTENTE — indice de tri des compositions. */
  activeContextLabels: string[]
  getSessionCount: (label: string) => SquadRailSessionCount | undefined
  getSessionShownCount: (label: string) => number | undefined
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** Retire les options dont le libellé est resté un UUID brut (référentiel incomplet). */
function cleanCascadeOptions(
  raw:
    | {
        playlists?: LabelValue[] | null
        modes?: LabelValue[] | null
        maps?: LabelValue[] | null
        experience_types?: LabelValue[] | null
      }
    | null
    | undefined,
): SquadCascadeOptions | undefined {
  if (!raw) return undefined
  // Défense : un slice Go nil sérialise en JSON null. `?? []` empêche un
  // crash si le contrat est violé. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
  const filterUUIDs = (opts: LabelValue[] | null | undefined): LabelValue[] =>
    (opts ?? []).filter((o) => !UUID_RE.test(o.label.trim()))
  return {
    playlists: filterUUIDs(raw.playlists),
    modes: filterUUIDs(raw.modes),
    maps: filterUUIDs(raw.maps),
    experience_types: filterUUIDs(raw.experience_types),
  }
}

export function useSquadFilterBarState({
  playerSlug,
  locale,
  pickedSquadSessionLabels,
  compositionSessions,
}: SquadFilterBarStateInput): SquadFilterBarState {
  const filterContext = useSquadFilterStore((s) => s.filterContext)
  const filterContextHash = useSquadFilterStore((s) => s.filterContextHash)
  const resolvedContext = useSquadFilterStore((s) => s.resolvedContext)
  const setFilterContext = useSquadFilterStore((s) => s.setFilterContext)
  const t = getSquadText(locale)

  // ── Filtres en attente (période + cascade) — commités via « Analyser » ────
  // Recalés sur le commité PENDANT le rendu, pas dans un effet : sinon la barre
  // paraît « sale » un commit après chaque snap ou clic du rail (aperçu relancé).
  const [pending, setPending] = useState<FilterContextInput>(() => filterContext)
  const [pendingBaseHash, setPendingBaseHash] = useState(filterContextHash)
  if (pendingBaseHash !== filterContextHash) {
    setPendingBaseHash(filterContextHash)
    setPending(filterContext)
  }

  const [activePopover, setActivePopover] = useState<SquadActivePopover>(null)

  const pendingCascade = pending.cascade ?? DEFAULT_CASCADE
  const pendingPeriod = pending.period

  function setPendingPeriod(p: PeriodInput) {
    const isPeriodSet = !!(p?.start_date || p?.end_date)
    setPending((prev) => ({
      ...prev,
      period: p,
      filter_mode: isPeriodSet ? 'period' : 'sessions',
      sessions: isPeriodSet ? DEFAULT_SESSIONS : prev.sessions,
    }))
  }
  function setPendingCascade(c: CascadeInput) {
    setPending((prev) => ({ ...prev, cascade: c }))
  }
  const togglePopover = (which: Exclude<SquadActivePopover, null>) =>
    setActivePopover((cur) => (cur === which ? null : which))
  const closeAllPopovers = () => setActivePopover(null)

  // Preview live : dérive un FilterContextInput depuis pending +
  // pickedSquadSessionLabels (cf. deriveSquadPending pour la sémantique).
  const squadPending = useMemo(
    () => deriveSquadPending(pending, pickedSquadSessionLabels),
    [pending, pickedSquadSessionLabels],
  )
  // Aperçu seulement filtres EN ATTENTE (D4.3) ; sinon le résolu commité (squad) sert.
  const isDirty = filterContextHash !== computePendingHash(pending)
  const { data: previewData } = useFiltersPreview(playerSlug, squadPending, { enabled: isDirty })
  const previewResolve = isDirty ? previewData : undefined

  const rawAvailable = previewResolve?.available_options ?? resolvedContext?.available_options
  const available = useMemo(() => cleanCascadeOptions(rawAvailable), [rawAvailable])

  const presetCounts = previewResolve?.period_presets ?? resolvedContext?.period_presets

  // ── Saisons (cascade-aware counts + détection saison active) ─────────────
  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)
  const seasonCounts = previewResolve?.season_counts ?? resolvedContext?.season_counts

  const cascadeCount = (['playlists', 'modes', 'maps', 'experience_types'] as const).reduce(
    (n, k) => n + ((pendingCascade[k] as string[] | undefined)?.length ?? 0),
    0,
  )

  const analyser = () => {
    setFilterContext(pending)
  }

  // Labels des playlists/modes du filtre courant → tri-en-tête des escouades dont
  // les contextes habituels matchent (indice souple).
  const activeContextLabels = useMemo(() => {
    const labels: string[] = []
    const collect = (opts: LabelValue[] | undefined, sel: string[] | undefined) => {
      if (!opts || !sel || sel.length === 0) return
      const selSet = new Set(sel)
      for (const o of opts) if (selSet.has(o.value)) labels.push(o.label)
    }
    collect(available?.playlists, pendingCascade.playlists as string[] | undefined)
    collect(available?.modes, pendingCascade.modes as string[] | undefined)
    return labels
  }, [available, pendingCascade])

  // Counts par session label — SOURCE UNIQUE en contexte escouade (ADR 0033) :
  // le compte « commencés ensemble » servi par teammates
  // (composition_sessions.match_count), exactement la population des tableaux
  // et graphes de la page. Les counts de /filters/resolve (population du
  // joueur principal, cascade seule) ne servent plus que de repli tant que la
  // réponse teammates n'est pas arrivée pour CE label — c'est cette double
  // source qui donnait 11/8/6/5 sur une même session (rail vs page).
  const sessionCountFallback = useMemo(
    () => resolveSquadSessionFallback(previewResolve, resolvedContext),
    [previewResolve, resolvedContext],
  )
  // {shown, total, hint} — alimente la L2 (PeriodSessionRail.sessionCount) :
  // « 4 sur 7 » quand la composition exacte écarte des matchs (D1), EXPLIQUÉ
  // au survol par la liste des matchs écartés (phase A3 — critère de succès
  // n°3 : l'écart doit être lisible ET expliqué, pas seulement visible).
  const getSessionCount = useMemo(
    () => (label: string) => {
      const count = squadSessionCount(label, compositionSessions, sessionCountFallback)
      if (!count) return undefined
      return {
        shown: count.shown,
        total: count.total,
        hint: buildCompositionGapHint(count.excluded, locale, t.compositionGap),
      }
    },
    [compositionSessions, sessionCountFallback, locale, t],
  )
  // Nombre seul — alimente SessionMultiSelect (masque les sessions vides +
  // affiche le compte par ligne), même module, même règle.
  const getSessionShownCount = useMemo(
    () => (label: string) =>
      squadSessionShownCount(label, compositionSessions, sessionCountFallback),
    [compositionSessions, sessionCountFallback],
  )

  return {
    pendingCascade,
    pendingPeriod,
    setPendingCascade,
    setPendingPeriod,
    activePopover,
    togglePopover,
    closeAllPopovers,
    available,
    presetCounts: presetCounts ?? undefined,
    seasons,
    activeSeason,
    seasonCounts: seasonCounts ?? undefined,
    cascadeCount,
    isDirty,
    analyser,
    activeContextLabels,
    getSessionCount,
    getSessionShownCount,
  }
}
