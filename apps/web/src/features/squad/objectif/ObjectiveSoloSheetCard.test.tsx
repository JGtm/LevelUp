/**
 * ObjectiveSoloSheetCard.test.tsx — « Ma part à l'objectif » : une seule fiche (le joueur affiché),
 * sa barre = sa part de son camp, un zéro atténué sans barre, l'emblème ou l'initiale, le rôle
 * dominant et le pied de fiche.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { FORMES_TEXT } from '../formes/i18n'
import { block2209 } from './objectif.fixtures'
import { buildSoloObjectiveSheet } from './objectif.logic'
import { ObjectiveSoloSheetCard, type SoloSheetText } from './ObjectiveSoloSheetCard'

const T: SoloSheetText = {
  title: 'Ma part à l’objectif',
  info: 'La fiche du joueur affiché.',
  dominantRole: 'Rôle dominant',
  roles: { take: 'Prendre', defend: 'Défendre', hold: 'Tenir' },
  pctFmt: (v) => `${v.toFixed(1)} %`,
  durationFmt: (s) => `${Math.round(s)} s`,
  lineTip: (player, family, column, value, camp, pct) => `${player} · ${family}\n${column} : ${value} des ${camp}${pct ? ` (${pct})` : ''}`,
}

const sheet = buildSoloObjectiveSheet({ ...block2209(), squad: [{ xuid: 'xj', gamertag: 'JGtm' }] })!

function renderCard(emblemUrl?: string) {
  render(
    <ObjectiveSoloSheetCard
      sheet={sheet}
      name="JGtm"
      emblemUrl={emblemUrl}
      familyLabel={(f) => (f === 'ctf' ? 'Drapeau' : f)}
      columns={FORMES_TEXT.fr.columns}
      t={T}
    />,
  )
}

describe('ObjectiveSoloSheetCard', () => {
  it('une seule fiche, au nom du joueur, sans « reste du camp »', () => {
    renderCard()
    expect(screen.getByTestId('objective-solo-sheet-xj')).toBeTruthy()
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.queryByText(/reste/i)).toBeNull()
  })

  it('la barre d’une action = ma part de mon camp (3 vols sur 14)', () => {
    renderCard()
    const bar = screen.getByTestId('objective-solo-bar-flag_steals') as HTMLElement
    expect(parseFloat(bar.style.width)).toBeCloseTo((3 / 14) * 100)
    expect((screen.getByTestId('objective-solo-bar-flag_captures') as HTMLElement).style.width).toBe('100%')
  })

  it('un zéro reste une ligne atténuée, sans barre', () => {
    renderCard()
    expect(screen.getByTestId('objective-solo-line-flag_capture_assists').dataset.zero).toBe('true')
    expect(screen.queryByTestId('objective-solo-bar-flag_capture_assists')).toBeNull()
  })

  it('l’infobulle dit la valeur et le total de mon camp', () => {
    renderCard()
    fireEvent.mouseEnter(screen.getByTestId('objective-solo-line-flag_steals').parentElement!)
    expect(screen.getByRole('tooltip').textContent).toContain('3 des 14')
  })

  it('emblème servi : l’image ; sans emblème : l’initiale', () => {
    renderCard('https://exemple.test/emblem.png')
    expect(screen.getByRole('img', { name: 'JGtm' }).getAttribute('src')).toBe('https://exemple.test/emblem.png')
  })

  it('sans emblème, l’initiale du joueur', () => {
    renderCard()
    expect(screen.queryByRole('img', { name: 'JGtm' })).toBeNull()
    expect(screen.getByText('J')).toBeTruthy()
  })

  it('rôle dominant et pied de fiche Prendre / Défendre / Tenir', () => {
    renderCard()
    expect(screen.getAllByText('Tenir').length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText('Rôle dominant')).toBeTruthy()
    // La ligne « temps de porteur » et le pied « Tenir » portent les mêmes 64,9 s.
    expect(screen.getAllByText('65 s')).toHaveLength(2)
  })
})
