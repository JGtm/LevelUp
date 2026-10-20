/**
 * SquadFragSection.test.tsx — « Outils de destruction » sur `weapon_tools` (lot L2 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : légende des joueurs en pied de carte, chaque
 * ligne du serveur nommée dans la langue de l'interface, aucune ligne « Autres ».
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import { SquadFragSection } from './SquadFragSection'
import { getSquadText } from './i18n'
import type { SquadWeaponTools } from '@/lib/api/types'

type Option = { yAxis?: { data: string[]; axisLabel: { formatter: (v: string, i: number) => string } } }
const options: Option[] = []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: Option }) => {
    options.push(props.option)
    return <div data-testid="echarts-mock" />
  },
}))

const T = getSquadText('fr')

const TOOLS: SquadWeaponTools = {
  players: ['JGtm', 'Chocoboflor'],
  lines: [
    { kind: 'weapon', weapon_key: 'hinf_br75', label: 'BR75', label_en: 'BR75', class: 'shoulder', kills_by_player: { JGtm: 22, Chocoboflor: 22 }, total_squad: 44 },
    { kind: 'melee', class: 'melee', kills_by_player: { JGtm: 6, Chocoboflor: 13 }, total_squad: 19 },
    { kind: 'explosive_object', class: 'unattributed', kills_by_player: { JGtm: 2 }, total_squad: 2 },
    { kind: 'weapon', weapon_key: 'hinf_mutilator', label: 'Mutilateur', label_en: 'Mutilator', class: 'shoulder', kills_by_player: { JGtm: 1 }, total_squad: 1 },
    { kind: 'unattributed', class: 'unattributed', kills_by_player: { Chocoboflor: 1 }, total_squad: 1 },
  ],
}

describe('SquadFragSection — Outils de destruction', () => {
  it('légende des joueurs en pied de carte, lignes nommées, reliquat en bas', async () => {
    options.length = 0
    render(
      <SquadFragSection
        fragClassesByPlayer={{}}
        weaponTools={TOOLS}
        weaponAccuracy={null}
        playerColors={{ JGtm: 'var(--p1)', Chocoboflor: 'var(--p2)' }}
        playerOrder={['JGtm', 'Chocoboflor']}
        locale="fr"
        t={T}
      />,
    )
    const legends = screen.getAllByTestId('chart-legend')
    const players = legends.find((l) => within(l).queryByText('JGtm'))!
    expect(within(players).getAllByRole('listitem').map((li) => li.textContent)).toEqual(['JGtm', 'Chocoboflor'])

    await waitFor(() => expect(options.some((o) => o.yAxis)).toBe(true))
    const opt = options.filter((o) => o.yAxis).pop()!
    const names = opt.yAxis!.data.map((v, i) => opt.yAxis!.axisLabel.formatter(v, i))
    // Première catégorie EN BAS : le reliquat, puis … jusqu'au BR75 en haut.
    expect(names.map((n) => n.replace(/\{[a-z_]+\|\}|\{name\||\}/g, ''))).toEqual([
      'Non attribué',
      'Mutilateur',
      T.weaponKills.explosiveObject,
      'Mêlée',
      'BR75',
    ])
    expect(names.some((n) => /autres/i.test(n))).toBe(false)
  })
})
