/// <reference types="node" />
/**
 * Attente d'un serveur qui démarre (serverStartup.ts) : quelles erreurs déclenchent la
 * réinterrogation chaque seconde, jusqu'où, et le libellé de chaque étape annoncée par le
 * serveur — y compris toutes celles que le serveur Go déclare (boot_gate.go).
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { commonManifest } from '@/lib/i18n/generated/common'
import {
  BOOTSTRAP_OTHER_ERROR_RETRIES,
  SERVER_STARTUP_MAX_RETRIES,
  SERVER_STARTUP_MAX_WAIT_MS,
  SERVER_STARTUP_POLL_MS,
  bootstrapRetryDelay,
  isServerStartingError,
  serverStartingStepKey,
  shouldRetryBootstrap,
} from './serverStartup'

const starting = (step?: string) => ({
  code: 'server_starting',
  message: 'server starting',
  retryable: true,
  status: 503,
  details: step === undefined ? undefined : { step },
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

describe('shouldRetryBootstrap / bootstrapRetryDelay', () => {
  it('serveur qui démarre : chaque seconde, jusqu’au plafond de 120 s', () => {
    expect(SERVER_STARTUP_MAX_RETRIES * SERVER_STARTUP_POLL_MS).toBeGreaterThanOrEqual(SERVER_STARTUP_MAX_WAIT_MS)
    for (const error of [new TypeError('Failed to fetch'), { status: 502 }, starting('migrations')]) {
      expect(shouldRetryBootstrap(0, error)).toBe(true)
      expect(shouldRetryBootstrap(SERVER_STARTUP_MAX_RETRIES - 1, error)).toBe(true)
      expect(shouldRetryBootstrap(SERVER_STARTUP_MAX_RETRIES, error)).toBe(false)
      for (const attempt of [0, 1, 5, 50]) {
        expect(bootstrapRetryDelay(attempt, error)).toBe(SERVER_STARTUP_POLL_MS)
      }
    }
  })

  it('autre erreur : six rejeux espacés de 0,5 s à 4 s', () => {
    const error = { code: 'bootstrap_error', status: 500, retryable: true }
    expect(shouldRetryBootstrap(BOOTSTRAP_OTHER_ERROR_RETRIES - 1, error)).toBe(true)
    expect(shouldRetryBootstrap(BOOTSTRAP_OTHER_ERROR_RETRIES, error)).toBe(false)
    expect([0, 1, 2, 3, 4, 5].map((n) => bootstrapRetryDelay(n, error))).toEqual([500, 1000, 2000, 4000, 4000, 4000])
  })
})
