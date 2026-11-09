/// <reference types="node" />
/**
 * Attente d'un serveur qui démarre (serverStartup.ts) : quelles erreurs déclenchent la
 * réinterrogation chaque seconde, jusqu'où (en temps), comment les rejeux des erreurs
 * ordinaires se comptent malgré le compteur cumulé de TanStack, et le libellé de chaque
 * étape annoncée par le serveur — y compris toutes celles que le serveur Go déclare
 * (boot_gate.go).
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { commonManifest } from '@/lib/i18n/generated/common'
import {
  BOOTSTRAP_OTHER_ERROR_RETRIES,
  SERVER_STARTUP_MAX_WAIT_MS,
  SERVER_STARTUP_POLL_MS,
  createBootstrapRetryPolicy,
  isDesktopEntryRequiredError,
  isServerStartFailedError,
  isServerStartingError,
  serverStartFailure,
  serverStartingStepKey,
} from './serverStartup'

const starting = (step?: string) => ({
  code: 'server_starting',
  message: 'server starting',
  retryable: true,
  status: 503,
  details: step === undefined ? undefined : { step },
})

const failed = (state?: string) => ({
  code: 'server_start_failed',
  message: 'server start failed',
  retryable: false,
  status: 503,
  details: { step: 'migrations', ...(state === undefined ? {} : { state }) },
})

describe('isServerStartingError', () => {
  it('erreur réseau, 502 du proxy et 503 server_starting : le serveur démarre', () => {
    expect(isServerStartingError(new TypeError('Failed to fetch'))).toBe(true)
    expect(isServerStartingError({ code: 'unknown_error', status: 502, retryable: true })).toBe(true)
    expect(isServerStartingError(starting('accounts'))).toBe(true)
  })

  it('toute autre erreur : pas un démarrage', () => {
    expect(isServerStartingError({ code: 'db_busy', status: 503, retryable: true })).toBe(false)
    expect(isServerStartingError({ code: 'bootstrap_error', status: 500, retryable: true })).toBe(false)
    expect(isServerStartingError({ code: 'unknown_error', status: 504, retryable: true })).toBe(false)
    expect(isServerStartingError({ code: 'auth_required', status: 401, retryable: false })).toBe(false)
    expect(isServerStartingError(new DOMException('aborted', 'AbortError'))).toBe(false)
    expect(isServerStartingError(new SyntaxError('Unexpected token <'))).toBe(false)
    expect(isServerStartingError(null)).toBe(false)
    expect(isServerStartingError(undefined)).toBe(false)
  })
})

describe('démarrage échoué (503 server_start_failed, non réessayable)', () => {
  it('reconnu comme échec, jamais comme « le serveur démarre »', () => {
    expect(isServerStartFailedError(failed('migration_failed'))).toBe(true)
    expect(isServerStartingError(failed('migration_failed'))).toBe(false)
    expect(isServerStartFailedError(starting('migrations'))).toBe(false)
    expect(isServerStartFailedError({ code: 'server_start_failed', status: 500, retryable: false })).toBe(false)
    expect(isServerStartFailedError({ code: 'server_start_failed', status: 503, retryable: true })).toBe(false)
    expect(isServerStartFailedError(new TypeError('Failed to fetch'))).toBe(false)
    expect(isServerStartFailedError(null)).toBe(false)
  })

  it('chaque état : un libellé présent en FR et en EN', () => {
    for (const state of ['locked', 'migration_failed', 'schema_newer', 'resources_incompatible']) {
      const failure = serverStartFailure(failed(state))
      expect(failure?.stateKey, `état « ${state} » sans libellé`).toBe(`common.root.server_failed.${state}`)
      expect(commonManifest[failure!.stateKey!].fr).not.toBe('')
      expect(commonManifest[failure!.stateKey!].en).not.toBe('')
    }
    expect(commonManifest['common.root.server_failed.title'].fr).not.toBe('')
    expect(commonManifest['common.root.server_failed.title'].en).not.toBe('')
  })

  it('état absent ou inconnu : l’écran d’échec sans détail ; autre erreur : pas d’écran', () => {
    expect(serverStartFailure(failed())).toEqual({ stateKey: undefined })
    expect(serverStartFailure(failed('constructor'))).toEqual({ stateKey: undefined })
    expect(serverStartFailure(failed('toString'))).toEqual({ stateKey: undefined })
    expect(serverStartFailure(starting('migrations'))).toBeNull()
    expect(serverStartFailure(new TypeError('Failed to fetch'))).toBeNull()
  })

  it('chaque état déclaré par le serveur Go a un libellé', () => {
    const goSource = readFileSync(
      resolve(__dirname, '../../../go-api/cmd/server/boot_gate.go'),
      'utf8',
    )
    const goStates = [...goSource.matchAll(/bootFailure\w+\s+bootFailureState\s*=\s*"([a-z_]+)"/g)].map((m) => m[1])
    expect(goStates.length).toBeGreaterThan(0)
    for (const state of goStates) {
      expect(serverStartFailure(failed(state))?.stateKey, `état Go « ${state} » sans libellé web`).toBeDefined()
    }
  })
})

describe('serverStartingStepKey', () => {
  it('étape connue : son libellé, présent en FR et en EN', () => {
    for (const step of ['migrations', 'databases', 'accounts', 'services']) {
      const key = serverStartingStepKey(starting(step))
      expect(key).toBe(`common.root.server_step.${step}`)
      expect(commonManifest[key!].fr).not.toBe('')
      expect(commonManifest[key!].en).not.toBe('')
    }
  })

  it('étape absente ou inconnue : aucun libellé', () => {
    expect(serverStartingStepKey(starting())).toBeUndefined()
    expect(serverStartingStepKey(starting('reticulating_splines'))).toBeUndefined()
    expect(serverStartingStepKey(starting('constructor'))).toBeUndefined()
    expect(serverStartingStepKey(starting('toString'))).toBeUndefined()
    expect(serverStartingStepKey(new TypeError('Failed to fetch'))).toBeUndefined()
    expect(serverStartingStepKey({ status: 503, code: 'server_starting', details: { step: 42 } })).toBeUndefined()
  })

  it('chaque étape déclarée par le serveur Go a un libellé', () => {
    const goSource = readFileSync(
      resolve(__dirname, '../../../go-api/cmd/server/boot_gate.go'),
      'utf8',
    )
    const goSteps = [...goSource.matchAll(/bootStep\w+\s+bootStep\s*=\s*"([a-z_]+)"/g)].map((m) => m[1])
    expect(goSteps.length).toBeGreaterThan(0)
    for (const step of goSteps) {
      expect(serverStartingStepKey(starting(step)), `étape Go « ${step} » sans libellé web`).toBeDefined()
    }
  })
})

/**
 * Rejoue la boucle de rejeu de TanStack Query : pour chaque échec, retryDelay PUIS retry,
 * avec le compteur d'échecs CUMULÉ de la série ; l'horloge avance du délai de chaque rejeu.
 * Rend les délais des rejeux accordés et le rang de l'échec qui arrête la série.
 */
