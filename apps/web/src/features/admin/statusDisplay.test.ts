import { describe, expect, it } from 'vitest'

import {
  jobStatusToAdminStatus,
  jobTypeLabel,
  schedulerOutcomeToAdminStatus,
  statusDisplay,
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

describe('jobTypeLabel', () => {
  it('traduit les types connus en FR et EN', () => {
    expect(jobTypeLabel('forced_sync_cycle', 'fr')).toBe('Cycle de sync forcé')
    expect(jobTypeLabel('forced_sync_cycle', 'en')).toBe('Forced sync cycle')
    expect(jobTypeLabel('scan_media', 'fr')).toBe('Scan médias')
  })

  it('retourne le type brut pour un type inconnu (repérable sans crash)', () => {
    expect(jobTypeLabel('nouveau_type_inconnu', 'fr')).toBe('nouveau_type_inconnu')
  })
})
