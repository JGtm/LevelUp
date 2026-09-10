/**
 * Tests — ReplayGroundWeaponTip : UNE LIGNE, ET AUCUN SÉPARATEUR ORPHELIN.
 *
 * L'infobulle assemble jusqu'à quatre fragments (arme, origine, munitions, reprise) dont deux
 * sont optionnels. Ce fichier verrouille le seul comportement qui puisse se casser en silence :
 * un fragment absent doit DISPARAÎTRE avec son séparateur, jamais laisser un « · » pendant.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { GroundWeaponHover } from '../layers/useReplayGroundWeapons'
import { ReplayGroundWeaponTip } from './ReplayGroundWeaponTip'

function survol(over: Partial<GroundWeaponHover> = {}): GroundWeaponHover {
  return {
    item: {
      t0: 0, t1: 10, t1max: 10, x: 0, y: 0,
      w: '2b1824d5', origin: 'dropped', dropper: 7, end: 'seen', picker: -1,
    },
    at: { x: 10, y: 10 },
    weaponName: 'BR75',
    originLine: 'lâchée par DinoR00',
    pickupLine: null,
    ammoLine: null,
    ...over,
  }
}

describe('ReplayGroundWeaponTip', () => {
  it('assemble les quatre fragments dans l’ordre du geste', () => {
    render(
      <ReplayGroundWeaponTip
        locale="fr"
        width={400}
        hover={survol({
          ammoLine: '≈ 12 au chargeur et 24 en réserve, lues 9.0 s avant le lâcher',
          pickupLine: 'reprise par SHROOM',
        })}
      />,
    )
    expect(screen.getByRole('tooltip')).toHaveTextContent(
      'BR75 · lâchée par DinoR00 · ≈ 12 au chargeur et 24 en réserve, lues 9.0 s avant le lâcher · reprise par SHROOM',
    )
  })

  it('sans munitions ni reprise : deux fragments, aucun séparateur pendant', () => {
    render(<ReplayGroundWeaponTip locale="fr" width={400} hover={survol()} />)
    expect(screen.getByRole('tooltip').textContent).toBe('BR75 · lâchée par DinoR00')
  })

  it('reprise SANS munitions : la reprise remonte, elle ne saute pas une place', () => {
    render(
      <ReplayGroundWeaponTip
        locale="fr"
        width={400}
        hover={survol({ pickupLine: 'reprise par SHROOM' })}
      />,
    )
    expect(screen.getByRole('tooltip').textContent).toBe(
      'BR75 · lâchée par DinoR00 · reprise par SHROOM',
    )
  })
})
