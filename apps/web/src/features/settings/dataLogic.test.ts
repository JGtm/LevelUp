/**
 * Tests — logique pure de la carte « Données » (lot L22).
 */
import { describe, it, expect } from 'vitest'
import {
  BACKUP_INTERVALS,
  DATA_STATUS_FAST_MS,
  DATA_STATUS_SLOW_MS,
  DEFAULT_BACKUP_INTERVAL,
  dataStatusRefetchMs,
  errorText,
  fieldOfCode,
  knownInterval,
  parseWholeNumber,
} from './dataLogic'
import { getDataText } from './dataText'
import { fill } from './fillTemplate'

/** Tous les codes de refus que le serveur peut renvoyer pour les réglages de données et la sauvegarde. */
const SERVER_CODES = [
  'backup_dir_not_absolute',
  'backup_dir_missing',
  'backup_dir_not_writable',
  'backup_dir_inside_data',
  'backup_interval_invalid',
  'backup_keep_invalid',
  'memory_limit_invalid',
  'data_setting_not_offered',
  'backup_disabled',
  'backup_already_running',
]

describe('dataLogic', () => {
  it('relit l’état toutes les 5 s pendant une sauvegarde, toutes les 30 s sinon', () => {
    expect(dataStatusRefetchMs({ backup: { enabled: true, running: true, same_volume: false } })).toBe(
      DATA_STATUS_FAST_MS,
    )
    expect(dataStatusRefetchMs({ backup: { enabled: true, running: false, same_volume: false } })).toBe(
      DATA_STATUS_SLOW_MS,
    )
    expect(dataStatusRefetchMs({})).toBe(DATA_STATUS_SLOW_MS)
    expect(dataStatusRefetchMs(undefined)).toBe(DATA_STATUS_SLOW_MS)
  })

  it('offre les six fréquences, 24h par défaut ; une valeur inconnue retombe sur le défaut', () => {
    expect([...BACKUP_INTERVALS].sort()).toEqual(['12h', '168h', '1h', '24h', '6h', '72h'])
    expect(knownInterval('6h')).toBe('6h')
    expect(knownInterval('5h')).toBe(DEFAULT_BACKUP_INTERVAL)
    expect(knownInterval(undefined)).toBe('24h')
  })

  it('lit un entier, refuse le reste', () => {
    expect(parseWholeNumber(' 2048 ')).toBe(2048)
    expect(parseWholeNumber('0')).toBe(0)
    expect(parseWholeNumber('12,5')).toBeNull()
    expect(parseWholeNumber('')).toBeNull()
    expect(parseWholeNumber('abc')).toBeNull()
  })

  it('rattache chaque code de refus de réglage à son champ', () => {
    expect(fieldOfCode('backup_dir_missing')).toBe('backup_dir')
    expect(fieldOfCode('backup_interval_invalid')).toBe('backup_interval')
    expect(fieldOfCode('backup_keep_invalid')).toBe('backup_keep')
    expect(fieldOfCode('memory_limit_invalid')).toBe('memory_limit_mb')
    expect(fieldOfCode('backup_already_running')).toBeUndefined()
    expect(fieldOfCode(undefined)).toBeUndefined()
  })

  it.each(['fr', 'en'] as const)('chaque code du serveur a son texte en %s', (locale) => {
    const t = getDataText(locale)
    for (const code of SERVER_CODES) {
      expect(errorText(t, code), `${locale}/${code}`).toBeTruthy()
    }
  })

  it('remplit un gabarit et laisse tel quel un nom absent', () => {
    expect(fill('{a} et {b}', { a: 1, b: 'deux' })).toBe('1 et deux')
    expect(fill('{a} {x}', { a: 1 })).toBe('1 {x}')
  })
})
