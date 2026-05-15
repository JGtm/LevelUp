/**
 * Tests unitaires — DataFreshnessIndicator.
 *
 * Couvre :
 *  - snapshotAt nil / undefined → composant ne rend rien (null)
 *  - snapshotAt invalide (string non parsable) → ne rend rien
 *  - snapshotAt valide → icône + aria-label avec date formatée locale
 *  - className override → applique les classes custom sur l'icône
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DataFreshnessIndicator } from './data-freshness-indicator'

const buildLabelFR = (date: string) => `Dernière synchronisation réussie le ${date}`

describe('DataFreshnessIndicator', () => {
  it('ne rend rien quand snapshotAt est null', () => {
    const { container } = render(
      <DataFreshnessIndicator snapshotAt={null} buildLabel={buildLabelFR} locale="fr-FR" />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('ne rend rien quand snapshotAt est undefined', () => {
    const { container } = render(
      <DataFreshnessIndicator snapshotAt={undefined} buildLabel={buildLabelFR} locale="fr-FR" />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('ne rend rien quand snapshotAt est une chaîne non parsable', () => {
    const { container } = render(
      <DataFreshnessIndicator snapshotAt="pas une date" buildLabel={buildLabelFR} locale="fr-FR" />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('rend une icône avec aria-label localisé en FR pour un snapshot valide', () => {
    render(
      <DataFreshnessIndicator
        snapshotAt="2026-05-10T14:32:00Z"
        buildLabel={buildLabelFR}
        locale="fr-FR"
      />,
    )
    const indicator = screen.getByTestId('data-freshness-indicator')
    expect(indicator).toBeInTheDocument()
    // Le aria-label doit contenir la phrase + une date formatée FR (jour/mois/année).
    const label = indicator.getAttribute('aria-label') ?? ''
    expect(label).toMatch(/Dernière synchronisation réussie le /)
    expect(label).toMatch(/10\/05\/2026/)
  })

  it('respecte la locale EN pour le format de date', () => {
    render(
      <DataFreshnessIndicator
        snapshotAt="2026-05-10T14:32:00Z"
        buildLabel={(date) => `Last sync ${date}`}
        locale="en-GB"
      />,
    )
    const label = screen.getByTestId('data-freshness-indicator').getAttribute('aria-label') ?? ''
    expect(label).toMatch(/^Last sync /)
    // en-GB → DD/MM/YYYY identique au FR mais avec d'autres séparateurs possibles ; on vérifie juste la date.
    expect(label).toMatch(/10\/05\/2026/)
  })

  it('applique className override sur la zone d\'icône', () => {
    render(
      <DataFreshnessIndicator
        snapshotAt="2026-05-10T14:32:00Z"
        buildLabel={buildLabelFR}
        locale="fr-FR"
        className="text-rose-500 hover:text-rose-300"
      />,
    )
    const indicator = screen.getByTestId('data-freshness-indicator')
    expect(indicator.className).toContain('text-rose-500')
    expect(indicator.className).toContain('hover:text-rose-300')
    // Et NE doit PAS contenir le défaut quand un override est fourni.
    expect(indicator.className).not.toContain('text-muted-foreground/35')
  })
})
