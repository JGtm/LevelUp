/**
 * Tests — ReplayKillFeed (kill feed de la page de rejeu).
 *
 * Ce qu'ils protègent, dans l'ordre :
 *  1. LA SYNCHRONISATION. Un kill ne s'affiche pas avant son instant dans le rejeu, et il
 *     sort du feed quand il a dépassé la fenêtre. C'est toute la raison d'être du composant.
 *  2. LE RECALAGE DES DEUX HORLOGES. Les events arrivent sur l'horloge du gameplay, le rejeu
 *     tourne sur celle du film : sans `t0_ms`, le feed a ~18 s de retard.
 *  3. AUCUNE ICÔNE FAUSSE : une arme non résolue rend un repère neutre, jamais l'icône d'un
 *     autre kill (même règle que le kill feed de la Match View).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const T0 = 18_465

function kill(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000,
    xuid: 'me',
    ally: true,
    teamID: 0,
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    ...over,
  }
}

const META = new Map([
  ['me', { gamertag: 'JGtm', ally: true }],
  ['foe', { gamertag: 'Cobra01', ally: false }],
])

const SCOREBOARD: MatchScoreboardRow[] = []

function renderFeed(
  kills: KillEvent[],
  nowMs: number,
  t0Ms = T0,
  victims?: { killer_xuid: string; victim_gamertag: string; time_ms?: number | null }[],
) {
  return render(
    <ReplayKillFeed
      kills={kills}
      victims={victims}
      t0Ms={t0Ms}
      nowMs={nowMs}
      scoreboard={SCOREBOARD}
      xuidMeta={META}
      locale="fr"
    />,
  )
}

describe('ReplayKillFeed — synchronisation', () => {
  const kills = [kill({ tMs: 16_841, xuid: 'me' }), kill({ tMs: 40_000, xuid: 'foe' })]

  it("n'affiche pas un kill qui n'a pas encore eu lieu", () => {
    // 16 841 ms d'horloge gameplay = 35 306 ms d'horloge rejeu. À 30 s de rejeu, rien.
    renderFeed(kills, 30_000)
    expect(screen.queryByText('JGtm')).toBeNull()
    expect(screen.getByText(/Rien à cet instant/)).toBeTruthy()
  })

  it("l'affiche dès que le rejeu atteint son instant", () => {
    renderFeed(kills, 35_400)
    expect(screen.getByText('JGtm')).toBeTruthy()
    // Et pas encore l'autre, 40 s plus loin sur l'horloge gameplay.
    expect(screen.queryByText('Cobra01')).toBeNull()
  })

  it('le laisse sortir du feed passé la fenêtre', () => {
    renderFeed(kills, 45_000)
    expect(screen.queryByText('JGtm')).toBeNull()
  })

  it('SANS RECALAGE, le même kill serait affiché ~18 s trop tôt — le témoin le montre', () => {
    // Le contre-test qui départage : avec t0 = 0, le kill apparaît à 16,8 s de rejeu, un
    // instant où il n'a pas eu lieu. C'est exactement le défaut que `t0_ms` corrige.
    renderFeed(kills, 17_000, 0)
    expect(screen.getByText('JGtm')).toBeTruthy()
  })
})

describe('ReplayKillFeed — arme du kill', () => {
  it("sert l'icône quand le backend a résolu l'arme", () => {
    renderFeed(
      [
        kill({
          tMs: 1_000,
          weaponImageUrl: '/static/weapons/br75.png',
          weaponLabel: 'BR75',
          weaponTinted: true,
        }),
      ],
      20_000,
    )
    expect(screen.getByRole('img', { name: 'BR75' })).toBeTruthy()
  })

  it("ne sert AUCUNE icône quand l'arme n'est pas résolue", () => {
    const { container } = renderFeed([kill({ tMs: 1_000 })], 20_000)
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.queryAllByRole('img')).toEqual([])
    // Le repère neutre, lui, est bien là : la ligne existe, seule l'arme manque.
    expect(container.querySelector('.rounded-full')).toBeTruthy()
  })

  it("n'écrit aucun hex de couleur (règle color-tokens)", () => {
    const { container } = renderFeed([kill({ tMs: 1_000 })], 20_000)
    expect(container.innerHTML).not.toMatch(/#[0-9a-fA-F]{6}/)
  })
})

describe('ReplayKillFeed — la victime, jointe par (tueur, instant)', () => {
  it('nomme la victime quand la paire existe', () => {
    renderFeed([kill({ tMs: 1_000 })], 20_000, T0, [
      { killer_xuid: 'me', victim_gamertag: 'Cobra01', time_ms: 1_000 },
    ])
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.getByText('Cobra01')).toBeTruthy()
  })

  it('sans paire, la ligne vit sans victime — rien n’est inventé', () => {
    renderFeed([kill({ tMs: 1_000 })], 20_000, T0, [])
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.queryByText('→')).toBeNull()
  })
})

describe('ReplayKillFeed — les TROIS états de l’assistance, jamais confondus', () => {
  it('assistant NOMMÉ : le nom, sa part, la part du tueur — et le fond affirme la contribution', () => {
    const { container } = renderFeed(
      [
        kill({
          tMs: 1_000,
          assistState: 'named',
          assistGamertag: 'Aidant77',
          assistTeamID: 0,
          killerDamagePct: 63,
          assistDamagePct: 37,
        }),
      ],
      20_000,
    )
    expect(screen.getByText('Aidant77')).toBeTruthy()
    expect(screen.getByText('37 %')).toBeTruthy()
    expect(screen.getByText(/tueur 63 %/)).toBeTruthy()
    expect(container.querySelector('li')?.getAttribute('style')).toContain('color-mix')
  })

  it('« aucun » MESURÉ : rien d’affiché — l’information vit en infobulle, distincte d’« inconnu »', () => {
    const { container } = renderFeed(
      [kill({ tMs: 1_000, assistState: 'none', killerDamagePct: 100 })],
      20_000,
    )
    expect(screen.queryByText(/assistant inconnu/)).toBeNull()
    const line = container.querySelector('li')
    expect(line?.getAttribute('title')).toMatch(/MESURÉ/)
    expect(line?.getAttribute('style') ?? '').not.toContain('color-mix')
  })

  it('INCONNU : la lacune se signale — « assistant inconnu », sans fond', () => {
    renderFeed([kill({ tMs: 1_000, assistState: '' })], 20_000)
    expect(screen.getByText('assistant inconnu')).toBeTruthy()
  })
})
