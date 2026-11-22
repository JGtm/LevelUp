/**
 * Tests — logique pure de l'encadré « Synchronisation » : cadence de relecture (jamais au repos),
 * lignes « dernière passe » et « prochaine passe ».
 */
import { describe, it, expect } from 'vitest'
import {
  HISTORY_POLL_CATCHING_UP_MS,
  buttonLabel,
  formatRemaining,
  manualStatusLine,
  remainingSeconds,
  HISTORY_POLL_RUNNING_MS,
  SYNC_CYCLE_POLL_MS,
  historyRefetchMs,
  lastPassAddedLine,
  lastPassLine,
  nextPassText,
  startFailureText,
  syncCycleRefetchMs,
} from './syncNowLogic'
import type { HistoryPass, HistorySyncStatus, TitleHistoryStatus } from './syncNowQueries'
import { getSyncNowText } from './syncNowText'

const T_FR = getSyncNowText('fr')
const T_EN = getSyncNowText('en')

function title(state: TitleHistoryStatus['state']): TitleHistoryStatus {
  return { title_slug: 'halo_infinite', title_name: 'Halo Infinite', state, state_label: state, match_count: 10 }
}

function status(over: Partial<HistorySyncStatus> = {}): HistorySyncStatus {
  return { auto_sync_enabled: true, titles: [title('complete')], ...over }
}

describe('cadence de relecture', () => {
  it('l’état du cycle ne se relit que pendant un cycle', () => {
    expect(syncCycleRefetchMs(undefined)).toBe(false)
    expect(syncCycleRefetchMs({ running: false })).toBe(false)
    expect(syncCycleRefetchMs({ running: true })).toBe(SYNC_CYCLE_POLL_MS)
  })

  it('l’historique se relit vite pendant un cycle, lentement pendant un rattrapage, jamais au repos', () => {
    expect(historyRefetchMs(status({ titles: [title('complete')] }), true)).toBe(HISTORY_POLL_RUNNING_MS)
    expect(historyRefetchMs(undefined, true)).toBe(HISTORY_POLL_RUNNING_MS)
    expect(historyRefetchMs(status({ titles: [title('complete'), title('catching_up')] }), false)).toBe(
      HISTORY_POLL_CATCHING_UP_MS,
    )
    expect(historyRefetchMs(status({ titles: [title('complete')] }), false)).toBe(false)
    expect(historyRefetchMs(status({ titles: [title('paused'), title('unknown')] }), false)).toBe(false)
    expect(historyRefetchMs(status({ titles: [] }), false)).toBe(false)
    expect(historyRefetchMs(undefined, false)).toBe(false)
  })
})

describe('refus d’un lancement', () => {
  it('un cycle déjà en cours a son texte, tout autre refus le texte général', () => {
    expect(startFailureText(T_FR, { code: 'sync_cycle_busy', message: 'x', status: 409 })).toBe(T_FR.toastBusy)
    expect(startFailureText(T_FR, { code: 'scheduler_unavailable', message: 'x', status: 503 })).toBe(T_FR.toastFailed)
    expect(startFailureText(T_EN, new Error('réseau'))).toBe(T_EN.toastFailed)
  })
})

describe('dernière passe', () => {
  const pass = (over: Partial<HistoryPass> = {}): HistoryPass => ({
    at: '2026-11-22T14:00:00Z',
    matches_added: 200,
    status: 'ok',
    status_label: 'Réussie',
    ...over,
  })

  it('dit l’issue résolue par le serveur et la date', () => {
    expect(lastPassLine(T_FR, 'fr', pass())).toMatch(/^Réussie, le .*2026/)
    expect(lastPassLine(T_EN, 'en', pass({ status_label: 'Succeeded' }))).toMatch(/^Succeeded, on .*2026/)
  })

  it('compte les matchs ajoutés pour une passe qui a tourné, pas pour un échec ni une passe différée', () => {
    expect(lastPassAddedLine(T_FR, pass())).toBe('Matchs ajoutés : 200')
    expect(lastPassAddedLine(T_EN, pass({ status: 'partial', matches_added: 3 }))).toBe('Matches added: 3')
    expect(lastPassAddedLine(T_FR, pass({ status: 'failed', matches_added: 0 }))).toBeNull()
    expect(lastPassAddedLine(T_FR, pass({ status: 'skipped', matches_added: 0 }))).toBeNull()
  })
})

