/**
 * WeaponRecordsGapCard — « Records de distance par arme » sans frag mesuré : la carte garde son titre et
 * dit pourquoi, dans son bloc (convention de l'application : bloc parent + état dedans).
 */
import { afterEach, describe, expect, it } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { WeaponRecordsGapCard } from './WeaponRecordsGapCard'

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('WeaponRecordsGapCard', () => {
  it('analyse en cours : le titre de la carte et l’indicateur d’attente', () => {
    renderWithProviders(<WeaponRecordsGapCard gap="pending" />)
    const bloc = screen.getByTestId('weapon-records-gap')
    expect(within(bloc).getByTestId('analysis-pending-slot').textContent).toContain('Analyse du film en cours')
    expect(screen.getAllByText('Records de distance par arme').length).toBeGreaterThan(0)
  })

  it('film non analysé : le pointillé est dans le bloc, sans action ni mot interne', () => {
    renderWithProviders(<WeaponRecordsGapCard gap="missing" />)
    const bloc = screen.getByTestId('weapon-records-gap')
    expect(bloc.textContent).toContain('Film non analysé')
    expect(bloc.querySelector('.border-dashed')).not.toBeNull()
    expect(bloc.textContent).not.toMatch(/réanalys|relanc|cuisson|artefact/i)
  })

  it('anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<WeaponRecordsGapCard gap="missing" />)
    expect(screen.getByTestId('weapon-records-gap').textContent).toContain('Film not analysed')
  })
})