function runRetryLoop(errors: unknown[], policy = createClockedPolicy()) {
  const delays: number[] = []
  for (let k = 0; k < errors.length; k++) {
    const delay = policy.policy.retryDelay(k, errors[k])
    if (!policy.policy.retry(k, errors[k])) return { delays, stoppedAt: k as number | null }
    delays.push(delay)
    policy.advance(delay)
  }
  return { delays, stoppedAt: null as number | null }
}

function createClockedPolicy() {
  let t = 0
  return {
    policy: createBootstrapRetryPolicy(() => t),
    advance: (ms: number) => {
      t += ms
    },
  }
}

const ordinary = () => ({ code: 'bootstrap_error', status: 500, retryable: true })
const repeat = (n: number, make: () => unknown) => Array.from({ length: n }, make)

const desktopUnauthorized = () => ({
  code: 'desktop_unauthorized',
  message: 'entrée de la coquille requise',
  retryable: false,
  status: 401,
})

describe('entrée de la coquille desktop manquante (401 desktop_unauthorized)', () => {
  it('reconnue par son statut ET son code, jamais comme un démarrage ni un démarrage échoué', () => {
    expect(isDesktopEntryRequiredError(desktopUnauthorized())).toBe(true)
    expect(isDesktopEntryRequiredError({ code: 'auth_required', status: 401, retryable: false })).toBe(false)
    expect(isDesktopEntryRequiredError({ code: 'desktop_unauthorized', status: 403, retryable: false })).toBe(false)
    expect(isDesktopEntryRequiredError(new TypeError('Failed to fetch'))).toBe(false)
    expect(isDesktopEntryRequiredError(null)).toBe(false)
    expect(isServerStartingError(desktopUnauthorized())).toBe(false)
    expect(isServerStartFailedError(desktopUnauthorized())).toBe(false)
  })

  it('aucun rejeu', () => {
    expect(runRetryLoop([desktopUnauthorized()])).toEqual({ delays: [], stoppedAt: 0 })
  })

  it('libellés de l’écran présents en FR et en EN', () => {
    for (const key of ['common.root.desktop_entry_required.title', 'common.root.desktop_entry_required.body'] as const) {
      expect(commonManifest[key].fr).not.toBe('')
      expect(commonManifest[key].en).not.toBe('')
      expect(commonManifest[key].fr).not.toBe(commonManifest[key].en)
    }
  })
})

