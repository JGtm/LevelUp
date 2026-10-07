/**
 * ResourceControlCard.test.tsx — « Contrôle des ressources » en pleine page (compte · part) et dans
 * la vue compacte du tiroir de comparaison (part entière seule, compte au survol ; maquette Sessions,
 * `makeControl` avec `cp`).
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { ResourceControlCard } from './ResourceControlCard'
import { EMPRISE_TEXT } from './empriseStrings'

const ROWS = [{ resource: 'powerup', us: 12, them: 8, share: 0.6 }]

function segmentTexts() {
  return [...document.querySelectorAll<HTMLElement>('[data-fit-label]')].map((l) => l.textContent)
}

describe('ResourceControlCard', () => {
  it('pleine page : compte · part dans chaque segment', () => {
    render(<ResourceControlCard rows={ROWS} t={EMPRISE_TEXT.fr} />)
    expect(segmentTexts()).toEqual(['12 · 60 %', '40 % · 8'])
  })

  it('compact : la part entière seule', () => {
    render(<ResourceControlCard rows={ROWS} t={EMPRISE_TEXT.fr} compact />)
    expect(segmentTexts()).toEqual(['60 %', '40 %'])
  })
})
