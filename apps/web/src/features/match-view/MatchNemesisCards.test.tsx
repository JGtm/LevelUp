/**
 * Tests — MatchNemesisCards : les deux cartes lisent les éliminations du fil (`highlight_events`)
 * par la fonction commune des antagonistes ; dessin, textes et état vide inchangés.
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MatchHighlightEvent, MatchScoreboardRow } from '@/lib/api/types'
import { MATCH_VIEW_TEXT } from './i18n'
import { MatchNemesisCards } from './MatchNemesisCards'

const t = MATCH_VIEW_TEXT.fr

function row(xuid: string, teamSide: string, gamertag: string): MatchScoreboardRow {
  return { xuid, gamertag, is_me: xuid === 'me', outcome: 'win', team_side: teamSide }
}

const SCOREBOARD = [row('me', 'eagle', 'Moi'), row('e1', 'cobra', 'Rival Un'), row('e2', 'cobra', 'Rival Deux')]

function kill(actor: string, victim: string, tMs: number): MatchHighlightEvent {
  return { event_type: 'kill', actor_xuid: actor, victim_xuid: victim, event_time_ms: tMs }
}

describe('MatchNemesisCards', () => {
  it('nomme la némésis et le souffre-douleur avec leurs comptes', () => {
    const events = [kill('e1', 'me', 1000), kill('e1', 'me', 2000), kill('me', 'e2', 3000)]
    const { container } = render(<MatchNemesisCards events={events} scoreboard={SCOREBOARD} meXUID="me" t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain(t.combatNemesisTitle)
    expect(text).toContain('Rival Un')
    expect(text).toContain(t.combatKilledMeFmt(2))
    expect(text).toContain(t.combatBullyTitle)
    expect(text).toContain('Rival Deux')
    expect(text).toContain(t.combatIKilledFmt(1))
  })

  it('garde les deux cartes titrées et leur état vide quand rien ne se compte', () => {
    const { container } = render(<MatchNemesisCards events={[]} scoreboard={SCOREBOARD} meXUID="me" t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain(t.combatNemesisTitle)
    expect(text).toContain(t.combatBullyTitle)
    expect(text.split(t.combatNoNemesis).length - 1).toBe(2)
    expect(text).not.toContain(t.combatKilledMeFmt(0))
    expect(text).not.toContain(t.combatIKilledFmt(0))
  })

  it('sur une égalité finale, la carte porte le tenant du titre', () => {
    const events = [kill('e1', 'me', 1000), kill('e2', 'me', 2000), kill('e2', 'me', 3000), kill('e1', 'me', 4000)]
    const { container } = render(<MatchNemesisCards events={events} scoreboard={SCOREBOARD} meXUID="me" t={t} />)
    expect(container.textContent).toContain('Rival Deux')
    expect(container.textContent).not.toContain('Rival Un')
  })
})
