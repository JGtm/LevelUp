/**
 * EquipmentOutcomesCard.test.tsx — « Équipement pris, et ce que j'en ai fait » : une ligne par famille
 * dans l'ordre du Go ; mesurées : servi / gardé / lâché pour moi (comptes dans les segments), barre
 * fine et ligne de parts pour le reste du camp ; non mesurées : « Non mesuré » et les lâchers du joueur.
 */
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type * as SegmentLabelFit from '@/components/charts/segmentLabelFit'

import { soloEmprise } from './usages.fixtures'
import { buildEquipmentRows } from './usages.logic'
import { EquipmentOutcomesCard } from './EquipmentOutcomesCard'
import { USAGES_TEXT } from './usagesText'

/**
 * La mesure au pixel (jsdom : aucune largeur, tout part au repli) est remplaçable par test : `fit.hidden`
 * fixe les segments dont l'étiquette ne tient pas.
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

const NAMES: Record<string, string> = { wall: 'Mur de protection', sensor: 'Capteur de menaces', shroud_screen: 'Écran occultant', grapple: 'Grappin', thruster: 'Propulseur' }

function renderCard() {
  render(<EquipmentOutcomesCard rows={buildEquipmentRows(soloEmprise())} familyLabel={(f) => NAMES[f] ?? f} player="JGtm" ut={USAGES_TEXT.fr.cards} />)
}

const width = (id: string) => parseFloat((screen.getByTestId(id) as HTMLElement).style.width)

describe('EquipmentOutcomesCard', () => {
  it('une ligne par famille, dans l’ordre du Go', () => {
    renderCard()
    const rows = screen.getAllByTestId(/^usages-equip-row-/).map((n) => n.getAttribute('data-testid'))
    expect(rows).toEqual(['usages-equip-row-grapple', 'usages-equip-row-wall', 'usages-equip-row-sensor', 'usages-equip-row-shroud_screen', 'usages-equip-row-thruster'])
  })

  it('mur : 84 objets dont 23 pris sur la carte ; servi 52 et lâché 32 dans leurs segments, aucun gardé', () => {
    renderCard()
    expect(screen.getByTestId('usages-equip-sub-wall').textContent).toBe('84 objets, dont 23 pris sur la carte')
    expect(width('usages-equip-me-wall-used')).toBeCloseTo((52 / 84) * 100)
    expect(screen.getByTestId('usages-equip-me-wall-used').textContent).toBe('52')
    expect(screen.queryByTestId('usages-equip-me-wall-kept')).toBeNull()
    expect(width('usages-equip-me-wall-dropped')).toBeCloseTo((32 / 84) * 100)
  })

  it('repli : un compte qui ne tient pas dans son segment monte au-dessus, seul (S2)', () => {
    fit.hidden = new Set(['usages-equip-me-wall-dropped'])
    renderCard()
    expect(screen.getByTestId('usages-equip-repli-wall').textContent).toBe('32')
    const label = (id: string) => screen.getByTestId(id).querySelector<HTMLElement>('[data-fit-label]')!.style.visibility
    expect(label('usages-equip-me-wall-dropped')).toBe('hidden')
    expect(label('usages-equip-me-wall-used')).toBe('visible')
    expect(screen.queryByTestId('usages-equip-repli-sensor')).toBeNull()
  })

  it('repli : aligné sur le début de SON segment (40 servis · 1 gardé · 2 lâchés : « gardé » à ~93 %)', () => {
    fit.hidden = new Set(['usages-equip-me-wall-kept'])
    const rows = buildEquipmentRows(soloEmprise()).map((r) => (r.family === 'wall' && r.measured ? { ...r, me: [40, 1, 2] as typeof r.me } : r))
    render(<EquipmentOutcomesCard rows={rows} familyLabel={(f) => NAMES[f] ?? f} player="JGtm" ut={USAGES_TEXT.fr.cards} />)
    const repli = screen.getByTestId('usages-equip-repli-wall')
    expect(repli.textContent).toBe('1')
    expect(parseFloat(repli.style.paddingLeft)).toBeCloseTo((40 / 43) * 100)
  })

  it('repli : rien au-dessus quand tous les comptes tiennent', () => {
    fit.hidden = new Set()
    renderCard()
    expect(screen.queryAllByTestId(/^usages-equip-repli-/)).toEqual([])
  })

  it('reste du camp : barre fine et ligne de parts', () => {
    renderCard()
    expect(width('usages-equip-rest-wall-used')).toBeCloseTo((146 / 304) * 100)
    expect(screen.getByTestId('usages-equip-restline-wall').textContent).toBe('reste du camp : 146 servis · 7 gardés · 151 lâchés48 % servis')
  })

  it('non mesurées : le libellé, les lâchers du joueur, « Non mesuré »', () => {
    renderCard()
    expect(screen.getByTestId('usages-equip-sub-grapple').textContent).toBe('84 lâchés')
    expect(screen.getByTestId('usages-equip-row-grapple').textContent).toContain('Non mesuré : ni prise ni usage publiés pour cette famille')
    expect(screen.getByTestId('usages-equip-sub-thruster').textContent).toBe('65 lâchés')
  })

  it('famille sans objet : « 0 objet », piste vide, reste à zéro dit', () => {
    renderCard()
    expect(screen.getByTestId('usages-equip-sub-shroud_screen').textContent).toBe('0 objet')
    expect(screen.queryByTestId('usages-equip-me-shroud_screen-used')).toBeNull()
    expect(screen.getByTestId('usages-equip-restline-shroud_screen').textContent).toBe('reste du camp : 0 objet')
  })

  it('légende', () => {
    renderCard()
    for (const l of ['Servi', 'Gardé sans servir', 'Lâché', 'Barre fine : reste du camp']) expect(screen.getByText(l)).toBeTruthy()
  })
})