describe('prochaine passe', () => {
  it('un cycle en cours prime sur tout le reste', () => {
    expect(nextPassText(T_FR, 'fr', status({ auto_sync_enabled: false }), true)).toBe(T_FR.nextPassRunning)
  })

  it('synchronisation automatique désactivée : le dit, sans heure', () => {
    expect(nextPassText(T_EN, 'en', status({ auto_sync_enabled: false, next_cycle_at: '2026-11-22T14:10:00Z' }), false)).toBe(
      T_EN.nextPassManualOnly,
    )
  })

  it('heure prévue, ou pas encore planifiée', () => {
    expect(nextPassText(T_FR, 'fr', status({ next_cycle_at: '2026-11-22T14:10:00Z' }), false)).toMatch(/^Vers .*2026/)
    expect(nextPassText(T_FR, 'fr', status(), false)).toBe(T_FR.nextPassUnknown)
  })
})

describe('délai minimal avant un lancement manuel', () => {
  const NOW = Date.parse('2026-11-22T14:02:30Z')

  it('secondes restantes : arrondies au-dessus, zéro une fois l’heure passée ou sans heure', () => {
    expect(remainingSeconds('2026-11-22T14:04:10Z', NOW)).toBe(100)
    expect(remainingSeconds('2026-11-22T14:02:30.400Z', NOW)).toBe(1)
    expect(remainingSeconds('2026-11-22T14:02:30Z', NOW)).toBe(0)
    expect(remainingSeconds('2026-11-22T14:00:00Z', NOW)).toBe(0)
    expect(remainingSeconds(undefined, NOW)).toBe(0)
    expect(remainingSeconds('pas une date', NOW)).toBe(0)
  })

  it('« 1 min 40 » au-delà d’une minute, « 40 s » en dessous', () => {
    expect(formatRemaining(100)).toBe('1 min 40')
    expect(formatRemaining(125)).toBe('2 min 05')
    expect(formatRemaining(60)).toBe('1 min 00')
    expect(formatRemaining(59)).toBe('59 s')
    expect(formatRemaining(1)).toBe('1 s')
  })

  it('libellé du bouton : en cours, délai, ou action', () => {
    expect(buttonLabel(T_FR, true, 0)).toBe('Synchronisation en cours…')
    expect(buttonLabel(T_FR, true, 40)).toBe('Synchronisation en cours…')
    expect(buttonLabel(T_FR, false, 100)).toBe('Disponible dans 1 min 40')
    expect(buttonLabel(T_EN, false, 100)).toBe('Available in 1 min 40')
    expect(buttonLabel(T_EN, false, 0)).toBe('Sync now')
  })

  it('ligne sous le bouton : cycle en cours, délai, ou rien', () => {
    const cycle = { last_cycle_at: '2026-11-22T14:02:00Z', manual_available_at: '2026-11-22T14:04:00Z' }
    expect(manualStatusLine(T_FR, 'fr', cycle, true, 0)).toBe('Synchronisation en cours.')
    expect(manualStatusLine(T_EN, 'en', cycle, true, 0)).toBe('Sync in progress.')
    expect(manualStatusLine(T_FR, 'fr', cycle, false, 0)).toBeNull()
    expect(manualStatusLine(T_FR, 'fr', undefined, false, 0)).toBeNull()
    expect(manualStatusLine(T_FR, 'fr', cycle, false, 90)).toMatch(
      /^Dernière synchronisation terminée à \d{2}:\d{2}\. Prochain lancement possible à \d{2}:\d{2}\.$/,
    )
    expect(manualStatusLine(T_EN, 'en', cycle, false, 90)).toMatch(/^Last sync finished at .+\. Next run available at .+\.$/)
  })
})
