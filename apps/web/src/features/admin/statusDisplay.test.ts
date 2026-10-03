import { describe, expect, it } from 'vitest'

import {
  jobStatusToAdminStatus,
  jobTypeLabel,
  schedulerOutcomeToAdminStatus,
  statusDisplay,
  watcherLivenessStatus,
  WATCHER_STALE_MS,
} from './statusDisplay'

describe('statusDisplay', () => {
  it('mappe les statuts actifs sur info + pulse', () => {
    expect(statusDisplay('running')).toEqual({ token: 'info', labelKey: 'admin.status.running', pulse: true })
    expect(statusDisplay('queued').pulse).toBe(true)
  })

  it('mappe les terminaux sur les bons tokens sans pulse', () => {
    expect(statusDisplay('succeeded').token).toBe('success')
    expect(statusDisplay('failed').token).toBe('destructive')
    expect(statusDisplay('interrupted').token).toBe('warning')
    expect(statusDisplay('succeeded').pulse).toBe(false)
  })

  it('retombe sur idle (neutre) pour une valeur inconnue', () => {
    const out = statusDisplay('statut-inconnu')
    expect(out.labelKey).toBe('admin.status.idle')
    expect(out.token).toBeUndefined()
    expect(out.pulse).toBe(false)
  })
})

describe('jobStatusToAdminStatus', () => {
  it('couvre les 6 statuts de job', () => {
    expect(jobStatusToAdminStatus('queued')).toBe('queued')
    expect(jobStatusToAdminStatus('running')).toBe('running')
    expect(jobStatusToAdminStatus('succeeded')).toBe('succeeded')
    expect(jobStatusToAdminStatus('failed')).toBe('failed')
    expect(jobStatusToAdminStatus('cancelled')).toBe('cancelled')
    expect(jobStatusToAdminStatus('interrupted')).toBe('interrupted')
  })

  it('valeur inconnue → idle', () => {
    expect(jobStatusToAdminStatus('autre')).toBe('idle')
  })
})

describe('schedulerOutcomeToAdminStatus', () => {
  it('mappe ok/skipped/failed et dégrade le reste', () => {
    expect(schedulerOutcomeToAdminStatus('ok')).toBe('ok')
    expect(schedulerOutcomeToAdminStatus('skipped')).toBe('skipped')
    expect(schedulerOutcomeToAdminStatus('failed')).toBe('failed')
    expect(schedulerOutcomeToAdminStatus('')).toBe('idle')
  })
})

describe('watcherLivenessStatus', () => {
  const now = new Date('2026-06-12T12:00:00Z').getTime()

  it('daemon arrêté → idle quel que soit le dernier event', () => {
    expect(watcherLivenessStatus('2026-06-12T11:59:55Z', false, now)).toBe('idle')
    expect(watcherLivenessStatus(undefined, false, now)).toBe('idle')
  })

  it('daemon actif sans aucun event → warning (en attente du premier snapshot)', () => {
    expect(watcherLivenessStatus(undefined, true, now)).toBe('warning')
  })

  it('event récent (< seuil) → ok', () => {
    const fresh = new Date(now - 10_000).toISOString() // 10 s, un cycle de poll
    expect(watcherLivenessStatus(fresh, true, now)).toBe('ok')
  })

  it('event juste sous le seuil → ok ; juste au-dessus → error', () => {
    const justUnder = new Date(now - (WATCHER_STALE_MS - 1)).toISOString()
    const justOver = new Date(now - (WATCHER_STALE_MS + 1)).toISOString()
    expect(watcherLivenessStatus(justUnder, true, now)).toBe('ok')
    expect(watcherLivenessStatus(justOver, true, now)).toBe('error')
  })

  it('timestamp illisible → warning (jamais de crash)', () => {
    expect(watcherLivenessStatus('pas-une-date', true, now)).toBe('warning')
  })
})

describe('jobTypeLabel', () => {
  it('traduit les types connus en FR et EN', () => {
    expect(jobTypeLabel('forced_sync_cycle', 'fr')).toBe('Cycle de sync forcé')
    expect(jobTypeLabel('forced_sync_cycle', 'en')).toBe('Forced sync cycle')
    expect(jobTypeLabel('scan_media', 'fr')).toBe('Scan médias')
    expect(jobTypeLabel('replay_build', 'fr')).toBe('Construction du rejeu 2D')
    expect(jobTypeLabel('replay_build', 'en')).toBe('2D replay build')
  })

  it('retourne le type brut pour un type inconnu (repérable sans crash)', () => {
    expect(jobTypeLabel('nouveau_type_inconnu', 'fr')).toBe('nouveau_type_inconnu')
  })
})
