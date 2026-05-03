/**
 * _hooks.ts — helpers et constantes partagés par les pills de FilterOmnibar.
 *
 * Extrait de FilterOmnibar.tsx pour respecter la limite 500L/module.
 */
import { useEffect, useRef } from 'react'
import { DEFAULT_GAP_MINUTES } from '@/stores/filterDefaults'
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

export const PERIOD_PRESETS = [
  { id: '7d', label: '7 jours', days: 7 },
  { id: '30d', label: '30 jours', days: 30 },
  { id: '90d', label: '90 jours', days: 90 },
  { id: 'all', label: 'Toutes', days: 0 },
] as const

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

export function computePendingHash(ctx: FilterContextInput): string {
  try {
    return btoa(JSON.stringify(ctx)).slice(0, 32)
  } catch {
    return 'default'
  }
}
