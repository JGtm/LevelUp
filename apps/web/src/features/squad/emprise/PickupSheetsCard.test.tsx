/**
 * PickupSheetsCard.test.tsx — la carte encadrée (Vue match) et la variante à même la section
 * (Escouade › Emprise, `bare`) : mêmes fiches et même légende, sans cadre ni titre en `bare`.
 */
import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import type { PickupSheets } from './emprise.logic'
import { EMPRISE_TEXT } from './empriseStrings'
import { PickupSheetsCard } from './PickupSheetsCard'

const T = EMPRISE_TEXT.fr
const SHEETS: PickupSheets = { owners: [{ xuid: 'x1', gamertag: 'JGtm' }], sections: [], dominant: [null], losses: null }
const IDS = [{ label: 'JGtm', color: 'var(--ac-squad-player-1)' }]

describe('PickupSheetsCard', () => {
  it('encadrée : titre de la carte, légende des deux marques', () => {
    render(<PickupSheetsCard sheets={SHEETS} identities={IDS} itemName={() => ''} t={T} />)
    expect(screen.getByText(T.sheets.title)).toBeInTheDocument()
    expect(screen.getByText(T.sheets.legendTaken)).toBeInTheDocument()
    expect(screen.getByText(T.sheets.legendLost)).toBeInTheDocument()
  })

  it('`bare` : ni titre ni cadre (le titre est celui de la section), la légende reste sous les fiches', () => {
    render(<PickupSheetsCard sheets={SHEETS} identities={IDS} itemName={() => ''} bare t={T} />)
    expect(screen.queryByText(T.sheets.title)).toBeNull()
    const card = screen.getByTestId('emprise-sheets')
    expect(card.className).not.toContain('border')
    expect(within(card).getByTestId('objectif-legend').textContent).toContain(T.sheets.legendTaken)
  })
})
