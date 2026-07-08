/**
 * _hooks.ts — helpers et constantes partagés par les pills de FilterOmnibar.
 *
 * Extrait de FilterOmnibar.tsx pour respecter la limite 500L/module.
 */
import { useEffect, useRef } from 'react'
import { DEFAULT_GAP_MINUTES } from '@/stores/filterDefaults'
import type { CommonManifestKey } from '@/lib/i18n/generated/common'
import type {
  CascadeInput,
  FilterContextInput,
  PeriodInput,
  SessionsInput,
} from '@/lib/api/types'

// ─── Constantes par défaut ───────────────────────────────────────────────────

export const DEFAULT_CASCADE: CascadeInput = {
  experience_types: [],
  playlists: [],
  modes: [],
  maps: [],
}
export const DEFAULT_SESSIONS: SessionsInput = {
  picked_sessions: [],
  gap_minutes: DEFAULT_GAP_MINUTES,
}
export const DEFAULT_PERIOD: PeriodInput = { start_date: null, end_date: null }

// ─── Période — presets et helpers ────────────────────────────────────────────

// labelKey : clé i18n résolue par le consommateur (PeriodePill) via `t()`.
export const PERIOD_PRESETS = [
  { id: '7d', labelKey: 'common.period.preset_7d', days: 7 },
  { id: '30d', labelKey: 'common.period.preset_30d', days: 30 },
  { id: '90d', labelKey: 'common.period.preset_90d', days: 90 },
  { id: 'all', labelKey: 'common.period.preset_all', days: 0 },
] as const satisfies readonly { id: string; labelKey: CommonManifestKey; days: number }[]

export type PresetId = (typeof PERIOD_PRESETS)[number]['id'] | 'custom'

export function isoDate(d: Date): string {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

export function presetPeriod(days: number): PeriodInput | null {
  if (days <= 0) return null
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - days)
  return { start_date: isoDate(start), end_date: isoDate(end) }
}

export function detectActivePreset(period: PeriodInput | undefined): PresetId {
  if (!period) return 'all'
  if (!period.start_date && !period.end_date) return 'all'
  for (const p of PERIOD_PRESETS) {
    const expected = presetPeriod(p.days)
    if (!expected) continue
    if (
      period.start_date === expected.start_date &&
      period.end_date === expected.end_date
    ) {
      return p.id
    }
  }
  return 'custom'
}

// ─── Labels UUID — masquer les options non traduites ─────────────────────────

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
export function isUUIDLabel(label: string): boolean {
  return UUID_RE.test(label.trim())
}

// ─── Hook : popover dismissable (click-outside + Escape) ─────────────────────

export function useDismissable(open: boolean, onClose: () => void) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    function onDocClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', onDocClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open, onClose])
  return ref
}

// ─── Hash de l'état pending — détection de dirty ─────────────────────────────
//
// Doit utiliser EXACTEMENT le même algo que `computeHash` dans
// `globalFilterStore.ts` : `isDirty` compare `filterContextHash` (calculé par
// le store) à `computePendingHash(pending)` (calculé ici). Un algo divergent
// rendrait le bouton « Analyser » faussement actif/inactif.

export function computePendingHash(ctx: FilterContextInput): string {
  const s = JSON.stringify(ctx) ?? ''
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h.toString(16).padStart(8, '0')
}
