/**
 * PickupSheetsCard.test.tsx — l'entrée « reste » de la légende ne se pose qu'avec son encre : la Vue
 * match n'a pas de fiche du reste de l'équipe et ne passe pas `restColor`.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { PickupSheets } from './emprise.logic'
import { EMPRISE_TEXT } from './empriseStrings'
import { PickupSheetsCard } from './PickupSheetsCard'

const T = EMPRISE_TEXT.fr
const SHEETS: PickupSheets = { owners: [{ xuid: 'x1', gamertag: 'JGtm' }], sections: [], dominant: [null], losses: null }
const IDS = [{ label: 'JGtm', color: 'var(--ac-squad-player-1)' }]

describe('PickupSheetsCard — `restColor` optionnel', () => {
  it('avec `restColor` : la légende nomme le reste de l’équipe', () => {
    render(<PickupSheetsCard sheets={SHEETS} identities={IDS} itemName={() => ''} restColor="var(--ac-muted)" t={T} />)
    expect(screen.getByText(T.sheets.legendRest)).toBeInTheDocument()
  })

  it('sans `restColor` : aucune entrée « reste », les deux marques gardées', () => {
    render(<PickupSheetsCard sheets={SHEETS} identities={IDS} itemName={() => ''} t={T} />)
    expect(screen.queryByText(T.sheets.legendRest)).toBeNull()
    expect(screen.getByText(T.sheets.legendTaken)).toBeInTheDocument()
    expect(screen.getByText(T.sheets.legendLost)).toBeInTheDocument()
  })
})