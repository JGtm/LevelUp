/**
 * SessionUsageSection.gate.test.tsx — LES DEUX PORTES DU BLOC « usages d'équipement,
 * socles et objectifs » (règle du 2026-09-05, registre L4).
 *
 * Le bloc affichait, sur un titre sans décodeur de film, une carte « Ce titre ne publie pas
 * de résumé d'usage des films » : `unsupported` était traité comme `empty` au lieu de
 * `hidden`. Une carte qui annonce une absence définitive occupe une place, ne dit rien
 * d'actionnable et ne disparaîtra jamais — un bloc mort.
 *
 * Ce fichier fixe la distinction, sur le COMPOSANT (usageLogic.test.ts la fixe sur la
 * fonction) :
 *   - `unsupported` (le TITRE ne le publie pas)   → RIEN ;
 *   - `load_failed` (CETTE lecture a échoué)      → carte d'état vide AVEC la raison ;
 *   - 0 match mesuré (le titre publie, pas ici)   → carte d'état vide « aucun film ».
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SessionUsageBlock } from '@/lib/api/types'

import { SessionUsageSection } from './SessionUsageSection'

const BASE: SessionUsageBlock = { available: true, matches_measured: 4, matches_total: 6 }

describe('SessionUsageSection — porte de titre', () => {
  it("titre qui ne publie pas de résumé d'usage (unsupported) : RIEN n'est rendu", () => {
    const { container } = render(
      <SessionUsageSection
        usage={{ ...BASE, available: false, unavailable_reason: 'unsupported' }}
        meLabel="moi"
      />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('bloc absent du payload (vieux serveur) : RIEN non plus', () => {
    const { container } = render(<SessionUsageSection usage={undefined} meLabel="moi" />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('SessionUsageSection — porte de donnée (le titre publie)', () => {
  it('lecture échouée : la carte dit la raison, elle est transitoire', () => {
    render(
      <SessionUsageSection
        usage={{ ...BASE, available: false, unavailable_reason: 'load_failed' }}
        meLabel="moi"
      />,
    )
    expect(screen.getByText(/La lecture du résumé d'usage a échoué/)).toBeInTheDocument()
  })

  it('aucun match mesuré : la carte le dit, avec le dénominateur', () => {
    render(<SessionUsageSection usage={{ ...BASE, matches_measured: 0 }} meLabel="moi" />)
    expect(screen.getByText(/Aucun match de cette session n'a de film mesuré/)).toBeInTheDocument()
  })
})
