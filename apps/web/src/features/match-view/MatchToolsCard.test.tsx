/**
 * Tests — MatchToolsCard (« Outils de destruction », carte B) : les outils du joueur de la page sur le
 * témoin du 22/09 (MK50 Sidekick 7, Mêlée 2, Grenade frag 1, VK78 Commando 1), nommés par la source
 * unique des natures, sous le titre et l'aide de la Vue match. Le graphe est mocké : il expose ce qu'il
 * reçoit.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SquadWeaponTools } from '@/lib/api/types'

import { MATCH_EMPRISE_TEXT } from './matchEmpriseText'
import { MatchToolsCard } from './MatchToolsCard'

type Recu = { title: React.ReactNode; data: { players: string[]; rows: { label: string; total: number }[] } | null; soloByClass?: boolean }
const recu = vi.hoisted(() => ({ props: null as null | Recu }))

vi.mock('@/features/squad/SquadWeaponKillsChart', () => ({
  SquadWeaponKillsChart: (p: Recu) => {
    recu.props = p
    return <div data-testid="outils">{p.title}</div>
  },
}))

const line = (kind: string, cls: string, n: number, label?: string) => ({ kind, class: cls, label, kills_by_player: { JGtm: n }, total_squad: n })

const TEMOIN: SquadWeaponTools = {
  players: ['JGtm'],
  lines: [line('weapon', 'sidearm', 7, 'MK50 Sidekick'), line('melee', 'melee', 2), line('weapon', 'grenade', 1, 'Grenade frag'), line('weapon', 'shoulder', 1, 'VK78 Commando')],
}

describe('MatchToolsCard', () => {
  it('22/09 : une ligne par outil, mêlée nommée par le manifeste, ordre du serveur renversé pour le graphe', () => {
    render(<MatchToolsCard tools={TEMOIN} locale="fr" />)
    expect(recu.props?.data?.players).toEqual(['JGtm'])
    expect(recu.props?.data?.rows.map((r) => `${r.label} ${r.total}`)).toEqual(['VK78 Commando 1', 'Grenade frag 1', 'Mêlée 2', 'MK50 Sidekick 7'])
  })

  it('barres à la couleur de la classe de l’outil, sans gamertag (un seul joueur sur la page)', () => {
    render(<MatchToolsCard tools={TEMOIN} locale="fr" />)
    expect(recu.props?.soloByClass).toBe(true)
  })

  it('titre de l’Escouade, aide portée « le match »', () => {
    render(<MatchToolsCard tools={TEMOIN} locale="fr" />)
    expect(screen.getByTestId('outils').textContent).toContain(MATCH_EMPRISE_TEXT.fr.squad.weaponKills.title)
  })
})
