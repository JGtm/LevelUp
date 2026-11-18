/**
 * ReplayAbilityCell.test.tsx — LES CHARGES dans l'infobulle de la vignette de capacité.
 *
 * La cellule n'écrit rien : aucune largeur ajoutée à la rangée, aucun chiffre en surimpression,
 * le compte et le « plein » se lisent dans l'infobulle de la vignette.
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { GABARIT_COMPACT } from '../model/cardGabarit'
import { testReplayDoc } from '../test/testDoc'
import { ReplayAbilityCell } from './ReplayAbilityCell'

const SLOT = 512

/** Une vie de 1 000 frames, un grappin lu à la frame 0, une lecture de charges à la frame 30. */
function document() {
  return testReplayDoc({
    frameCount: 1_000,
    tracks: [
      { slot: SLOT, team: -1, xuid: 'A', points: [{ t: 0, x: 1, y: 1 }], startFrame: 0, endFrame: 999 },
    ],
    abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot', img: '/static/grappin.png', tinted: true } },
    abilities: [{ t: 0, slot: SLOT, r: 20, src: 'i48' }],
    abilityCharges: [{ t: 30, slot: SLOT, family: 'grapple', charges: 2 }],
  })
}

function rendre(frame: number, readingFull = 20) {
  const vue = render(
    <ReplayAbilityCell
      doc={document()}
      slot={SLOT}
      frame={frame}
      readingFull={readingFull}
      locale="fr"
      gabarit={GABARIT_COMPACT}
    />,
  )
  const cellule = vue.container.firstElementChild as HTMLElement
  const chiffre = cellule.querySelector('[aria-hidden="true"]') as HTMLElement | null
  const vignette = cellule.querySelector('[title]') as HTMLElement
  return { vue, cellule, chiffre, vignette }
}

describe('ReplayAbilityCell — les charges dans l’infobulle de la vignette', () => {
  it('une seule cellule, à la largeur de la vignette, sans chiffre : tout passe par l’infobulle', () => {
    const { vue, cellule, chiffre, vignette } = rendre(42)
    expect(vue.container.children).toHaveLength(1)
    expect(cellule.style.width).toBe(`${GABARIT_COMPACT.iconAbilityPx}px`)
    expect(chiffre).toBeNull()
    expect(cellule.textContent).toBe('')
    expect(vignette.title).toContain('Charges restantes : 2')
    expect(vignette.title).toContain('Charges lues')
  })

  it('« plein » ne s’écrit pas à l’écran : il est dans l’infobulle de la vignette', () => {
    // Avant la première lecture de charges (frame 30) : le canal n’a rien transmis, donc « plein ».
    const { vue, cellule, chiffre, vignette } = rendre(10)
    expect(chiffre).toBeNull()
    expect(cellule.textContent).toBe('')
    expect(vue.queryByText(/plein/i)).toBeNull()
    expect(vignette.title).toContain('Charges pleines')
  })
})
