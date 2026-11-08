/**
 * ReplayAbilityCell.test.tsx — le COMPTE DE CHARGES EN SURIMPRESSION sur la vignette de capacité.
 *
 * Le compte posé à côté de la vignette tombait hors de la rangée (cellules fixes + marque =
 * plus que la largeur utile de la fiche) et était rogné : le chiffre vit désormais DANS la boîte
 * de la vignette. Ces tests épinglent ce qui rend cela vrai — aucune largeur ajoutée, un chiffre
 * lisible et sans couleur en dur, « plein » dans l'infobulle seule, l'estompage propre au
 * chiffre — et que la tuile compacte garde son comportement (infobulle seule).
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { GABARIT_COMPACT, GABARIT_NORMAL } from '../model/cardGabarit'
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

function rendre(frame: number, gabarit = GABARIT_NORMAL, readingFull = 20) {
  const vue = render(
    <ReplayAbilityCell
      doc={document()}
      slot={SLOT}
      frame={frame}
      readingFull={readingFull}
      locale="fr"
      gabarit={gabarit}
    />,
  )
  const cellule = vue.container.firstElementChild as HTMLElement
  const chiffre = cellule.querySelector('[aria-hidden="true"]') as HTMLElement | null
  const vignette = cellule.querySelector('[title]') as HTMLElement
  return { vue, cellule, chiffre, vignette }
}

describe('ReplayAbilityCell — le chiffre des charges sur la vignette', () => {
  it('le chiffre est dans la boîte de la vignette et n’ajoute AUCUNE largeur à la rangée', () => {
    const { vue, cellule, chiffre } = rendre(30)
    // Une seule cellule, à la largeur de la vignette : aucun frère après elle.
    expect(vue.container.children).toHaveLength(1)
    expect(cellule.style.width).toBe(`${GABARIT_NORMAL.iconAbilityPx}px`)
    expect(cellule.className).toContain('relative')
    expect(chiffre?.textContent).toBe('2')
    expect(chiffre?.parentElement).toBe(cellule)
    // Hors du flux, ancré au coin bas droit de la vignette, sans prise au pointeur.
    expect(chiffre?.className).toContain('absolute')
    expect(chiffre?.className).toContain('bottom-0')
    expect(chiffre?.className).toContain('right-0')
    expect(chiffre?.className).toContain('pointer-events-none')
  })

  it('lisible en thème clair comme sombre : jetons sémantiques, aucune couleur en dur', () => {
    const { chiffre } = rendre(30)
    expect(chiffre?.className).toContain('bg-background')
    expect(chiffre?.className).toContain('text-foreground')
    expect(chiffre?.className).not.toMatch(/#[0-9a-f]{3,6}|(text|bg|border)-(red|green|blue|yellow|amber|rose|slate|gray|zinc|neutral|white|black)\b/i)
    expect(chiffre?.getAttribute('style') ?? '').not.toMatch(/#|rgb/i)
  })

  it('l’infobulle de la vignette porte le compte et l’âge de sa lecture', () => {
    const { vignette } = rendre(42)
    expect(vignette.title).toContain('Charges restantes : 2')
    expect(vignette.title).toContain('Charges lues')
  })

  it('le chiffre pâlit avec l’âge de SA lecture, pas avec celui de la capacité', () => {
    // Capacité lue à 0 (âgée de 30 images, au-delà de 20 : pâlie) ; charges lues à 30 (âge 0).
    const { chiffre, vignette } = rendre(30, GABARIT_NORMAL, 20)
    expect(chiffre?.style.opacity).toBe('1')
    expect(vignette.style.opacity).toBe('0.5')
    // 600 images plus tard, le chiffre a pâli à son tour.
    const tard = rendre(630, GABARIT_NORMAL, 20)
    expect(tard.chiffre?.style.opacity).toBe('0.5')
  })

  it('« plein » ne s’écrit plus à l’écran : il est dans l’infobulle de la vignette', () => {
    // Avant la première lecture de charges (frame 30) : le canal n’a rien transmis, donc « plein ».
    const { vue, cellule, chiffre, vignette } = rendre(10)
    expect(chiffre).toBeNull()
    expect(cellule.textContent).toBe('')
    expect(vue.queryByText(/plein/i)).toBeNull()
    expect(vignette.title).toContain('Charges pleines')
  })

  it('la tuile compacte garde son comportement : aucun chiffre, tout dans l’infobulle', () => {
    const { vue, cellule, chiffre, vignette } = rendre(42, GABARIT_COMPACT)
    expect(chiffre).toBeNull()
    expect(cellule.textContent).toBe('')
    expect(vue.container.children).toHaveLength(1)
    expect(vignette.title).toContain('Charges restantes : 2')
  })
})
