import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { WeaponsTable } from './WeaponsTable'
import type { WeaponsTableRow } from '../types'

const baseLabels = {
  weapon: 'Weapon',
  total: 'Total',
  minKills: (n: number) => `Min: ${n}`,
  grenadeMelee: 'Grenade/Melee',
}

describe('WeaponsTable', () => {
  const rows: WeaponsTableRow[] = [
    {
      weapon_id: 100,
      label: 'BR75',
      kills_by_xuid: { main: 10, f1: 6 },
      total: 16,
    },
    {
      weapon_id: 200,
      label: 'Sidekick',
      kills_by_xuid: { main: 3 },
      total: 3,
    },
    {
      weapon_id: 0,
      label: 'Grenade',
      kills_by_xuid: { main: 2 },
      total: 2,
      is_grenade_melee: true,
    },
  ]

  it('rend toutes les armes par défaut', () => {
    render(<WeaponsTable rows={rows} squadOrder={['main', 'f1']} labels={baseLabels} />)
    expect(screen.getByTestId('weapons-table')).toBeTruthy()
    expect(screen.getByText('BR75')).toBeTruthy()
    expect(screen.getByText('Sidekick')).toBeTruthy()
    expect(screen.getByText('Grenade')).toBeTruthy()
  })

  it('le slider filtre les armes en dessous de minKills', () => {
    render(<WeaponsTable rows={rows} squadOrder={['main']} labels={baseLabels} />)
    const slider = screen.getByTestId('weapons-table-slider') as HTMLInputElement
    fireEvent.change(slider, { target: { value: '5' } })
    // Sidekick (3) et Grenade (2) doivent disparaitre, BR75 (16) reste
    expect(screen.getByText('BR75')).toBeTruthy()
    expect(screen.queryByText('Sidekick')).toBeNull()
    expect(screen.queryByText('Grenade')).toBeNull()
  })

  it('affiche le marker grenade/melee', () => {
    render(<WeaponsTable rows={rows} squadOrder={['main']} labels={baseLabels} />)
    expect(screen.getByText('(Grenade/Melee)')).toBeTruthy()
  })

  it('rows vides → composant absent', () => {
    const { container } = render(
      <WeaponsTable rows={[]} squadOrder={['main']} labels={baseLabels} />,
    )
    expect(container.firstChild).toBeNull()
  })
})
