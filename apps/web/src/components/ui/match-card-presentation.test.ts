import { describe, expect, it } from 'vitest'
import {
  analysisPendingLabelKey,
  analysisPendingPartKeys,
  getMatchCardOutcomeStyle,
  getMatchNarrativeBadgeMeta,
} from './match-card-presentation'

describe('match-card-presentation', () => {
  it('retourne des CSS vars de palette pour les scores victoire et défaite', () => {
    expect(getMatchCardOutcomeStyle('win').scoreColor).toBe('var(--ac-outcome-win)')
    expect(getMatchCardOutcomeStyle('loss').scoreColor).toBe('var(--ac-outcome-loss)')
  })

  it('mappe les badges narratifs supportés avec des CSS vars de palette', () => {
    expect(getMatchNarrativeBadgeMeta('dominant')).toEqual({
      label: 'DOMINATION',
      color: 'var(--ac-narrative-dominant)',
      textColor: 'var(--ac-narrative-dominant-text)',
    })
    expect(getMatchNarrativeBadgeMeta('remontada')).toEqual({
      label: 'REMONTADA',
      color: 'var(--ac-narrative-remontada)',
      textColor: 'var(--ac-narrative-remontada-text)',
    })
    expect(getMatchNarrativeBadgeMeta('unknown')).toBeNull()
  })

  it('sert les libellés anglais quand la locale est en', () => {
    expect(getMatchNarrativeBadgeMeta('sabordage', 'en')?.label).toBe('SCUTTLED')
    expect(getMatchNarrativeBadgeMeta('abnegation', 'en')?.label).toBe('SELFLESS')
    expect(getMatchNarrativeBadgeMeta('remontada', 'en')?.label).toBe('COMEBACK')
    expect(getMatchNarrativeBadgeMeta('contre_remontada', 'en')?.label).toBe('COUNTER-COMEBACK')
    expect(getMatchNarrativeBadgeMeta('debacle', 'en')?.label).toBe('COLLAPSE')
  })

  it('garde les libellés français inchangés, locale explicite ou par défaut', () => {
    expect(getMatchNarrativeBadgeMeta('sabordage', 'fr')?.label).toBe('SABORDAGE')
    expect(getMatchNarrativeBadgeMeta('abnegation', 'fr')?.label).toBe('ABNÉGATION')
    expect(getMatchNarrativeBadgeMeta('debacle', 'fr')?.label).toBe('DÉBÂCLE')
    expect(getMatchNarrativeBadgeMeta('debacle')?.label).toBe('DÉBÂCLE')
  })
})

describe('indicateur d\'analyse — libellés selon l\'état', () => {
  it.each(['badge', 'tooltip_title', 'assisted_frags', 'citations', 'replay'] as const)(
    '%s : « en cours » et « en attente » sont deux clés distinctes',
    (label) => {
      const enCours = analysisPendingLabelKey(label, true)
      const enAttente = analysisPendingLabelKey(label, false)
      expect(enCours).toBe(`common.match_card.analysis_pending.${label}`)
      expect(enAttente).toBe(`common.match_card.analysis_pending.${label}_queued`)
    },
  )

  it('parties en attente : noms courts, ordre servi, causes inconnues ignorées', () => {
    expect(analysisPendingPartKeys({ analysis_pending_parts: ['replay', 'inconnue', 'kill_source'] })).toEqual([
      'common.match_card.analysis_pending.part_replay',
      'common.match_card.analysis_pending.part_kill_source',
    ])
    expect(analysisPendingPartKeys({ analysis_pending_parts: null })).toEqual([])
    expect(analysisPendingPartKeys({})).toEqual([])
    expect(analysisPendingPartKeys(undefined)).toEqual([])
  })
})
