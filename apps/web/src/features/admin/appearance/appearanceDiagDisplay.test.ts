/**
 * Tests appearanceDiagDisplay — logique PURE (mapping verdict/detail/composant →
 * statut de badge + clés i18n). Vérifie aussi que TOUTE clé retournée existe
 * réellement dans le manifest admin (parité FR/EN garantie par le manifest).
 */
import { describe, it, expect } from 'vitest'
import { adminManifest } from '@/lib/i18n/generated/admin'
import {
  componentLabelKey,
  detailExplanationKey,
  fetchStatusKey,
  isImageComponent,
  servedFromKey,
  verdictActionKey,
  verdictActionKind,
  verdictBadgeStatus,
  verdictLabelKey,
} from './appearanceDiagDisplay'

const VERDICTS = ['ok', 'upstream_missing', 'transient', 'auth_required', 'not_supported'] as const

const DETAILS = [
  'mapping_hit',
  'mapping_miss',
  'image_resolved',
  'service_tag_present',
  'no_positive_cfg',
  'cms_http_error',
  'image_unresolved',
  'no_service_tag',
  'no_emblem_path',
  'non_emblem_path',
  'no_banner_field',
] as const

const COMPONENTS = ['banner', 'emblem', 'backdrop', 'service_tag'] as const

/** Garde-fou : la clé i18n retournée doit exister (sinon typecheck OK mais
 *  runtime rend la clé brute). */
function assertKey(key: string) {
  expect(Object.prototype.hasOwnProperty.call(adminManifest, key)).toBe(true)
}

describe('appearanceDiagDisplay — badge par verdict', () => {
  it('mappe chaque verdict vers le bon statut de badge (token)', () => {
    expect(verdictBadgeStatus('ok')).toBe('ok')
    expect(verdictBadgeStatus('transient')).toBe('warning')
    expect(verdictBadgeStatus('auth_required')).toBe('error')
    expect(verdictBadgeStatus('upstream_missing')).toBe('idle')
    expect(verdictBadgeStatus('not_supported')).toBe('idle')
  })

  it('verdict inconnu → neutre (jamais un faux « cassé »)', () => {
    expect(verdictBadgeStatus('wat')).toBe('idle')
  })

  it('libellé de verdict : clé i18n existante pour les 5 verdicts + inconnu', () => {
    for (const v of VERDICTS) assertKey(verdictLabelKey(v))
    assertKey(verdictLabelKey('wat'))
  })
})

describe('appearanceDiagDisplay — action « quoi faire »', () => {
  it('nature de l’action par verdict', () => {
    expect(verdictActionKind('ok')).toBe('none')
    expect(verdictActionKind('upstream_missing')).toBe('none')
    expect(verdictActionKind('not_supported')).toBe('none')
    expect(verdictActionKind('transient')).toBe('wait')
    expect(verdictActionKind('auth_required')).toBe('reauth')
  })

  it('texte d’action : clé existante pour les 5 verdicts + inconnu', () => {
    for (const v of VERDICTS) assertKey(verdictActionKey(v))
    assertKey(verdictActionKey('wat'))
  })
})

describe('appearanceDiagDisplay — POURQUOI (detail)', () => {
  it('chaque detail technique connu a une explication dédiée existante', () => {
    for (const d of DETAILS) assertKey(detailExplanationKey('transient', d))
  })

  it('detail vide → repli par verdict (chemins uniformes auth_required/not_supported)', () => {
    assertKey(detailExplanationKey('auth_required', ''))
    assertKey(detailExplanationKey('not_supported', ''))
    // Le repli auth_required diffère du repli not_supported.
    expect(detailExplanationKey('auth_required', '')).not.toBe(
      detailExplanationKey('not_supported', ''),
    )
  })

  it('detail inconnu + verdict inconnu → repli générique existant', () => {
    assertKey(detailExplanationKey('wat', 'nope'))
    expect(detailExplanationKey('wat', 'nope')).toBe('admin.appearance.detail.fallback_unknown')
  })
})

describe('appearanceDiagDisplay — composants', () => {
  it('libellé existant pour les 4 composants + inconnu', () => {
    for (const c of COMPONENTS) assertKey(componentLabelKey(c))
    assertKey(componentLabelKey('wat'))
  })

  it('banner/emblem/backdrop = image ; service_tag = texte', () => {
    expect(isImageComponent('banner')).toBe(true)
    expect(isImageComponent('emblem')).toBe(true)
    expect(isImageComponent('backdrop')).toBe(true)
    expect(isImageComponent('service_tag')).toBe(false)
  })
})

describe('appearanceDiagDisplay — served_from & fetch_status', () => {
  it('served_from live vs carry', () => {
    expect(servedFromKey('live')).toBe('admin.appearance.served_from.live')
    expect(servedFromKey('carry')).toBe('admin.appearance.served_from.carry')
    assertKey(servedFromKey('live'))
    assertKey(servedFromKey('carry'))
  })

  it('fetch_status : les 5 valeurs connues + vide → « jamais tenté »', () => {
    for (const s of ['ok', 'api_empty', 'forbidden_403', 'auth_missing', 'failed']) {
      assertKey(fetchStatusKey(s))
    }
    expect(fetchStatusKey('')).toBe('admin.appearance.fetch_status.none')
    assertKey(fetchStatusKey(''))
  })
})
