/**
 * Tests MatchKillFeed — kill-feed chronologique (timeline canonique d'events).
 * Le hook useMatchEvents est mocké : on teste le rendu (noms via chokepoint,
 * temps mm:ss, kill environnemental) + l'auto-masquage (erreur / vide).
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MatchKillFeed } from './MatchKillFeed'
import { MATCH_VIEW_TEXT } from './i18n'

vi.mock('./queries', () => ({ useMatchEvents: vi.fn() }))
import { useMatchEvents } from './queries'

const t = MATCH_VIEW_TEXT.fr
const mockUse = useMatchEvents as unknown as ReturnType<typeof vi.fn>

function setResult(result: unknown) {
  mockUse.mockReturnValue(result)
}

function renderFeed(meXUID: string | null = 'x1') {
  return render(
    <MatchKillFeed playerSlug="JGtm" matchId="m1" meXUID={meXUID} t={t} />,
  )
}

beforeEach(() => mockUse.mockReset())

describe('MatchKillFeed', () => {
  it('rend les kills killer → victim avec temps mm:ss', () => {
    setResult({
      data: {
        match_id: 'm1',
        events: [
          {
            type: 'kill',
            time_ms: 65000,
            killer: { Gamertag: 'JGtm', XUID: 'x1' },
            victim: { Gamertag: 'Foe', XUID: 'x2' },
          },
        ],
        limitations: [],
      },
      isPending: false,
      isError: false,
    })
    renderFeed()
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.getByText('Foe')).toBeTruthy()
    expect(screen.getByText('1:05')).toBeTruthy() // 65000 ms
  })

  it('503 capability_not_supported → message « indisponible pour ce titre »', () => {
    setResult({
      data: undefined,
      isPending: false,
      isError: true,
      error: { code: 'capability_not_supported' },
    })
    renderFeed(null)
    expect(screen.getByText(t.killFeedUnsupported)).toBeTruthy()
  })

  it('erreur réseau (autre code) → section masquée silencieusement', () => {
    setResult({
      data: undefined,
      isPending: false,
      isError: true,
      error: { code: 'network_error' },
    })
    const { container } = renderFeed(null)
    expect(container.firstChild).toBeNull()
  })

  it('aucun kill → état vide explicite (titre supporté, match sans event)', () => {
    setResult({ data: { match_id: 'm1', events: [], limitations: [] }, isPending: false, isError: false })
    renderFeed()
    expect(screen.getByText(t.killFeedNoData)).toBeTruthy()
  })

  it('kill environnemental (killer absent) → libellé Environnement', () => {
    setResult({
      data: {
        match_id: 'm1',
        events: [{ type: 'kill', time_ms: 1000, victim: { Gamertag: 'Foe', XUID: 'x2' } }],
        limitations: [],
      },
      isPending: false,
      isError: false,
    })
    renderFeed(null)
    expect(screen.getByText(t.killFeedEnvironment)).toBeTruthy()
  })

  it('affiche la note dégradée quand des limitations sont présentes', () => {
    setResult({
      data: {
        match_id: 'm1',
        events: [
          {
            type: 'kill',
            time_ms: 1000,
            killer: { Gamertag: 'A', XUID: 'x1' },
            victim: { Gamertag: 'B', XUID: 'x2' },
          },
        ],
        limitations: [{ CapabilityKey: 'match.killfeed.per_kill' }],
      },
      isPending: false,
      isError: false,
    })
    renderFeed()
    expect(screen.getByText(t.killFeedDegradedNote)).toBeTruthy()
  })

  it('ne rend rien pendant le chargement', () => {
    setResult({ data: undefined, isPending: true, isError: false })
    const { container } = renderFeed()
    expect(container.firstChild).toBeNull()
  })

  it('bilingue : message « indisponible » rendu en anglais (503)', () => {
    setResult({
      data: undefined,
      isPending: false,
      isError: true,
      error: { code: 'capability_not_supported' },
    })
    const en = MATCH_VIEW_TEXT.en
    render(<MatchKillFeed playerSlug="JGtm" matchId="m1" meXUID={null} t={en} />)
    expect(screen.getByText(en.killFeedUnsupported)).toBeTruthy()
  })
})
