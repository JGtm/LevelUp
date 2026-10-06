/**
 * PisteCampsForm.test.tsx — la piste camp contre camp (lot L5.3) : adversaire en `team-enemy`
 * PLEIN (plus de hachure), « compte · part » dans chaque segment (le nôtre à gauche, le sien à
 * droite), repli S3 au-dessus seulement pour la valeur qui ne tient pas (mesure au pixel), trait
 * 50 % pointillé `warning`, axe 0-100 %.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { PisteCampsForm, type PisteCampsRow } from './PisteCampsForm'
import { EMPRISE_TEXT } from './empriseStrings'

const T = EMPRISE_TEXT.fr

const ROW: PisteCampsRow = {
  key: 'power_weapon',
  label: 'Armes spéciales',
  sublabel: 'prises sur les socles',
  dot: 'var(--ac-resource-power-weapon)',
  us: 23,
  them: 29,
  usTip: 'Notre camp',
  themTip: 'Adversaire',
}

/** Largeurs simulées : notre segment large, celui de l'adversaire étroit, étiquettes de 60 px. */
function mockWidths(us: number, them: number) {
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    const key = this.dataset.fitKey
    const width = key?.endsWith('|us') ? us : key?.endsWith('|them') ? them : this.hasAttribute('data-fit-label') ? 60 : 0
    return { width, height: 22, top: 0, left: 0, right: width, bottom: 22, x: 0, y: 0, toJSON: () => ({}) } as DOMRect
  })
}

afterEach(() => {
  vi.restoreAllMocks()
})

function mount(rows: PisteCampsRow[] = [ROW]) {
  return render(<PisteCampsForm rows={rows} pctFmt={T.pctFmt} axisMaxLabel={T.pctIntFmt(100)} />)
}

describe('PisteCampsForm', () => {
  it('les deux valeurs tiennent : écrites dans leur segment, aucun repli', () => {
    mockWidths(300, 300)
    mount()
    const labels = document.querySelectorAll<HTMLElement>('[data-fit-label]')
    expect(labels[0].textContent).toBe('23 · 44,2 %')
    expect(labels[1].textContent).toBe('55,8 % · 29')
    expect([...labels].map((l) => l.style.visibility)).toEqual(['visible', 'visible'])
    expect(screen.queryByTestId('piste-camps-repli-power_weapon')).toBeNull()
  })

  it('la valeur qui ne tient pas (6 px de marge) passe au repli, de son côté, pastille devant', () => {
    mockWidths(300, 70)
    mount()
    const labels = document.querySelectorAll<HTMLElement>('[data-fit-label]')
    expect(labels[0].style.visibility).toBe('visible')
    expect(labels[1].style.visibility).toBe('hidden')
    const repli = screen.getByTestId('piste-camps-repli-power_weapon')
    expect(repli.textContent).toBe('29 · 55,8 %')
    // Notre camp à gauche (vide ici), l'adversaire à droite : deux emplacements.
    expect(repli.children).toHaveLength(2)
    expect(repli.children[0].textContent).toBe('')
  })

  it('adversaire en aplat `team-enemy` (plus de hachure), notre camp en `team-ally`, trait 50 % pointillé `warning`', () => {
    mockWidths(300, 300)
    const { container } = mount()
    const segs = container.querySelectorAll<HTMLElement>('[data-fit-key]')
    expect(segs[0].style.backgroundColor).toBe('var(--ac-team-ally)')
    expect(segs[1].style.backgroundColor).toBe('var(--ac-team-enemy)')
    expect(segs[1].style.backgroundImage).toBe('')
    expect(segs[0].style.width).toBe(`${(23 / 52) * 100}%`)
    const parity = container.querySelector<HTMLElement>('.border-dashed')!
    expect(parity.style.borderColor).toBe('var(--ac-warning)')
    expect(parity.className).toContain('left-1/2')
  })

  it('un camp à zéro : un seul segment plein, sa valeur au repli', () => {
    mockWidths(300, 300)
    mount([{ ...ROW, key: 'powerup', us: 4, them: 0 }])
    expect(document.querySelectorAll('[data-fit-key]')).toHaveLength(1)
    expect(screen.getByTestId('piste-camps-repli-powerup').textContent).toBe('0 · 0 %')
  })

  it('parts seules (vue compacte de Sessions) : la part dans le segment et au repli, le compte en infobulle seulement', () => {
    mockWidths(300, 70)
    render(<PisteCampsForm rows={[ROW]} pctFmt={T.pctIntFmt} axisMaxLabel={T.pctIntFmt(100)} pctOnly />)
    const labels = document.querySelectorAll<HTMLElement>('[data-fit-label]')
    expect(labels[0].textContent).toBe('44 %')
    expect(screen.getByTestId('piste-camps-repli-power_weapon').textContent).toBe('56 %')
  })

  it('nom précédé de la pastille de ressource (S8), sous-libellé, axe 0-100 %', () => {
    mockWidths(300, 300)
    mount()
    const row = screen.getByTestId('piste-camps-row-power_weapon')
    expect(row.textContent).toContain('Armes spéciales')
    expect(row.textContent).toContain('prises sur les socles')
    expect(row.querySelector<HTMLElement>('[aria-hidden]')!.style.backgroundColor).toBe('var(--ac-resource-power-weapon)')
    expect(screen.getByText('100 %')).toBeInTheDocument()
    expect(screen.getByText('50')).toBeInTheDocument()
  })
})