describe('createBootstrapRetryPolicy', () => {
  it('serveur qui démarre : chaque seconde, jusqu’à 120 s mesurées en temps', () => {
    const { delays, stoppedAt } = runRetryLoop(repeat(200, () => starting('accounts')))
    expect(stoppedAt).toBe(SERVER_STARTUP_MAX_WAIT_MS / SERVER_STARTUP_POLL_MS)
    expect(new Set(delays)).toEqual(new Set([SERVER_STARTUP_POLL_MS]))
  })

  it('démarrage échoué : aucun rejeu, même après des attentes de démarrage', () => {
    expect(runRetryLoop([failed('locked')])).toEqual({ delays: [], stoppedAt: 0 })
    const { delays, stoppedAt } = runRetryLoop([...repeat(5, () => starting('migrations')), failed('migration_failed')])
    expect(stoppedAt).toBe(5)
    expect(delays).toEqual(repeat(5, () => SERVER_STARTUP_POLL_MS))
  })

  it('migrations en cours : l’attente de 120 s repart à chaque réponse', () => {
    // 300 s de réponses « migrations » : jamais d’arrêt.
    expect(runRetryLoop(repeat(300, () => starting('migrations'))).stoppedAt).toBeNull()
    // Puis une autre étape : 120 s à compter de la dernière réponse « migrations » (rang 199).
    const errors = [...repeat(200, () => starting('migrations')), ...repeat(200, () => starting('accounts'))]
    expect(runRetryLoop(errors).stoppedAt).toBe(199 + SERVER_STARTUP_MAX_WAIT_MS / SERVER_STARTUP_POLL_MS)
  })

  it('réseau et 502 comptent aussi comme démarrage', () => {
    const errors = [new TypeError('Failed to fetch'), { status: 502 }, starting('migrations')]
    expect(runRetryLoop(errors).delays).toEqual([1000, 1000, 1000])
  })

  it('erreur ordinaire seule : six rejeux espacés de 0,5 s à 4 s', () => {
    const { delays, stoppedAt } = runRetryLoop(repeat(10, ordinary))
    expect(stoppedAt).toBe(BOOTSTRAP_OTHER_ERROR_RETRIES)
    expect(delays).toEqual([500, 1000, 2000, 4000, 4000, 4000])
  })

  it('séquence mixte : N attentes de démarrage puis erreurs ordinaires → six rejeux 0,5 → 4 s', () => {
    const errors = [...repeat(10, () => starting('services')), ...repeat(10, ordinary)]
    const { delays, stoppedAt } = runRetryLoop(errors)
    expect(stoppedAt).toBe(10 + BOOTSTRAP_OTHER_ERROR_RETRIES)
    expect(delays.slice(10)).toEqual([500, 1000, 2000, 4000, 4000, 4000])
  })

  it('erreurs ordinaires recomptées après chaque attente de démarrage', () => {
    const errors = [starting(), ...repeat(3, ordinary), starting(), ...repeat(10, ordinary)]
    const { delays, stoppedAt } = runRetryLoop(errors)
    expect(stoppedAt).toBe(1 + 3 + 1 + BOOTSTRAP_OTHER_ERROR_RETRIES)
    expect(delays).toEqual([1000, 500, 1000, 2000, 1000, 500, 1000, 2000, 4000, 4000, 4000])
  })

  it('nouvelle série (compteur qui repart à 0) : attente et rejeux remis à zéro', () => {
    const clocked = createClockedPolicy()
    expect(runRetryLoop(repeat(200, () => starting()), clocked).stoppedAt).toBe(120)
    // Refetch plus tard : nouvelle série, l'attente de 120 s repart de son premier échec.
    const again = runRetryLoop(repeat(5, () => starting()), clocked)
    expect(again.stoppedAt).toBeNull()
    expect(runRetryLoop([...repeat(3, () => starting()), ...repeat(8, ordinary)], clocked).stoppedAt).toBe(
      3 + BOOTSTRAP_OTHER_ERROR_RETRIES,
    )
  })
})
