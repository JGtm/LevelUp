/**
 * matchMedals.test.ts — la lecture « médailles du match » depuis le document de rejeu, et le fil
 * qu'elle alimente.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import type { ReplayMatchContext } from '../../../lib/replay/replayMatchContext'
import { testReplayDoc } from '../test/testDoc'
import { readMatchMedals } from './matchMedals'
import { buildReplayModel } from './replayModel'

const MEDALS: NonNullable<ReplayDocument['medals']> = [
  { t: 20, ms: 3_000, xuid: 'b', id: 7, name: 'Killjoy' },
  { t: 10, ms: 1_000, xuid: 'a', id: 7, name: 'Killjoy' },
  { t: 12, ms: 1_500, xuid: 'a', id: 99, name: 'Medaille Inconnue' },
  { t: 14, ms: 2_000, xuid: 'a', id: 5 },
  { t: 16, ms: 2_500, xuid: '', id: 7, name: 'Killjoy' },
]

const LABELS: NonNullable<ReplayDocument['medalLabels']> = {
  Killjoy: {
    en: 'Killjoy',
    fr: 'Rabat-joie',
    description: { en: 'End an enemy spree', fr: 'Mettre fin à une série' },
    img: '/static/medals/halo_infinite/101.png',
    difficulty: 'Heroic',
  },
}

describe('readMatchMedals', () => {
  it('lit libellé, description et image dans la langue demandée, dans l’ordre du temps', () => {
    const doc = testReplayDoc({ medals: MEDALS, medalLabels: LABELS })
    const fr = readMatchMedals(doc, 'fr')
    expect(fr.map((m) => [m.tMs, m.xuid, m.name])).toEqual([
      [1_000, 'a', 'Killjoy'],
      [1_500, 'a', 'Medaille Inconnue'],
      [3_000, 'b', 'Killjoy'],
    ])
    expect(fr[0]).toMatchObject({
      label: 'Rabat-joie',
      description: 'Mettre fin à une série',
      imageUrl: '/static/medals/halo_infinite/101.png',
    })
    expect(readMatchMedals(doc, 'en')[0].label).toBe('Killjoy')
  })

  it('une médaille hors table garde son nom, sans libellé ni image inventés', () => {
    const [, inconnue] = readMatchMedals(testReplayDoc({ medals: MEDALS, medalLabels: LABELS }), 'fr')
    expect(inconnue).toMatchObject({ name: 'Medaille Inconnue', label: '', description: '', imageUrl: '' })
  })

  it('un document sans `medals` n’affiche aucune médaille', () => {
    expect(readMatchMedals(testReplayDoc({ medalLabels: LABELS }), 'fr')).toEqual([])
    expect(readMatchMedals(null, 'fr')).toEqual([])
    expect(readMatchMedals(undefined)).toEqual([])
  })

  it('sans table de libellés, les médailles gardent leur nom', () => {
    const out = readMatchMedals(testReplayDoc({ medals: MEDALS }), 'fr')
    expect(out).toHaveLength(3)
    expect(out.every((m) => m.label === '' && m.imageUrl === '')).toBe(true)
  })
})

describe('le fil du rejeu lit les médailles du document', () => {
  // Un événement `medal` du contexte du match qui n'est PAS dans le document ne doit jamais apparaître.
  const contexte = {
    header: { t0_ms: 0, playable_duration_seconds: 200, start_time: '2026-09-01T12:00:00Z' },
    scoreboard: [{ xuid: 'a', gamertag: 'Aa', team_side: 't0', is_me: true }],
    highlight_events: [
      { event_type: 'medal', actor_xuid: 'a', event_time_ms: 40_000, medal_name: 'Ancienne Source', medal_label: 'X' },
    ],
  } as unknown as ReplayMatchContext

  const medailles = (feed: ReturnType<typeof buildReplayModel>['feed']) =>
    feed.flatMap((e) => [e.medal, ...(e.kill?.medals ?? [])]).flatMap((m) => (m ? [m] : []))

  it('alimente le fil depuis `medals` et ignore les événements `medal` du contexte', () => {
    const doc = testReplayDoc({ medals: MEDALS, medalLabels: LABELS })
    const { feed } = buildReplayModel(doc, contexte, { locale: 'fr' })
    const toutes = medailles(feed)
    expect(toutes.map((m) => m.name).sort()).toEqual(['Killjoy', 'Killjoy', 'Medaille Inconnue'])
    expect(toutes.map((m) => m.label)).toContain('Rabat-joie')
  })

  it('un document sans `medals` : aucune médaille au fil, aucun repli sur le contexte', () => {
    const { feed } = buildReplayModel(testReplayDoc(), contexte, { locale: 'fr' })
    expect(medailles(feed)).toEqual([])
  })
})
