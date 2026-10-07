/**
 * SessionToolsCard.test.tsx — « Outils de destruction » de Sessions (carte B) : tous les outils et
 * leurs comptes en pleine page ; en vue compacte (maquette `makeTools`, `cp`), les six premiers sans
 * « Non attribué », en part de TOUS mes frags (témoin MESURES §3 s2209 : 65 frags, BR75 22 = 33,8 %).
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import type { SquadBarRows } from '@/features/squad/charts/squadWeaponKillsChart'

import { ME_GAMERTAG, session2209 } from './sessionEmprise.fixtures'
import { SESSION_CARD_TEXT } from './sessionEmpriseText'
import { SessionToolsCard } from './SessionToolsCard'

interface Captured {
  data: SquadBarRows | null
  valueLabel?: string
  shareTotals?: Record<string, number>
  minLabelShare?: number
  soloByClass?: boolean
}

const captured: Captured[] = []

vi.mock('@/features/squad/SquadWeaponKillsChart', () => ({
  SquadWeaponKillsChart: (props: Captured) => {
    captured.push(props)
    return null
  },
}))

function monter(compact: boolean): Captured {
  captured.length = 0
  render(
    <SessionToolsCard
      tools={session2209().entry?.weapon_tools}
      player={ME_GAMERTAG}
      locale="fr"
      texts={compact ? SESSION_CARD_TEXT.fr.compact : SESSION_CARD_TEXT.fr.full}
      compact={compact}
    />,
  )
  return captured[captured.length - 1]
}

describe('SessionToolsCard', () => {
  it('pleine page : tous les outils, « Non attribué » compris, le compte au bout', () => {
    const p = monter(false)
    expect(p.data?.rows).toHaveLength(8)
    expect(p.valueLabel).toBe('count')
    expect(p.shareTotals).toBeUndefined()
  })

  it('les deux vues : barres à la couleur de la classe, sans gamertag (un seul joueur sur la page)', () => {
    for (const compact of [false, true]) expect(monter(compact).soloByClass).toBe(true)
  })

  it('vue compacte : les six premiers, sans « Non attribué », en part de mes 65 frags', () => {
    const p = monter(true)
    expect(p.data?.rows).toHaveLength(6)
    expect(p.data?.rows.map((r) => r.key)).not.toContain('unattributed')
    expect(p.valueLabel).toBe('share')
    expect(p.shareTotals).toEqual({ [ME_GAMERTAG]: 65 })
    // Toutes les parts sont écrites, même les petites (Grenade frag : 3 %).
    expect(p.minLabelShare).toBe(0)
  })
})
