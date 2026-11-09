/**
 * Tests — ReplayKillFeed, la ZONE DU TUEUR (gouttière après l'horloge des lignes de frag).
 *
 * CE QU'ILS PROTÈGENT : la zone s'écrit dans la langue de la page, tronquée, avec son nom complet
 * en infobulle ; la gouttière n'existe que sur les lignes de FRAG et que si la carte a des zones ;
 * une zone inconnue laisse la gouttière VIDE (colonnes alignées, aucun texte de repli) ; sans
 * `zoneOf`, la ligne est celle d'avant.
 */
import { render, renderHook, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { useKillerZones } from '../hooks/useKillerZones'
import { buildFeedEntries, type MedalEvent } from '../model/killFeedLogic'
import { testReplayDoc } from '../test/testDoc'
import { normalizeCalloutZones } from '@/lib/replay/calloutsPaint'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const META = new Map([
  ['me', { gamertag: 'JGtm', ally: true }],
  ['foe', { gamertag: 'Cobra01', ally: false }],
])

const ALLEGEANCE = filmAllegianceOf(
  testReplayDoc({
    roster: [
      { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
      { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
    ],
  }),
  [],
  'me',
)

function kill(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000,
    xuid: 'me',
    ally: true,
    teamID: 0,
    weaponKey: '',
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: '',
    victimGamertag: '',
    ...over,
  }
}

const MEDAL: MedalEvent = {
  tMs: 5_000,
  xuid: 'me',
  gamertag: 'JGtm',
  name: 'No Scope',
  label: 'Sans lunette',
  description: '',
  imageUrl: '/m.png',
}

/** Un fil d'un frag (clé connue) et d'une médaille seule, lu à 20 s : les deux lignes sont visibles. */
const ENTRIES = buildFeedEntries([kill({ tMs: 1_000 })], [MEDAL], 0, null)
const KILL_KEY = ENTRIES.find((e) => e.kill)!.key

function renderFeed(locale: 'fr' | 'en', zoneOf?: (key: string) => string) {
  return render(
    <ReplayKillFeed
      entries={ENTRIES}
      nowMs={20_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={ALLEGEANCE}
      locale={locale}
      zoneOf={zoneOf}
    />,
  )
}

const gutters = (container: HTMLElement) => container.querySelectorAll('span.w-\\[5\\.5rem\\]')

describe('ReplayKillFeed — zone du tueur', () => {
  it('fr : le nom de la zone s’écrit après l’horloge, son nom complet en infobulle', () => {
    const { container } = renderFeed('fr', (key) => (key === KILL_KEY ? 'Fer à cheval' : ''))
    const zone = screen.getByText('Fer à cheval')
    expect(zone.getAttribute('title')).toBe('Zone du tueur : Fer à cheval')
    expect(zone.className).toContain('truncate')
    expect(zone.className).toContain('text-muted-foreground')
    const row = zone.closest('li') as HTMLElement
    expect(Array.from(row.children).indexOf(zone)).toBe(1)
    expect(gutters(container)).toHaveLength(1)
  })

  it('en : même gouttière, libellé de l’infobulle dans la langue de la page', () => {
    renderFeed('en', (key) => (key === KILL_KEY ? 'Horseshoe' : ''))
    expect(screen.getByText('Horseshoe').getAttribute('title')).toBe("Killer's zone: Horseshoe")
  })

  it('zone INCONNUE : la gouttière reste, vide — aucun texte, aucune infobulle', () => {
    const { container } = renderFeed('fr', () => '')
    const g = gutters(container)
    expect(g).toHaveLength(1)
    expect(g[0].textContent).toBe('')
    expect(g[0].getAttribute('title')).toBeNull()
  })

  it('la gouttière ne concerne QUE les lignes de frag : la médaille seule n’en reçoit pas', () => {
    const { container } = renderFeed('fr', () => 'Zone')
    expect(container.querySelectorAll('li')).toHaveLength(2)
    expect(gutters(container)).toHaveLength(1)
  })

  it('SANS zoneOf (carte sans zone) : aucune gouttière réservée', () => {
    const { container } = renderFeed('fr')
    expect(gutters(container)).toHaveLength(0)
  })
})

describe('useKillerZones', () => {
  const doc = testReplayDoc({
    tracks: [
      { slot: 1, team: 0, xuid: 'me', points: [{ t: 0, x: 5, y: 5, z: 1 }, { t: 100, x: 5, y: 5, z: 1 }], startFrame: 0, endFrame: 100 },
    ] as never,
  })
  const zones = normalizeCalloutZones([
    {
      name: 'horses', en: 'Horseshoe', fr: 'Fer à cheval', x: 5, y: 5, z: 1, volume_index: 1,
      polygon: [[0, 0], [10, 0], [10, 10], [0, 10]], z_bottom: 0, z_top: 3,
    },
  ])

  it('carte sans zone : undefined (pas de gouttière)', () => {
    const { result } = renderHook(() => useKillerZones(ENTRIES, doc, [], 'fr'))
    expect(result.current).toBeUndefined()
  })

  it('nomme la zone du tueur dans la langue de la page, vide pour une ligne sans zone', () => {
    const fr = renderHook(() => useKillerZones(ENTRIES, doc, zones, 'fr'))
    expect(fr.result.current?.(KILL_KEY)).toBe('Fer à cheval')
    expect(fr.result.current?.('inconnue')).toBe('')
    const en = renderHook(() => useKillerZones(ENTRIES, doc, zones, 'en'))
    expect(en.result.current?.(KILL_KEY)).toBe('Horseshoe')
  })

  it('un tueur BOT a sa zone quand la feuille le relie à sa piste, pas sans elle', () => {
    const botDoc = testReplayDoc({
      tracks: [
        { slot: 4, team: 1, bot: 'Spark [bot]', points: [{ t: 0, x: 5, y: 5, z: 1 }, { t: 100, x: 5, y: 5, z: 1 }], startFrame: 0, endFrame: 100 },
      ] as never,
    })
    const botEntries = buildFeedEntries([kill({ tMs: 1_000, xuid: 'bid(1.0)' })], [], 0, null)
    const key = botEntries[0].key
    const row = { xuid: 'bid(1.0)', gamertag: 'Spark', is_bot: true } as unknown as MatchScoreboardRow
    const avec = renderHook(() => useKillerZones(botEntries, botDoc, zones, 'fr', [row]))
    expect(avec.result.current?.(key)).toBe('Fer à cheval')
    const sans = renderHook(() => useKillerZones(botEntries, botDoc, zones, 'fr'))
    expect(sans.result.current?.(key)).toBe('')
  })

  it('document absent : gouttière vide, jamais une erreur', () => {
    const { result } = renderHook(() => useKillerZones(ENTRIES, undefined, zones, 'fr'))
    expect(result.current?.(KILL_KEY)).toBe('')
  })
})
