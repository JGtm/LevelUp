/**
 * La colonne « Cartes jouées » de l'écran unique (plan Tactique v2, L4.3).
 *
 * Ce que ces tests cadenassent :
 *   - une vignette par carte OUVRABLE, la plus jouée en tête, la carte active `aria-pressed` ;
 *   - la vignette compacte dit ce qu'elle fait (« Sélectionner <carte> ») et résume la carte
 *     sur une ligne (« 24 · 14 V / 9 D ») ;
 *   - la recherche filtre au fil de la frappe, sans casse ni accents ;
 *   - les cartes sous le plancher vivent dans un repli, « nom » et « 9 sur 10 », résumé
 *     « x sur N » pendant une recherche, ouvert quand seules elles correspondent ;
 *   - un filtre sans carte ouvrable le dit.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { TacticalMapCard } from '@/lib/api/types'

import { getTacticalText } from './i18n'
import { TacticalMapsColumn } from './TacticalMapsColumn'

vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      get: () => Promise.reject(new Error('pas de cadre')),
      getBlob: () => Promise.reject(new Error('pas de fond')),
    },
  }
})

function carte(map_id: string, nom: string, nomFR: string, matchs: number, sous_plancher: boolean): TacticalMapCard {
  return {
    map_id,
    map_name: nom,
    map_name_fr: nomFR,
    matchs,
    victoires: Math.floor(matchs / 2) + 2,
    defaites: Math.floor(matchs / 2) - 3,
    sous_plancher,
  }
}

const CARTES = [
  carte('aquarius', 'Aquarius', 'Aquarius', 9, true),
  carte('lock', 'Lock', 'Écluse', 18, false),
  carte('behemoth', 'Behemoth', 'Béhémoth', 4, true),
  carte('streets', 'Streets', 'Ruelles', 24, false),
]

const t = getTacticalText('fr')
let onSelect = vi.fn()

function rendre(cartes: TacticalMapCard[] = CARTES, carteActive = 'streets') {
  return renderWithProviders(
    <TacticalMapsColumn
      cartes={cartes}
      plancher={10}
      carteActive={carteActive}
      playerSlug="JGtm"
      locale="fr"
      t={t}
      onSelect={onSelect}
      enRelecture={false}
    />,
  )
}

function vignettes(): string[] {
  return screen.queryAllByRole('button', { name: /^Sélectionner / }).map((b) => b.getAttribute('aria-label') ?? '')
}

beforeEach(() => {
  onSelect = vi.fn()
})

describe('TacticalMapsColumn', () => {
  it('une vignette par carte ouvrable, la plus jouée en tête, l’active pressée', () => {
    rendre()
    expect(vignettes()).toEqual(['Sélectionner Ruelles', 'Sélectionner Écluse'])
    expect(screen.getByRole('button', { name: 'Sélectionner Ruelles' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Sélectionner Écluse' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('la vignette résume la carte sur une ligne et sa barre dit le bilan', () => {
    rendre()
    const tuile = screen.getByRole('button', { name: 'Sélectionner Ruelles' })
    expect(within(tuile).getByText('24 · 14 V / 9 D')).toBeInTheDocument()
    expect(within(tuile).getByRole('img', { name: '14 victoires et 9 défaites sur 24 matchs' })).toBeInTheDocument()
  })

  it('le clic sélectionne la carte', () => {
    rendre()
    fireEvent.click(screen.getByRole('button', { name: 'Sélectionner Écluse' }))
    expect(onSelect).toHaveBeenCalledWith('lock')
  })

  it('la recherche filtre au fil de la frappe, sans casse ni accents', () => {
    rendre()
    const champ = screen.getByRole('searchbox', { name: 'Rechercher une carte' })
    expect(champ).toHaveAttribute('placeholder', 'Carte')
    fireEvent.change(champ, { target: { value: 'ECL' } })
    expect(vignettes()).toEqual(['Sélectionner Écluse'])
    fireEvent.change(champ, { target: { value: 'stree' } })
    expect(vignettes()).toEqual(['Sélectionner Ruelles'])
  })

  it('le repli des cartes sous le plancher : nom et « n sur plancher », fermé par défaut', () => {
    rendre()
    const repli = screen.getByTestId('tactical-maps-floor')
    expect(within(repli).getByText('2 cartes sous le plancher')).toBeInTheDocument()
    expect(repli).not.toHaveAttribute('open')
    const ligne = screen.getByTestId('tactical-map-plancher-aquarius')
    expect(ligne).toHaveTextContent('Aquarius')
    expect(ligne).toHaveTextContent('9 sur 10')
  })

  it('seules des cartes sous le plancher correspondent : « x sur N », repli ouvert, liste vide dite', () => {
    rendre()
    fireEvent.change(screen.getByRole('searchbox', { name: 'Rechercher une carte' }), { target: { value: 'béhé' } })
    const repli = screen.getByTestId('tactical-maps-floor')
    expect(within(repli).getByText('1 sur 2 cartes sous le plancher')).toBeInTheDocument()
    expect(repli).toHaveAttribute('open')
    expect(vignettes()).toEqual([])
    expect(screen.getByText('Aucune carte ouvrable ne correspond')).toBeInTheDocument()
  })

  it('aucune carte ouvrable dans le filtre : la colonne le dit, le repli est ouvert', () => {
    rendre([CARTES[0], CARTES[2]], '')
    expect(vignettes()).toEqual([])
    expect(screen.getByText('Aucune carte ouvrable')).toBeInTheDocument()
    expect(screen.getByTestId('tactical-maps-floor')).toHaveAttribute('open')
  })
})
