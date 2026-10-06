/**
 * MinePickupsCard.compact.test.tsx — « Mes prises dans mon camp » en vue compacte (tiroir de
 * comparaison de Sessions, maquette `renderMineCompact`) : une barre par RESSOURCE, ma part et celle
 * du reste de mon camp en pourcentage (comptes au survol), bonus perdus des deux camps en pourcentage ;
 * une part qui ne tient pas dans son segment monte sur une ligne de repli alignée sur lui (règle S2).
 */
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type * as SegmentLabelFit from '@/components/charts/segmentLabelFit'

import { soloEmprise } from './usages.fixtures'
import { buildMinePickups, mineByResource } from './usages.logic'
import { MinePickupsCard } from './MinePickupsCard'
import { EMPRISE_TEXT_SOLO, USAGES_TEXT } from './usagesText'

/**
 * La mesure au pixel (jsdom : aucune largeur) est remplaçable par test : `fit.hidden` fixe les
 * segments dont l'étiquette ne tient pas.
 */
const fit = vi.hoisted(() => ({ hidden: null as ReadonlySet<string> | null }))
vi.mock('@/components/charts/segmentLabelFit', async (importOriginal) => {
  const mod = await importOriginal<typeof SegmentLabelFit>()
  return {
    ...mod,
    useSegmentLabelFit: (...args: Parameters<typeof mod.useSegmentLabelFit>) => {
      const measured = mod.useSegmentLabelFit(...args)
      return fit.hidden ?? measured
    },
  }
})
afterEach(() => {
  fit.hidden = null
})

const nameOf = (o: { key: string; label?: string }) => o.label ?? o.key
const mine = buildMinePickups(soloEmprise(), nameOf)!

describe('mineByResource — mes prises, ressource par ressource', () => {
  it('ma part et celle de mon camp, sommées sur les objets de la ressource, dans l’ordre du bilan', () => {
    expect(mineByResource(mine)).toEqual([
      { resource: 'powerup', me: 31, camp: 111 },
      { resource: 'power_weapon', me: 30, camp: 229 },
      { resource: 'rack', me: 2, camp: 5 },
    ])
  })
})

describe('MinePickupsCard — compact', () => {
  function renderCompact() {
    render(
      <MinePickupsCard mine={mine} itemName={nameOf} player="JGtm" t={EMPRISE_TEXT_SOLO.fr} ut={USAGES_TEXT.fr.cards} compact={{ resourceSub: 'prises de mon camp' }} />,
    )
  }

  it('une ligne par ressource, aucune ligne d’objet ni bouton de râteliers', () => {
    renderCompact()
    expect(screen.getAllByText('prises de mon camp')).toHaveLength(3)
    expect(screen.queryByTestId('usages-mine-row-spnkr')).toBeNull()
    expect(screen.queryByTestId('usages-mine-racks-toggle')).toBeNull()
  })

  it('ma part et celle du reste en pourcentage dans les segments', () => {
    fit.hidden = new Set()
    renderCompact()
    expect(screen.getByTestId('usages-mine-me-power_weapon').textContent).toBe('13 %')
    expect(screen.getByTestId('usages-mine-rest-power_weapon').textContent).toBe('87 %')
    expect(screen.getByTestId('usages-mine-me-powerup').textContent).toBe('28 %')
    expect(screen.getByTestId('usages-mine-rest-powerup').textContent).toBe('72 %')
  })

  it('repli : une part qui ne tient pas monte au-dessus, seule, alignée sur SON segment (S2)', () => {
    fit.hidden = new Set(['usages-mine-rest-rack'])
    renderCompact()
    const repli = screen.getByTestId('usages-mine-repli-rack')
    expect(repli.textContent).toBe('60 %')
    expect(parseFloat(repli.style.paddingLeft)).toBeCloseTo((2 / 5) * 100)
    const label = (id: string) => screen.getByTestId(id).querySelector<HTMLElement>('[data-fit-label]')!.style.visibility
    expect(label('usages-mine-rest-rack')).toBe('hidden')
    expect(label('usages-mine-me-rack')).toBe('visible')
    expect(screen.queryByTestId('usages-mine-repli-powerup')).toBeNull()
  })

  it('repli : ma part masquée part du bord gauche', () => {
    fit.hidden = new Set(['usages-mine-me-power_weapon'])
    renderCompact()
    const repli = screen.getByTestId('usages-mine-repli-power_weapon')
    expect(repli.textContent).toBe('13 %')
    expect(parseFloat(repli.style.paddingLeft)).toBe(0)
  })

  it('repli : rien au-dessus quand toutes les parts tiennent', () => {
    fit.hidden = new Set()
    renderCompact()
    expect(screen.queryAllByTestId(/^usages-mine-repli-/)).toEqual([])
  })

  it('bonus perdus des deux camps en pourcentage', () => {
    renderCompact()
    expect(screen.getByTestId('usages-mine-losses').textContent).toBe('Bonus perdus11 %8 %')
  })
})
