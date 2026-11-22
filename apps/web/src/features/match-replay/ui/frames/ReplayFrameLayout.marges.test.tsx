/**
 * Tests — LES MARGES DU CORPS, UNE COTE PAR CADRE (`model/cotesCorps.ts`) : le Cinéma pose 24 px de chaque côté du
 * corps et 8 px dessous ; le Plein cadre, dont le cadre d'écran pose déjà 12 px, ajoute 4 px de marge propre
 * (16 px entre le bord de l'écran et le contenu) et descend le bas du cadre à 6 px. Le corps, l'en-tête et la règle
 * de tenue de la rangée lisent le même tableau.
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { COTES_CORPS_CADRE, margeCorpsDe } from '../../model/cotesCorps'
import { ReplayFrameLayout } from './ReplayFrameLayout'

function mountLayout(frame: 'cinema' | 'plein-cadre') {
  return render(
    <ReplayFrameLayout
      frame={frame}
      tete={<header />}
      avis={null}
      rangee={<div data-testid="rangee" />}
      scene={<div />}
      colonne={<div />}
    />,
  )
}

describe('les marges du corps par cadre', () => {
  it('Cinéma : 24 px de chaque côté du corps, 8 px dessous', () => {
    const { container } = mountLayout('cinema')
    const corps = container.querySelector<HTMLElement>('[data-replay-body="cinema"]')!
    expect(corps.style.paddingInline).toBe('24px')
    expect(corps.style.paddingBottom).toBe('8px')
  })

  it('Plein cadre : 4 px propres au corps, qui s’ajoutent aux 12 px du cadre d’écran (16 px au bord) ; bas du cadre à 6 px', () => {
    const { container } = mountLayout('plein-cadre')
    const racine = container.querySelector<HTMLElement>('[data-replay-frame="plein-cadre"]')!
    const corps = container.querySelector<HTMLElement>('[data-replay-body="plein-cadre"]')!
    expect(corps.style.paddingInline).toBe('4px')
    expect(racine.classList.contains('p-3')).toBe(true)
    expect(racine.style.paddingBottom).toBe('6px')
    // 12 px de cadre + 4 px de corps : la marge latérale de 16 px.
    expect(12 + COTES_CORPS_CADRE['plein-cadre'].margeLateralePx).toBe(16)
  })

  it('la marge que lisent les règles de largeur est celle que le corps pose', () => {
    expect(margeCorpsDe('cinema')).toBe(COTES_CORPS_CADRE.cinema.margeLateralePx)
    expect(margeCorpsDe('plein-cadre')).toBe(COTES_CORPS_CADRE['plein-cadre'].margeLateralePx)
    const { container } = mountLayout('plein-cadre')
    expect(container.querySelector<HTMLElement>('[data-replay-body]')!.style.paddingInline).toBe(`${margeCorpsDe('plein-cadre')}px`)
  })
})
